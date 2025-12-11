package ecs

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/ecs"
	"infra-operator/internal/ports"
)

// ClusterUseCase handles ECS cluster business logic
type ClusterUseCase struct {
	repo ports.ECSRepository
}

// NewClusterUseCase creates a new ECS cluster use case
func NewClusterUseCase(repo ports.ECSRepository) *ClusterUseCase {
	return &ClusterUseCase{
		repo: repo,
	}
}

// SyncCluster creates or updates an ECS cluster
func (uc *ClusterUseCase) SyncCluster(ctx context.Context, cluster *ecs.Cluster) error {
	// Set defaults
	cluster.SetDefaults()

	// Validate
	if err := cluster.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if cluster exists
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

		// Update cluster object with current state
		cluster.ClusterARN = current.ClusterARN
		cluster.Status = current.Status
		cluster.RegisteredContainerInstancesCount = current.RegisteredContainerInstancesCount
		cluster.RunningTasksCount = current.RunningTasksCount
		cluster.PendingTasksCount = current.PendingTasksCount
		cluster.ActiveServicesCount = current.ActiveServicesCount

		// Update settings if changed
		if settingsChanged(cluster.Settings, current.Settings) {
			if err := uc.repo.UpdateSettings(ctx, cluster); err != nil {
				return fmt.Errorf("failed to update cluster settings: %w", err)
			}
		}

		// Update capacity providers if changed
		if capacityProvidersChanged(cluster, current) {
			if err := uc.repo.UpdateCapacityProviders(ctx, cluster); err != nil {
				return fmt.Errorf("failed to update capacity providers: %w", err)
			}
		}

		// Update tags if provided
		if len(cluster.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, cluster.ClusterARN, cluster.Tags); err != nil {
				return fmt.Errorf("failed to update tags: %w", err)
			}
		}

		return nil
	}

	// Create new cluster
	if err := uc.repo.Create(ctx, cluster); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	return nil
}

// DeleteCluster deletes an ECS cluster
func (uc *ClusterUseCase) DeleteCluster(ctx context.Context, cluster *ecs.Cluster) error {
	// Check deletion policy
	if !cluster.ShouldDelete() {
		return nil // Skip deletion based on policy
	}

	// Skip if cluster ARN is empty (never created or already deleted)
	if cluster.ClusterARN == "" {
		return nil
	}

	// Delete the cluster
	if err := uc.repo.Delete(ctx, cluster.ClusterARN); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}

// settingsChanged checks if cluster settings have changed
func settingsChanged(new, old []ecs.ClusterSetting) bool {
	if len(new) != len(old) {
		return true
	}

	// Create maps for comparison
	newMap := make(map[string]string)
	for _, setting := range new {
		newMap[setting.Name] = setting.Value
	}

	oldMap := make(map[string]string)
	for _, setting := range old {
		oldMap[setting.Name] = setting.Value
	}

	// Compare maps
	for name, value := range newMap {
		if oldMap[name] != value {
			return true
		}
	}

	return false
}

// capacityProvidersChanged checks if capacity providers have changed
func capacityProvidersChanged(new, old *ecs.Cluster) bool {
	// Check capacity providers list
	if len(new.CapacityProviders) != len(old.CapacityProviders) {
		return true
	}

	newCPMap := make(map[string]bool)
	for _, cp := range new.CapacityProviders {
		newCPMap[cp] = true
	}

	for _, cp := range old.CapacityProviders {
		if !newCPMap[cp] {
			return true
		}
	}

	// Check default capacity provider strategy
	if len(new.DefaultCapacityProviderStrategy) != len(old.DefaultCapacityProviderStrategy) {
		return true
	}

	for i, newStrat := range new.DefaultCapacityProviderStrategy {
		oldStrat := old.DefaultCapacityProviderStrategy[i]
		if newStrat.CapacityProvider != oldStrat.CapacityProvider ||
			newStrat.Weight != oldStrat.Weight ||
			newStrat.Base != oldStrat.Base {
			return true
		}
	}

	return false
}
