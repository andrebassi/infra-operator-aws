package ecr

import (
	"errors"
	"time"
)

var (
	ErrInvalidRepositoryName = errors.New("repository name cannot be empty")
	ErrInvalidTagMutability  = errors.New("image tag mutability must be MUTABLE or IMMUTABLE")
	ErrInvalidEncryptionType = errors.New("encryption type must be AES256 or KMS")
	ErrKmsKeyRequiredForKMS  = errors.New("KMS key ARN is required when encryption type is KMS")
)

// Repository represents an ECR repository in the domain model
type Repository struct {
	// Identification
	RepositoryName string
	RepositoryArn  string
	RepositoryUri  string
	RegistryId     string

	// Configuration
	ImageTagMutability string // MUTABLE or IMMUTABLE
	ScanOnPush         bool

	// Encryption
	EncryptionType string // AES256 or KMS
	KmsKey         string

	// Lifecycle
	LifecyclePolicyText string

	// Deletion
	DeletionPolicy string

	// Tags
	Tags map[string]string

	// State
	ImageCount   int64
	CreatedAt    *time.Time
	LastSyncTime *time.Time
}

// Validate checks if the repository configuration is valid
func (r *Repository) Validate() error {
	if r.RepositoryName == "" {
		return ErrInvalidRepositoryName
	}

	// Validate image tag mutability
	if r.ImageTagMutability != "" {
		if r.ImageTagMutability != "MUTABLE" && r.ImageTagMutability != "IMMUTABLE" {
			return ErrInvalidTagMutability
		}
	}

	// Validate encryption configuration
	if r.EncryptionType != "" {
		if r.EncryptionType != "AES256" && r.EncryptionType != "KMS" {
			return ErrInvalidEncryptionType
		}

		if r.EncryptionType == "KMS" && r.KmsKey == "" {
			return ErrKmsKeyRequiredForKMS
		}
	}

	return nil
}

// SetDefaults sets default values for optional fields
func (r *Repository) SetDefaults() {
	if r.ImageTagMutability == "" {
		r.ImageTagMutability = "MUTABLE"
	}

	if r.EncryptionType == "" {
		r.EncryptionType = "AES256"
	}

	if r.DeletionPolicy == "" {
		r.DeletionPolicy = "Delete"
	}
}

// IsReady checks if the repository is ready for use
func (r *Repository) IsReady() bool {
	return r.RepositoryArn != "" && r.RepositoryUri != ""
}
