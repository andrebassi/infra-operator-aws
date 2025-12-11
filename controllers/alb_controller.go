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

const albFinalizerName = "alb.infra.operator.aws.io/finalizer"

// ALBReconciler reconciles an ALB object
type ALBReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=infra.operator.aws.io,resources=albs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.operator.aws.io,resources=albs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infra.operator.aws.io,resources=albs/finalizers,verbs=update

func (r *ALBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ALB instance
	albCR := &infrav1alpha1.ALB{}
	if err := r.Get(ctx, req.NamespacedName, albCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get ALB use case
	albUseCase, err := r.AWSClientFactory.GetALBUseCase(ctx, albCR.Spec.ProviderRef, albCR.Namespace)
	if err != nil {
		logger.Error(err, "failed to get ALB use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion
	if !albCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(albCR, albFinalizerName) {
			// Convert CR to domain model
			lb := mapper.CRToDomainALB(albCR)

			// Delete load balancer
			if err := albUseCase.DeleteLoadBalancer(ctx, lb); err != nil {
				logger.Error(err, "failed to delete load balancer")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(albCR, albFinalizerName)
			if err := r.Update(ctx, albCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(albCR, albFinalizerName) {
		controllerutil.AddFinalizer(albCR, albFinalizerName)
		if err := r.Update(ctx, albCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	lb := mapper.CRToDomainALB(albCR)

	// Sync load balancer
	if err := albUseCase.SyncLoadBalancer(ctx, lb); err != nil {
		logger.Error(err, "failed to sync load balancer")
		albCR.Status.Ready = false
		r.Status().Update(ctx, albCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusALB(lb, albCR)
	if err := r.Status().Update(ctx, albCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ALBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ALB{}).
		Complete(r)
}
