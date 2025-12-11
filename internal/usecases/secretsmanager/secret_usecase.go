package secretsmanager

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/secretsmanager"
	"infra-operator/internal/ports"
)

type SecretUseCase struct {
	repo ports.SecretsManagerRepository
}

func NewSecretUseCase(repo ports.SecretsManagerRepository) *SecretUseCase {
	return &SecretUseCase{repo: repo}
}

func (uc *SecretUseCase) SyncSecret(ctx context.Context, secret *secretsmanager.Secret) error {
	secret.SetDefaults()
	if err := secret.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	exists, err := uc.repo.Exists(ctx, secret.SecretName)
	if err != nil {
		return fmt.Errorf("failed to check if secret exists: %w", err)
	}

	if !exists {
		if err := uc.repo.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
		if len(secret.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, secret.ARN, secret.Tags); err != nil {
				return fmt.Errorf("failed to tag secret: %w", err)
			}
		}
		if secret.HasRotation() {
			if err := uc.repo.UpdateRotation(ctx, secret.SecretName, secret.RotationLambdaARN, secret.AutomaticallyAfterDays); err != nil {
				return fmt.Errorf("failed to enable rotation: %w", err)
			}
		}
	} else {
		versionId, err := uc.repo.UpdateSecretValue(ctx, secret.SecretName, secret.SecretString, secret.SecretBinary)
		if err != nil {
			return fmt.Errorf("failed to update secret value: %w", err)
		}
		secret.VersionId = versionId

		if secret.HasRotation() {
			if err := uc.repo.UpdateRotation(ctx, secret.SecretName, secret.RotationLambdaARN, secret.AutomaticallyAfterDays); err != nil {
				return fmt.Errorf("failed to update rotation: %w", err)
			}
		}
	}

	updated, err := uc.repo.Get(ctx, secret.SecretName)
	if err != nil {
		return fmt.Errorf("failed to get updated secret: %w", err)
	}

	secret.ARN = updated.ARN
	secret.CreatedAt = updated.CreatedAt
	now := time.Now()
	secret.LastSyncTime = &now

	return nil
}

func (uc *SecretUseCase) DeleteSecret(ctx context.Context, secret *secretsmanager.Secret) error {
	if !secret.ShouldDelete() {
		return nil
	}

	exists, err := uc.repo.Exists(ctx, secret.SecretName)
	if err != nil {
		return fmt.Errorf("failed to check if secret exists: %w", err)
	}

	if !exists {
		return nil
	}

	forceDelete := secret.RecoveryWindowInDays == 0
	if err := uc.repo.Delete(ctx, secret.SecretName, secret.RecoveryWindowInDays, forceDelete); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
