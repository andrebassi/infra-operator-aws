package routetable

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"infra-operator/internal/domain/routetable"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsec2.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, routeTableID string) (bool, error) {
	_, err := r.client.DescribeRouteTables(ctx, &awsec2.DescribeRouteTablesInput{
		RouteTableIds: []string{routeTableID},
	})
	if err != nil {
		// Check if error is not found
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to describe route table: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, rt *routetable.RouteTable) error {
	// Create route table
	output, err := r.client.CreateRouteTable(ctx, &awsec2.CreateRouteTableInput{
		VpcId: aws.String(rt.VpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to create route table: %w", err)
	}

	rt.RouteTableID = aws.ToString(output.RouteTable.RouteTableId)

	// Add routes
	for _, route := range rt.Routes {
		if err := r.CreateRoute(ctx, rt.RouteTableID, route); err != nil {
			return fmt.Errorf("failed to create route: %w", err)
		}
	}

	// Associate subnets
	for _, subnetID := range rt.SubnetAssociations {
		if err := r.AssociateSubnet(ctx, rt.RouteTableID, subnetID); err != nil {
			return fmt.Errorf("failed to associate subnet: %w", err)
		}
	}

	// Add tags
	if len(rt.Tags) > 0 {
		if err := r.TagResource(ctx, rt.RouteTableID, rt.Tags); err != nil {
			return fmt.Errorf("failed to tag route table: %w", err)
		}
	}

	return nil
}

func (r *Repository) CreateRoute(ctx context.Context, routeTableID string, route routetable.Route) error {
	input := &awsec2.CreateRouteInput{
		RouteTableId: aws.String(routeTableID),
	}

	if route.DestinationCidrBlock != "" {
		input.DestinationCidrBlock = aws.String(route.DestinationCidrBlock)
	}

	if route.GatewayID != "" {
		input.GatewayId = aws.String(route.GatewayID)
	} else if route.NatGatewayID != "" {
		input.NatGatewayId = aws.String(route.NatGatewayID)
	} else if route.InstanceID != "" {
		input.InstanceId = aws.String(route.InstanceID)
	} else if route.NetworkInterfaceID != "" {
		input.NetworkInterfaceId = aws.String(route.NetworkInterfaceID)
	} else if route.VpcPeeringConnectionID != "" {
		input.VpcPeeringConnectionId = aws.String(route.VpcPeeringConnectionID)
	}

	_, err := r.client.CreateRoute(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}

	return nil
}

func (r *Repository) AssociateSubnet(ctx context.Context, routeTableID, subnetID string) error {
	_, err := r.client.AssociateRouteTable(ctx, &awsec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableID),
		SubnetId:     aws.String(subnetID),
	})
	if err != nil {
		return fmt.Errorf("failed to associate subnet: %w", err)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, routeTableID string) (*routetable.RouteTable, error) {
	output, err := r.client.DescribeRouteTables(ctx, &awsec2.DescribeRouteTablesInput{
		RouteTableIds: []string{routeTableID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe route table: %w", err)
	}

	if len(output.RouteTables) == 0 {
		return nil, fmt.Errorf("route table not found: %s", routeTableID)
	}

	awsRT := output.RouteTables[0]

	rt := &routetable.RouteTable{
		RouteTableID:       aws.ToString(awsRT.RouteTableId),
		VpcID:              aws.ToString(awsRT.VpcId),
		Routes:             []routetable.Route{},
		SubnetAssociations: []string{},
		Tags:               make(map[string]string),
	}

	// Convert routes
	for _, awsRoute := range awsRT.Routes {
		if awsRoute.DestinationCidrBlock != nil {
			rt.Routes = append(rt.Routes, routetable.Route{
				DestinationCidrBlock:   aws.ToString(awsRoute.DestinationCidrBlock),
				GatewayID:              aws.ToString(awsRoute.GatewayId),
				NatGatewayID:           aws.ToString(awsRoute.NatGatewayId),
				InstanceID:             aws.ToString(awsRoute.InstanceId),
				NetworkInterfaceID:     aws.ToString(awsRoute.NetworkInterfaceId),
				VpcPeeringConnectionID: aws.ToString(awsRoute.VpcPeeringConnectionId),
			})
		}
	}

	// Convert subnet associations
	for _, assoc := range awsRT.Associations {
		if assoc.SubnetId != nil {
			rt.SubnetAssociations = append(rt.SubnetAssociations, aws.ToString(assoc.SubnetId))
			rt.AssociatedSubnets = append(rt.AssociatedSubnets, aws.ToString(assoc.SubnetId))
		}
	}

	// Convert tags
	for _, tag := range awsRT.Tags {
		rt.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return rt, nil
}

func (r *Repository) Delete(ctx context.Context, routeTableID string) error {
	// First, disassociate all subnets
	output, err := r.client.DescribeRouteTables(ctx, &awsec2.DescribeRouteTablesInput{
		RouteTableIds: []string{routeTableID},
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to describe route table: %w", err)
	}

	if len(output.RouteTables) > 0 {
		for _, assoc := range output.RouteTables[0].Associations {
			if assoc.RouteTableAssociationId != nil && !aws.ToBool(assoc.Main) {
				_, err := r.client.DisassociateRouteTable(ctx, &awsec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				})
				if err != nil {
					return fmt.Errorf("failed to disassociate route table: %w", err)
				}
			}
		}
	}

	// Delete the route table
	_, err = r.client.DeleteRouteTable(ctx, &awsec2.DeleteRouteTableInput{
		RouteTableId: aws.String(routeTableID),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete route table: %w", err)
	}

	return nil
}

func (r *Repository) TagResource(ctx context.Context, routeTableID string, tags map[string]string) error {
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
		Resources: []string{routeTableID},
		Tags:      ec2Tags,
	})
	if err != nil {
		return fmt.Errorf("failed to tag route table: %w", err)
	}

	return nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() != "" && err.Error() == "InvalidRouteTableID.NotFound"
}
