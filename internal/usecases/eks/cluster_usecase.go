package eks

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/eks"
	"infra-operator/internal/ports"
)

type ClusterUseCase struct {
	repo ports.EKSRepository
}

func NewClusterUseCase(repo ports.EKSRepository) *ClusterUseCase {
	return &ClusterUseCase{repo: repo}
}

func (uc *ClusterUseCase) SyncCluster(ctx context.Context, cluster *eks.Cluster) error {
	cluster.SetDefaults()

	if err := cluster.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	exists, err := uc.repo.Exists(ctx, cluster.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if exists {
		// Get current cluster state
		current, err := uc.repo.Get(ctx, cluster.ClusterName)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}

		// Update cluster with current state
		cluster.ARN = current.ARN
		cluster.Endpoint = current.Endpoint
		cluster.Status = current.Status
		cluster.PlatformVersion = current.PlatformVersion
		cluster.CertificateAuthority = current.CertificateAuthority
		cluster.LastSyncTime = current.LastSyncTime

		// Check if version upgrade is needed
		if cluster.Version != current.Version {
			if err := uc.repo.UpdateVersion(ctx, cluster.ClusterName, cluster.Version); err != nil {
				return fmt.Errorf("failed to update cluster version: %w", err)
			}
		}

		return nil
	}

	// Create new cluster
	if err := uc.repo.Create(ctx, cluster); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Tag the cluster
	if cluster.ARN != "" && len(cluster.Tags) > 0 {
		if err := uc.repo.TagResource(ctx, cluster.ARN, cluster.Tags); err != nil {
			return fmt.Errorf("failed to tag cluster: %w", err)
		}
	}

	return nil
}

func (uc *ClusterUseCase) DeleteCluster(ctx context.Context, cluster *eks.Cluster) error {
	if !cluster.ShouldDelete() {
		return nil
	}

	exists, err := uc.repo.Exists(ctx, cluster.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		return nil
	}

	if err := uc.repo.Delete(ctx, cluster.ClusterName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}

func (uc *ClusterUseCase) WaitForClusterReady(ctx context.Context, clusterName string) error {
	timeout := 20 * time.Minute
	return uc.repo.WaitForActive(ctx, clusterName, timeout)
}
