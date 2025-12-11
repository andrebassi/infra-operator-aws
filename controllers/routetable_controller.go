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

const routeTableFinalizerName = "routetable.aws-infra-operator.runner.codes/finalizer"

type RouteTableReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *RouteTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	rtCR := &infrav1alpha1.RouteTable{}
	if err := r.Get(ctx, req.NamespacedName, rtCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	rtUseCase, err := r.AWSClientFactory.GetRouteTableUseCase(ctx, rtCR.Spec.ProviderRef, rtCR.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get RouteTable use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion with finalizer
	if !rtCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(rtCR, routeTableFinalizerName) {
			rt := mapper.CRToDomainRouteTable(rtCR)
			if err := rtUseCase.DeleteRouteTable(ctx, rt); err != nil {
				logger.Error(err, "Failed to delete route table")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(rtCR, routeTableFinalizerName)
			if err := r.Update(ctx, rtCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(rtCR, routeTableFinalizerName) {
		controllerutil.AddFinalizer(rtCR, routeTableFinalizerName)
		if err := r.Update(ctx, rtCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync route table
	rt := mapper.CRToDomainRouteTable(rtCR)
	if err := rtUseCase.SyncRouteTable(ctx, rt); err != nil {
		logger.Error(err, "Failed to sync route table")
		rtCR.Status.Ready = false
		r.Status().Update(ctx, rtCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusRouteTable(rt, rtCR)
	if err := r.Status().Update(ctx, rtCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *RouteTableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.RouteTable{}).
		Complete(r)
}
