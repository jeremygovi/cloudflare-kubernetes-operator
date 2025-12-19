/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/zones"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudflarev1 "github.com/jeremygovi/cloudflare-kubernetes-operator/api/v1"
)

const (
	cloudflareRecordFinalizer = "cloudflare.io/record-finalizer"
	conditionTypeReady        = "Ready"
	conditionTypeError        = "Error"
)

// CloudflareRecordReconciler reconciles a CloudflareRecord object
type CloudflareRecordReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CloudflareAPI   *cloudflare.Client
	RequeueDuration time.Duration
}

// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarerecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarerecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarerecords/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CloudflareRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cloudflarerecord", req.NamespacedName)

	// Fetch the CloudflareRecord instance
	record := &cloudflarev1.CloudflareRecord{}
	if err := r.Get(ctx, req.NamespacedName, record); err != nil {
		if errors.IsNotFound(err) {
			// Resource already deleted, nothing to do
			log.V(1).Info("CloudflareRecord resource not found. Ignoring because object is already deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get CloudflareRecord")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !record.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, record)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(record, cloudflareRecordFinalizer) {
		log.V(1).Info("Adding finalizer to CloudflareRecord")
		controllerutil.AddFinalizer(record, cloudflareRecordFinalizer)
		if err := r.Update(ctx, record); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Skip if already reconciled (generation check)
	if record.Status.ObservedGeneration == record.Generation && record.Status.State == cloudflarev1.RecordStateActive {
		log.V(1).Info("Record already reconciled, skipping")
		return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
	}

	// Reconcile the DNS record
	return r.reconcileDNSRecord(ctx, record)
}

// reconcileDNSRecord creates or updates the DNS record in Cloudflare
func (r *CloudflareRecordReconciler) reconcileDNSRecord(ctx context.Context, record *cloudflarev1.CloudflareRecord) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Set initial pending state
	if record.Status.State == "" {
		record.Status.State = cloudflarev1.RecordStatePending
		if err := r.Status().Update(ctx, record); err != nil {
			log.Error(err, "Failed to update status to Pending")
			return ctrl.Result{}, err
		}
	}

	// Resolve ZoneID from domain if not already resolved
	zoneID := record.Status.ZoneID
	if zoneID == "" {
		log.V(1).Info("Resolving ZoneID for domain", "domain", record.Spec.Domain)
		resolvedZoneID, err := r.resolveZoneID(ctx, record.Spec.Domain)
		if err != nil {
			return r.handleReconcileError(ctx, record, fmt.Errorf("failed to resolve zone ID for domain %s: %w", record.Spec.Domain, err))
		}
		zoneID = resolvedZoneID
		record.Status.ZoneID = zoneID
		log.Info("Resolved ZoneID", "domain", record.Spec.Domain, "zoneID", zoneID)
	}

	// Build full DNS name
	fullName := r.buildFullDNSName(record.Spec.Name, record.Spec.Domain)

	var recordID string

	if record.Status.RecordID != "" {
		// Update existing record
		log.Info("Updating existing DNS record", "recordID", record.Status.RecordID, "zoneID", zoneID)

		bodyNew, err := r.buildDNSBody(record, fullName)
		if err != nil {
			return r.handleReconcileError(ctx, record, err)
		}
		// buildDNSBody returns a RecordNewParamsBodyUnion; assert underlying value implements the Update union
		bodyUpdate, ok := bodyNew.(dns.RecordUpdateParamsBodyUnion)
		if !ok {
			return r.handleReconcileError(ctx, record, fmt.Errorf("internal: DNS body type mismatch for update"))
		}

		_, err = r.CloudflareAPI.DNS.Records.Update(ctx, record.Status.RecordID, dns.RecordUpdateParams{
			ZoneID: cloudflare.F(zoneID),
			Body:   bodyUpdate,
		})
		if err != nil {
			return r.handleReconcileError(ctx, record, fmt.Errorf("failed to update DNS record: %w", err))
		}
		recordID = record.Status.RecordID
	} else {
		// Create new record
		log.Info("Creating new DNS record", "name", record.Spec.Name, "type", record.Spec.Type, "zoneID", zoneID)

		body, err := r.buildDNSBody(record, fullName)
		if err != nil {
			return r.handleReconcileError(ctx, record, err)
		}

		response, err := r.CloudflareAPI.DNS.Records.New(ctx, dns.RecordNewParams{
			ZoneID: cloudflare.F(zoneID),
			Body:   body,
		})
		if err != nil {
			return r.handleReconcileError(ctx, record, fmt.Errorf("failed to create DNS record: %w", err))
		}
		recordID = response.ID
	}

	// Update status to Active
	now := metav1.NewTime(time.Now())
	record.Status.RecordID = recordID
	record.Status.State = cloudflarev1.RecordStateActive
	record.Status.Message = "DNS record synchronized successfully"
	record.Status.LastSync = &now
	record.Status.ObservedGeneration = record.Generation

	// Update condition
	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: record.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationSucceeded",
		Message:            "DNS record successfully synchronized with Cloudflare",
	})
	meta.RemoveStatusCondition(&record.Status.Conditions, conditionTypeError)

	if err := r.Status().Update(ctx, record); err != nil {
		log.Error(err, "Failed to update CloudflareRecord status")
		return ctrl.Result{}, err
	}

	log.Info("DNS record synchronized", "recordID", recordID)
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// handleDeletion handles the deletion of a CloudflareRecord
func (r *CloudflareRecordReconciler) handleDeletion(ctx context.Context, record *cloudflarev1.CloudflareRecord) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(record, cloudflareRecordFinalizer) {
		return ctrl.Result{}, nil
	}

	// Delete the DNS record from Cloudflare if it exists
	if record.Status.RecordID != "" && record.Status.ZoneID != "" {
		log.Info("Deleting DNS record", "domain", record.Spec.Domain, "name", record.Spec.Name)

		_, err := r.CloudflareAPI.DNS.Records.Delete(ctx, record.Status.RecordID, dns.RecordDeleteParams{
			ZoneID: cloudflare.F(record.Status.ZoneID),
		})
		if err != nil {
			// Check if record already deleted (404)
			if !isNotFoundError(err) {
				log.Error(err, "Failed to delete DNS record from Cloudflare")
				return ctrl.Result{}, err
			}
			log.Info("DNS record not found in Cloudflare, assuming already deleted")
		} else {
			log.Info("Successfully deleted DNS record from Cloudflare")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(record, cloudflareRecordFinalizer)
	if err := r.Update(ctx, record); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Finalizer removed, resource will be deleted")
	return ctrl.Result{}, nil
}

// handleReconcileError updates the status when reconciliation fails
func (r *CloudflareRecordReconciler) handleReconcileError(ctx context.Context, record *cloudflarev1.CloudflareRecord, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "Reconciliation failed")

	now := metav1.NewTime(time.Now())
	record.Status.State = cloudflarev1.RecordStateError
	record.Status.Message = err.Error()
	record.Status.ObservedGeneration = record.Generation

	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionTypeError,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: record.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})
	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: record.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})

	if statusErr := r.Status().Update(ctx, record); statusErr != nil {
		log.Error(statusErr, "Failed to update status")
		return ctrl.Result{}, statusErr
	}

	// Requeue with backoff
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.RequeueDuration == 0 {
		r.RequeueDuration = 5 * time.Minute
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1.CloudflareRecord{}).
		Complete(r)
}

// resolveZoneID looks up the Cloudflare Zone ID for a given domain
func (r *CloudflareRecordReconciler) resolveZoneID(ctx context.Context, domain string) (string, error) {
	// List zones and find the one matching the domain
	iter := r.CloudflareAPI.Zones.ListAutoPaging(ctx, zones.ZoneListParams{
		Name: cloudflare.F(domain),
	})

	for iter.Next() {
		zone := iter.Current()
		if zone.Name == domain {
			return zone.ID, nil
		}
	}

	if err := iter.Err(); err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	return "", fmt.Errorf("no zone found for domain %s", domain)
}

// buildFullDNSName constructs the full DNS record name from the subdomain and domain
func (r *CloudflareRecordReconciler) buildFullDNSName(name, domain string) string {
	// Handle special cases
	if name == "" || name == "@" {
		return domain
	}
	return name + "." + domain
}

// buildDNSBody construit le body adapté pour la création/mise à jour d'un enregistrement DNS
func (r *CloudflareRecordReconciler) buildDNSBody(record *cloudflarev1.CloudflareRecord, fullName string) (dns.RecordNewParamsBodyUnion, error) {
	switch record.Spec.Type {
	case "A":
		return dns.ARecordParam{
			Name:    cloudflare.F(fullName),
			Type:    cloudflare.F(dns.ARecordTypeA),
			Content: cloudflare.F(record.Spec.Content),
			TTL:     cloudflare.F(dns.TTL(intPtrToInt(record.Spec.TTL, 1))),
			Proxied: cloudflare.F(*boolPtrToBool(record.Spec.Proxied, false)),
			Comment: cloudflare.F(record.Spec.Comment),
		}, nil
	case "AAAA":
		return dns.AAAARecordParam{
			Name:    cloudflare.F(fullName),
			Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
			Content: cloudflare.F(record.Spec.Content),
			TTL:     cloudflare.F(dns.TTL(intPtrToInt(record.Spec.TTL, 1))),
			Proxied: cloudflare.F(*boolPtrToBool(record.Spec.Proxied, false)),
			Comment: cloudflare.F(record.Spec.Comment),
		}, nil
	case "CNAME":
		return dns.CNAMERecordParam{
			Name:    cloudflare.F(fullName),
			Type:    cloudflare.F(dns.CNAMERecordTypeCNAME),
			Content: cloudflare.F(record.Spec.Content),
			TTL:     cloudflare.F(dns.TTL(intPtrToInt(record.Spec.TTL, 1))),
			Proxied: cloudflare.F(*boolPtrToBool(record.Spec.Proxied, false)),
			Comment: cloudflare.F(record.Spec.Comment),
		}, nil
	case "TXT":
		return dns.TXTRecordParam{
			Name:    cloudflare.F(fullName),
			Type:    cloudflare.F(dns.TXTRecordTypeTXT),
			Content: cloudflare.F(record.Spec.Content),
			TTL:     cloudflare.F(dns.TTL(intPtrToInt(record.Spec.TTL, 1))),
			Comment: cloudflare.F(record.Spec.Comment),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported DNS record type: %s", record.Spec.Type)
	}
}
