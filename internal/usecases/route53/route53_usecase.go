// Package route53 implementa os casos de uso da aplicação.
//
// Contém a lógica de negócio que orquestra repositórios e aplica regras de domínio,
// atuando como camada de aplicação na Clean Architecture.
package route53


import (
	"context"

	"infra-operator/internal/domain/route53"
	"infra-operator/internal/ports"
)

// Route53UseCaseImpl implements the Route53UseCase interface
type Route53UseCaseImpl struct {
	hostedZoneUC *HostedZoneUseCase
	recordSetUC  *RecordSetUseCase
}

// NewRoute53UseCase creates a new Route53 use case
func NewRoute53UseCase(repo ports.Route53Repository) ports.Route53UseCase {
	return &Route53UseCaseImpl{
		hostedZoneUC: NewHostedZoneUseCase(repo),
		recordSetUC:  NewRecordSetUseCase(repo),
	}
}

// SyncHostedZone creates or updates a hosted zone
func (uc *Route53UseCaseImpl) SyncHostedZone(ctx context.Context, hz *route53.HostedZone) error {
	return uc.hostedZoneUC.SyncHostedZone(ctx, hz)
}

// DeleteHostedZone deletes a hosted zone
func (uc *Route53UseCaseImpl) DeleteHostedZone(ctx context.Context, hz *route53.HostedZone) error {
	return uc.hostedZoneUC.DeleteHostedZone(ctx, hz)
}

// SyncRecordSet creates or updates a record set
func (uc *Route53UseCaseImpl) SyncRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	return uc.recordSetUC.SyncRecordSet(ctx, rs)
}

// DeleteRecordSet deletes a record set
func (uc *Route53UseCaseImpl) DeleteRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	return uc.recordSetUC.DeleteRecordSet(ctx, rs)
}
