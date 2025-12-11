// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/elasticip"
)

// ElasticIPRepository defines the interface for Elastic IP operations
type ElasticIPRepository interface {
	Allocate(ctx context.Context, addr *elasticip.Address) error
	Exists(ctx context.Context, allocationID string) (bool, error)
	Get(ctx context.Context, allocationID string) (*elasticip.Address, error)
	Release(ctx context.Context, allocationID string) error
	TagResource(ctx context.Context, allocationID string, tags map[string]string) error
}

// ElasticIPUseCase defines the use case interface for Elastic IP operations
type ElasticIPUseCase interface {
	SyncAddress(ctx context.Context, addr *elasticip.Address) error
	ReleaseAddress(ctx context.Context, addr *elasticip.Address) error
}
