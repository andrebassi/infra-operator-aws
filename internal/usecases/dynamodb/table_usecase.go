package dynamodb

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/dynamodb"
	"infra-operator/internal/ports"
)

// TableUseCase implements DynamoDB table business logic
type TableUseCase struct {
	repo ports.DynamoDBRepository
}

// NewTableUseCase creates a new table use case
func NewTableUseCase(repo ports.DynamoDBRepository) ports.DynamoDBUseCase {
	return &TableUseCase{repo: repo}
}

// SyncTable ensures table exists and matches desired state (idempotent)
func (uc *TableUseCase) SyncTable(ctx context.Context, table *dynamodb.Table) error {
	// Validate first
	if err := table.Validate(); err != nil {
		return fmt.Errorf("invalid table configuration: %w", err)
	}

	// Check if table exists
	exists, err := uc.repo.Exists(ctx, table.Name)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !exists {
		// Create table
		if err := uc.repo.Create(ctx, table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Get current state
	current, err := uc.repo.Get(ctx, table.Name)
	if err != nil {
		return fmt.Errorf("failed to get table: %w", err)
	}

	// Update table from current state
	table.ARN = current.ARN
	table.Status = current.Status
	table.ItemCount = current.ItemCount
	table.TableSizeBytes = current.TableSizeBytes
	table.StreamARN = current.StreamARN

	// Apply PITR if needed
	if table.PointInTimeRecovery {
		if err := uc.repo.UpdatePointInTimeRecovery(ctx, table.Name, true); err != nil {
			// Log but don't fail - PITR might not be supported in LocalStack
			fmt.Printf("Warning: failed to enable PITR: %v\n", err)
		}
	}

	// Apply tags if provided
	if len(table.Tags) > 0 && table.ARN != "" {
		if err := uc.repo.TagResource(ctx, table.ARN, table.Tags); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to apply tags: %v\n", err)
		}
	}

	return nil
}

// DeleteTable removes a table
func (uc *TableUseCase) DeleteTable(ctx context.Context, table *dynamodb.Table) error {
	// Check deletion policy
	if table.DeletionPolicy == "Retain" {
		return nil // Don't delete
	}

	// Check if table exists
	exists, err := uc.repo.Exists(ctx, table.Name)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !exists {
		return nil // Already deleted
	}

	// Delete table
	return uc.repo.Delete(ctx, table.Name)
}
