package ports

import (
	"context"
	"infra-operator/internal/domain/lambda"
)

// LambdaRepository defines the interface for Lambda function operations
type LambdaRepository interface {
	// Exists checks if a Lambda function exists
	Exists(ctx context.Context, functionName string) (bool, error)

	// Create creates a new Lambda function
	Create(ctx context.Context, function *lambda.Function) error

	// Get retrieves a Lambda function by name
	Get(ctx context.Context, functionName string) (*lambda.Function, error)

	// GetByARN retrieves a Lambda function by ARN
	GetByARN(ctx context.Context, functionARN string) (*lambda.Function, error)

	// Update updates an existing Lambda function configuration
	UpdateConfiguration(ctx context.Context, function *lambda.Function) error

	// UpdateCode updates the Lambda function code
	UpdateCode(ctx context.Context, function *lambda.Function) error

	// Delete deletes a Lambda function
	Delete(ctx context.Context, functionName string) error

	// TagFunction adds or updates tags on a Lambda function
	TagFunction(ctx context.Context, functionARN string, tags map[string]string) error

	// UntagFunction removes tags from a Lambda function
	UntagFunction(ctx context.Context, functionARN string, tagKeys []string) error
}

// LambdaUseCase defines the use case interface for Lambda function operations
type LambdaUseCase interface {
	// SyncFunction creates or updates a Lambda function
	SyncFunction(ctx context.Context, function *lambda.Function) error

	// DeleteFunction deletes a Lambda function
	DeleteFunction(ctx context.Context, function *lambda.Function) error
}
