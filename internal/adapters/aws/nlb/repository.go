package nlb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awselbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"infra-operator/internal/domain/nlb"
)

// Repository handles NLB operations using AWS SDK
type Repository struct {
	client *awselbv2.Client
}

// NewRepository creates a new NLB repository
func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awselbv2.NewFromConfig(cfg),
	}
}

// Exists checks if an NLB exists
func (r *Repository) Exists(ctx context.Context, lbName string) (bool, error) {
	input := &awselbv2.DescribeLoadBalancersInput{
		Names: []string{lbName},
	}

	output, err := r.client.DescribeLoadBalancers(ctx, input)
	if err != nil {
		return false, nil
	}

	return len(output.LoadBalancers) > 0, nil
}

// Create creates a new NLB
func (r *Repository) Create(ctx context.Context, lb *nlb.LoadBalancer) error {
	input := &awselbv2.CreateLoadBalancerInput{
		Name:    aws.String(lb.LoadBalancerName),
		Subnets: lb.Subnets,
		Scheme:  types.LoadBalancerSchemeEnum(lb.Scheme),
		Type:    types.LoadBalancerTypeEnumNetwork,
	}

	if lb.IPAddressType != "" {
		input.IpAddressType = types.IpAddressType(lb.IPAddressType)
	}

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
		return fmt.Errorf("failed to create NLB: %w", err)
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
		return fmt.Errorf("failed to set NLB attributes: %w", err)
	}

	return nil
}

// Get retrieves an NLB
func (r *Repository) Get(ctx context.Context, lbName string) (*nlb.LoadBalancer, error) {
	input := &awselbv2.DescribeLoadBalancersInput{
		Names: []string{lbName},
	}

	output, err := r.client.DescribeLoadBalancers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe NLB: %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, fmt.Errorf("NLB not found")
	}

	lbData := output.LoadBalancers[0]
	lb := &nlb.LoadBalancer{
		LoadBalancerName:      lbName,
		LoadBalancerARN:       aws.ToString(lbData.LoadBalancerArn),
		DNSName:               aws.ToString(lbData.DNSName),
		State:                 string(lbData.State.Code),
		VpcID:                 aws.ToString(lbData.VpcId),
		CanonicalHostedZoneID: aws.ToString(lbData.CanonicalHostedZoneId),
	}

	return lb, nil
}

// Delete deletes an NLB
func (r *Repository) Delete(ctx context.Context, lbARN string) error {
	input := &awselbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(lbARN),
	}

	_, err := r.client.DeleteLoadBalancer(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete NLB: %w", err)
	}

	return nil
}

// SetAttributes sets NLB attributes
func (r *Repository) SetAttributes(ctx context.Context, lb *nlb.LoadBalancer) error {
	attributes := []types.LoadBalancerAttribute{
		{
			Key:   aws.String("deletion_protection.enabled"),
			Value: aws.String(fmt.Sprintf("%t", lb.EnableDeletionProtection)),
		},
		{
			Key:   aws.String("load_balancing.cross_zone.enabled"),
			Value: aws.String(fmt.Sprintf("%t", lb.EnableCrossZoneLoadBalancing)),
		},
	}

	input := &awselbv2.ModifyLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(lb.LoadBalancerARN),
		Attributes:      attributes,
	}

	_, err := r.client.ModifyLoadBalancerAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to modify NLB attributes: %w", err)
	}

	return nil
}

// GetAttributes gets NLB attributes
func (r *Repository) GetAttributes(ctx context.Context, lb *nlb.LoadBalancer) error {
	input := &awselbv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(lb.LoadBalancerARN),
	}

	output, err := r.client.DescribeLoadBalancerAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe NLB attributes: %w", err)
	}

	for _, attr := range output.Attributes {
		key := aws.ToString(attr.Key)
		value := aws.ToString(attr.Value)

		switch key {
		case "deletion_protection.enabled":
			lb.EnableDeletionProtection = value == "true"
		case "load_balancing.cross_zone.enabled":
			lb.EnableCrossZoneLoadBalancing = value == "true"
		}
	}

	return nil
}

// TagResource adds or updates tags on an NLB
func (r *Repository) TagResource(ctx context.Context, lbARN string, tags map[string]string) error {
	var awsTags []types.Tag
	for key, value := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	input := &awselbv2.AddTagsInput{
		ResourceArns: []string{lbARN},
		Tags:         awsTags,
	}

	_, err := r.client.AddTags(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag NLB: %w", err)
	}

	return nil
}
