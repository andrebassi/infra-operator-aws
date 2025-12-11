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

const elasticIPFinalizerName = "elasticip.infra.operator.aws.io/finalizer"

// ElasticIPReconciler reconciles an ElasticIP object
type ElasticIPReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticips/finalizers,verbs=update

func (r *ElasticIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ElasticIP instance
	eipCR := &infrav1alpha1.ElasticIP{}
	if err := r.Get(ctx, req.NamespacedName, eipCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get Elastic IP use case
	eipUseCase, err := r.AWSClientFactory.GetElasticIPUseCase(ctx, eipCR.Spec.ProviderRef, eipCR.Namespace)
	if err != nil {
		logger.Error(err, "failed to get Elastic IP use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion
	if !eipCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(eipCR, elasticIPFinalizerName) {
			// Convert CR to domain model
			addr := mapper.CRToDomainElasticIP(eipCR)

			// Release EIP
			if err := eipUseCase.ReleaseAddress(ctx, addr); err != nil {
				logger.Error(err, "failed to release Elastic IP")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(eipCR, elasticIPFinalizerName)
			if err := r.Update(ctx, eipCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(eipCR, elasticIPFinalizerName) {
		controllerutil.AddFinalizer(eipCR, elasticIPFinalizerName)
		if err := r.Update(ctx, eipCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	addr := mapper.CRToDomainElasticIP(eipCR)

	// Sync Elastic IP
	if err := eipUseCase.SyncAddress(ctx, addr); err != nil {
		logger.Error(err, "failed to sync Elastic IP")
		eipCR.Status.Ready = false
		r.Status().Update(ctx, eipCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusElasticIP(addr, eipCR)
	if err := r.Status().Update(ctx, eipCR); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to keep checking status
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ElasticIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ElasticIP{}).
		Complete(r)
}
