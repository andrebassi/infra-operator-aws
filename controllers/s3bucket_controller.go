package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awspkg "infra-operator/pkg/aws"
)

const s3BucketFinalizer = "aws-infra-operator.runner.codes/s3bucket-finalizer"

// S3BucketReconciler reconciles a S3Bucket object
type S3BucketReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=s3buckets/finalizers,verbs=update

func (r *S3BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the S3Bucket instance
	bucket := &infrav1alpha1.S3Bucket{}
	if err := r.Get(ctx, req.NamespacedName, bucket); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get AWS config from provider
	awsConfig, provider, err := awspkg.GetAWSConfigFromProvider(ctx, r.Client, bucket.Namespace, bucket.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Failed to get AWS config from provider")
		return r.updateStatus(ctx, bucket, false, fmt.Sprintf("Provider error: %v", err))
	}

	s3Client := s3.NewFromConfig(awsConfig)

	// Handle deletion
	if !bucket.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
			if err := r.deleteBucket(ctx, s3Client, bucket); err != nil {
				logger.Error(err, "Failed to delete S3 bucket")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(bucket, s3BucketFinalizer)
			if err := r.Update(ctx, bucket); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
		controllerutil.AddFinalizer(bucket, s3BucketFinalizer)
		if err := r.Update(ctx, bucket); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if bucket exists
	exists, err := r.bucketExists(ctx, s3Client, bucket.Spec.BucketName)
	if err != nil {
		return r.updateStatus(ctx, bucket, false, fmt.Sprintf("Failed to check bucket: %v", err))
	}

	// Create bucket if it doesn't exist
	if !exists {
		if err := r.createBucket(ctx, s3Client, bucket, provider); err != nil {
			logger.Error(err, "Failed to create S3 bucket")
			return r.updateStatus(ctx, bucket, false, fmt.Sprintf("Creation failed: %v", err))
		}
		logger.Info("Created S3 bucket", "bucket", bucket.Spec.BucketName)
	}

	// Configure bucket (versioning, encryption, etc.)
	if err := r.configureBucket(ctx, s3Client, bucket); err != nil {
		logger.Error(err, "Failed to configure S3 bucket")
		return r.updateStatus(ctx, bucket, false, fmt.Sprintf("Configuration failed: %v", err))
	}

	// Update status
	bucket.Status.Ready = true
	bucket.Status.ARN = fmt.Sprintf("arn:aws:s3:::%s", bucket.Spec.BucketName)
	bucket.Status.Region = provider.Spec.Region
	bucket.Status.BucketDomainName = fmt.Sprintf("%s.s3.amazonaws.com", bucket.Spec.BucketName)
	now := metav1.Now()
	bucket.Status.LastSyncTime = &now

	bucket.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "BucketReady",
			Message:            "S3 bucket is ready",
		},
	}

	if err := r.Status().Update(ctx, bucket); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled S3Bucket", "bucket", bucket.Spec.BucketName)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *S3BucketReconciler) bucketExists(ctx context.Context, s3Client *s3.Client, bucketName string) (bool, error) {
	_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Check if error is 404
		return false, nil
	}
	return true, nil
}

func (r *S3BucketReconciler) createBucket(ctx context.Context, s3Client *s3.Client, bucket *infrav1alpha1.S3Bucket, provider *infrav1alpha1.AWSProvider) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket.Spec.BucketName),
	}

	// Set location constraint if not us-east-1
	if provider.Spec.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(provider.Spec.Region),
		}
	}

	_, err := s3Client.CreateBucket(ctx, input)
	return err
}

func (r *S3BucketReconciler) configureBucket(ctx context.Context, s3Client *s3.Client, bucket *infrav1alpha1.S3Bucket) error {
	// Configure versioning
	if bucket.Spec.Versioning != nil {
		status := types.BucketVersioningStatusSuspended
		if bucket.Spec.Versioning.Enabled {
			status = types.BucketVersioningStatusEnabled
		}

		_, err := s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucket.Spec.BucketName),
			VersioningConfiguration: &types.VersioningConfiguration{
				Status: status,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to configure versioning: %w", err)
		}
	}

	// Configure encryption
	if bucket.Spec.Encryption != nil {
		var rule types.ServerSideEncryptionRule
		if bucket.Spec.Encryption.Algorithm == "AES256" {
			rule = types.ServerSideEncryptionRule{
				ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
					SSEAlgorithm: types.ServerSideEncryptionAes256,
				},
			}
		} else if bucket.Spec.Encryption.Algorithm == "aws:kms" {
			rule = types.ServerSideEncryptionRule{
				ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
					SSEAlgorithm:   types.ServerSideEncryptionAwsKms,
					KMSMasterKeyID: aws.String(bucket.Spec.Encryption.KMSKeyID),
				},
			}
		}

		_, err := s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
			Bucket: aws.String(bucket.Spec.BucketName),
			ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
				Rules: []types.ServerSideEncryptionRule{rule},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to configure encryption: %w", err)
		}
	}

	// Configure public access block
	if bucket.Spec.PublicAccessBlock != nil {
		_, err := s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
			Bucket: aws.String(bucket.Spec.BucketName),
			PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(bucket.Spec.PublicAccessBlock.BlockPublicAcls),
				IgnorePublicAcls:      aws.Bool(bucket.Spec.PublicAccessBlock.IgnorePublicAcls),
				BlockPublicPolicy:     aws.Bool(bucket.Spec.PublicAccessBlock.BlockPublicPolicy),
				RestrictPublicBuckets: aws.Bool(bucket.Spec.PublicAccessBlock.RestrictPublicBuckets),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to configure public access block: %w", err)
		}
	}

	// Add tags
	if len(bucket.Spec.Tags) > 0 {
		var tags []types.Tag
		for k, v := range bucket.Spec.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}

		_, err := s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
			Bucket: aws.String(bucket.Spec.BucketName),
			Tagging: &types.Tagging{
				TagSet: tags,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to configure tags: %w", err)
		}
	}

	return nil
}

func (r *S3BucketReconciler) deleteBucket(ctx context.Context, s3Client *s3.Client, bucket *infrav1alpha1.S3Bucket) error {
	// Check deletion policy
	if bucket.Spec.DeletionPolicy == "Retain" || bucket.Spec.DeletionPolicy == "Orphan" {
		return nil // Don't delete the bucket
	}

	// Empty the bucket first
	// Note: In production, you'd want to list and delete all objects first
	// This is a simplified version

	_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket.Spec.BucketName),
	})
	return err
}

func (r *S3BucketReconciler) updateStatus(ctx context.Context, bucket *infrav1alpha1.S3Bucket, ready bool, message string) (ctrl.Result, error) {
	bucket.Status.Ready = ready

	conditionStatus := metav1.ConditionTrue
	reason := "BucketReady"
	if !ready {
		conditionStatus = metav1.ConditionFalse
		reason = "BucketNotReady"
	}

	bucket.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             conditionStatus,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		},
	}

	if err := r.Status().Update(ctx, bucket); err != nil {
		return ctrl.Result{}, err
	}

	if !ready {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *S3BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.S3Bucket{}).
		Complete(r)
}
