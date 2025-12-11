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

const cloudfrontFinalizer = "aws-infra-operator.runner.codes/cloudfront-finalizer"

type CloudFrontReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *CloudFrontReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cloudfront infrav1alpha1.CloudFront
	if err := r.Get(ctx, req.NamespacedName, &cloudfront); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	useCase, err := r.AWSClientFactory.GetCloudFrontUseCase(ctx, cloudfront.Spec.ProviderRef, cloudfront.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get CloudFront use case")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if !cloudfront.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cloudfront, cloudfrontFinalizer) {
			domainDist := mapper.CRToDomainCloudFront(&cloudfront)
			if err := useCase.DeleteDistribution(ctx, domainDist); err != nil {
				logger.Error(err, "Failed to delete distribution")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}

			controllerutil.RemoveFinalizer(&cloudfront, cloudfrontFinalizer)
			if err := r.Update(ctx, &cloudfront); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cloudfront, cloudfrontFinalizer) {
		controllerutil.AddFinalizer(&cloudfront, cloudfrontFinalizer)
		if err := r.Update(ctx, &cloudfront); err != nil {
			return ctrl.Result{}, err
		}
	}

	domainDist := mapper.CRToDomainCloudFront(&cloudfront)
	if err := useCase.SyncDistribution(ctx, domainDist); err != nil {
		logger.Error(err, "Failed to sync distribution")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	mapper.DomainToStatusCloudFront(domainDist, &cloudfront)
	if err := r.Status().Update(ctx, &cloudfront); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *CloudFrontReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.CloudFront{}).
		Complete(r)
}
