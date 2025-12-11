package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const kmsKeyFinalizerName = "kmskey.aws-infra-operator.runner.codes/finalizer"

// KMSKeyReconciler reconciles a KMSKey object
type KMSKeyReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=kmskeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=kmskeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=kmskeys/finalizers,verbs=update

func (r *KMSKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the KMSKey instance
	kmsKey := &infrav1alpha1.KMSKey{}
	if err := r.Get(ctx, req.NamespacedName, kmsKey); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get KMS use case
	kmsUseCase, err := r.AWSClientFactory.GetKMSUseCase(ctx, kmsKey.Spec.ProviderRef, kmsKey.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get KMS use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Check if the resource is being deleted
	if !kmsKey.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(kmsKey, kmsKeyFinalizerName) {
			// Convert CR to domain model
			key := mapper.CRToDomainKMSKey(kmsKey)

			// Delete KMS key
			if err := kmsUseCase.DeleteKey(ctx, key); err != nil {
				logger.Error(err, "Failed to delete KMS key")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(kmsKey, kmsKeyFinalizerName)
			if err := r.Update(ctx, kmsKey); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(kmsKey, kmsKeyFinalizerName) {
		controllerutil.AddFinalizer(kmsKey, kmsKeyFinalizerName)
		if err := r.Update(ctx, kmsKey); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	key := mapper.CRToDomainKMSKey(kmsKey)

	// Sync KMS key
	if err := kmsUseCase.SyncKey(ctx, key); err != nil {
		logger.Error(err, "Failed to sync KMS key")
		kmsKey.Status.Ready = false
		if updateErr := r.Status().Update(ctx, kmsKey); updateErr != nil {
			logger.Error(updateErr, "Failed to update KMSKey status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusKMSKey(key, kmsKey)
	if err := r.Status().Update(ctx, kmsKey); err != nil {
		logger.Error(err, "Failed to update KMSKey status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled KMSKey",
		"keyId", kmsKey.Status.KeyId,
		"arn", kmsKey.Status.Arn,
		"state", kmsKey.Status.KeyState)

	// Requeue after 5 minutes for continuous sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *KMSKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.KMSKey{}).
		Complete(r)
}
