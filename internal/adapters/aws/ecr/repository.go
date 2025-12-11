package ecr

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"infra-operator/internal/domain/ecr"
	"infra-operator/internal/ports"
)

type Repository struct {
	client *awsecr.Client
}

func NewRepository(awsConfig aws.Config) ports.ECRRepository {
	var options []func(*awsecr.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awsecr.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	return &Repository{
		client: awsecr.NewFromConfig(awsConfig, options...),
	}
}

func (r *Repository) Exists(ctx context.Context, repositoryName string) (bool, error) {
	_, err := r.client.DescribeRepositories(ctx, &awsecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repositoryName},
	})
	if err != nil {
		var notFoundErr *types.RepositoryNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if repository exists: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, repository *ecr.Repository) error {
	input := &awsecr.CreateRepositoryInput{
		RepositoryName: aws.String(repository.RepositoryName),
		Tags:           convertTags(repository.Tags),
	}

	// Set image tag mutability
	if repository.ImageTagMutability != "" {
		input.ImageTagMutability = types.ImageTagMutability(repository.ImageTagMutability)
	}

	// Set image scanning configuration
	input.ImageScanningConfiguration = &types.ImageScanningConfiguration{
		ScanOnPush: repository.ScanOnPush,
	}

	// Set encryption configuration
	if repository.EncryptionType != "" {
		input.EncryptionConfiguration = &types.EncryptionConfiguration{
			EncryptionType: types.EncryptionType(repository.EncryptionType),
		}
		if repository.EncryptionType == "KMS" && repository.KmsKey != "" {
			input.EncryptionConfiguration.KmsKey = aws.String(repository.KmsKey)
		}
	}

	output, err := r.client.CreateRepository(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Update repository with created information
	if output.Repository != nil {
		repository.RepositoryArn = aws.ToString(output.Repository.RepositoryArn)
		repository.RepositoryUri = aws.ToString(output.Repository.RepositoryUri)
		repository.RegistryId = aws.ToString(output.Repository.RegistryId)
		if output.Repository.CreatedAt != nil {
			createdAt := *output.Repository.CreatedAt
			repository.CreatedAt = &createdAt
		}
	}

	// Set lifecycle policy if provided
	if repository.LifecyclePolicyText != "" {
		if err := r.PutLifecyclePolicy(ctx, repository.RepositoryName, repository.LifecyclePolicyText); err != nil {
			return fmt.Errorf("failed to set lifecycle policy: %w", err)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, repositoryName string) (*ecr.Repository, error) {
	output, err := r.client.DescribeRepositories(ctx, &awsecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repositoryName},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if len(output.Repositories) == 0 {
		return nil, fmt.Errorf("repository not found")
	}

	return mapToRepository(&output.Repositories[0]), nil
}

func (r *Repository) UpdateImageScanning(ctx context.Context, repositoryName string, scanOnPush bool) error {
	_, err := r.client.PutImageScanningConfiguration(ctx, &awsecr.PutImageScanningConfigurationInput{
		RepositoryName: aws.String(repositoryName),
		ImageScanningConfiguration: &types.ImageScanningConfiguration{
			ScanOnPush: scanOnPush,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update image scanning: %w", err)
	}
	return nil
}

func (r *Repository) UpdateImageTagMutability(ctx context.Context, repositoryName string, mutability string) error {
	_, err := r.client.PutImageTagMutability(ctx, &awsecr.PutImageTagMutabilityInput{
		RepositoryName:     aws.String(repositoryName),
		ImageTagMutability: types.ImageTagMutability(mutability),
	})
	if err != nil {
		return fmt.Errorf("failed to update image tag mutability: %w", err)
	}
	return nil
}

func (r *Repository) PutLifecyclePolicy(ctx context.Context, repositoryName, policyText string) error {
	_, err := r.client.PutLifecyclePolicy(ctx, &awsecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(repositoryName),
		LifecyclePolicyText: aws.String(policyText),
	})
	if err != nil {
		return fmt.Errorf("failed to put lifecycle policy: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, repositoryName string) error {
	_, err := r.client.DeleteRepository(ctx, &awsecr.DeleteRepositoryInput{
		RepositoryName: aws.String(repositoryName),
		Force:          true, // Delete even if contains images
	})
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awsecr.TagResourceInput{
		ResourceArn: aws.String(arn),
		Tags:        convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag resource: %w", err)
	}
	return nil
}

func (r *Repository) GetImageCount(ctx context.Context, repositoryName string) (int64, error) {
	output, err := r.client.ListImages(ctx, &awsecr.ListImagesInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list images: %w", err)
	}
	return int64(len(output.ImageIds)), nil
}

func convertTags(tags map[string]string) []types.Tag {
	var ecrTags []types.Tag
	for k, v := range tags {
		ecrTags = append(ecrTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return ecrTags
}

func mapToRepository(repo *types.Repository) *ecr.Repository {
	repository := &ecr.Repository{
		RepositoryName: aws.ToString(repo.RepositoryName),
		RepositoryArn:  aws.ToString(repo.RepositoryArn),
		RepositoryUri:  aws.ToString(repo.RepositoryUri),
		RegistryId:     aws.ToString(repo.RegistryId),
	}

	if repo.ImageTagMutability != "" {
		repository.ImageTagMutability = string(repo.ImageTagMutability)
	}

	if repo.ImageScanningConfiguration != nil {
		repository.ScanOnPush = repo.ImageScanningConfiguration.ScanOnPush
	}

	if repo.EncryptionConfiguration != nil {
		repository.EncryptionType = string(repo.EncryptionConfiguration.EncryptionType)
		repository.KmsKey = aws.ToString(repo.EncryptionConfiguration.KmsKey)
	}

	if repo.CreatedAt != nil {
		createdAt := *repo.CreatedAt
		repository.CreatedAt = &createdAt
	}

	return repository
}
