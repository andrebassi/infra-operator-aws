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

const apigatewayFinalizer = "aws-infra-operator.runner.codes/apigateway-finalizer"

type APIGatewayReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

func (r *APIGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var apigateway infrav1alpha1.APIGateway
	if err := r.Get(ctx, req.NamespacedName, &apigateway); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize use case
	useCase, err := r.AWSClientFactory.GetAPIGatewayUseCase(ctx, apigateway.Spec.ProviderRef, apigateway.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get API Gateway use case")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Handle deletion
	if !apigateway.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&apigateway, apigatewayFinalizer) {
			domainAPI := mapper.CRToDomainAPIGateway(&apigateway)
			if err := useCase.DeleteAPI(ctx, domainAPI); err != nil {
				logger.Error(err, "Failed to delete API")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}

			controllerutil.RemoveFinalizer(&apigateway, apigatewayFinalizer)
			if err := r.Update(ctx, &apigateway); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(&apigateway, apigatewayFinalizer) {
		controllerutil.AddFinalizer(&apigateway, apigatewayFinalizer)
		if err := r.Update(ctx, &apigateway); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync API
	domainAPI := mapper.CRToDomainAPIGateway(&apigateway)
	if err := useCase.SyncAPI(ctx, domainAPI); err != nil {
		logger.Error(err, "Failed to sync API")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Update status
	mapper.DomainToStatusAPIGateway(domainAPI, &apigateway)
	if err := r.Status().Update(ctx, &apigateway); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *APIGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.APIGateway{}).
		Complete(r)
}
