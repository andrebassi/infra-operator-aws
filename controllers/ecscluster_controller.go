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
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const ecsClusterFinalizerName = "ecscluster.infra.operator.aws.io/finalizer"

// ECSClusterReconciler reconciles an ECSCluster object
type ECSClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecsclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecsclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ecsclusters/finalizers,verbs=update

func (r *ECSClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ECSCluster instance
	ecsClusterCR := &infrav1alpha1.ECSCluster{}
	if err := r.Get(ctx, req.NamespacedName, ecsClusterCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get ECS use case
	ecsUseCase, err := r.AWSClientFactory.GetECSUseCase(ctx, ecsClusterCR.Spec.ProviderRef, ecsClusterCR.Namespace)
	if err != nil {
		logger.Error(err, "failed to get ECS use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion
	if !ecsClusterCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(ecsClusterCR, ecsClusterFinalizerName) {
			// Convert CR to domain model
			cluster := mapper.CRToDomainECSCluster(ecsClusterCR)

			// Delete cluster
			if err := ecsUseCase.DeleteCluster(ctx, cluster); err != nil {
				logger.Error(err, "failed to delete ECS cluster")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(ecsClusterCR, ecsClusterFinalizerName)
			if err := r.Update(ctx, ecsClusterCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ecsClusterCR, ecsClusterFinalizerName) {
		controllerutil.AddFinalizer(ecsClusterCR, ecsClusterFinalizerName)
		if err := r.Update(ctx, ecsClusterCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	cluster := mapper.CRToDomainECSCluster(ecsClusterCR)

	// Sync cluster
	if err := ecsUseCase.SyncCluster(ctx, cluster); err != nil {
		logger.Error(err, "failed to sync ECS cluster")
		ecsClusterCR.Status.Ready = false
		r.Status().Update(ctx, ecsClusterCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusECSCluster(cluster, ecsClusterCR)
	if err := r.Status().Update(ctx, ecsClusterCR); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to keep checking cluster status
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ECSClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ECSCluster{}).
		Complete(r)
}
