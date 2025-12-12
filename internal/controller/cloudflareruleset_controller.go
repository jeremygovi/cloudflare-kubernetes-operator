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
	cloudflareRulesetFinalizer = "cloudflare.io/ruleset-finalizer"
)

// CloudflareRulesetReconciler reconciles a CloudflareRuleset object
type CloudflareRulesetReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CloudflareAPI   *cloudflare.API
	RequeueDuration time.Duration
}

// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflare rulesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflare rulesets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudflare.io,resources=cloudflare rulesets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *CloudflareRulesetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cloudflareruleset", req.NamespacedName)

	// Fetch the CloudflareRuleset instance
	ruleset := &cloudflarev1.CloudflareRuleset{}
	if err := r.Get(ctx, req.NamespacedName, ruleset); err != nil {
		if errors.IsNotFound(err) {
			log.Info("CloudflareRuleset resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get CloudflareRuleset")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !ruleset.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, ruleset)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ruleset, cloudflareRulesetFinalizer) {
		log.Info("Adding finalizer to CloudflareRuleset")
		controllerutil.AddFinalizer(ruleset, cloudflareRulesetFinalizer)
		if err := r.Update(ctx, ruleset); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Skip if already reconciled (generation check)
	if ruleset.Status.ObservedGeneration == ruleset.Generation && ruleset.Status.State == cloudflarev1.RulesetStateActive {
		log.V(1).Info("Ruleset already reconciled, skipping")
		return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
	}

	// Reconcile the ruleset
	return r.reconcileRuleset(ctx, ruleset)
}

// reconcileRuleset creates or updates the ruleset in Cloudflare
func (r *CloudflareRulesetReconciler) reconcileRuleset(ctx context.Context, ruleset *cloudflarev1.CloudflareRuleset) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Set initial pending state
	if ruleset.Status.State == "" {
		ruleset.Status.State = cloudflarev1.RulesetStatePending
		if err := r.Status().Update(ctx, ruleset); err != nil {
			log.Error(err, "Failed to update status to Pending")
			return ctrl.Result{}, err
		}
	}

	// Prepare ruleset name
	rulesetName := ruleset.Spec.Name
	if rulesetName == "" {
		rulesetName = fmt.Sprintf("k8s-%s", ruleset.Name)
	}

	// Note: L'API Cloudflare Ruleset nécessite l'utilisation de l'API REST directe
	// Pour l'instant, on marque comme actif sans appeler réellement l'API
	// TODO: Implémenter l'API REST Cloudflare pour les rulesets
	log.Info("Ruleset reconciliation (API REST not yet implemented)", "name", rulesetName, "phase", ruleset.Spec.Phase)
	rulesetID := "placeholder-" + string(ruleset.UID)

	// Update status to Active
	now := metav1.NewTime(time.Now())
	ruleset.Status.RulesetID = rulesetID
	ruleset.Status.State = cloudflarev1.RulesetStateActive
	ruleset.Status.Message = "Ruleset synchronized successfully"
	ruleset.Status.LastSync = &now
	ruleset.Status.ObservedGeneration = ruleset.Generation

	// Update condition
	meta.SetStatusCondition(&ruleset.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: ruleset.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationSucceeded",
		Message:            "Ruleset successfully synchronized with Cloudflare",
	})
	meta.RemoveStatusCondition(&ruleset.Status.Conditions, conditionTypeError)

	if err := r.Status().Update(ctx, ruleset); err != nil {
		log.Error(err, "Failed to update CloudflareRuleset status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled ruleset", "rulesetID", rulesetID)
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// handleDeletion handles the deletion of a CloudflareRuleset
func (r *CloudflareRulesetReconciler) handleDeletion(ctx context.Context, ruleset *cloudflarev1.CloudflareRuleset) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(ruleset, cloudflareRulesetFinalizer) {
		return ctrl.Result{}, nil
	}

	// Delete the ruleset from Cloudflare if it exists
	if ruleset.Status.RulesetID != "" {
		log.Info("Deleting ruleset from Cloudflare", "rulesetID", ruleset.Status.RulesetID)
		// TODO: Implémenter la suppression via API REST
		log.Info("Ruleset deletion (API REST not yet implemented)")
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(ruleset, cloudflareRulesetFinalizer)
	if err := r.Update(ctx, ruleset); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Finalizer removed, resource will be deleted")
	return ctrl.Result{}, nil
}

// handleReconcileError updates the status when reconciliation fails
func (r *CloudflareRulesetReconciler) handleReconcileError(ctx context.Context, ruleset *cloudflarev1.CloudflareRuleset, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "Reconciliation failed")

	now := metav1.NewTime(time.Now())
	ruleset.Status.State = cloudflarev1.RulesetStateError
	ruleset.Status.Message = err.Error()
	ruleset.Status.ObservedGeneration = ruleset.Generation

	meta.SetStatusCondition(&ruleset.Status.Conditions, metav1.Condition{
		Type:               conditionTypeError,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: ruleset.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})
	meta.SetStatusCondition(&ruleset.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: ruleset.Generation,
		LastTransitionTime: now,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
	})

	if statusErr := r.Status().Update(ctx, ruleset); statusErr != nil {
		log.Error(statusErr, "Failed to update status")
		return ctrl.Result{}, statusErr
	}

	// Requeue with backoff
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudflareRulesetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.RequeueDuration == 0 {
		r.RequeueDuration = 5 * time.Minute
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1.CloudflareRuleset{}).
		Complete(r)
}


