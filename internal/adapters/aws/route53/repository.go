// Package route53 implementa adaptadores para integração com AWS.
//
// Integra com AWS SDK e implementa as interfaces de repositório definidas em ports,
// isolando detalhes de infraestrutura da lógica de negócio.
package route53


import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"infra-operator/internal/domain/route53"
)

// Repository handles Route53 operations using AWS SDK
type Repository struct {
	client *awsroute53.Client
}

// NewRepository creates a new Route53 repository
func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsroute53.NewFromConfig(cfg),
	}
}

// ===== Hosted Zone Operations =====

// CreateHostedZone creates a new hosted zone
func (r *Repository) CreateHostedZone(ctx context.Context, hz *route53.HostedZone) error {
	input := &awsroute53.CreateHostedZoneInput{
		Name:            aws.String(hz.Name),
		CallerReference: aws.String(fmt.Sprintf("infra-operator-%d", aws.ToTime(nil).Unix())),
		HostedZoneConfig: &types.HostedZoneConfig{
			Comment: aws.String(hz.Comment),
		},
	}

	// Configure private zone if needed
	if hz.PrivateZone {
		input.VPC = &types.VPC{
			VPCId:     aws.String(hz.VPCId),
			VPCRegion: types.VPCRegion(hz.VPCRegion),
		}
		input.HostedZoneConfig.PrivateZone = true
	}

	output, err := r.client.CreateHostedZone(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create hosted zone: %w", err)
	}

	// Extract hosted zone ID (remove /hostedzone/ prefix)
	hz.HostedZoneID = extractHostedZoneID(aws.ToString(output.HostedZone.Id))
	hz.ResourceRecordSetCount = aws.ToInt64(output.HostedZone.ResourceRecordSetCount)

	// Extract name servers
	if output.DelegationSet != nil && len(output.DelegationSet.NameServers) > 0 {
		hz.NameServers = output.DelegationSet.NameServers
	}

	// Tag the hosted zone if tags are provided
	if len(hz.Tags) > 0 {
		if err := r.TagHostedZone(ctx, hz.HostedZoneID, hz.Tags); err != nil {
			return fmt.Errorf("failed to tag hosted zone: %w", err)
		}
	}

	return nil
}

// GetHostedZone retrieves a hosted zone by ID
func (r *Repository) GetHostedZone(ctx context.Context, hostedZoneID string) (*route53.HostedZone, error) {
	input := &awsroute53.GetHostedZoneInput{
		Id: aws.String(formatHostedZoneID(hostedZoneID)),
	}

	output, err := r.client.GetHostedZone(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get hosted zone: %w", err)
	}

	hz := &route53.HostedZone{
		HostedZoneID:           extractHostedZoneID(aws.ToString(output.HostedZone.Id)),
		Name:                   aws.ToString(output.HostedZone.Name),
		ResourceRecordSetCount: aws.ToInt64(output.HostedZone.ResourceRecordSetCount),
	}

	if output.HostedZone.Config != nil {
		hz.Comment = aws.ToString(output.HostedZone.Config.Comment)
		hz.PrivateZone = output.HostedZone.Config.PrivateZone
	}

	// Extract name servers
	if output.DelegationSet != nil && len(output.DelegationSet.NameServers) > 0 {
		hz.NameServers = output.DelegationSet.NameServers
	}

	// Get VPC information if private zone
	if hz.PrivateZone && len(output.VPCs) > 0 {
		hz.VPCId = aws.ToString(output.VPCs[0].VPCId)
		hz.VPCRegion = string(output.VPCs[0].VPCRegion)
	}

	return hz, nil
}

// DeleteHostedZone deletes a hosted zone
func (r *Repository) DeleteHostedZone(ctx context.Context, hostedZoneID string) error {
	input := &awsroute53.DeleteHostedZoneInput{
		Id: aws.String(formatHostedZoneID(hostedZoneID)),
	}

	_, err := r.client.DeleteHostedZone(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete hosted zone: %w", err)
	}

	return nil
}

// HostedZoneExists checks if a hosted zone exists
func (r *Repository) HostedZoneExists(ctx context.Context, hostedZoneID string) (bool, error) {
	_, err := r.GetHostedZone(ctx, hostedZoneID)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchHostedZone") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// TagHostedZone adds or updates tags on a hosted zone
func (r *Repository) TagHostedZone(ctx context.Context, hostedZoneID string, tags map[string]string) error {
	var route53Tags []types.Tag
	for key, value := range tags {
		route53Tags = append(route53Tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	input := &awsroute53.ChangeTagsForResourceInput{
		ResourceType: types.TagResourceTypeHostedzone,
		ResourceId:   aws.String(hostedZoneID),
		AddTags:      route53Tags,
	}

	_, err := r.client.ChangeTagsForResource(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag hosted zone: %w", err)
	}

	return nil
}

// ===== Record Set Operations =====

// CreateRecordSet creates a new record set
func (r *Repository) CreateRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	change := buildChangeInput(rs, types.ChangeActionCreate)
	return r.executeChange(ctx, rs.HostedZoneID, change, rs)
}

// UpdateRecordSet updates an existing record set
func (r *Repository) UpdateRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	change := buildChangeInput(rs, types.ChangeActionUpsert)
	return r.executeChange(ctx, rs.HostedZoneID, change, rs)
}

// DeleteRecordSet deletes a record set
func (r *Repository) DeleteRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	change := buildChangeInput(rs, types.ChangeActionDelete)
	return r.executeChange(ctx, rs.HostedZoneID, change, rs)
}

// GetRecordSet retrieves a record set
func (r *Repository) GetRecordSet(ctx context.Context, hostedZoneID, name, recordType string) (*route53.RecordSet, error) {
	input := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(formatHostedZoneID(hostedZoneID)),
		StartRecordName: aws.String(name),
		StartRecordType: types.RRType(recordType),
		MaxItems:        aws.Int32(1),
	}

	output, err := r.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list record sets: %w", err)
	}

	if len(output.ResourceRecordSets) == 0 {
		return nil, fmt.Errorf("record set not found")
	}

	awsRRS := output.ResourceRecordSets[0]
	if aws.ToString(awsRRS.Name) != name || string(awsRRS.Type) != recordType {
		return nil, fmt.Errorf("record set not found")
	}

	return convertFromAWSRecordSet(&awsRRS, hostedZoneID), nil
}

// RecordSetExists checks if a record set exists
func (r *Repository) RecordSetExists(ctx context.Context, hostedZoneID, name, recordType string) (bool, error) {
	_, err := r.GetRecordSet(ctx, hostedZoneID, name, recordType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetChangeStatus gets the status of a change batch
func (r *Repository) GetChangeStatus(ctx context.Context, changeID string) (string, error) {
	input := &awsroute53.GetChangeInput{
		Id: aws.String(formatChangeID(changeID)),
	}

	output, err := r.client.GetChange(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get change status: %w", err)
	}

	return string(output.ChangeInfo.Status), nil
}

// ===== Helper Functions =====

// executeChange executes a change batch
func (r *Repository) executeChange(ctx context.Context, hostedZoneID string, change types.Change, rs *route53.RecordSet) error {
	input := &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(formatHostedZoneID(hostedZoneID)),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{change},
		},
	}

	output, err := r.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to change resource record sets: %w", err)
	}

	rs.ChangeID = extractChangeID(aws.ToString(output.ChangeInfo.Id))
	rs.ChangeStatus = string(output.ChangeInfo.Status)

	return nil
}

// buildChangeInput builds a Change input for Route53 API
func buildChangeInput(rs *route53.RecordSet, action types.ChangeAction) types.Change {
	rrs := &types.ResourceRecordSet{
		Name: aws.String(rs.Name),
		Type: types.RRType(rs.Type),
	}

	// Alias record
	if rs.AliasTarget != nil {
		rrs.AliasTarget = &types.AliasTarget{
			HostedZoneId:         aws.String(rs.AliasTarget.HostedZoneID),
			DNSName:              aws.String(rs.AliasTarget.DNSName),
			EvaluateTargetHealth: rs.AliasTarget.EvaluateTargetHealth,
		}
	} else {
		// Regular record
		rrs.TTL = rs.TTL
		var resourceRecords []types.ResourceRecord
		for _, value := range rs.ResourceRecords {
			resourceRecords = append(resourceRecords, types.ResourceRecord{
				Value: aws.String(value),
			})
		}
		rrs.ResourceRecords = resourceRecords
	}

	// Routing policy
	if rs.SetIdentifier != "" {
		rrs.SetIdentifier = aws.String(rs.SetIdentifier)
	}
	if rs.Weight != nil {
		rrs.Weight = rs.Weight
	}
	if rs.Region != "" {
		rrs.Region = types.ResourceRecordSetRegion(rs.Region)
	}
	if rs.GeoLocation != nil {
		rrs.GeoLocation = &types.GeoLocation{
			ContinentCode:   aws.String(rs.GeoLocation.ContinentCode),
			CountryCode:     aws.String(rs.GeoLocation.CountryCode),
			SubdivisionCode: aws.String(rs.GeoLocation.SubdivisionCode),
		}
	}
	if rs.Failover != "" {
		rrs.Failover = types.ResourceRecordSetFailover(rs.Failover)
	}
	if rs.MultiValueAnswer {
		rrs.MultiValueAnswer = aws.Bool(rs.MultiValueAnswer)
	}
	if rs.HealthCheckID != "" {
		rrs.HealthCheckId = aws.String(rs.HealthCheckID)
	}

	return types.Change{
		Action:            action,
		ResourceRecordSet: rrs,
	}
}

// convertFromAWSRecordSet converts AWS RecordSet to domain RecordSet
func convertFromAWSRecordSet(awsRRS *types.ResourceRecordSet, hostedZoneID string) *route53.RecordSet {
	rs := &route53.RecordSet{
		HostedZoneID: hostedZoneID,
		Name:         aws.ToString(awsRRS.Name),
		Type:         string(awsRRS.Type),
	}

	// Alias target
	if awsRRS.AliasTarget != nil {
		rs.AliasTarget = &route53.AliasTarget{
			HostedZoneID:         aws.ToString(awsRRS.AliasTarget.HostedZoneId),
			DNSName:              aws.ToString(awsRRS.AliasTarget.DNSName),
			EvaluateTargetHealth: awsRRS.AliasTarget.EvaluateTargetHealth,
		}
	} else {
		// Regular record
		rs.TTL = awsRRS.TTL
		for _, rr := range awsRRS.ResourceRecords {
			rs.ResourceRecords = append(rs.ResourceRecords, aws.ToString(rr.Value))
		}
	}

	// Routing policy
	rs.SetIdentifier = aws.ToString(awsRRS.SetIdentifier)
	rs.Weight = awsRRS.Weight
	rs.Region = string(awsRRS.Region)
	if awsRRS.GeoLocation != nil {
		rs.GeoLocation = &route53.GeoLocation{
			ContinentCode:   aws.ToString(awsRRS.GeoLocation.ContinentCode),
			CountryCode:     aws.ToString(awsRRS.GeoLocation.CountryCode),
			SubdivisionCode: aws.ToString(awsRRS.GeoLocation.SubdivisionCode),
		}
	}
	rs.Failover = string(awsRRS.Failover)
	rs.MultiValueAnswer = aws.ToBool(awsRRS.MultiValueAnswer)
	rs.HealthCheckID = aws.ToString(awsRRS.HealthCheckId)

	return rs
}

// formatHostedZoneID ensures hosted zone ID has the correct format
func formatHostedZoneID(id string) string {
	if strings.HasPrefix(id, "/hostedzone/") {
		return id
	}
	return "/hostedzone/" + id
}

// extractHostedZoneID removes the /hostedzone/ prefix
func extractHostedZoneID(id string) string {
	return strings.TrimPrefix(id, "/hostedzone/")
}

// formatChangeID ensures change ID has the correct format
func formatChangeID(id string) string {
	if strings.HasPrefix(id, "/change/") {
		return id
	}
	return "/change/" + id
}

// extractChangeID removes the /change/ prefix
func extractChangeID(id string) string {
	return strings.TrimPrefix(id, "/change/")
}
