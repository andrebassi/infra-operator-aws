package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsadapter "infra-operator/internal/adapters/aws/s3"
	s3usecase "infra-operator/internal/usecases/s3"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const s3BucketFinalizerClean = "aws-infra-operator.runner.codes/s3bucket-finalizer-clean"

// S3BucketReconcilerClean reconciles S3Bucket using Clean Architecture
type S3BucketReconcilerClean struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets/finalizers,verbs=update
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=awsproviders,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *S3BucketReconcilerClean) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch S3Bucket CR
	bucketCR := &infrav1alpha1.S3Bucket{}
	if err := r.Get(ctx, req.NamespacedName, bucketCR); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Get AWS configuration from provider
	awsConfig, provider, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(
		ctx,
		bucketCR.Namespace,
		bucketCR.Spec.ProviderRef,
	)
	if err != nil {
		logger.Error(err, "Failed to get AWS config from provider")
		return r.updateStatus(ctx, bucketCR, false, fmt.Sprintf("Provider error: %v", err))
	}

	// 3. Create AWS S3 Repository (Adapter)
	s3Repo := awsadapter.NewRepository(awsConfig)

	// 4. Create Use Case with injected repository
	s3UseCase := s3usecase.NewBucketUseCase(s3Repo)

	// 5. Convert CR to Domain model
	domainBucket := mapper.CRToDomainBucket(bucketCR)
	mapper.SetBucketRegionFromProvider(domainBucket, provider)

	// 6. Handle deletion
	if !bucketCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(bucketCR, s3BucketFinalizerClean) {
			// Execute deletion through use case
			if err := s3UseCase.DeleteBucket(ctx, domainBucket); err != nil {
				logger.Error(err, "Failed to delete S3 bucket")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(bucketCR, s3BucketFinalizerClean)
			if err := r.Update(ctx, bucketCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 7. Add finalizer if not present
	if !controllerutil.ContainsFinalizer(bucketCR, s3BucketFinalizerClean) {
		controllerutil.AddFinalizer(bucketCR, s3BucketFinalizerClean)
		if err := r.Update(ctx, bucketCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 8. Execute business logic through use case (idempotent)
	if err := s3UseCase.SyncBucket(ctx, domainBucket); err != nil {
		logger.Error(err, "Failed to sync S3 bucket")
		return r.updateStatus(ctx, bucketCR, false, fmt.Sprintf("Sync failed: %v", err))
	}

	// 9. Update CR status from domain model
	mapper.DomainBucketToCRStatus(domainBucket, bucketCR)
	bucketCR.Status.Ready = true
	bucketCR.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "BucketReady",
			Message:            "S3 bucket is ready",
		},
	}

	if err := r.Status().Update(ctx, bucketCR); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled S3Bucket",
		"bucket", domainBucket.Name,
		"region", domainBucket.Region)

	// Requeue after 5 minutes for drift detection
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// updateStatus updates the CR status
func (r *S3BucketReconcilerClean) updateStatus(ctx context.Context, bucketCR *infrav1alpha1.S3Bucket, ready bool, message string) (ctrl.Result, error) {
	bucketCR.Status.Ready = ready

	conditionStatus := metav1.ConditionTrue
	reason := "BucketReady"
	if !ready {
		conditionStatus = metav1.ConditionFalse
		reason = "BucketNotReady"
	}

	bucketCR.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             conditionStatus,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		},
	}

	if err := r.Status().Update(ctx, bucketCR); err != nil {
		return ctrl.Result{}, err
	}

	if !ready {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *S3BucketReconcilerClean) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.S3Bucket{}).
		Complete(r)
}
