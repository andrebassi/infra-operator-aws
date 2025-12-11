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

const nlbFinalizerName = "nlb.infra.operator.aws.io/finalizer"

// NLBReconciler reconciles an NLB object
type NLBReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=nlbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=nlbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=nlbs/finalizers,verbs=update

func (r *NLBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nlbCR := &infrav1alpha1.NLB{}
	if err := r.Get(ctx, req.NamespacedName, nlbCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	nlbUseCase, err := r.AWSClientFactory.GetNLBUseCase(ctx, nlbCR.Spec.ProviderRef, nlbCR.Namespace)
	if err != nil {
		logger.Error(err, "failed to get NLB use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !nlbCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(nlbCR, nlbFinalizerName) {
			lb := mapper.CRToDomainNLB(nlbCR)
			if err := nlbUseCase.DeleteLoadBalancer(ctx, lb); err != nil {
				logger.Error(err, "failed to delete NLB")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(nlbCR, nlbFinalizerName)
			if err := r.Update(ctx, nlbCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(nlbCR, nlbFinalizerName) {
		controllerutil.AddFinalizer(nlbCR, nlbFinalizerName)
		if err := r.Update(ctx, nlbCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	lb := mapper.CRToDomainNLB(nlbCR)
	if err := nlbUseCase.SyncLoadBalancer(ctx, lb); err != nil {
		logger.Error(err, "failed to sync NLB")
		nlbCR.Status.Ready = false
		r.Status().Update(ctx, nlbCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusNLB(lb, nlbCR)
	if err := r.Status().Update(ctx, nlbCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *NLBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.NLB{}).
		Complete(r)
}
