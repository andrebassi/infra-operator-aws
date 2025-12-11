// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/apigateway"
)

// APIGatewayRepository defines the interface for API Gateway operations
type APIGatewayRepository interface {
	Exists(ctx context.Context, apiID string) (bool, error)
	Create(ctx context.Context, api *apigateway.API) error
	Get(ctx context.Context, apiID string) (*apigateway.API, error)
	Update(ctx context.Context, api *apigateway.API) error
	Delete(ctx context.Context, apiID string) error
	TagResource(ctx context.Context, apiARN string, tags map[string]string) error
}

// APIGatewayUseCase defines the use case interface for API Gateway operations
type APIGatewayUseCase interface {
	SyncAPI(ctx context.Context, api *apigateway.API) error
	DeleteAPI(ctx context.Context, api *apigateway.API) error
}
