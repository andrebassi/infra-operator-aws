package kms

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"infra-operator/internal/domain/kms"
)

type Repository struct {
	client *awskms.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awskms.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, keyId string) (bool, error) {
	_, err := r.client.DescribeKey(ctx, &awskms.DescribeKeyInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		var nf *types.NotFoundException
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if key exists: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, key *kms.Key) error {
	input := &awskms.CreateKeyInput{
		Description: aws.String(key.Description),
		KeyUsage:    types.KeyUsageType(key.KeyUsage),
		KeySpec:     types.KeySpec(key.KeySpec),
		MultiRegion: aws.Bool(key.MultiRegion),
	}

	if key.KeyPolicy != "" {
		input.Policy = aws.String(key.KeyPolicy)
	}

	if len(key.Tags) > 0 {
		input.Tags = convertTags(key.Tags)
	}

	output, err := r.client.CreateKey(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	key.KeyId = aws.ToString(output.KeyMetadata.KeyId)
	key.Arn = aws.ToString(output.KeyMetadata.Arn)
	key.KeyState = string(output.KeyMetadata.KeyState)
	if output.KeyMetadata.CreationDate != nil {
		t := *output.KeyMetadata.CreationDate
		key.CreatedAt = &t
	}

	// Enable key rotation if requested
	if key.EnableKeyRotation && key.IsSymmetric() {
		if err := r.EnableKeyRotation(ctx, key.KeyId); err != nil {
			return fmt.Errorf("failed to enable key rotation: %w", err)
		}
	}

	// Disable key if not enabled
	if !key.Enabled {
		if err := r.DisableKey(ctx, key.KeyId); err != nil {
			return fmt.Errorf("failed to disable key: %w", err)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, keyId string) (*kms.Key, error) {
	output, err := r.client.DescribeKey(ctx, &awskms.DescribeKeyInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe key: %w", err)
	}

	key := &kms.Key{
		KeyId:       aws.ToString(output.KeyMetadata.KeyId),
		Arn:         aws.ToString(output.KeyMetadata.Arn),
		Description: aws.ToString(output.KeyMetadata.Description),
		KeyUsage:    string(output.KeyMetadata.KeyUsage),
		KeySpec:     string(output.KeyMetadata.KeySpec),
		MultiRegion: aws.ToBool(output.KeyMetadata.MultiRegion),
		Enabled:     output.KeyMetadata.Enabled,
		KeyState:    string(output.KeyMetadata.KeyState),
	}

	if output.KeyMetadata.CreationDate != nil {
		t := *output.KeyMetadata.CreationDate
		key.CreatedAt = &t
	}

	// Check if key rotation is enabled (only for symmetric keys)
	if key.IsSymmetric() {
		rotationOutput, err := r.client.GetKeyRotationStatus(ctx, &awskms.GetKeyRotationStatusInput{
			KeyId: aws.String(keyId),
		})
		if err == nil {
			key.EnableKeyRotation = rotationOutput.KeyRotationEnabled
		}
	}

	return key, nil
}

func (r *Repository) Update(ctx context.Context, key *kms.Key) error {
	// Update description
	if key.Description != "" {
		_, err := r.client.UpdateKeyDescription(ctx, &awskms.UpdateKeyDescriptionInput{
			KeyId:       aws.String(key.KeyId),
			Description: aws.String(key.Description),
		})
		if err != nil {
			return fmt.Errorf("failed to update key description: %w", err)
		}
	}

	return nil
}

func (r *Repository) EnableKey(ctx context.Context, keyId string) error {
	_, err := r.client.EnableKey(ctx, &awskms.EnableKeyInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return fmt.Errorf("failed to enable key: %w", err)
	}
	return nil
}

func (r *Repository) DisableKey(ctx context.Context, keyId string) error {
	_, err := r.client.DisableKey(ctx, &awskms.DisableKeyInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return fmt.Errorf("failed to disable key: %w", err)
	}
	return nil
}

func (r *Repository) EnableKeyRotation(ctx context.Context, keyId string) error {
	_, err := r.client.EnableKeyRotation(ctx, &awskms.EnableKeyRotationInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return fmt.Errorf("failed to enable key rotation: %w", err)
	}
	return nil
}

func (r *Repository) DisableKeyRotation(ctx context.Context, keyId string) error {
	_, err := r.client.DisableKeyRotation(ctx, &awskms.DisableKeyRotationInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return fmt.Errorf("failed to disable key rotation: %w", err)
	}
	return nil
}

func (r *Repository) ScheduleKeyDeletion(ctx context.Context, keyId string, pendingWindowInDays int32) error {
	_, err := r.client.ScheduleKeyDeletion(ctx, &awskms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(keyId),
		PendingWindowInDays: aws.Int32(pendingWindowInDays),
	})
	if err != nil {
		return fmt.Errorf("failed to schedule key deletion: %w", err)
	}
	return nil
}

func (r *Repository) CancelKeyDeletion(ctx context.Context, keyId string) error {
	_, err := r.client.CancelKeyDeletion(ctx, &awskms.CancelKeyDeletionInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		return fmt.Errorf("failed to cancel key deletion: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, keyId string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awskms.TagResourceInput{
		KeyId: aws.String(keyId),
		Tags:  convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag key: %w", err)
	}
	return nil
}

func convertTags(tags map[string]string) []types.Tag {
	kmsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		kmsTags = append(kmsTags, types.Tag{
			TagKey:   aws.String(k),
			TagValue: aws.String(v),
		})
	}
	return kmsTags
}
