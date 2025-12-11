// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/natgateway"
)

// NATGatewayRepository defines the interface for NAT Gateway operations
type NATGatewayRepository interface {
	Exists(ctx context.Context, natGwID string) (bool, error)
	Create(ctx context.Context, g *natgateway.Gateway) error
	Get(ctx context.Context, natGwID string) (*natgateway.Gateway, error)
	Delete(ctx context.Context, natGwID string) error
	TagResource(ctx context.Context, natGwID string, tags map[string]string) error
}

// NATGatewayUseCase defines the use case interface for NAT Gateway operations
type NATGatewayUseCase interface {
	SyncGateway(ctx context.Context, g *natgateway.Gateway) error
	DeleteGateway(ctx context.Context, g *natgateway.Gateway) error
}
