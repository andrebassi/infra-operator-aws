package s3

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"infra-operator/internal/domain/s3"
	"infra-operator/internal/ports"
)

// Repository implements ports.S3Repository using AWS SDK
// This is an Adapter in Hexagonal Architecture
type Repository struct {
	client *awss3.Client
}

// NewRepository creates a new S3 repository
func NewRepository(awsConfig aws.Config) ports.S3Repository {
	// Check for custom endpoint (LocalStack)
	var options []func(*awss3.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awss3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // LocalStack requires path-style URLs
		})
	}

	return &Repository{
		client: awss3.NewFromConfig(awsConfig, options...),
	}
}

// Exists checks if bucket exists
func (r *Repository) Exists(ctx context.Context, name, region string) (bool, error) {
	_, err := r.client.HeadBucket(ctx, &awss3.HeadBucketInput{
		Bucket: aws.String(name),
	})

	if err != nil {
		// Check if error is 404
		return false, nil
	}

	return true, nil
}

// Create creates a new S3 bucket
func (r *Repository) Create(ctx context.Context, bucket *s3.Bucket) error {
	if err := bucket.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	input := &awss3.CreateBucketInput{
		Bucket: aws.String(bucket.Name),
	}

	// Set location constraint if not us-east-1
	if bucket.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(bucket.Region),
		}
	}

	_, err := r.client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// Get retrieves bucket information
func (r *Repository) Get(ctx context.Context, name, region string) (*s3.Bucket, error) {
	// Check if bucket exists
	exists, err := r.Exists(ctx, name, region)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, s3.ErrBucketNotFound
	}

	bucket := &s3.Bucket{
		Name:       name,
		Region:     region,
		ARN:        fmt.Sprintf("arn:aws:s3:::%s", name),
		DomainName: fmt.Sprintf("%s.s3.amazonaws.com", name),
	}

	// Get versioning
	versioning, err := r.getVersioning(ctx, name)
	if err == nil {
		bucket.Versioning = versioning
	}

	// Get encryption
	encryption, err := r.getEncryption(ctx, name)
	if err == nil {
		bucket.Encryption = encryption
	}

	// Get tags
	tags, err := r.getTags(ctx, name)
	if err == nil {
		bucket.Tags = tags
	}

	return bucket, nil
}

// Update updates bucket configuration
func (r *Repository) Update(ctx context.Context, bucket *s3.Bucket) error {
	return r.Configure(ctx, bucket)
}

// Delete deletes a bucket
func (r *Repository) Delete(ctx context.Context, name, region string) error {
	_, err := r.client.DeleteBucket(ctx, &awss3.DeleteBucketInput{
		Bucket: aws.String(name),
	})

	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// Configure applies all configurations to bucket
func (r *Repository) Configure(ctx context.Context, bucket *s3.Bucket) error {
	// Configure versioning
	if bucket.Versioning != nil {
		if err := r.ConfigureVersioning(ctx, bucket.Name, bucket.Region, bucket.Versioning); err != nil {
			return err
		}
	}

	// Configure encryption
	if bucket.Encryption != nil {
		if err := r.ConfigureEncryption(ctx, bucket.Name, bucket.Region, bucket.Encryption); err != nil {
			return err
		}
	}

	// Configure lifecycle
	if len(bucket.LifecycleRules) > 0 {
		if err := r.ConfigureLifecycle(ctx, bucket.Name, bucket.Region, bucket.LifecycleRules); err != nil {
			return err
		}
	}

	// Configure CORS
	if len(bucket.CORSRules) > 0 {
		if err := r.ConfigureCORS(ctx, bucket.Name, bucket.Region, bucket.CORSRules); err != nil {
			return err
		}
	}

	// Configure public access block
	if bucket.PublicAccessBlock != nil {
		if err := r.ConfigurePublicAccessBlock(ctx, bucket.Name, bucket.Region, bucket.PublicAccessBlock); err != nil {
			return err
		}
	}

	// Configure tags
	if len(bucket.Tags) > 0 {
		if err := r.ConfigureTags(ctx, bucket.Name, bucket.Region, bucket.Tags); err != nil {
			return err
		}
	}

	return nil
}

// ConfigureVersioning configures versioning
func (r *Repository) ConfigureVersioning(ctx context.Context, name, region string, config *s3.VersioningConfig) error {
	status := types.BucketVersioningStatusSuspended
	if config.Enabled {
		status = types.BucketVersioningStatusEnabled
	}

	_, err := r.client.PutBucketVersioning(ctx, &awss3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: status,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure versioning: %w", err)
	}

	return nil
}

// ConfigureEncryption configures encryption
func (r *Repository) ConfigureEncryption(ctx context.Context, name, region string, config *s3.EncryptionConfig) error {
	var rule types.ServerSideEncryptionRule

	if config.Algorithm == "AES256" {
		rule = types.ServerSideEncryptionRule{
			ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
				SSEAlgorithm: types.ServerSideEncryptionAes256,
			},
		}
	} else if config.Algorithm == "aws:kms" {
		rule = types.ServerSideEncryptionRule{
			ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
				SSEAlgorithm:   types.ServerSideEncryptionAwsKms,
				KMSMasterKeyID: aws.String(config.KMSKeyID),
			},
		}
	}

	_, err := r.client.PutBucketEncryption(ctx, &awss3.PutBucketEncryptionInput{
		Bucket: aws.String(name),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{rule},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure encryption: %w", err)
	}

	return nil
}

// ConfigureLifecycle configures lifecycle rules
func (r *Repository) ConfigureLifecycle(ctx context.Context, name, region string, rules []s3.LifecycleRule) error {
	// Convert domain rules to AWS types
	var awsRules []types.LifecycleRule

	for _, rule := range rules {
		awsRule := types.LifecycleRule{
			ID:     aws.String(rule.ID),
			Status: types.ExpirationStatusEnabled,
		}

		if !rule.Enabled {
			awsRule.Status = types.ExpirationStatusDisabled
		}

		if rule.Prefix != "" {
			awsRule.Filter = &types.LifecycleRuleFilterMemberPrefix{
				Value: rule.Prefix,
			}
		}

		if rule.Expiration != nil {
			awsRule.Expiration = &types.LifecycleExpiration{
				Days: aws.Int32(rule.Expiration.Days),
			}
		}

		if len(rule.Transitions) > 0 {
			var transitions []types.Transition
			for _, t := range rule.Transitions {
				transitions = append(transitions, types.Transition{
					Days:         aws.Int32(t.Days),
					StorageClass: types.TransitionStorageClass(t.StorageClass),
				})
			}
			awsRule.Transitions = transitions
		}

		awsRules = append(awsRules, awsRule)
	}

	_, err := r.client.PutBucketLifecycleConfiguration(ctx, &awss3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(name),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: awsRules,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure lifecycle: %w", err)
	}

	return nil
}

// ConfigureCORS configures CORS rules
func (r *Repository) ConfigureCORS(ctx context.Context, name, region string, rules []s3.CORSRule) error {
	var awsRules []types.CORSRule

	for _, rule := range rules {
		awsRules = append(awsRules, types.CORSRule{
			AllowedOrigins: rule.AllowedOrigins,
			AllowedMethods: rule.AllowedMethods,
			AllowedHeaders: rule.AllowedHeaders,
			ExposeHeaders:  rule.ExposeHeaders,
			MaxAgeSeconds:  aws.Int32(rule.MaxAgeSeconds),
		})
	}

	_, err := r.client.PutBucketCors(ctx, &awss3.PutBucketCorsInput{
		Bucket: aws.String(name),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: awsRules,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure CORS: %w", err)
	}

	return nil
}

// ConfigurePublicAccessBlock configures public access block
func (r *Repository) ConfigurePublicAccessBlock(ctx context.Context, name, region string, config *s3.PublicAccessBlockConfig) error {
	_, err := r.client.PutPublicAccessBlock(ctx, &awss3.PutPublicAccessBlockInput{
		Bucket: aws.String(name),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(config.BlockPublicAcls),
			IgnorePublicAcls:      aws.Bool(config.IgnorePublicAcls),
			BlockPublicPolicy:     aws.Bool(config.BlockPublicPolicy),
			RestrictPublicBuckets: aws.Bool(config.RestrictPublicBuckets),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure public access block: %w", err)
	}

	return nil
}

// ConfigureTags applies tags
func (r *Repository) ConfigureTags(ctx context.Context, name, region string, tags map[string]string) error {
	var awsTags []types.Tag
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := r.client.PutBucketTagging(ctx, &awss3.PutBucketTaggingInput{
		Bucket: aws.String(name),
		Tagging: &types.Tagging{
			TagSet: awsTags,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure tags: %w", err)
	}

	return nil
}

// Helper methods

func (r *Repository) getVersioning(ctx context.Context, name string) (*s3.VersioningConfig, error) {
	output, err := r.client.GetBucketVersioning(ctx, &awss3.GetBucketVersioningInput{
		Bucket: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	return &s3.VersioningConfig{
		Enabled: output.Status == types.BucketVersioningStatusEnabled,
	}, nil
}

func (r *Repository) getEncryption(ctx context.Context, name string) (*s3.EncryptionConfig, error) {
	output, err := r.client.GetBucketEncryption(ctx, &awss3.GetBucketEncryptionInput{
		Bucket: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	if len(output.ServerSideEncryptionConfiguration.Rules) == 0 {
		return nil, nil
	}

	rule := output.ServerSideEncryptionConfiguration.Rules[0]
	config := &s3.EncryptionConfig{
		Algorithm: string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm),
	}

	if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
		config.KMSKeyID = *rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID
	}

	return config, nil
}

func (r *Repository) getTags(ctx context.Context, name string) (map[string]string, error) {
	output, err := r.client.GetBucketTagging(ctx, &awss3.GetBucketTaggingInput{
		Bucket: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, tag := range output.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}
