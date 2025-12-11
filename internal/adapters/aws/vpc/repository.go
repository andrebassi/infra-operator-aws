package vpc

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/vpc"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsec2.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, vpcID string) (bool, error) {
	output, err := r.client.DescribeVpcs(ctx, &awsec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		return false, nil
	}
	return len(output.Vpcs) > 0, nil
}

func (r *Repository) Create(ctx context.Context, v *vpc.VPC) error {
	input := &awsec2.CreateVpcInput{
		CidrBlock: aws.String(v.CidrBlock),
	}

	if v.InstanceTenancy != "" {
		input.InstanceTenancy = types.Tenancy(v.InstanceTenancy)
	}

	output, err := r.client.CreateVpc(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}

	if output.Vpc != nil {
		v.VpcID = aws.ToString(output.Vpc.VpcId)
		v.State = string(output.Vpc.State)
		v.IsDefault = aws.ToBool(output.Vpc.IsDefault)
	}

	// Enable DNS support and hostnames if requested
	if v.EnableDnsSupport {
		if _, err := r.client.ModifyVpcAttribute(ctx, &awsec2.ModifyVpcAttributeInput{
			VpcId:            aws.String(v.VpcID),
			EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		}); err != nil {
			return fmt.Errorf("failed to enable DNS support: %w", err)
		}
	}

	if v.EnableDnsHostnames {
		if _, err := r.client.ModifyVpcAttribute(ctx, &awsec2.ModifyVpcAttributeInput{
			VpcId:              aws.String(v.VpcID),
			EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		}); err != nil {
			return fmt.Errorf("failed to enable DNS hostnames: %w", err)
		}
	}

	// Apply tags
	if len(v.Tags) > 0 {
		if err := r.TagResource(ctx, v.VpcID, v.Tags); err != nil {
			return fmt.Errorf("failed to tag VPC: %w", err)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, vpcID string) (*vpc.VPC, error) {
	output, err := r.client.DescribeVpcs(ctx, &awsec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC: %w", err)
	}

	if len(output.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC not found")
	}

	v := output.Vpcs[0]
	return &vpc.VPC{
		VpcID:     aws.ToString(v.VpcId),
		CidrBlock: aws.ToString(v.CidrBlock),
		State:     string(v.State),
		IsDefault: aws.ToBool(v.IsDefault),
	}, nil
}

func (r *Repository) Delete(ctx context.Context, vpcID string) error {
	_, err := r.client.DeleteVpc(ctx, &awsec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VPC: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, vpcID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{vpcID},
		Tags:      ec2Tags,
	})
	return err
}
