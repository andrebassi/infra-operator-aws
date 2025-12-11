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
	sqsadapter "infra-operator/internal/adapters/aws/sqs"
	"infra-operator/internal/domain/sqs"
	"infra-operator/internal/ports"
	sqsusecase "infra-operator/internal/usecases/sqs"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const sqsQueueFinalizer = "aws-infra-operator.runner.codes/sqsqueue-finalizer"

// SQSQueueReconciler reconciles a SQSQueue object
type SQSQueueReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=sqsqueues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=sqsqueues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=sqsqueues/finalizers,verbs=update

func (r *SQSQueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the SQSQueue instance
	var sqsQueue infrav1alpha1.SQSQueue
	if err := r.Get(ctx, req.NamespacedName, &sqsQueue); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return without error
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch SQSQueue")
		return ctrl.Result{}, err
	}

	// Get AWS configuration from provider
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(
		ctx,
		sqsQueue.Namespace,
		sqsQueue.Spec.ProviderRef,
	)
	if err != nil {
		logger.Error(err, "failed to get AWS config from provider")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Create SQS repository and use case
	sqsRepo := sqsadapter.NewRepository(awsConfig)
	sqsUseCase := sqsusecase.NewQueueUseCase(sqsRepo)

	// Convert CR to domain model
	queue := mapper.CRToDomainQueue(&sqsQueue)

	// Handle deletion
	if !sqsQueue.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &sqsQueue, queue, sqsUseCase)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&sqsQueue, sqsQueueFinalizer) {
		controllerutil.AddFinalizer(&sqsQueue, sqsQueueFinalizer)
		if err := r.Update(ctx, &sqsQueue); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Execute business logic through use case
	if err := sqsUseCase.SyncQueue(ctx, queue); err != nil {
		logger.Error(err, "failed to sync queue")
		sqsQueue.Status.Ready = false
		if statusErr := r.Status().Update(ctx, &sqsQueue); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update CR status with domain model state
	sqsQueue.Status = mapper.DomainQueueToStatus(queue)

	if err := r.Status().Update(ctx, &sqsQueue); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("successfully reconciled SQSQueue",
		"queueName", queue.Name,
		"queueURL", queue.URL,
		"queueARN", queue.ARN,
		"messages", queue.ApproximateNumberOfMessages)

	// Requeue after 5 minutes for periodic sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *SQSQueueReconciler) handleDeletion(ctx context.Context, sqsQueue *infrav1alpha1.SQSQueue, queue *sqs.Queue, useCase ports.SQSUseCase) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(sqsQueue, sqsQueueFinalizer) {
		return ctrl.Result{}, nil
	}

	// Check deletion policy
	deletionPolicy := sqsQueue.Spec.DeletionPolicy
	if deletionPolicy == "" {
		deletionPolicy = "Delete"
	}

	switch deletionPolicy {
	case "Delete":
		logger.Info("deleting queue from AWS", "queueName", queue.Name)
		if err := useCase.DeleteQueue(ctx, queue); err != nil {
			logger.Error(err, "failed to delete queue")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	case "Retain":
		logger.Info("retaining queue in AWS", "queueName", queue.Name)
	default:
		logger.Info(fmt.Sprintf("unknown deletion policy %s, retaining queue", deletionPolicy))
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(sqsQueue, sqsQueueFinalizer)
	if err := r.Update(ctx, sqsQueue); err != nil {
		logger.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SQSQueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.SQSQueue{}).
		Complete(r)
}
