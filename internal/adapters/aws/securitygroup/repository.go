package securitygroup

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/securitygroup"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsec2.NewFromConfig(cfg)}
}

func (r *Repository) Exists(ctx context.Context, groupID string) (bool, error) {
	output, err := r.client.DescribeSecurityGroups(ctx, &awsec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
	})
	if err != nil {
		return false, nil
	}
	return len(output.SecurityGroups) > 0, nil
}

func (r *Repository) Create(ctx context.Context, sg *securitygroup.SecurityGroup) error {
	// Create security group
	output, err := r.client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sg.GroupName),
		Description: aws.String(sg.Description),
		VpcId:       aws.String(sg.VpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to create security group: %w", err)
	}

	sg.GroupID = aws.ToString(output.GroupId)

	// Add ingress rules if specified
	if len(sg.IngressRules) > 0 {
		if err := r.AuthorizeIngress(ctx, sg.GroupID, sg.IngressRules); err != nil {
			return fmt.Errorf("failed to authorize ingress rules: %w", err)
		}
	}

	// Add egress rules if specified (AWS creates default allow-all egress, so we need to revoke it first if custom rules are specified)
	if len(sg.EgressRules) > 0 {
		// Revoke default egress rule (0.0.0.0/0 all traffic)
		if _, err := r.client.RevokeSecurityGroupEgress(ctx, &awsec2.RevokeSecurityGroupEgressInput{
			GroupId: aws.String(sg.GroupID),
			IpPermissions: []types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					IpRanges: []types.IpRange{
						{CidrIp: aws.String("0.0.0.0/0")},
					},
				},
			},
		}); err != nil {
			// Ignore error if rule doesn't exist
		}

		if err := r.AuthorizeEgress(ctx, sg.GroupID, sg.EgressRules); err != nil {
			return fmt.Errorf("failed to authorize egress rules: %w", err)
		}
	}

	// Apply tags
	if len(sg.Tags) > 0 {
		if err := r.TagResource(ctx, sg.GroupID, sg.Tags); err != nil {
			return fmt.Errorf("failed to tag security group: %w", err)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, groupID string) (*securitygroup.SecurityGroup, error) {
	output, err := r.client.DescribeSecurityGroups(ctx, &awsec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe security group: %w", err)
	}

	if len(output.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group not found")
	}

	ec2SG := output.SecurityGroups[0]
	sg := &securitygroup.SecurityGroup{
		GroupID:     aws.ToString(ec2SG.GroupId),
		GroupName:   aws.ToString(ec2SG.GroupName),
		Description: aws.ToString(ec2SG.Description),
		VpcID:       aws.ToString(ec2SG.VpcId),
		Tags:        make(map[string]string),
	}

	// Convert ingress rules
	for _, perm := range ec2SG.IpPermissions {
		rule := convertPermissionToRule(perm)
		sg.IngressRules = append(sg.IngressRules, rule...)
	}

	// Convert egress rules
	for _, perm := range ec2SG.IpPermissionsEgress {
		rule := convertPermissionToRule(perm)
		sg.EgressRules = append(sg.EgressRules, rule...)
	}

	// Convert tags
	for _, tag := range ec2SG.Tags {
		sg.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	return sg, nil
}

func (r *Repository) Delete(ctx context.Context, groupID string) error {
	if _, err := r.client.DeleteSecurityGroup(ctx, &awsec2.DeleteSecurityGroupInput{
		GroupId: aws.String(groupID),
	}); err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}
	return nil
}

func (r *Repository) AuthorizeIngress(ctx context.Context, groupID string, rules []securitygroup.Rule) error {
	permissions := convertRulesToPermissions(rules)
	if len(permissions) == 0 {
		return nil
	}

	if _, err := r.client.AuthorizeSecurityGroupIngress(ctx, &awsec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       aws.String(groupID),
		IpPermissions: permissions,
	}); err != nil {
		return err
	}
	return nil
}

func (r *Repository) AuthorizeEgress(ctx context.Context, groupID string, rules []securitygroup.Rule) error {
	permissions := convertRulesToPermissions(rules)
	if len(permissions) == 0 {
		return nil
	}

	if _, err := r.client.AuthorizeSecurityGroupEgress(ctx, &awsec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       aws.String(groupID),
		IpPermissions: permissions,
	}); err != nil {
		return err
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, groupID string, tags map[string]string) error {
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

	if _, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{groupID},
		Tags:      ec2Tags,
	}); err != nil {
		return err
	}
	return nil
}

// Helper functions

func convertRulesToPermissions(rules []securitygroup.Rule) []types.IpPermission {
	var permissions []types.IpPermission

	for _, rule := range rules {
		perm := types.IpPermission{
			IpProtocol: aws.String(rule.IpProtocol),
		}

		// Set ports if not -1 (all)
		if rule.FromPort > 0 {
			perm.FromPort = aws.Int32(rule.FromPort)
		}
		if rule.ToPort > 0 {
			perm.ToPort = aws.Int32(rule.ToPort)
		}

		// Add CIDR blocks
		for _, cidr := range rule.CidrBlocks {
			perm.IpRanges = append(perm.IpRanges, types.IpRange{
				CidrIp:      aws.String(cidr),
				Description: aws.String(rule.Description),
			})
		}

		// Add IPv6 CIDR blocks
		for _, cidr := range rule.Ipv6CidrBlocks {
			perm.Ipv6Ranges = append(perm.Ipv6Ranges, types.Ipv6Range{
				CidrIpv6:    aws.String(cidr),
				Description: aws.String(rule.Description),
			})
		}

		// Add source security group
		if rule.SourceSecurityGroupID != "" {
			perm.UserIdGroupPairs = append(perm.UserIdGroupPairs, types.UserIdGroupPair{
				GroupId:     aws.String(rule.SourceSecurityGroupID),
				Description: aws.String(rule.Description),
			})
		}

		permissions = append(permissions, perm)
	}

	return permissions
}

func convertPermissionToRule(perm types.IpPermission) []securitygroup.Rule {
	var rules []securitygroup.Rule

	protocol := aws.ToString(perm.IpProtocol)
	fromPort := int32(0)
	toPort := int32(0)

	if perm.FromPort != nil {
		fromPort = *perm.FromPort
	}
	if perm.ToPort != nil {
		toPort = *perm.ToPort
	}

	// Create rules for each CIDR block
	for _, ipRange := range perm.IpRanges {
		rules = append(rules, securitygroup.Rule{
			IpProtocol:  protocol,
			FromPort:    fromPort,
			ToPort:      toPort,
			CidrBlocks:  []string{aws.ToString(ipRange.CidrIp)},
			Description: aws.ToString(ipRange.Description),
		})
	}

	// Create rules for each IPv6 CIDR block
	for _, ipv6Range := range perm.Ipv6Ranges {
		rules = append(rules, securitygroup.Rule{
			IpProtocol:     protocol,
			FromPort:       fromPort,
			ToPort:         toPort,
			Ipv6CidrBlocks: []string{aws.ToString(ipv6Range.CidrIpv6)},
			Description:    aws.ToString(ipv6Range.Description),
		})
	}

	// Create rules for each source security group
	for _, pair := range perm.UserIdGroupPairs {
		rules = append(rules, securitygroup.Rule{
			IpProtocol:            protocol,
			FromPort:              fromPort,
			ToPort:                toPort,
			SourceSecurityGroupID: aws.ToString(pair.GroupId),
			Description:           aws.ToString(pair.Description),
		})
	}

	return rules
}
