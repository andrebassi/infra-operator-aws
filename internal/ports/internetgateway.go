// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/internetgateway"
)

// InternetGatewayRepository defines the interface for Internet Gateway operations
type InternetGatewayRepository interface {
	Exists(ctx context.Context, igwID string) (bool, error)
	Create(ctx context.Context, g *internetgateway.Gateway) error
	Get(ctx context.Context, igwID string) (*internetgateway.Gateway, error)
	Delete(ctx context.Context, igwID, vpcID string) error
	TagResource(ctx context.Context, igwID string, tags map[string]string) error
}

// InternetGatewayUseCase defines the use case interface for Internet Gateway operations
type InternetGatewayUseCase interface {
	SyncGateway(ctx context.Context, g *internetgateway.Gateway) error
	DeleteGateway(ctx context.Context, g *internetgateway.Gateway) error
}
