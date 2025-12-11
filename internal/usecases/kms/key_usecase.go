package kms

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/kms"
	"infra-operator/internal/ports"
)

type KeyUseCase struct {
	repo ports.KMSRepository
}

func NewKeyUseCase(repo ports.KMSRepository) *KeyUseCase {
	return &KeyUseCase{repo: repo}
}

func (uc *KeyUseCase) SyncKey(ctx context.Context, key *kms.Key) error {
	// Set defaults
	key.SetDefaults()

	// Validate
	if err := key.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if key exists
	exists, err := uc.repo.Exists(ctx, key.KeyId)
	if err != nil {
		return fmt.Errorf("failed to check if key exists: %w", err)
	}

	if !exists {
		// Create new key
		if err := uc.repo.Create(ctx, key); err != nil {
			return fmt.Errorf("failed to create key: %w", err)
		}

		// Tag the key
		if len(key.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, key.KeyId, key.Tags); err != nil {
				return fmt.Errorf("failed to tag key: %w", err)
			}
		}
	} else {
		// Update existing key
		existingKey, err := uc.repo.Get(ctx, key.KeyId)
		if err != nil {
			return fmt.Errorf("failed to get existing key: %w", err)
		}

		// Update description if changed
		if key.Description != existingKey.Description {
			if err := uc.repo.Update(ctx, key); err != nil {
				return fmt.Errorf("failed to update key: %w", err)
			}
		}

		// Sync enabled state
		if key.Enabled != existingKey.Enabled {
			if key.Enabled {
				if err := uc.repo.EnableKey(ctx, key.KeyId); err != nil {
					return fmt.Errorf("failed to enable key: %w", err)
				}
			} else {
				if err := uc.repo.DisableKey(ctx, key.KeyId); err != nil {
					return fmt.Errorf("failed to disable key: %w", err)
				}
			}
		}

		// Sync key rotation (only for symmetric keys)
		if key.IsSymmetric() {
			if key.EnableKeyRotation != existingKey.EnableKeyRotation {
				if key.EnableKeyRotation {
					if err := uc.repo.EnableKeyRotation(ctx, key.KeyId); err != nil {
						return fmt.Errorf("failed to enable key rotation: %w", err)
					}
				} else {
					if err := uc.repo.DisableKeyRotation(ctx, key.KeyId); err != nil {
						return fmt.Errorf("failed to disable key rotation: %w", err)
					}
				}
			}
		}

		// Update tags
		if len(key.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, key.KeyId, key.Tags); err != nil {
				return fmt.Errorf("failed to update tags: %w", err)
			}
		}
	}

	// Update last sync time
	now := time.Now()
	key.LastSyncTime = &now

	return nil
}

func (uc *KeyUseCase) DeleteKey(ctx context.Context, key *kms.Key) error {
	if !key.ShouldDelete() {
		// Retain policy - just return without deleting
		return nil
	}

	// Schedule key for deletion with pending window
	if err := uc.repo.ScheduleKeyDeletion(ctx, key.KeyId, key.PendingWindowInDays); err != nil {
		return fmt.Errorf("failed to schedule key deletion: %w", err)
	}

	return nil
}
