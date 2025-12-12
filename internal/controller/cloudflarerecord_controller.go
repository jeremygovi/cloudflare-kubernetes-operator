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
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
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
	CloudflareAPI   *cloudflare.API
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
			log.Info("CloudflareRecord resource not found. Ignoring since object must be deleted")
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
		log.Info("Adding finalizer to CloudflareRecord")
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

	// Prepare DNS record parameters
	zoneID := record.Spec.ZoneID
	recordParams := cloudflare.CreateDNSRecordParams{
		Name:     record.Spec.Name,
		Type:     string(record.Spec.Type),
		Content:  record.Spec.Content,
		TTL:      intPtrToInt(record.Spec.TTL, 1),
		Proxied:  boolPtrToBool(record.Spec.Proxied, false),
		Priority: uint16PtrToPtr(record.Spec.Priority),
		Comment:  record.Spec.Comment,
	}

	var recordID string
	var err error

	if record.Status.RecordID != "" {
		// Update existing record
		log.Info("Updating existing DNS record", "recordID", record.Status.RecordID, "zoneID", zoneID)
		
		updateParams := cloudflare.UpdateDNSRecordParams{
			ID:       record.Status.RecordID,
			Name:     recordParams.Name,
			Type:     recordParams.Type,
			Content:  recordParams.Content,
			TTL:      recordParams.TTL,
			Proxied:  recordParams.Proxied,
			Priority: recordParams.Priority,
			Comment:  stringToPtr(recordParams.Comment),
		}

		_, err = r.CloudflareAPI.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), updateParams)
		if err != nil {
			return r.handleReconcileError(ctx, record, fmt.Errorf("failed to update DNS record: %w", err))
		}
		recordID = record.Status.RecordID
	} else {
		// Create new record
		log.Info("Creating new DNS record", "name", record.Spec.Name, "type", record.Spec.Type, "zoneID", zoneID)
		
		response, err := r.CloudflareAPI.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), recordParams)
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

	log.Info("Successfully reconciled DNS record", "recordID", recordID)
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// handleDeletion handles the deletion of a CloudflareRecord
func (r *CloudflareRecordReconciler) handleDeletion(ctx context.Context, record *cloudflarev1.CloudflareRecord) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(record, cloudflareRecordFinalizer) {
		return ctrl.Result{}, nil
	}

	// Delete the DNS record from Cloudflare if it exists
	if record.Status.RecordID != "" {
		log.Info("Deleting DNS record from Cloudflare", "recordID", record.Status.RecordID)
		
		err := r.CloudflareAPI.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(record.Spec.ZoneID), record.Status.RecordID)
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

	log.Info("Finalizer removed, resource will be deleted")
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

// Helper functions

func intPtrToInt(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func boolPtrToBool(ptr *bool, defaultVal bool) *bool {
	if ptr != nil {
		return ptr
	}
	return &defaultVal
}

func uint16PtrToPtr(ptr *uint16) *uint16 {
	return ptr
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "not found" or similar indicators
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "not found") || 
	       strings.Contains(errorMsg, "404") ||
	       strings.Contains(errorMsg, "could not be found")
}
