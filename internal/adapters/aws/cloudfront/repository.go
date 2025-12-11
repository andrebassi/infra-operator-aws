package cloudfront

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscf "github.com/aws/aws-sdk-go-v2/service/cloudfront"

	"infra-operator/internal/domain/cloudfront"
)

type Repository struct {
	client *awscf.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awscf.NewFromConfig(cfg)}
}

func (r *Repository) Exists(ctx context.Context, distributionID string) (bool, error) {
	_, err := r.client.GetDistribution(ctx, &awscf.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, dist *cloudfront.Distribution) error {
	// Note: CloudFront CreateDistribution requires complex DistributionConfig
	// This is a simplified implementation stub
	return fmt.Errorf("CloudFront creation requires complex configuration - implement as needed")
}

func (r *Repository) Get(ctx context.Context, distributionID string) (*cloudfront.Distribution, error) {
	output, err := r.client.GetDistribution(ctx, &awscf.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution: %w", err)
	}

	dist := &cloudfront.Distribution{
		DistributionID: aws.ToString(output.Distribution.Id),
		DomainName:     aws.ToString(output.Distribution.DomainName),
		Status:         aws.ToString(output.Distribution.Status),
		ETag:           aws.ToString(output.ETag),
	}

	if output.Distribution.DistributionConfig != nil {
		dist.Comment = aws.ToString(output.Distribution.DistributionConfig.Comment)
		dist.Enabled = aws.ToBool(output.Distribution.DistributionConfig.Enabled)
	}

	return dist, nil
}

func (r *Repository) Update(ctx context.Context, dist *cloudfront.Distribution) error {
	// Note: CloudFront UpdateDistribution requires full DistributionConfig + ETag
	// This is a simplified implementation stub
	return fmt.Errorf("CloudFront update requires complex configuration - implement as needed")
}

func (r *Repository) Delete(ctx context.Context, distributionID, etag string) error {
	_, err := r.client.DeleteDistribution(ctx, &awscf.DeleteDistributionInput{
		Id:      aws.String(distributionID),
		IfMatch: aws.String(etag),
	})
	if err != nil {
		return fmt.Errorf("failed to delete distribution: %w", err)
	}
	return nil
}
