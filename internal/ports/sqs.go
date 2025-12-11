package ports

import (
	"context"
	"infra-operator/internal/domain/sqs"
)

// SQSRepository defines the interface for SQS queue operations
type SQSRepository interface {
	// Exists checks if a queue with the given name exists
	Exists(ctx context.Context, queueName string) (bool, error)

	// Create creates a new SQS queue
	Create(ctx context.Context, queue *sqs.Queue) error

	// Get retrieves queue details by URL
	Get(ctx context.Context, queueURL string) (*sqs.Queue, error)

	// GetByName retrieves queue details by name
	GetByName(ctx context.Context, queueName string) (*sqs.Queue, error)

	// Update updates queue attributes
	Update(ctx context.Context, queue *sqs.Queue) error

	// Delete deletes a queue
	Delete(ctx context.Context, queueURL string) error

	// TagQueue tags a queue
	TagQueue(ctx context.Context, queueURL string, tags map[string]string) error
}

// SQSUseCase defines the business logic interface for SQS operations
type SQSUseCase interface {
	// SyncQueue synchronizes the desired queue state with AWS
	SyncQueue(ctx context.Context, queue *sqs.Queue) error

	// DeleteQueue deletes a queue
	DeleteQueue(ctx context.Context, queue *sqs.Queue) error
}
