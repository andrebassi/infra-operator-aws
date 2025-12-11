package elasticache

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/elasticache"
	"infra-operator/internal/ports"
)

type ClusterUseCase struct {
	repo ports.ElastiCacheRepository
}

func NewClusterUseCase(repo ports.ElastiCacheRepository) *ClusterUseCase {
	return &ClusterUseCase{
		repo: repo,
	}
}

func (uc *ClusterUseCase) SyncCluster(ctx context.Context, cluster *elasticache.Cluster) error {
	cluster.SetDefaults()

	if err := cluster.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Determine if this is a Redis replication group or single cache cluster
	isRedisCluster := cluster.IsRedis() && (cluster.NumNodeGroups > 0 || cluster.NumCacheNodes > 1 || cluster.AutomaticFailoverEnabled)

	// Check if cluster already exists
	exists, err := uc.repo.Exists(ctx, cluster.ClusterID, isRedisCluster)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if exists {
		// Get current cluster state
		var current *elasticache.Cluster
		if isRedisCluster {
			current, err = uc.repo.GetReplicationGroup(ctx, cluster.ClusterID)
		} else {
			current, err = uc.repo.GetCacheCluster(ctx, cluster.ClusterID)
		}
		if err != nil {
			return fmt.Errorf("failed to get current cluster: %w", err)
		}

		// Copy status fields from current cluster
		cluster.ClusterStatus = current.ClusterStatus
		cluster.CacheClusterARN = current.CacheClusterARN
		cluster.ConfigurationEndpoint = current.ConfigurationEndpoint
		cluster.PrimaryEndpoint = current.PrimaryEndpoint
		cluster.ReaderEndpoint = current.ReaderEndpoint
		cluster.NodeEndpoints = current.NodeEndpoints
		cluster.MemberClusters = current.MemberClusters
		cluster.ClusterCreateTime = current.ClusterCreateTime

		// TODO: Implement update logic if needed (modify replication group, etc)
		return nil
	}

	// Create new cluster
	if isRedisCluster {
		if err := uc.repo.CreateReplicationGroup(ctx, cluster); err != nil {
			return fmt.Errorf("failed to create replication group: %w", err)
		}
	} else {
		if err := uc.repo.CreateCacheCluster(ctx, cluster); err != nil {
			return fmt.Errorf("failed to create cache cluster: %w", err)
		}
	}

	return nil
}

func (uc *ClusterUseCase) DeleteCluster(ctx context.Context, cluster *elasticache.Cluster) error {
	// Determine final snapshot identifier if needed
	var finalSnapshotID string
	if cluster.ShouldSnapshot() && cluster.FinalSnapshotIdentifier != "" {
		finalSnapshotID = cluster.FinalSnapshotIdentifier
	}

	// Skip deletion if retention policy
	if !cluster.ShouldDelete() && !cluster.ShouldSnapshot() {
		return nil // Retain policy
	}

	// Determine if this is a Redis replication group
	isRedisCluster := cluster.IsRedis() && (cluster.NumNodeGroups > 0 || cluster.NumCacheNodes > 1 || cluster.AutomaticFailoverEnabled)

	if isRedisCluster {
		return uc.repo.DeleteReplicationGroup(ctx, cluster.ClusterID, finalSnapshotID)
	}

	return uc.repo.DeleteCacheCluster(ctx, cluster.ClusterID, finalSnapshotID)
}
