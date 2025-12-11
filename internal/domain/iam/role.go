package iam

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrInvalidRoleName             = errors.New("role name is required")
	ErrInvalidAssumeRolePolicy     = errors.New("assume role policy document is required")
	ErrInvalidMaxSessionDuration   = errors.New("max session duration must be between 3600 and 43200 seconds")
	ErrInvalidPath                 = errors.New("path must start with / and end with /")
	ErrInvalidInlinePolicyName     = errors.New("inline policy name is required")
	ErrInvalidInlinePolicyDocument = errors.New("inline policy document is required")
)

// Role represents an IAM role in the domain model
type Role struct {
	// Core identifiers
	RoleName string
	RoleArn  string
	RoleId   string

	// Configuration
	Description              string
	AssumeRolePolicyDocument string
	MaxSessionDuration       int32
	Path                     string
	PermissionsBoundary      string

	// Policies
	ManagedPolicyArns    []string
	InlinePolicyName     string
	InlinePolicyDocument string

	// Tags
	Tags map[string]string

	// Deletion
	DeletionPolicy string

	// Metadata
	CreatedAt    *time.Time
	LastSyncTime *time.Time
}

// SetDefaults sets default values for the role
func (r *Role) SetDefaults() {
	if r.Path == "" {
		r.Path = "/"
	}
	if r.MaxSessionDuration == 0 {
		r.MaxSessionDuration = 3600 // 1 hour default
	}
	if r.DeletionPolicy == "" {
		r.DeletionPolicy = "Delete"
	}
	if r.Tags == nil {
		r.Tags = make(map[string]string)
	}
}

// Validate validates the role configuration
func (r *Role) Validate() error {
	if r.RoleName == "" {
		return ErrInvalidRoleName
	}

	if r.AssumeRolePolicyDocument == "" {
		return ErrInvalidAssumeRolePolicy
	}

	if r.MaxSessionDuration != 0 {
		if r.MaxSessionDuration < 3600 || r.MaxSessionDuration > 43200 {
			return ErrInvalidMaxSessionDuration
		}
	}

	if r.Path != "" {
		if len(r.Path) < 1 || r.Path[0] != '/' || r.Path[len(r.Path)-1] != '/' {
			return ErrInvalidPath
		}
	}

	// Validate inline policy if provided
	if r.InlinePolicyName != "" || r.InlinePolicyDocument != "" {
		if r.InlinePolicyName == "" {
			return ErrInvalidInlinePolicyName
		}
		if r.InlinePolicyDocument == "" {
			return ErrInvalidInlinePolicyDocument
		}
	}

	return nil
}

// ShouldDelete returns true if the role should be deleted when the CR is deleted
func (r *Role) ShouldDelete() bool {
	return r.DeletionPolicy == "Delete"
}
