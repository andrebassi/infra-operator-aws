// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"time"
	"infra-operator/internal/domain/eks"
)

// EKSRepository defines the interface for EKS operations
type EKSRepository interface {
	Exists(ctx context.Context, clusterName string) (bool, error)
	Create(ctx context.Context, cluster *eks.Cluster) error
	Get(ctx context.Context, clusterName string) (*eks.Cluster, error)
	Delete(ctx context.Context, clusterName string) error
	WaitForActive(ctx context.Context, clusterName string, timeout time.Duration) error
	UpdateVersion(ctx context.Context, clusterName, version string) error
	TagResource(ctx context.Context, arn string, tags map[string]string) error
}

// EKSUseCase defines the use case interface for EKS operations
type EKSUseCase interface {
	SyncCluster(ctx context.Context, cluster *eks.Cluster) error
	DeleteCluster(ctx context.Context, cluster *eks.Cluster) error
	WaitForClusterReady(ctx context.Context, clusterName string) error
}
