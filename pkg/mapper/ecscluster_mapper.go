package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/ecs"
)

// CRToDomainECSCluster converts ECSCluster CR to domain model
func CRToDomainECSCluster(cr *infrav1alpha1.ECSCluster) *ecs.Cluster {
	cluster := &ecs.Cluster{
		ClusterName:                     cr.Spec.ClusterName,
		CapacityProviders:               cr.Spec.CapacityProviders,
		DefaultCapacityProviderStrategy: convertCapacityProviderStrategy(cr.Spec.DefaultCapacityProviderStrategy),
		Settings:                        convertClusterSettings(cr.Spec.Settings),
		Configuration:                   convertClusterConfiguration(cr.Spec.Configuration),
		ServiceConnectDefaults:          convertServiceConnectDefaults(cr.Spec.ServiceConnectDefaults),
		Tags:                            cr.Spec.Tags,
		DeletionPolicy:                  cr.Spec.DeletionPolicy,
	}

	// Copy status fields if present
	if cr.Status.ClusterARN != "" {
		cluster.ClusterARN = cr.Status.ClusterARN
		cluster.Status = cr.Status.Status
		cluster.RegisteredContainerInstancesCount = cr.Status.RegisteredContainerInstancesCount
		cluster.RunningTasksCount = cr.Status.RunningTasksCount
		cluster.PendingTasksCount = cr.Status.PendingTasksCount
		cluster.ActiveServicesCount = cr.Status.ActiveServicesCount
	}

	return cluster
}

// DomainToStatusECSCluster updates CR status from domain model
func DomainToStatusECSCluster(cluster *ecs.Cluster, cr *infrav1alpha1.ECSCluster) {
	cr.Status.Ready = cluster.IsActive()
	cr.Status.ClusterARN = cluster.ClusterARN
	cr.Status.Status = cluster.Status
	cr.Status.RegisteredContainerInstancesCount = cluster.RegisteredContainerInstancesCount
	cr.Status.RunningTasksCount = cluster.RunningTasksCount
	cr.Status.PendingTasksCount = cluster.PendingTasksCount
	cr.Status.ActiveServicesCount = cluster.ActiveServicesCount

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}

// Helper functions

func convertCapacityProviderStrategy(crStrategy []infrav1alpha1.CapacityProviderStrategyItem) []ecs.CapacityProviderStrategyItem {
	if len(crStrategy) == 0 {
		return nil
	}

	var domainStrategy []ecs.CapacityProviderStrategyItem
	for _, item := range crStrategy {
		domainStrategy = append(domainStrategy, ecs.CapacityProviderStrategyItem{
			CapacityProvider: item.CapacityProvider,
			Weight:           item.Weight,
			Base:             item.Base,
		})
	}
	return domainStrategy
}

func convertClusterSettings(crSettings []infrav1alpha1.ClusterSetting) []ecs.ClusterSetting {
	if len(crSettings) == 0 {
		return nil
	}

	var domainSettings []ecs.ClusterSetting
	for _, setting := range crSettings {
		domainSettings = append(domainSettings, ecs.ClusterSetting{
			Name:  setting.Name,
			Value: setting.Value,
		})
	}
	return domainSettings
}

func convertClusterConfiguration(crConfig *infrav1alpha1.ClusterConfiguration) *ecs.ClusterConfiguration {
	if crConfig == nil {
		return nil
	}

	config := &ecs.ClusterConfiguration{}

	if crConfig.ExecuteCommandConfiguration != nil {
		config.ExecuteCommandConfiguration = &ecs.ExecuteCommandConfiguration{
			KmsKeyID: crConfig.ExecuteCommandConfiguration.KmsKeyID,
			Logging:  crConfig.ExecuteCommandConfiguration.Logging,
		}

		if crConfig.ExecuteCommandConfiguration.LogConfiguration != nil {
			config.ExecuteCommandConfiguration.LogConfiguration = &ecs.ExecuteCommandLogConfiguration{
				CloudWatchLogGroupName:      crConfig.ExecuteCommandConfiguration.LogConfiguration.CloudWatchLogGroupName,
				CloudWatchEncryptionEnabled: crConfig.ExecuteCommandConfiguration.LogConfiguration.CloudWatchEncryptionEnabled,
				S3BucketName:                crConfig.ExecuteCommandConfiguration.LogConfiguration.S3BucketName,
				S3EncryptionEnabled:         crConfig.ExecuteCommandConfiguration.LogConfiguration.S3EncryptionEnabled,
				S3KeyPrefix:                 crConfig.ExecuteCommandConfiguration.LogConfiguration.S3KeyPrefix,
			}
		}
	}

	return config
}

func convertServiceConnectDefaults(crDefaults *infrav1alpha1.ServiceConnectDefaults) *ecs.ServiceConnectDefaults {
	if crDefaults == nil {
		return nil
	}

	return &ecs.ServiceConnectDefaults{
		Namespace: crDefaults.Namespace,
	}
}
