// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/elasticache"
)

// ElastiCacheRepository defines the interface for ElastiCache operations
type ElastiCacheRepository interface {
	Exists(ctx context.Context, clusterID string, isRedisCluster bool) (bool, error)
	CreateReplicationGroup(ctx context.Context, cluster *elasticache.Cluster) error
	CreateCacheCluster(ctx context.Context, cluster *elasticache.Cluster) error
	GetReplicationGroup(ctx context.Context, clusterID string) (*elasticache.Cluster, error)
	GetCacheCluster(ctx context.Context, clusterID string) (*elasticache.Cluster, error)
	DeleteReplicationGroup(ctx context.Context, clusterID string, finalSnapshotID string) error
	DeleteCacheCluster(ctx context.Context, clusterID string, finalSnapshotID string) error
}

// ElastiCacheUseCase defines the use case interface for ElastiCache operations
type ElastiCacheUseCase interface {
	SyncCluster(ctx context.Context, cluster *elasticache.Cluster) error
	DeleteCluster(ctx context.Context, cluster *elasticache.Cluster) error
}
