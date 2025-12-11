package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	lambdaadapter "infra-operator/internal/adapters/aws/lambda"
	"infra-operator/internal/domain/lambda"
	"infra-operator/internal/ports"
	lambdausecase "infra-operator/internal/usecases/lambda"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const lambdaFunctionFinalizer = "aws-infra-operator.runner.codes/lambdafunction-finalizer"

// LambdaFunctionReconciler reconciles a LambdaFunction object
type LambdaFunctionReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=lambdafunctions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=lambdafunctions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=lambdafunctions/finalizers,verbs=update

func (r *LambdaFunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the LambdaFunction instance
	var lambdaFunction infrav1alpha1.LambdaFunction
	if err := r.Get(ctx, req.NamespacedName, &lambdaFunction); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return without error
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch LambdaFunction")
		return ctrl.Result{}, err
	}

	// Get AWS configuration from provider
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(
		ctx,
		lambdaFunction.Namespace,
		lambdaFunction.Spec.ProviderRef,
	)
	if err != nil {
		logger.Error(err, "failed to get AWS config from provider")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Create Lambda repository and use case
	lambdaRepo := lambdaadapter.NewRepository(awsConfig)
	lambdaUseCase := lambdausecase.NewFunctionUseCase(lambdaRepo)

	// Convert CR to domain model
	function := mapper.CRToDomainFunction(&lambdaFunction)

	// Handle deletion
	if !lambdaFunction.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &lambdaFunction, function, lambdaUseCase)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&lambdaFunction, lambdaFunctionFinalizer) {
		controllerutil.AddFinalizer(&lambdaFunction, lambdaFunctionFinalizer)
		if err := r.Update(ctx, &lambdaFunction); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Execute business logic through use case
	if err := lambdaUseCase.SyncFunction(ctx, function); err != nil {
		logger.Error(err, "failed to sync function")
		lambdaFunction.Status.Ready = false
		lambdaFunction.Status.StateReason = err.Error()
		if statusErr := r.Status().Update(ctx, &lambdaFunction); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update CR status with domain model state
	lambdaFunction.Status = mapper.DomainFunctionToStatus(function)

	if err := r.Status().Update(ctx, &lambdaFunction); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("successfully reconciled LambdaFunction",
		"functionName", function.Name,
		"functionARN", function.ARN,
		"state", function.State,
		"runtime", function.Runtime)

	// Requeue after 5 minutes for periodic sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *LambdaFunctionReconciler) handleDeletion(ctx context.Context, lambdaFunction *infrav1alpha1.LambdaFunction, function *lambda.Function, useCase ports.LambdaUseCase) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(lambdaFunction, lambdaFunctionFinalizer) {
		return ctrl.Result{}, nil
	}

	// Check deletion policy
	deletionPolicy := lambdaFunction.Spec.DeletionPolicy
	if deletionPolicy == "" {
		deletionPolicy = "Delete"
	}

	switch deletionPolicy {
	case "Delete":
		logger.Info("deleting function from AWS", "functionName", function.Name)
		if err := useCase.DeleteFunction(ctx, function); err != nil {
			logger.Error(err, "failed to delete function")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	case "Retain":
		logger.Info("retaining function in AWS", "functionName", function.Name)
	default:
		logger.Info(fmt.Sprintf("unknown deletion policy %s, retaining function", deletionPolicy))
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(lambdaFunction, lambdaFunctionFinalizer)
	if err := r.Update(ctx, lambdaFunction); err != nil {
		logger.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *LambdaFunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.LambdaFunction{}).
		Complete(r)
}
