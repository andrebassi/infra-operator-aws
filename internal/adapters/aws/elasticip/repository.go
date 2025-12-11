package elasticip

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/elasticip"
)

// Repository handles Elastic IP operations using AWS SDK
type Repository struct {
	client *awsec2.Client
}

// NewRepository creates a new Elastic IP repository
func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsec2.NewFromConfig(cfg),
	}
}

// Allocate allocates a new Elastic IP address
func (r *Repository) Allocate(ctx context.Context, addr *elasticip.Address) error {
	input := &awsec2.AllocateAddressInput{
		Domain: types.DomainType(addr.Domain),
	}

	if addr.NetworkBorderGroup != "" {
		input.NetworkBorderGroup = aws.String(addr.NetworkBorderGroup)
	}

	if addr.PublicIpv4Pool != "" {
		input.PublicIpv4Pool = aws.String(addr.PublicIpv4Pool)
	}

	if addr.CustomerOwnedIpv4Pool != "" {
		input.CustomerOwnedIpv4Pool = aws.String(addr.CustomerOwnedIpv4Pool)
	}

	if len(addr.Tags) > 0 {
		var tagSpecs []types.TagSpecification
		var tags []types.Tag
		for key, value := range addr.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}
		tagSpecs = append(tagSpecs, types.TagSpecification{
			ResourceType: types.ResourceTypeElasticIp,
			Tags:         tags,
		})
		input.TagSpecifications = tagSpecs
	}

	output, err := r.client.AllocateAddress(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to allocate elastic IP: %w", err)
	}

	addr.AllocationID = aws.ToString(output.AllocationId)
	addr.PublicIP = aws.ToString(output.PublicIp)
	addr.Domain = string(output.Domain)

	return nil
}

// Exists checks if an Elastic IP exists by allocation ID
func (r *Repository) Exists(ctx context.Context, allocationID string) (bool, error) {
	input := &awsec2.DescribeAddressesInput{
		AllocationIds: []string{allocationID},
	}

	output, err := r.client.DescribeAddresses(ctx, input)
	if err != nil {
		// Check if error is because address doesn't exist
		return false, nil
	}

	return len(output.Addresses) > 0, nil
}

// Get retrieves an Elastic IP by allocation ID
func (r *Repository) Get(ctx context.Context, allocationID string) (*elasticip.Address, error) {
	input := &awsec2.DescribeAddressesInput{
		AllocationIds: []string{allocationID},
	}

	output, err := r.client.DescribeAddresses(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe elastic IP: %w", err)
	}

	if len(output.Addresses) == 0 {
		return nil, fmt.Errorf("elastic IP not found")
	}

	awsAddr := output.Addresses[0]
	addr := &elasticip.Address{
		AllocationID:       aws.ToString(awsAddr.AllocationId),
		PublicIP:           aws.ToString(awsAddr.PublicIp),
		AssociationID:      aws.ToString(awsAddr.AssociationId),
		InstanceID:         aws.ToString(awsAddr.InstanceId),
		NetworkInterfaceID: aws.ToString(awsAddr.NetworkInterfaceId),
		PrivateIPAddress:   aws.ToString(awsAddr.PrivateIpAddress),
		Domain:             string(awsAddr.Domain),
		NetworkBorderGroup: aws.ToString(awsAddr.NetworkBorderGroup),
		PublicIpv4Pool:     aws.ToString(awsAddr.PublicIpv4Pool),
	}

	// Populate tags
	if len(awsAddr.Tags) > 0 {
		addr.Tags = make(map[string]string)
		for _, tag := range awsAddr.Tags {
			addr.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return addr, nil
}

// Release releases an Elastic IP address
func (r *Repository) Release(ctx context.Context, allocationID string) error {
	input := &awsec2.ReleaseAddressInput{
		AllocationId: aws.String(allocationID),
	}

	_, err := r.client.ReleaseAddress(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to release elastic IP: %w", err)
	}

	return nil
}

// TagResource adds or updates tags on an Elastic IP
func (r *Repository) TagResource(ctx context.Context, allocationID string, tags map[string]string) error {
	var ec2Tags []types.Tag
	for key, value := range tags {
		ec2Tags = append(ec2Tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	input := &awsec2.CreateTagsInput{
		Resources: []string{allocationID},
		Tags:      ec2Tags,
	}

	_, err := r.client.CreateTags(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag elastic IP: %w", err)
	}

	return nil
}
