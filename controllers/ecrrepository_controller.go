package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/ports"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const ecrFinalizerName = "aws-infra-operator.runner.codes/ecr-finalizer"

// ECRRepositoryReconciler reconciles an ECRRepository object
type ECRRepositoryReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecrrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecrrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecrrepositories/finalizers,verbs=update

func (r *ECRRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ECRRepository instance
	ecrRepo := &infrav1alpha1.ECRRepository{}
	if err := r.Get(ctx, req.NamespacedName, ecrRepo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ECRRepository")
		return ctrl.Result{}, err
	}

	// Get the ECR use case
	ecrUseCase, err := r.AWSClientFactory.GetECRUseCase(ctx, ecrRepo.Spec.ProviderRef, ecrRepo.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get ECR use case")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !ecrRepo.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, ecrRepo, ecrUseCase)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ecrRepo, ecrFinalizerName) {
		controllerutil.AddFinalizer(ecrRepo, ecrFinalizerName)
		if err := r.Update(ctx, ecrRepo); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	repository := mapper.CRToDomainECRRepository(ecrRepo)

	// Sync the repository
	if err := ecrUseCase.SyncRepository(ctx, repository); err != nil {
		logger.Error(err, "Failed to sync ECR repository")

		// Update status with error
		ecrRepo.Status.Ready = false
		if err := r.Status().Update(ctx, ecrRepo); err != nil {
			logger.Error(err, "Failed to update status")
		}

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update CR status from domain model
	mapper.UpdateCRStatusFromECRRepository(ecrRepo, repository)

	// Update the status
	if err := r.Status().Update(ctx, ecrRepo); err != nil {
		logger.Error(err, "Failed to update ECRRepository status")
		return ctrl.Result{}, err
	}

	logger.Info("successfully reconciled ECRRepository",
		"repositoryName", repository.RepositoryName,
		"repositoryUri", repository.RepositoryUri,
		"imageCount", repository.ImageCount)

	// Requeue to check status periodically
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ECRRepositoryReconciler) handleDeletion(ctx context.Context, ecrRepo *infrav1alpha1.ECRRepository, ecrUseCase ports.ECRUseCase) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(ecrRepo, ecrFinalizerName) {
		// Convert CR to domain model
		repository := mapper.CRToDomainECRRepository(ecrRepo)

		// Delete the repository
		if err := ecrUseCase.DeleteRepository(ctx, repository); err != nil {
			logger.Error(err, "Failed to delete ECR repository")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(ecrRepo, ecrFinalizerName)
		if err := r.Update(ctx, ecrRepo); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ECRRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ECRRepository{}).
		Complete(r)
}
