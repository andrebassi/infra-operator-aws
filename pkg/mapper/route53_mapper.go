// Package mapper contém funções de mapeamento entre camadas.
//
// Converte entre Custom Resources Kubernetes (CRDs) e objetos de domínio,
// mantendo as camadas desacopladas.
package mapper


import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/route53"
)

// ===== Hosted Zone Mappers =====

// CRToDomainRoute53HostedZone converts Route53HostedZone CR to domain model
func CRToDomainRoute53HostedZone(cr *infrav1alpha1.Route53HostedZone) *route53.HostedZone {
	hz := &route53.HostedZone{
		Name:           cr.Spec.Name,
		Comment:        cr.Spec.Comment,
		PrivateZone:    cr.Spec.PrivateZone,
		VPCId:          cr.Spec.VPCId,
		VPCRegion:      cr.Spec.VPCRegion,
		Tags:           cr.Spec.Tags,
		DeletionPolicy: cr.Spec.DeletionPolicy,
	}

	// Copy status fields if present
	if cr.Status.HostedZoneID != "" {
		hz.HostedZoneID = cr.Status.HostedZoneID
		hz.NameServers = cr.Status.NameServers
		hz.ResourceRecordSetCount = cr.Status.ResourceRecordSetCount
	}

	return hz
}

// DomainToStatusRoute53HostedZone updates CR status from domain model
func DomainToStatusRoute53HostedZone(hz *route53.HostedZone, cr *infrav1alpha1.Route53HostedZone) {
	cr.Status.Ready = hz.IsCreated()
	cr.Status.HostedZoneID = hz.HostedZoneID
	cr.Status.NameServers = hz.NameServers
	cr.Status.ResourceRecordSetCount = hz.ResourceRecordSetCount

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}

// ===== Record Set Mappers =====

// CRToDomainRoute53RecordSet converts Route53RecordSet CR to domain model
func CRToDomainRoute53RecordSet(cr *infrav1alpha1.Route53RecordSet) *route53.RecordSet {
	rs := &route53.RecordSet{
		HostedZoneID:     cr.Spec.HostedZoneID,
		Name:             cr.Spec.Name,
		Type:             cr.Spec.Type,
		TTL:              cr.Spec.TTL,
		ResourceRecords:  cr.Spec.ResourceRecords,
		SetIdentifier:    cr.Spec.SetIdentifier,
		Weight:           cr.Spec.Weight,
		Region:           cr.Spec.Region,
		Failover:         cr.Spec.Failover,
		MultiValueAnswer: cr.Spec.MultiValueAnswer,
		HealthCheckID:    cr.Spec.HealthCheckID,
		DeletionPolicy:   cr.Spec.DeletionPolicy,
	}

	// Map alias target if present
	if cr.Spec.AliasTarget != nil {
		rs.AliasTarget = &route53.AliasTarget{
			HostedZoneID:         cr.Spec.AliasTarget.HostedZoneID,
			DNSName:              cr.Spec.AliasTarget.DNSName,
			EvaluateTargetHealth: cr.Spec.AliasTarget.EvaluateTargetHealth,
		}
	}

	// Map geo location if present
	if cr.Spec.GeoLocation != nil {
		rs.GeoLocation = &route53.GeoLocation{
			ContinentCode:   cr.Spec.GeoLocation.ContinentCode,
			CountryCode:     cr.Spec.GeoLocation.CountryCode,
			SubdivisionCode: cr.Spec.GeoLocation.SubdivisionCode,
		}
	}

	// Copy status fields if present
	if cr.Status.ChangeID != "" {
		rs.ChangeID = cr.Status.ChangeID
		rs.ChangeStatus = cr.Status.ChangeStatus
	}

	return rs
}

// DomainToStatusRoute53RecordSet updates CR status from domain model
func DomainToStatusRoute53RecordSet(rs *route53.RecordSet, cr *infrav1alpha1.Route53RecordSet) {
	cr.Status.Ready = true
	cr.Status.ChangeID = rs.ChangeID
	cr.Status.ChangeStatus = rs.ChangeStatus

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
