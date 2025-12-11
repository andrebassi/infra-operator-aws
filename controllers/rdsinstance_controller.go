package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/ports"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const rdsFinalizerName = "aws-infra-operator.runner.codes/rds-finalizer"

// RDSInstanceReconciler reconciles a RDSInstance object
type RDSInstanceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=rdsinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=rdsinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=rdsinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *RDSInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the RDSInstance instance
	rdsInstance := &infrav1alpha1.RDSInstance{}
	if err := r.Get(ctx, req.NamespacedName, rdsInstance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RDSInstance")
		return ctrl.Result{}, err
	}

	// Get the RDS use case
	rdsUseCase, err := r.AWSClientFactory.GetRDSUseCase(ctx, rdsInstance.Spec.ProviderRef, rdsInstance.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get RDS use case")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !rdsInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, rdsInstance, rdsUseCase)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(rdsInstance, rdsFinalizerName) {
		controllerutil.AddFinalizer(rdsInstance, rdsFinalizerName)
		if err := r.Update(ctx, rdsInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get password from secret if specified
	password := rdsInstance.Spec.MasterUserPassword
	if rdsInstance.Spec.MasterUserPasswordSecretRef != nil {
		secret, err := r.getSecret(ctx, rdsInstance.Namespace, rdsInstance.Spec.MasterUserPasswordSecretRef.Name)
		if err != nil {
			logger.Error(err, "Failed to get master password secret")
			return ctrl.Result{}, err
		}
		passwordBytes, ok := secret.Data[rdsInstance.Spec.MasterUserPasswordSecretRef.Key]
		if !ok {
			return ctrl.Result{}, fmt.Errorf("key %s not found in secret %s",
				rdsInstance.Spec.MasterUserPasswordSecretRef.Key,
				rdsInstance.Spec.MasterUserPasswordSecretRef.Name)
		}
		password = string(passwordBytes)
	}

	// Convert CR to domain model
	instance := mapper.CRToDomainRDSInstance(rdsInstance)
	instance.MasterPassword = password

	// Sync the DB instance
	if err := rdsUseCase.SyncDBInstance(ctx, instance); err != nil {
		logger.Error(err, "Failed to sync RDS instance")

		// Update status with error
		rdsInstance.Status.Ready = false
		rdsInstance.Status.Status = "error"
		if err := r.Status().Update(ctx, rdsInstance); err != nil {
			logger.Error(err, "Failed to update status")
		}

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update CR status from domain model
	mapper.UpdateCRStatusFromRDSInstance(rdsInstance, instance)

	// Update the status
	if err := r.Status().Update(ctx, rdsInstance); err != nil {
		logger.Error(err, "Failed to update RDSInstance status")
		return ctrl.Result{}, err
	}

	// Requeue to check status periodically
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *RDSInstanceReconciler) handleDeletion(ctx context.Context, rdsInstance *infrav1alpha1.RDSInstance, rdsUseCase ports.RDSUseCase) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(rdsInstance, rdsFinalizerName) {
		// Convert CR to domain model
		instance := mapper.CRToDomainRDSInstance(rdsInstance)

		// Delete the DB instance
		if err := rdsUseCase.DeleteDBInstance(ctx, instance); err != nil {
			logger.Error(err, "Failed to delete RDS instance")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(rdsInstance, rdsFinalizerName)
		if err := r.Update(ctx, rdsInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getSecret retrieves a Kubernetes Secret
func (r *RDSInstanceReconciler) getSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RDSInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.RDSInstance{}).
		Complete(r)
}
