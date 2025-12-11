package secretsmanager

import (
	"errors"
	"time"
)

var (
	ErrInvalidSecretName       = errors.New("secret name is required")
	ErrNoSecretValue           = errors.New("either secret string or secret binary must be provided")
	ErrBothSecretValues        = errors.New("cannot specify both secret string and secret binary")
	ErrRotationLambdaRequired  = errors.New("rotation lambda ARN is required when rotation is enabled")
	ErrInvalidRotationInterval = errors.New("rotation interval must be at least 1 day")
	ErrInvalidRecoveryWindow   = errors.New("recovery window must be between 0 and 30 days")
)

// Secret represents an AWS Secrets Manager secret
type Secret struct {
	// Identifiers
	SecretName string
	ARN        string
	VersionId  string

	// Content
	Description  string
	SecretString string
	SecretBinary []byte

	// Encryption
	KmsKeyId string

	// Rotation
	RotationEnabled        bool
	RotationLambdaARN      string
	AutomaticallyAfterDays int32

	// Tags
	Tags map[string]string

	// Deletion
	DeletionPolicy       string
	RecoveryWindowInDays int32

	// Metadata
	CreatedAt    *time.Time
	LastSyncTime *time.Time
}

// SetDefaults sets default values
func (s *Secret) SetDefaults() {
	if s.DeletionPolicy == "" {
		s.DeletionPolicy = "Delete"
	}
	if s.RecoveryWindowInDays == 0 {
		s.RecoveryWindowInDays = 30
	}
	if s.Tags == nil {
		s.Tags = make(map[string]string)
	}
}

// Validate validates the secret configuration
func (s *Secret) Validate() error {
	if s.SecretName == "" {
		return ErrInvalidSecretName
	}

	// Must have either string or binary value
	hasString := s.SecretString != ""
	hasBinary := len(s.SecretBinary) > 0

	if !hasString && !hasBinary {
		return ErrNoSecretValue
	}

	if hasString && hasBinary {
		return ErrBothSecretValues
	}

	// Rotation validation
	if s.RotationEnabled {
		if s.RotationLambdaARN == "" {
			return ErrRotationLambdaRequired
		}
		if s.AutomaticallyAfterDays < 1 {
			return ErrInvalidRotationInterval
		}
	}

	// Recovery window validation
	if s.RecoveryWindowInDays < 0 || s.RecoveryWindowInDays > 30 {
		return ErrInvalidRecoveryWindow
	}

	return nil
}

// ShouldDelete returns true if the secret should be deleted when CR is deleted
func (s *Secret) ShouldDelete() bool {
	return s.DeletionPolicy == "Delete"
}

// HasRotation returns true if rotation is configured
func (s *Secret) HasRotation() bool {
	return s.RotationEnabled && s.RotationLambdaARN != ""
}
