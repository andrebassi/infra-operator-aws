package subnet

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/subnet"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsec2.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, subnetID string) (bool, error) {
	output, err := r.client.DescribeSubnets(ctx, &awsec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		return false, nil
	}
	return len(output.Subnets) > 0, nil
}

func (r *Repository) Create(ctx context.Context, s *subnet.Subnet) error {
	input := &awsec2.CreateSubnetInput{
		VpcId:     aws.String(s.VpcID),
		CidrBlock: aws.String(s.CidrBlock),
	}

	if s.AvailabilityZone != "" {
		input.AvailabilityZone = aws.String(s.AvailabilityZone)
	}

	output, err := r.client.CreateSubnet(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	if output.Subnet != nil {
		s.SubnetID = aws.ToString(output.Subnet.SubnetId)
		s.State = string(output.Subnet.State)
		s.AvailabilityZone = aws.ToString(output.Subnet.AvailabilityZone)
		s.AvailableIpAddressCount = aws.ToInt32(output.Subnet.AvailableIpAddressCount)
	}

	// Set MapPublicIpOnLaunch if requested
	if s.MapPublicIpOnLaunch {
		if _, err := r.client.ModifySubnetAttribute(ctx, &awsec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(s.SubnetID),
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		}); err != nil {
			return fmt.Errorf("failed to set MapPublicIpOnLaunch: %w", err)
		}
	}

	// Apply tags
	if len(s.Tags) > 0 {
		if err := r.TagResource(ctx, s.SubnetID, s.Tags); err != nil {
			return fmt.Errorf("failed to tag subnet: %w", err)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, subnetID string) (*subnet.Subnet, error) {
	output, err := r.client.DescribeSubnets(ctx, &awsec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnet: %w", err)
	}

	if len(output.Subnets) == 0 {
		return nil, fmt.Errorf("subnet not found")
	}

	s := output.Subnets[0]
	return &subnet.Subnet{
		SubnetID:                aws.ToString(s.SubnetId),
		VpcID:                   aws.ToString(s.VpcId),
		CidrBlock:               aws.ToString(s.CidrBlock),
		AvailabilityZone:        aws.ToString(s.AvailabilityZone),
		State:                   string(s.State),
		AvailableIpAddressCount: aws.ToInt32(s.AvailableIpAddressCount),
	}, nil
}

func (r *Repository) Delete(ctx context.Context, subnetID string) error {
	_, err := r.client.DeleteSubnet(ctx, &awsec2.DeleteSubnetInput{
		SubnetId: aws.String(subnetID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete subnet: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, subnetID string, tags map[string]string) error {
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
		Resources: []string{subnetID},
		Tags:      ec2Tags,
	})
	return err
}
