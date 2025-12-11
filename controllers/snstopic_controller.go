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
	snsadapter "infra-operator/internal/adapters/aws/sns"
	"infra-operator/internal/domain/sns"
	"infra-operator/internal/ports"
	snsusecase "infra-operator/internal/usecases/sns"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const snsTopicFinalizer = "aws-infra-operator.runner.codes/snstopic-finalizer"

// SNSTopicReconciler reconciles a SNSTopic object
type SNSTopicReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=snstopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=snstopics/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=snstopics/finalizers,verbs=update

func (r *SNSTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the SNSTopic instance
	var snsTopic infrav1alpha1.SNSTopic
	if err := r.Get(ctx, req.NamespacedName, &snsTopic); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return without error
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch SNSTopic")
		return ctrl.Result{}, err
	}

	// Get AWS configuration from provider
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(
		ctx,
		snsTopic.Namespace,
		snsTopic.Spec.ProviderRef,
	)
	if err != nil {
		logger.Error(err, "failed to get AWS config from provider")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Create SNS repository and use case
	snsRepo := snsadapter.NewRepository(awsConfig)
	snsUseCase := snsusecase.NewTopicUseCase(snsRepo)

	// Convert CR to domain model
	topic := mapper.CRToDomainTopic(&snsTopic)

	// Handle deletion
	if !snsTopic.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &snsTopic, topic, snsUseCase)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&snsTopic, snsTopicFinalizer) {
		controllerutil.AddFinalizer(&snsTopic, snsTopicFinalizer)
		if err := r.Update(ctx, &snsTopic); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Execute business logic through use case
	if err := snsUseCase.SyncTopic(ctx, topic); err != nil {
		logger.Error(err, "failed to sync topic")
		snsTopic.Status.Ready = false
		if statusErr := r.Status().Update(ctx, &snsTopic); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update CR status with domain model state
	snsTopic.Status = mapper.DomainTopicToStatus(topic)

	if err := r.Status().Update(ctx, &snsTopic); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("successfully reconciled SNSTopic",
		"topicName", topic.Name,
		"topicARN", topic.ARN,
		"subscriptionsConfirmed", topic.SubscriptionsConfirmed,
		"subscriptionsPending", topic.SubscriptionsPending)

	// Requeue after 5 minutes for periodic sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *SNSTopicReconciler) handleDeletion(ctx context.Context, snsTopic *infrav1alpha1.SNSTopic, topic *sns.Topic, useCase ports.SNSUseCase) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(snsTopic, snsTopicFinalizer) {
		return ctrl.Result{}, nil
	}

	// Check deletion policy
	deletionPolicy := snsTopic.Spec.DeletionPolicy
	if deletionPolicy == "" {
		deletionPolicy = "Delete"
	}

	switch deletionPolicy {
	case "Delete":
		logger.Info("deleting topic from AWS", "topicName", topic.Name)
		if err := useCase.DeleteTopic(ctx, topic); err != nil {
			logger.Error(err, "failed to delete topic")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	case "Retain":
		logger.Info("retaining topic in AWS", "topicName", topic.Name)
	default:
		logger.Info(fmt.Sprintf("unknown deletion policy %s, retaining topic", deletionPolicy))
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(snsTopic, snsTopicFinalizer)
	if err := r.Update(ctx, snsTopic); err != nil {
		logger.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SNSTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.SNSTopic{}).
		Complete(r)
}
