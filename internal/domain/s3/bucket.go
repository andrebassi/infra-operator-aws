package s3

import "time"

// Bucket represents the domain model for an S3 bucket
// This is the core business entity, independent of AWS SDK or Kubernetes
type Bucket struct {
	Name   string
	Region string

	// Configuration
	Versioning        *VersioningConfig
	Encryption        *EncryptionConfig
	LifecycleRules    []LifecycleRule
	CORSRules         []CORSRule
	PublicAccessBlock *PublicAccessBlockConfig
	Tags              map[string]string

	// State
	ARN          string
	DomainName   string
	CreationTime *time.Time
	LastSyncTime *time.Time

	// Policy
	DeletionPolicy DeletionPolicy
}

// VersioningConfig represents versioning settings
type VersioningConfig struct {
	Enabled bool
}

// EncryptionConfig represents encryption settings
type EncryptionConfig struct {
	Algorithm string // AES256 or aws:kms
	KMSKeyID  string
}

// LifecycleRule represents a lifecycle rule
type LifecycleRule struct {
	ID          string
	Enabled     bool
	Prefix      string
	Expiration  *Expiration
	Transitions []Transition
}

// Expiration defines expiration settings
type Expiration struct {
	Days int32
}

// Transition defines transition settings
type Transition struct {
	Days         int32
	StorageClass string
}

// CORSRule defines CORS settings
type CORSRule struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	ExposeHeaders  []string
	MaxAgeSeconds  int32
}

// PublicAccessBlockConfig defines public access block settings
type PublicAccessBlockConfig struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

// DeletionPolicy defines what happens when bucket is deleted
type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyRetain DeletionPolicy = "Retain"
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)

// BucketStatus represents the current status
type BucketStatus string

const (
	BucketStatusPending  BucketStatus = "Pending"
	BucketStatusCreating BucketStatus = "Creating"
	BucketStatusActive   BucketStatus = "Active"
	BucketStatusDeleting BucketStatus = "Deleting"
	BucketStatusFailed   BucketStatus = "Failed"
)

// Validate performs domain validation
func (b *Bucket) Validate() error {
	if b.Name == "" {
		return ErrBucketNameRequired
	}

	if len(b.Name) < 3 || len(b.Name) > 63 {
		return ErrInvalidBucketNameLength
	}

	if b.Region == "" {
		return ErrRegionRequired
	}

	return nil
}

// IsEncrypted checks if bucket has encryption enabled
func (b *Bucket) IsEncrypted() bool {
	return b.Encryption != nil && b.Encryption.Algorithm != ""
}

// IsVersioned checks if versioning is enabled
func (b *Bucket) IsVersioned() bool {
	return b.Versioning != nil && b.Versioning.Enabled
}

// HasPublicAccessBlocked checks if all public access is blocked
func (b *Bucket) HasPublicAccessBlocked() bool {
	if b.PublicAccessBlock == nil {
		return false
	}

	return b.PublicAccessBlock.BlockPublicAcls &&
		b.PublicAccessBlock.IgnorePublicAcls &&
		b.PublicAccessBlock.BlockPublicPolicy &&
		b.PublicAccessBlock.RestrictPublicBuckets
}
