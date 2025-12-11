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

const securityGroupFinalizerName = "securitygroup.aws-infra-operator.runner.codes/finalizer"

type SecurityGroupReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *SecurityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	sgCR := &infrav1alpha1.SecurityGroup{}
	if err := r.Get(ctx, req.NamespacedName, sgCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sgUseCase, err := r.AWSClientFactory.GetSecurityGroupUseCase(ctx, sgCR.Spec.ProviderRef, sgCR.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get SecurityGroup use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion
	if !sgCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(sgCR, securityGroupFinalizerName) {
			sg := mapper.CRToDomainSecurityGroup(sgCR)
			if err := sgUseCase.DeleteSecurityGroup(ctx, sg); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(sgCR, securityGroupFinalizerName)
			if err := r.Update(ctx, sgCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(sgCR, securityGroupFinalizerName) {
		controllerutil.AddFinalizer(sgCR, securityGroupFinalizerName)
		if err := r.Update(ctx, sgCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync security group
	sg := mapper.CRToDomainSecurityGroup(sgCR)
	if err := sgUseCase.SyncSecurityGroup(ctx, sg); err != nil {
		sgCR.Status.Ready = false
		r.Status().Update(ctx, sgCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusSecurityGroup(sg, sgCR)
	if err := r.Status().Update(ctx, sgCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *SecurityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&infrav1alpha1.SecurityGroup{}).Complete(r)
}
