// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/route53"
)

// Route53Repository defines the interface for Route53 operations
type Route53Repository interface {
	// Hosted Zone operations
	CreateHostedZone(ctx context.Context, hz *route53.HostedZone) error
	GetHostedZone(ctx context.Context, hostedZoneID string) (*route53.HostedZone, error)
	DeleteHostedZone(ctx context.Context, hostedZoneID string) error
	HostedZoneExists(ctx context.Context, hostedZoneID string) (bool, error)
	TagHostedZone(ctx context.Context, hostedZoneID string, tags map[string]string) error

	// Record Set operations
	CreateRecordSet(ctx context.Context, rs *route53.RecordSet) error
	UpdateRecordSet(ctx context.Context, rs *route53.RecordSet) error
	DeleteRecordSet(ctx context.Context, rs *route53.RecordSet) error
	GetRecordSet(ctx context.Context, hostedZoneID, name, recordType string) (*route53.RecordSet, error)
	RecordSetExists(ctx context.Context, hostedZoneID, name, recordType string) (bool, error)
	GetChangeStatus(ctx context.Context, changeID string) (string, error)
}

// Route53UseCase defines the use case interface for Route53 operations
type Route53UseCase interface {
	// Hosted Zone use cases
	SyncHostedZone(ctx context.Context, hz *route53.HostedZone) error
	DeleteHostedZone(ctx context.Context, hz *route53.HostedZone) error

	// Record Set use cases
	SyncRecordSet(ctx context.Context, rs *route53.RecordSet) error
	DeleteRecordSet(ctx context.Context, rs *route53.RecordSet) error
}
