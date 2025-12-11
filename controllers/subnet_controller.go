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

const subnetFinalizerName = "subnet.aws-infra-operator.runner.codes/finalizer"

type SubnetReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *SubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	cr := &infrav1alpha1.Subnet{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	useCase, err := r.AWSClientFactory.GetSubnetUseCase(ctx, cr.Spec.ProviderRef, cr.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cr, subnetFinalizerName) {
			obj := mapper.CRToDomainSubnet(cr)
			if err := useCase.DeleteSubnet(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(cr, subnetFinalizerName)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(cr, subnetFinalizerName) {
		controllerutil.AddFinalizer(cr, subnetFinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	obj := mapper.CRToDomainSubnet(cr)
	if err := useCase.SyncSubnet(ctx, obj); err != nil {
		cr.Status.Ready = false
		r.Status().Update(ctx, cr)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusSubnet(obj, cr)
	if err := r.Status().Update(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *SubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&infrav1alpha1.Subnet{}).Complete(r)
}
