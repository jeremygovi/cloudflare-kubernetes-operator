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
	cloudflareZoneFinalizer = "cloudflare.io/zone-finalizer"
)

// CloudflareZoneReconciler reconciles a CloudflareZone object
type CloudflareZoneReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CloudflareAPI   *cloudflare.API
	RequeueDuration time.Duration
	AccountID       string // Default account ID from environment variable
}

// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarezones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarezones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflarezones/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CloudflareZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cloudflarezone", req.NamespacedName)

	// Fetch the CloudflareZone instance
	zone := &cloudflarev1.CloudflareZone{}
	if err := r.Get(ctx, req.NamespacedName, zone); err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("CloudflareZone resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get CloudflareZone")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !zone.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, zone)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(zone, cloudflareZoneFinalizer) {
		log.V(1).Info("Adding finalizer to CloudflareZone")
		controllerutil.AddFinalizer(zone, cloudflareZoneFinalizer)
		if err := r.Update(ctx, zone); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Skip if already reconciled (generation check)
	if zone.Status.ObservedGeneration == zone.Generation &&
		(zone.Status.State == cloudflarev1.ZoneStateActive || zone.Status.State == cloudflarev1.ZoneStatePaused) {
		log.V(1).Info("Zone already reconciled, skipping")
		return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
	}

	// Reconcile the zone
	return r.reconcileZone(ctx, zone)
}

// reconcileZone creates or updates the zone in Cloudflare
func (r *CloudflareZoneReconciler) reconcileZone(ctx context.Context, zone *cloudflarev1.CloudflareZone) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Set initial pending state
	if zone.Status.State == "" {
		zone.Status.State = cloudflarev1.ZoneStatePending
		if err := r.Status().Update(ctx, zone); err != nil {
			log.Error(err, "Failed to update status to Pending")
			return ctrl.Result{}, err
		}
	}

	var zoneID string
	var cfZone cloudflare.Zone
	var err error

	if zone.Status.ZoneID != "" {
		// Check if zone exists and update if needed
		log.V(1).Info("Checking existing zone", "zoneID", zone.Status.ZoneID)

		cfZone, err = r.CloudflareAPI.ZoneDetails(ctx, zone.Status.ZoneID)
		if err != nil {
			if isNotFoundError(err) {
				// Zone doesn't exist in Cloudflare anymore, need to recreate
				log.Info("Zone not found in Cloudflare, will recreate")
				zone.Status.ZoneID = ""
			} else {
				return r.handleReconcileError(ctx, zone, fmt.Errorf("failed to get zone details: %w", err))
			}
		} else {
			// Update zone settings if needed
			log.V(1).Info("Updating zone settings", "zoneID", zone.Status.ZoneID)

			paused := boolPtrToBool(zone.Spec.Paused, false)
			if cfZone.Paused != *paused {
				zoneOptions := cloudflare.ZoneOptions{
					Paused: paused,
				}
				cfZone, err = r.CloudflareAPI.EditZone(ctx, zone.Status.ZoneID, zoneOptions)
				if err != nil {
					return r.handleReconcileError(ctx, zone, fmt.Errorf("failed to update zone: %w", err))
				}
			}
			zoneID = zone.Status.ZoneID
		}
	}

	// Create new zone if needed
	if zone.Status.ZoneID == "" {
		// Determine account ID: use spec value or fall back to environment variable
		accountID := zone.Spec.AccountID
		if accountID == "" {
			accountID = r.AccountID
			if accountID == "" {
				return r.handleReconcileError(ctx, zone, fmt.Errorf("accountId must be specified in spec or CLOUDFLARE_ACCOUNT_ID environment variable must be set"))
			}
			log.V(1).Info("Using account ID from environment variable", "accountID", accountID)
		}

		log.Info("Creating new zone", "name", zone.Spec.Name, "accountID", accountID)

		jumpStart := boolPtrToBool(zone.Spec.JumpStart, false)
		zoneType := string(zone.Spec.Type)
		if zoneType == "" {
			zoneType = string(cloudflarev1.ZoneTypeFull)
		}

		account := cloudflare.Account{ID: accountID}
		cfZone, err = r.CloudflareAPI.CreateZone(ctx, zone.Spec.Name, *jumpStart, account, zoneType)
		if err != nil {
			return r.handleReconcileError(ctx, zone, fmt.Errorf("failed to create zone: %w", err))
		}
		zoneID = cfZone.ID

		// Apply paused setting if specified
		if zone.Spec.Paused != nil && *zone.Spec.Paused {
			zoneOptions := cloudflare.ZoneOptions{
				Paused: zone.Spec.Paused,
			}
			cfZone, err = r.CloudflareAPI.EditZone(ctx, zoneID, zoneOptions)
			if err != nil {
				log.Error(err, "Failed to pause zone, but zone was created")
			}
		}
	}

	// Update status
	now := metav1.NewTime(time.Now())
	zone.Status.ZoneID = zoneID
	zone.Status.Status = cfZone.Status
	zone.Status.NameServers = cfZone.NameServers
	zone.Status.OriginalNameServers = cfZone.OriginalNS
	zone.Status.VerificationKey = cfZone.VerificationKey
	zone.Status.ObservedGeneration = zone.Generation
	zone.Status.LastSync = &now

	// Set state based on zone status and paused flag
	if cfZone.Paused {
		zone.Status.State = cloudflarev1.ZoneStatePaused
		zone.Status.Message = "Zone is paused"
	} else if cfZone.Status == "active" {
		zone.Status.State = cloudflarev1.ZoneStateActive
		zone.Status.Message = "Zone synchronized successfully"
	} else {
		zone.Status.State = cloudflarev1.ZoneStatePending
		zone.Status.Message = fmt.Sprintf("Zone status: %s", cfZone.Status)
	}

	// Update condition
	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: zone.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationSucceeded",
		Message:            "Zone successfully synchronized with Cloudflare",
	})
	meta.RemoveStatusCondition(&zone.Status.Conditions, conditionTypeError)

	if err := r.Status().Update(ctx, zone); err != nil {
		log.Error(err, "Failed to update CloudflareZone status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled zone", "zoneID", zoneID, "status", cfZone.Status)
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// handleDeletion handles the deletion of a CloudflareZone
func (r *CloudflareZoneReconciler) handleDeletion(ctx context.Context, zone *cloudflarev1.CloudflareZone) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(zone, cloudflareZoneFinalizer) {
		return ctrl.Result{}, nil
	}

	// Delete the zone from Cloudflare if it exists
	if zone.Status.ZoneID != "" {
		log.Info("Deleting zone", "name", zone.Spec.Name)

		_, err := r.CloudflareAPI.DeleteZone(ctx, zone.Status.ZoneID)
		if err != nil {
			// Check if zone already deleted (404)
			if !isNotFoundError(err) {
				log.Error(err, "Failed to delete zone from Cloudflare")
				return ctrl.Result{}, err
			}
			log.V(1).Info("Zone not found in Cloudflare, assuming already deleted")
		} else {
			log.Info("Successfully deleted zone from Cloudflare")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(zone, cloudflareZoneFinalizer)
	if err := r.Update(ctx, zone); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Finalizer removed, resource will be deleted")
	return ctrl.Result{}, nil
}

// handleReconcileError updates the status when reconciliation fails
func (r *CloudflareZoneReconciler) handleReconcileError(ctx context.Context, zone *cloudflarev1.CloudflareZone, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "Reconciliation failed")

	now := metav1.NewTime(time.Now())
	zone.Status.State = cloudflarev1.ZoneStateError
	zone.Status.Message = err.Error()
	zone.Status.ObservedGeneration = zone.Generation

	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionTypeError,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: zone.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})
	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: zone.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})

	if statusErr := r.Status().Update(ctx, zone); statusErr != nil {
		log.Error(statusErr, "Failed to update status")
		return ctrl.Result{}, statusErr
	}

	// Requeue with backoff
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.RequeueDuration == 0 {
		r.RequeueDuration = 5 * time.Minute
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1.CloudflareZone{}).
		Complete(r)
}
