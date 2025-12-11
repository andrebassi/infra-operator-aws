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

const eksClusterFinalizerName = "ekscluster.aws-infra-operator.runner.codes/finalizer"

type EKSClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *EKSClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	eksCR := &infrav1alpha1.EKSCluster{}
	if err := r.Get(ctx, req.NamespacedName, eksCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	eksUseCase, err := r.AWSClientFactory.GetEKSUseCase(ctx, eksCR.Spec.ProviderRef, eksCR.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get EKS use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Handle deletion with finalizer
	if !eksCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(eksCR, eksClusterFinalizerName) {
			cluster := mapper.CRToDomainEKSCluster(eksCR)
			if err := eksUseCase.DeleteCluster(ctx, cluster); err != nil {
				logger.Error(err, "Failed to delete EKS cluster")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(eksCR, eksClusterFinalizerName)
			if err := r.Update(ctx, eksCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(eksCR, eksClusterFinalizerName) {
		controllerutil.AddFinalizer(eksCR, eksClusterFinalizerName)
		if err := r.Update(ctx, eksCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync EKS cluster
	cluster := mapper.CRToDomainEKSCluster(eksCR)
	if err := eksUseCase.SyncCluster(ctx, cluster); err != nil {
		logger.Error(err, "Failed to sync EKS cluster")
		eksCR.Status.Ready = false
		eksCR.Status.Status = "FAILED"
		r.Status().Update(ctx, eksCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusEKSCluster(cluster, eksCR)
	if err := r.Status().Update(ctx, eksCR); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue based on cluster status
	if cluster.IsCreating() {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *EKSClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.EKSCluster{}).
		Complete(r)
}
