package ecr

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/ecr"
	"infra-operator/internal/ports"
)

type RepositoryUseCase struct {
	repo ports.ECRRepository
}

func NewRepositoryUseCase(repo ports.ECRRepository) ports.ECRUseCase {
	return &RepositoryUseCase{
		repo: repo,
	}
}

func (uc *RepositoryUseCase) SyncRepository(ctx context.Context, repository *ecr.Repository) error {
	// Set defaults before validation
	repository.SetDefaults()

	// Validate the repository
	if err := repository.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if repository exists
	exists, err := uc.repo.Exists(ctx, repository.RepositoryName)
	if err != nil {
		return fmt.Errorf("failed to check if repository exists: %w", err)
	}

	if !exists {
		// Create new repository
		if err := uc.repo.Create(ctx, repository); err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}

		// Tag the resource if ARN is available and tags are provided
		if repository.RepositoryArn != "" && len(repository.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, repository.RepositoryArn, repository.Tags); err != nil {
				return fmt.Errorf("failed to tag repository: %w", err)
			}
		}
	} else {
		// Get existing repository
		existing, err := uc.repo.Get(ctx, repository.RepositoryName)
		if err != nil {
			return fmt.Errorf("failed to get existing repository: %w", err)
		}

		// Update repository info from existing
		repository.RepositoryArn = existing.RepositoryArn
		repository.RepositoryUri = existing.RepositoryUri
		repository.RegistryId = existing.RegistryId
		repository.CreatedAt = existing.CreatedAt

		// Check if image scanning needs update
		if existing.ScanOnPush != repository.ScanOnPush {
			if err := uc.repo.UpdateImageScanning(ctx, repository.RepositoryName, repository.ScanOnPush); err != nil {
				return fmt.Errorf("failed to update image scanning: %w", err)
			}
		}

		// Check if image tag mutability needs update
		if existing.ImageTagMutability != repository.ImageTagMutability {
			if err := uc.repo.UpdateImageTagMutability(ctx, repository.RepositoryName, repository.ImageTagMutability); err != nil {
				return fmt.Errorf("failed to update image tag mutability: %w", err)
			}
		}

		// Update lifecycle policy if provided
		if repository.LifecyclePolicyText != "" {
			if err := uc.repo.PutLifecyclePolicy(ctx, repository.RepositoryName, repository.LifecyclePolicyText); err != nil {
				return fmt.Errorf("failed to update lifecycle policy: %w", err)
			}
		}

		// Update tags if they differ
		if len(repository.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, repository.RepositoryArn, repository.Tags); err != nil {
				return fmt.Errorf("failed to update tags: %w", err)
			}
		}
	}

	// Get image count
	imageCount, err := uc.repo.GetImageCount(ctx, repository.RepositoryName)
	if err != nil {
		// Don't fail if we can't get image count
		imageCount = 0
	}
	repository.ImageCount = imageCount

	// Update last sync time
	now := time.Now()
	repository.LastSyncTime = &now

	return nil
}

func (uc *RepositoryUseCase) DeleteRepository(ctx context.Context, repository *ecr.Repository) error {
	// Check if repository exists
	exists, err := uc.repo.Exists(ctx, repository.RepositoryName)
	if err != nil {
		return fmt.Errorf("failed to check if repository exists: %w", err)
	}

	if !exists {
		// Already deleted
		return nil
	}

	// Delete the repository
	if err := uc.repo.Delete(ctx, repository.RepositoryName); err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}
