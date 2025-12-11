package alb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awselbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"infra-operator/internal/domain/alb"
)

type Repository struct {
	client *awselbv2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awselbv2.NewFromConfig(cfg),
	}
}

// Exists checks if a load balancer exists
func (r *Repository) Exists(ctx context.Context, lbName string) (bool, error) {
	input := &awselbv2.DescribeLoadBalancersInput{
		Names: []string{lbName},
	}

	_, err := r.client.DescribeLoadBalancers(ctx, input)
	if err != nil {
		// Check if it's a not found error
		return false, nil
	}

	return true, nil
}

// Create creates a new Application Load Balancer
func (r *Repository) Create(ctx context.Context, lb *alb.LoadBalancer) error {
	input := &awselbv2.CreateLoadBalancerInput{
		Name:    aws.String(lb.LoadBalancerName),
		Subnets: lb.Subnets,
		Scheme:  types.LoadBalancerSchemeEnum(lb.Scheme),
		Type:    types.LoadBalancerTypeEnumApplication,
	}

	// Add security groups if provided
	if len(lb.SecurityGroups) > 0 {
		input.SecurityGroups = lb.SecurityGroups
	}

	// Set IP address type
	if lb.IPAddressType != "" {
		input.IpAddressType = types.IpAddressType(lb.IPAddressType)
	}

	// Add tags
	if len(lb.Tags) > 0 {
		var tags []types.Tag
		for key, value := range lb.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}
		input.Tags = tags
	}

	output, err := r.client.CreateLoadBalancer(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	if len(output.LoadBalancers) > 0 {
		lbData := output.LoadBalancers[0]
		lb.LoadBalancerARN = aws.ToString(lbData.LoadBalancerArn)
		lb.DNSName = aws.ToString(lbData.DNSName)
		lb.CanonicalHostedZoneID = aws.ToString(lbData.CanonicalHostedZoneId)
		lb.State = string(lbData.State.Code)
		lb.VpcID = aws.ToString(lbData.VpcId)
	}

	// Set load balancer attributes
	if err := r.SetAttributes(ctx, lb); err != nil {
		return fmt.Errorf("failed to set load balancer attributes: %w", err)
	}

	return nil
}

// Get retrieves load balancer details
func (r *Repository) Get(ctx context.Context, lbName string) (*alb.LoadBalancer, error) {
	input := &awselbv2.DescribeLoadBalancersInput{
		Names: []string{lbName},
	}

	output, err := r.client.DescribeLoadBalancers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe load balancer: %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, fmt.Errorf("load balancer not found")
	}

	lbData := output.LoadBalancers[0]
	lb := &alb.LoadBalancer{
		LoadBalancerName:      aws.ToString(lbData.LoadBalancerName),
		LoadBalancerARN:       aws.ToString(lbData.LoadBalancerArn),
		DNSName:               aws.ToString(lbData.DNSName),
		Scheme:                string(lbData.Scheme),
		VpcID:                 aws.ToString(lbData.VpcId),
		State:                 string(lbData.State.Code),
		CanonicalHostedZoneID: aws.ToString(lbData.CanonicalHostedZoneId),
		IPAddressType:         string(lbData.IpAddressType),
	}

	// Get subnets
	for _, az := range lbData.AvailabilityZones {
		lb.Subnets = append(lb.Subnets, aws.ToString(az.SubnetId))
	}

	// Get security groups
	if len(lbData.SecurityGroups) > 0 {
		lb.SecurityGroups = lbData.SecurityGroups
	}

	// Get attributes
	if err := r.GetAttributes(ctx, lb); err != nil {
		return nil, fmt.Errorf("failed to get load balancer attributes: %w", err)
	}

	return lb, nil
}

// Delete deletes the load balancer
func (r *Repository) Delete(ctx context.Context, lbARN string) error {
	input := &awselbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(lbARN),
	}

	_, err := r.client.DeleteLoadBalancer(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	return nil
}

// SetAttributes sets load balancer attributes
func (r *Repository) SetAttributes(ctx context.Context, lb *alb.LoadBalancer) error {
	attributes := []types.LoadBalancerAttribute{
		{
			Key:   aws.String("deletion_protection.enabled"),
			Value: aws.String(fmt.Sprintf("%t", lb.EnableDeletionProtection)),
		},
		{
			Key:   aws.String("routing.http2.enabled"),
			Value: aws.String(fmt.Sprintf("%t", lb.EnableHttp2)),
		},
		{
			Key:   aws.String("waf.fail_open.enabled"),
			Value: aws.String(fmt.Sprintf("%t", lb.EnableWafFailOpen)),
		},
		{
			Key:   aws.String("idle_timeout.timeout_seconds"),
			Value: aws.String(fmt.Sprintf("%d", lb.IdleTimeout)),
		},
	}

	input := &awselbv2.ModifyLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(lb.LoadBalancerARN),
		Attributes:      attributes,
	}

	_, err := r.client.ModifyLoadBalancerAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to modify load balancer attributes: %w", err)
	}

	return nil
}

// GetAttributes retrieves load balancer attributes
func (r *Repository) GetAttributes(ctx context.Context, lb *alb.LoadBalancer) error {
	input := &awselbv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(lb.LoadBalancerARN),
	}

	output, err := r.client.DescribeLoadBalancerAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe load balancer attributes: %w", err)
	}

	for _, attr := range output.Attributes {
		switch aws.ToString(attr.Key) {
		case "deletion_protection.enabled":
			lb.EnableDeletionProtection = aws.ToString(attr.Value) == "true"
		case "routing.http2.enabled":
			lb.EnableHttp2 = aws.ToString(attr.Value) == "true"
		case "waf.fail_open.enabled":
			lb.EnableWafFailOpen = aws.ToString(attr.Value) == "true"
		case "idle_timeout.timeout_seconds":
			fmt.Sscanf(aws.ToString(attr.Value), "%d", &lb.IdleTimeout)
		}
	}

	return nil
}

// TagResource tags the load balancer
func (r *Repository) TagResource(ctx context.Context, lbARN string, tags map[string]string) error {
	var tagList []types.Tag
	for key, value := range tags {
		tagList = append(tagList, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	input := &awselbv2.AddTagsInput{
		ResourceArns: []string{lbARN},
		Tags:         tagList,
	}

	_, err := r.client.AddTags(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag load balancer: %w", err)
	}

	return nil
}
