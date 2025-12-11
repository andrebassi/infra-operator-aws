package s3

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/s3"
	"infra-operator/internal/ports"
)

// BucketUseCase implements business logic for S3 buckets
type BucketUseCase struct {
	repo ports.S3Repository
}

// NewBucketUseCase creates a new use case
func NewBucketUseCase(repo ports.S3Repository) ports.S3UseCase {
	return &BucketUseCase{
		repo: repo,
	}
}

// CreateBucket creates and fully configures a new S3 bucket
func (uc *BucketUseCase) CreateBucket(ctx context.Context, bucket *s3.Bucket) error {
	// Domain validation
	if err := bucket.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if bucket already exists
	exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if exists {
		return s3.ErrBucketAlreadyExists
	}

	// Create bucket
	if err := uc.repo.Create(ctx, bucket); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Apply configuration
	if err := uc.repo.Configure(ctx, bucket); err != nil {
		return fmt.Errorf("failed to configure bucket: %w", err)
	}

	// Update timestamps
	now := time.Now()
	bucket.CreationTime = &now
	bucket.LastSyncTime = &now

	return nil
}

// GetBucket retrieves bucket with full configuration
func (uc *BucketUseCase) GetBucket(ctx context.Context, name, region string) (*s3.Bucket, error) {
	if name == "" {
		return nil, s3.ErrBucketNameRequired
	}

	if region == "" {
		return nil, s3.ErrRegionRequired
	}

	bucket, err := uc.repo.Get(ctx, name, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return bucket, nil
}

// UpdateBucket updates bucket configuration
func (uc *BucketUseCase) UpdateBucket(ctx context.Context, bucket *s3.Bucket) error {
	// Domain validation
	if err := bucket.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if bucket exists
	exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		return s3.ErrBucketNotFound
	}

	// Update configuration
	if err := uc.repo.Configure(ctx, bucket); err != nil {
		return fmt.Errorf("failed to update bucket: %w", err)
	}

	// Update timestamp
	now := time.Now()
	bucket.LastSyncTime = &now

	return nil
}

// DeleteBucket deletes bucket according to deletion policy
func (uc *BucketUseCase) DeleteBucket(ctx context.Context, bucket *s3.Bucket) error {
	// Check deletion policy
	if bucket.DeletionPolicy == s3.DeletionPolicyRetain {
		// Don't delete, just orphan
		return nil
	}

	if bucket.DeletionPolicy == s3.DeletionPolicyOrphan {
		// Don't delete, just remove from operator management
		return nil
	}

	// DeletionPolicy = Delete
	exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		// Already deleted, nothing to do
		return nil
	}

	// Delete bucket
	if err := uc.repo.Delete(ctx, bucket.Name, bucket.Region); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// SyncBucket ensures bucket matches desired state (idempotent operation)
// This is the main method called by the Kubernetes controller
func (uc *BucketUseCase) SyncBucket(ctx context.Context, bucket *s3.Bucket) error {
	// Domain validation
	if err := bucket.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if bucket exists
	exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		// Create new bucket
		return uc.CreateBucket(ctx, bucket)
	}

	// Bucket exists, update configuration to match desired state
	if err := uc.repo.Configure(ctx, bucket); err != nil {
		return fmt.Errorf("failed to sync bucket configuration: %w", err)
	}

	// Update timestamp
	now := time.Now()
	bucket.LastSyncTime = &now

	return nil
}

// ValidateConfiguration validates complex bucket configuration rules
func (uc *BucketUseCase) ValidateConfiguration(bucket *s3.Bucket) error {
	// Validate encryption configuration
	if bucket.Encryption != nil {
		if bucket.Encryption.Algorithm == "aws:kms" && bucket.Encryption.KMSKeyID == "" {
			return s3.ErrInvalidEncryption
		}
	}

	// Validate lifecycle rules
	for _, rule := range bucket.LifecycleRules {
		if rule.ID == "" {
			return s3.ErrInvalidLifecycleRule
		}

		// At least one action (expiration or transition) must be set
		if rule.Expiration == nil && len(rule.Transitions) == 0 {
			return s3.ErrInvalidLifecycleRule
		}
	}

	return nil
}

// EnsureBucketCompliance ensures bucket meets compliance requirements
func (uc *BucketUseCase) EnsureBucketCompliance(bucket *s3.Bucket) error {
	// Example: Enforce encryption
	if bucket.Encryption == nil {
		return fmt.Errorf("encryption is required for compliance")
	}

	// Example: Enforce public access block
	if !bucket.HasPublicAccessBlocked() {
		return fmt.Errorf("public access must be blocked for compliance")
	}

	// Example: Enforce versioning for production
	if bucket.Tags["environment"] == "production" && !bucket.IsVersioned() {
		return fmt.Errorf("versioning is required for production buckets")
	}

	return nil
}
