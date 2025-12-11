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

const vpcFinalizerName = "vpc.aws-infra-operator.runner.codes/finalizer"

type VPCReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *VPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	vpcCR := &infrav1alpha1.VPC{}
	if err := r.Get(ctx, req.NamespacedName, vpcCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	vpcUseCase, err := r.AWSClientFactory.GetVPCUseCase(ctx, vpcCR.Spec.ProviderRef, vpcCR.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get VPC use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !vpcCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(vpcCR, vpcFinalizerName) {
			v := mapper.CRToDomainVPC(vpcCR)
			if err := vpcUseCase.DeleteVPC(ctx, v); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(vpcCR, vpcFinalizerName)
			if err := r.Update(ctx, vpcCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(vpcCR, vpcFinalizerName) {
		controllerutil.AddFinalizer(vpcCR, vpcFinalizerName)
		if err := r.Update(ctx, vpcCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	v := mapper.CRToDomainVPC(vpcCR)
	if err := vpcUseCase.SyncVPC(ctx, v); err != nil {
		vpcCR.Status.Ready = false
		r.Status().Update(ctx, vpcCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusVPC(v, vpcCR)
	if err := r.Status().Update(ctx, vpcCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *VPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&infrav1alpha1.VPC{}).Complete(r)
}
