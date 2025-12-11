package ports

import (
	"context"

	"infra-operator/internal/domain/s3"
)

// S3Repository defines the interface for S3 operations
// This is a Port in Hexagonal Architecture - defines WHAT we need, not HOW
type S3Repository interface {
	// Create creates a new S3 bucket
	Create(ctx context.Context, bucket *s3.Bucket) error

	// Get retrieves bucket information
	Get(ctx context.Context, name, region string) (*s3.Bucket, error)

	// Update updates bucket configuration
	Update(ctx context.Context, bucket *s3.Bucket) error

	// Delete deletes a bucket
	Delete(ctx context.Context, name, region string) error

	// Exists checks if bucket exists
	Exists(ctx context.Context, name, region string) (bool, error)

	// Configure applies configuration to existing bucket
	Configure(ctx context.Context, bucket *s3.Bucket) error

	// ConfigureVersioning enables/disables versioning
	ConfigureVersioning(ctx context.Context, name, region string, config *s3.VersioningConfig) error

	// ConfigureEncryption configures encryption
	ConfigureEncryption(ctx context.Context, name, region string, config *s3.EncryptionConfig) error

	// ConfigureLifecycle configures lifecycle rules
	ConfigureLifecycle(ctx context.Context, name, region string, rules []s3.LifecycleRule) error

	// ConfigureCORS configures CORS rules
	ConfigureCORS(ctx context.Context, name, region string, rules []s3.CORSRule) error

	// ConfigurePublicAccessBlock configures public access block
	ConfigurePublicAccessBlock(ctx context.Context, name, region string, config *s3.PublicAccessBlockConfig) error

	// ConfigureTags applies tags to bucket
	ConfigureTags(ctx context.Context, name, region string, tags map[string]string) error
}

// S3UseCase defines business logic operations for S3
type S3UseCase interface {
	// CreateBucket creates and configures a bucket
	CreateBucket(ctx context.Context, bucket *s3.Bucket) error

	// GetBucket retrieves bucket with full configuration
	GetBucket(ctx context.Context, name, region string) (*s3.Bucket, error)

	// UpdateBucket updates bucket configuration
	UpdateBucket(ctx context.Context, bucket *s3.Bucket) error

	// DeleteBucket deletes bucket according to deletion policy
	DeleteBucket(ctx context.Context, bucket *s3.Bucket) error

	// SyncBucket ensures bucket matches desired state
	SyncBucket(ctx context.Context, bucket *s3.Bucket) error
}
