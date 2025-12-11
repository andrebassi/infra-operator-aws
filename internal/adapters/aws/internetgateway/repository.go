package internetgateway

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/internetgateway"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsec2.NewFromConfig(cfg)}
}

func (r *Repository) Exists(ctx context.Context, igwID string) (bool, error) {
	output, err := r.client.DescribeInternetGateways(ctx, &awsec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{igwID},
	})
	if err != nil {
		return false, nil
	}
	return len(output.InternetGateways) > 0, nil
}

func (r *Repository) Create(ctx context.Context, g *internetgateway.Gateway) error {
	output, err := r.client.CreateInternetGateway(ctx, &awsec2.CreateInternetGatewayInput{})
	if err != nil {
		return fmt.Errorf("failed to create internet gateway: %w", err)
	}

	if output.InternetGateway != nil {
		g.InternetGatewayID = aws.ToString(output.InternetGateway.InternetGatewayId)
	}

	// Attach to VPC
	if _, err := r.client.AttachInternetGateway(ctx, &awsec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(g.InternetGatewayID),
		VpcId:             aws.String(g.VpcID),
	}); err != nil {
		return fmt.Errorf("failed to attach internet gateway: %w", err)
	}

	g.State = "available"

	// Apply tags
	if len(g.Tags) > 0 {
		r.TagResource(ctx, g.InternetGatewayID, g.Tags)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, igwID string) (*internetgateway.Gateway, error) {
	output, err := r.client.DescribeInternetGateways(ctx, &awsec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{igwID},
	})
	if err != nil {
		return nil, err
	}
	if len(output.InternetGateways) == 0 {
		return nil, fmt.Errorf("internet gateway not found")
	}

	igw := output.InternetGateways[0]
	g := &internetgateway.Gateway{
		InternetGatewayID: aws.ToString(igw.InternetGatewayId),
	}

	if len(igw.Attachments) > 0 {
		g.VpcID = aws.ToString(igw.Attachments[0].VpcId)
		g.State = string(igw.Attachments[0].State)
	}

	return g, nil
}

func (r *Repository) Delete(ctx context.Context, igwID, vpcID string) error {
	// Detach from VPC first
	if vpcID != "" {
		if _, err := r.client.DetachInternetGateway(ctx, &awsec2.DetachInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
			VpcId:             aws.String(vpcID),
		}); err != nil {
			return fmt.Errorf("failed to detach internet gateway: %w", err)
		}
	}

	// Delete
	if _, err := r.client.DeleteInternetGateway(ctx, &awsec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
	}); err != nil {
		return fmt.Errorf("failed to delete internet gateway: %w", err)
	}

	return nil
}

func (r *Repository) TagResource(ctx context.Context, igwID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	_, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{igwID},
		Tags:      ec2Tags,
	})
	return err
}
