package lambda

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidFunctionName = errors.New("function name cannot be empty")
	ErrInvalidRuntime      = errors.New("runtime cannot be empty")
	ErrInvalidHandler      = errors.New("handler cannot be empty")
	ErrInvalidRole         = errors.New("role ARN cannot be empty")
	ErrInvalidCode         = errors.New("code source must be specified")
	ErrInvalidTimeout      = errors.New("timeout must be between 1 and 900 seconds")
	ErrInvalidMemory       = errors.New("memory size must be between 128 and 10240 MB")
)

// Function represents a Lambda function in the domain model
type Function struct {
	// Identification
	Name        string
	ARN         string
	Description string

	// Configuration
	Runtime     string
	Handler     string
	Role        string
	Timeout     int32
	MemorySize  int32
	Environment map[string]string
	Layers      []string
	Tags        map[string]string

	// Code
	Code Code

	// VPC Configuration
	VpcConfig *VpcConfig

	// State
	State          string
	StateReason    string
	LastModified   time.Time
	Version        string
	CodeSize       int64
	DeletionPolicy string
}

// Code represents the Lambda function code
type Code struct {
	// For inline code (base64 zip)
	ZipFile string

	// For S3-based code
	S3Bucket        string
	S3Key           string
	S3ObjectVersion string

	// For container images
	ImageUri string
}

// VpcConfig represents VPC configuration for Lambda
type VpcConfig struct {
	SecurityGroupIds []string
	SubnetIds        []string
}

// Validate checks if the Lambda function configuration is valid
func (f *Function) Validate() error {
	if f.Name == "" {
		return ErrInvalidFunctionName
	}

	if f.Runtime == "" {
		return ErrInvalidRuntime
	}

	if f.Handler == "" && !strings.Contains(f.Code.ImageUri, ":") {
		// Handler is required unless using container images
		return ErrInvalidHandler
	}

	if f.Role == "" {
		return ErrInvalidRole
	}

	// Validate code source - at least one must be specified
	if f.Code.ZipFile == "" && f.Code.S3Bucket == "" && f.Code.ImageUri == "" {
		return ErrInvalidCode
	}

	// S3-based code needs both bucket and key
	if f.Code.S3Bucket != "" && f.Code.S3Key == "" {
		return errors.New("S3 bucket specified but S3 key is missing")
	}

	// Validate timeout
	if f.Timeout != 0 && (f.Timeout < 1 || f.Timeout > 900) {
		return ErrInvalidTimeout
	}

	// Validate memory size
	if f.MemorySize != 0 && (f.MemorySize < 128 || f.MemorySize > 10240) {
		return ErrInvalidMemory
	}

	// VPC config validation
	if f.VpcConfig != nil {
		if len(f.VpcConfig.SubnetIds) == 0 {
			return errors.New("VPC config requires at least one subnet")
		}
		if len(f.VpcConfig.SecurityGroupIds) == 0 {
			return errors.New("VPC config requires at least one security group")
		}
	}

	return nil
}

// IsActive checks if the function is in Active state
func (f *Function) IsActive() bool {
	return f.State == "Active"
}

// SetDefaults sets default values for optional fields
func (f *Function) SetDefaults() {
	if f.Timeout == 0 {
		f.Timeout = 3
	}
	if f.MemorySize == 0 {
		f.MemorySize = 128
	}
	if f.DeletionPolicy == "" {
		f.DeletionPolicy = "Delete"
	}
}
