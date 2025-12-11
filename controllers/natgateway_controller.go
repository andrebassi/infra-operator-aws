package controllers

import (
	"context"
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const natgatewayFinalizerName = "natgateway.aws-infra-operator.runner.codes/finalizer"

type NATGatewayReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *NATGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	cr := &infrav1alpha1.NATGateway{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	useCase, err := r.AWSClientFactory.GetNATGatewayUseCase(ctx, cr.Spec.ProviderRef, cr.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cr, natgatewayFinalizerName) {
			obj := mapper.CRToDomainNATGateway(cr)
			if err := useCase.DeleteGateway(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(cr, natgatewayFinalizerName)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(cr, natgatewayFinalizerName) {
		controllerutil.AddFinalizer(cr, natgatewayFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	obj := mapper.CRToDomainNATGateway(cr)
	if err := useCase.SyncGateway(ctx, obj); err != nil {
		cr.Status.Ready = false
		r.Status().Update(ctx, cr)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusNATGateway(obj, cr)
	if err := r.Status().Update(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *NATGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&infrav1alpha1.NATGateway{}).Complete(r)
}
