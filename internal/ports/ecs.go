// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/ecs"
)

// ECSRepository defines the interface for ECS Cluster operations
type ECSRepository interface {
	Exists(ctx context.Context, clusterName string) (bool, error)
	Create(ctx context.Context, cluster *ecs.Cluster) error
	Get(ctx context.Context, clusterName string) (*ecs.Cluster, error)
	Delete(ctx context.Context, clusterARN string) error
	UpdateSettings(ctx context.Context, cluster *ecs.Cluster) error
	UpdateCapacityProviders(ctx context.Context, cluster *ecs.Cluster) error
	TagResource(ctx context.Context, clusterARN string, tags map[string]string) error
}

// ECSUseCase defines the use case interface for ECS operations
type ECSUseCase interface {
	SyncCluster(ctx context.Context, cluster *ecs.Cluster) error
	DeleteCluster(ctx context.Context, cluster *ecs.Cluster) error
}
