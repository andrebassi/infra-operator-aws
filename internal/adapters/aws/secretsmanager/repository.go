package secretsmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"infra-operator/internal/domain/secretsmanager"
)

type Repository struct {
	client *awssm.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awssm.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, secretName string) (bool, error) {
	_, err := r.client.DescribeSecret(ctx, &awssm.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		var nf *types.ResourceNotFoundException
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if secret exists: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, secret *secretsmanager.Secret) error {
	input := &awssm.CreateSecretInput{
		Name:        aws.String(secret.SecretName),
		Description: aws.String(secret.Description),
	}

	if secret.SecretString != "" {
		input.SecretString = aws.String(secret.SecretString)
	} else if len(secret.SecretBinary) > 0 {
		input.SecretBinary = secret.SecretBinary
	}

	if secret.KmsKeyId != "" {
		input.KmsKeyId = aws.String(secret.KmsKeyId)
	}

	if len(secret.Tags) > 0 {
		input.Tags = convertTags(secret.Tags)
	}

	output, err := r.client.CreateSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	secret.ARN = aws.ToString(output.ARN)
	secret.VersionId = aws.ToString(output.VersionId)

	return nil
}

func (r *Repository) Get(ctx context.Context, secretName string) (*secretsmanager.Secret, error) {
	descOutput, err := r.client.DescribeSecret(ctx, &awssm.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe secret: %w", err)
	}

	secret := &secretsmanager.Secret{
		SecretName:  aws.ToString(descOutput.Name),
		ARN:         aws.ToString(descOutput.ARN),
		Description: aws.ToString(descOutput.Description),
		KmsKeyId:    aws.ToString(descOutput.KmsKeyId),
	}

	if descOutput.RotationEnabled != nil {
		secret.RotationEnabled = *descOutput.RotationEnabled
	}
	if descOutput.RotationLambdaARN != nil {
		secret.RotationLambdaARN = aws.ToString(descOutput.RotationLambdaARN)
	}

	if descOutput.CreatedDate != nil {
		t := *descOutput.CreatedDate
		secret.CreatedAt = &t
	}

	return secret, nil
}

func (r *Repository) UpdateSecretValue(ctx context.Context, secretName, secretString string, secretBinary []byte) (string, error) {
	input := &awssm.PutSecretValueInput{
		SecretId: aws.String(secretName),
	}

	if secretString != "" {
		input.SecretString = aws.String(secretString)
	} else if len(secretBinary) > 0 {
		input.SecretBinary = secretBinary
	}

	output, err := r.client.PutSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to update secret value: %w", err)
	}

	return aws.ToString(output.VersionId), nil
}

func (r *Repository) UpdateRotation(ctx context.Context, secretName, lambdaARN string, days int32) error {
	_, err := r.client.RotateSecret(ctx, &awssm.RotateSecretInput{
		SecretId:          aws.String(secretName),
		RotationLambdaARN: aws.String(lambdaARN),
		RotationRules: &types.RotationRulesType{
			AutomaticallyAfterDays: aws.Int64(int64(days)),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update rotation: %w", err)
	}
	return nil
}

func (r *Repository) DisableRotation(ctx context.Context, secretName string) error {
	_, err := r.client.CancelRotateSecret(ctx, &awssm.CancelRotateSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return fmt.Errorf("failed to disable rotation: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, secretName string, recoveryWindowInDays int32, forceDelete bool) error {
	input := &awssm.DeleteSecretInput{
		SecretId: aws.String(secretName),
	}

	if forceDelete {
		input.ForceDeleteWithoutRecovery = aws.Bool(true)
	} else {
		input.RecoveryWindowInDays = aws.Int64(int64(recoveryWindowInDays))
	}

	_, err := r.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, secretARN string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awssm.TagResourceInput{
		SecretId: aws.String(secretARN),
		Tags:     convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag secret: %w", err)
	}
	return nil
}

func convertTags(tags map[string]string) []types.Tag {
	smTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		smTags = append(smTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return smTags
}
