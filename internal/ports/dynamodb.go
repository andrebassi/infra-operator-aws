package ports

import (
	"context"
	"infra-operator/internal/domain/dynamodb"
)

// DynamoDBRepository defines the port for DynamoDB data access
type DynamoDBRepository interface {
	// Exists checks if a table exists
	Exists(ctx context.Context, tableName string) (bool, error)

	// Create creates a new table
	Create(ctx context.Context, table *dynamodb.Table) error

	// Get retrieves table information
	Get(ctx context.Context, tableName string) (*dynamodb.Table, error)

	// Update updates table configuration
	Update(ctx context.Context, table *dynamodb.Table) error

	// Delete deletes a table
	Delete(ctx context.Context, tableName string) error

	// UpdateTimeToLive enables/disables TTL
	UpdateTimeToLive(ctx context.Context, tableName string, attributeName string, enabled bool) error

	// UpdateStreaming updates streaming configuration
	UpdateStreaming(ctx context.Context, tableName string, enabled bool, viewType string) error

	// UpdatePointInTimeRecovery updates PITR configuration
	UpdatePointInTimeRecovery(ctx context.Context, tableName string, enabled bool) error

	// TagResource applies tags to table
	TagResource(ctx context.Context, tableARN string, tags map[string]string) error
}

// DynamoDBUseCase defines the port for DynamoDB business logic
type DynamoDBUseCase interface {
	// SyncTable ensures table exists and matches desired state
	SyncTable(ctx context.Context, table *dynamodb.Table) error

	// DeleteTable removes a table
	DeleteTable(ctx context.Context, table *dynamodb.Table) error
}
