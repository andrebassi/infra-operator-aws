package natgateway

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/natgateway"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsec2.NewFromConfig(cfg)}
}

func (r *Repository) Exists(ctx context.Context, natGwID string) (bool, error) {
	output, err := r.client.DescribeNatGateways(ctx, &awsec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{natGwID},
	})
	if err != nil {
		return false, nil
	}
	return len(output.NatGateways) > 0, nil
}

func (r *Repository) Create(ctx context.Context, g *natgateway.Gateway) error {
	input := &awsec2.CreateNatGatewayInput{
		SubnetId:         aws.String(g.SubnetID),
		ConnectivityType: types.ConnectivityType(g.ConnectivityType),
	}

	// Allocate EIP if needed for public NAT gateway
	if g.AllocationID == "" && g.ConnectivityType == "public" {
		eipOutput, err := r.client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{
			Domain: types.DomainTypeVpc,
		})
		if err != nil {
			return fmt.Errorf("failed to allocate EIP: %w", err)
		}
		g.AllocationID = aws.ToString(eipOutput.AllocationId)
	}

	if g.AllocationID != "" {
		input.AllocationId = aws.String(g.AllocationID)
	}

	output, err := r.client.CreateNatGateway(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create NAT gateway: %w", err)
	}

	if output.NatGateway != nil {
		g.NatGatewayID = aws.ToString(output.NatGateway.NatGatewayId)
		g.State = string(output.NatGateway.State)
		g.VpcID = aws.ToString(output.NatGateway.VpcId)

		if len(output.NatGateway.NatGatewayAddresses) > 0 {
			g.PublicIP = aws.ToString(output.NatGateway.NatGatewayAddresses[0].PublicIp)
			g.PrivateIP = aws.ToString(output.NatGateway.NatGatewayAddresses[0].PrivateIp)
		}
	}

	// Apply tags
	if len(g.Tags) > 0 {
		r.TagResource(ctx, g.NatGatewayID, g.Tags)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, natGwID string) (*natgateway.Gateway, error) {
	output, err := r.client.DescribeNatGateways(ctx, &awsec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{natGwID},
	})
	if err != nil {
		return nil, err
	}
	if len(output.NatGateways) == 0 {
		return nil, fmt.Errorf("NAT gateway not found")
	}

	nat := output.NatGateways[0]
	g := &natgateway.Gateway{
		NatGatewayID: aws.ToString(nat.NatGatewayId),
		SubnetID:     aws.ToString(nat.SubnetId),
		VpcID:        aws.ToString(nat.VpcId),
		State:        string(nat.State),
	}

	if len(nat.NatGatewayAddresses) > 0 {
		g.PublicIP = aws.ToString(nat.NatGatewayAddresses[0].PublicIp)
		g.PrivateIP = aws.ToString(nat.NatGatewayAddresses[0].PrivateIp)
	}

	return g, nil
}

func (r *Repository) Delete(ctx context.Context, natGwID string) error {
	_, err := r.client.DeleteNatGateway(ctx, &awsec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(natGwID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete NAT gateway: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, natGwID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	_, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{natGwID},
		Tags:      ec2Tags,
	})
	return err
}
