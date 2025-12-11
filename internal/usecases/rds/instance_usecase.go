package rds

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/rds"
	"infra-operator/internal/ports"
)

type InstanceUseCase struct {
	repo ports.RDSRepository
}

func NewInstanceUseCase(repo ports.RDSRepository) ports.RDSUseCase {
	return &InstanceUseCase{
		repo: repo,
	}
}

func (uc *InstanceUseCase) SyncDBInstance(ctx context.Context, instance *rds.DBInstance) error {
	// Set defaults before validation
	instance.SetDefaults()

	// Validate the instance
	if err := instance.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if instance exists
	exists, err := uc.repo.Exists(ctx, instance.DBInstanceIdentifier)
	if err != nil {
		return fmt.Errorf("failed to check if instance exists: %w", err)
	}

	if !exists {
		// Create new instance
		if err := uc.repo.Create(ctx, instance); err != nil {
			return fmt.Errorf("failed to create DB instance: %w", err)
		}

		// Tag the resource if ARN is available and tags are provided
		if instance.DBInstanceArn != "" && len(instance.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, instance.DBInstanceArn, instance.Tags); err != nil {
				return fmt.Errorf("failed to tag DB instance: %w", err)
			}
		}
	} else {
		// Get existing instance
		existing, err := uc.repo.Get(ctx, instance.DBInstanceIdentifier)
		if err != nil {
			return fmt.Errorf("failed to get existing DB instance: %w", err)
		}

		// Update instance ARN and status from existing
		instance.DBInstanceArn = existing.DBInstanceArn
		instance.Status = existing.Status
		instance.Endpoint = existing.Endpoint

		// Check if update is needed (compare modifiable fields)
		needsUpdate := false

		if existing.AllocatedStorage != instance.AllocatedStorage {
			needsUpdate = true
		}
		if existing.DBInstanceClass != instance.DBInstanceClass {
			needsUpdate = true
		}
		if existing.BackupRetentionPeriod != instance.BackupRetentionPeriod {
			needsUpdate = true
		}
		if existing.PreferredBackupWindow != instance.PreferredBackupWindow {
			needsUpdate = true
		}

		// Perform update if needed
		if needsUpdate {
			if err := uc.repo.Update(ctx, instance); err != nil {
				return fmt.Errorf("failed to update DB instance: %w", err)
			}
		}

		// Update tags if they differ
		if len(instance.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, instance.DBInstanceArn, instance.Tags); err != nil {
				return fmt.Errorf("failed to update tags: %w", err)
			}
		}
	}

	// Update last sync time
	now := time.Now()
	instance.LastSyncTime = &now

	return nil
}

func (uc *InstanceUseCase) DeleteDBInstance(ctx context.Context, instance *rds.DBInstance) error {
	// Check if instance exists
	exists, err := uc.repo.Exists(ctx, instance.DBInstanceIdentifier)
	if err != nil {
		return fmt.Errorf("failed to check if instance exists: %w", err)
	}

	if !exists {
		// Already deleted
		return nil
	}

	// Determine skip final snapshot based on deletion policy
	skipFinalSnapshot := instance.SkipFinalSnapshot
	if instance.DeletionPolicy == "Delete" {
		skipFinalSnapshot = true
	}

	// Delete the instance
	if err := uc.repo.Delete(ctx, instance.DBInstanceIdentifier, skipFinalSnapshot); err != nil {
		return fmt.Errorf("failed to delete DB instance: %w", err)
	}

	return nil
}
