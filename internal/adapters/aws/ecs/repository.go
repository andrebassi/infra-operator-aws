package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"infra-operator/internal/domain/ecs"
)

// Repository handles ECS cluster operations using AWS SDK
type Repository struct {
	client *awsecs.Client
}

// NewRepository creates a new ECS repository
func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsecs.NewFromConfig(cfg),
	}
}

// Exists checks if a cluster exists
func (r *Repository) Exists(ctx context.Context, clusterName string) (bool, error) {
	input := &awsecs.DescribeClustersInput{
		Clusters: []string{clusterName},
		Include:  []types.ClusterField{types.ClusterFieldTags, types.ClusterFieldSettings, types.ClusterFieldConfigurations},
	}

	output, err := r.client.DescribeClusters(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to describe clusters: %w", err)
	}

	// Check if cluster was found and is not in INACTIVE state
	for _, cluster := range output.Clusters {
		if aws.ToString(cluster.ClusterName) == clusterName && cluster.Status != aws.String("INACTIVE") {
			return true, nil
		}
	}

	return false, nil
}

// Create creates a new ECS cluster
func (r *Repository) Create(ctx context.Context, cluster *ecs.Cluster) error {
	input := &awsecs.CreateClusterInput{
		ClusterName: aws.String(cluster.ClusterName),
	}

	// Add capacity providers
	if len(cluster.CapacityProviders) > 0 {
		input.CapacityProviders = cluster.CapacityProviders
	}

	// Add default capacity provider strategy
	if len(cluster.DefaultCapacityProviderStrategy) > 0 {
		var strategy []types.CapacityProviderStrategyItem
		for _, item := range cluster.DefaultCapacityProviderStrategy {
			strategy = append(strategy, types.CapacityProviderStrategyItem{
				CapacityProvider: aws.String(item.CapacityProvider),
				Weight:           item.Weight,
				Base:             item.Base,
			})
		}
		input.DefaultCapacityProviderStrategy = strategy
	}

	// Add settings
	if len(cluster.Settings) > 0 {
		var settings []types.ClusterSetting
		for _, setting := range cluster.Settings {
			settings = append(settings, types.ClusterSetting{
				Name:  types.ClusterSettingName(setting.Name),
				Value: aws.String(setting.Value),
			})
		}
		input.Settings = settings
	}

	// Add configuration
	if cluster.Configuration != nil && cluster.Configuration.ExecuteCommandConfiguration != nil {
		executeCmd := cluster.Configuration.ExecuteCommandConfiguration
		config := &types.ClusterConfiguration{
			ExecuteCommandConfiguration: &types.ExecuteCommandConfiguration{},
		}

		if executeCmd.KmsKeyID != "" {
			config.ExecuteCommandConfiguration.KmsKeyId = aws.String(executeCmd.KmsKeyID)
		}

		if executeCmd.Logging != "" {
			config.ExecuteCommandConfiguration.Logging = types.ExecuteCommandLogging(executeCmd.Logging)
		}

		if executeCmd.LogConfiguration != nil {
			logConfig := &types.ExecuteCommandLogConfiguration{}

			if executeCmd.LogConfiguration.CloudWatchLogGroupName != "" {
				logConfig.CloudWatchLogGroupName = aws.String(executeCmd.LogConfiguration.CloudWatchLogGroupName)
				logConfig.CloudWatchEncryptionEnabled = executeCmd.LogConfiguration.CloudWatchEncryptionEnabled
			}

			if executeCmd.LogConfiguration.S3BucketName != "" {
				logConfig.S3BucketName = aws.String(executeCmd.LogConfiguration.S3BucketName)
				logConfig.S3EncryptionEnabled = executeCmd.LogConfiguration.S3EncryptionEnabled
				if executeCmd.LogConfiguration.S3KeyPrefix != "" {
					logConfig.S3KeyPrefix = aws.String(executeCmd.LogConfiguration.S3KeyPrefix)
				}
			}

			config.ExecuteCommandConfiguration.LogConfiguration = logConfig
		}

		input.Configuration = config
	}

	// Add service connect defaults
	if cluster.ServiceConnectDefaults != nil {
		input.ServiceConnectDefaults = &types.ClusterServiceConnectDefaultsRequest{
			Namespace: aws.String(cluster.ServiceConnectDefaults.Namespace),
		}
	}

	// Add tags
	if len(cluster.Tags) > 0 {
		var tags []types.Tag
		for key, value := range cluster.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}
		input.Tags = tags
	}

	output, err := r.client.CreateCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Update cluster with created data
	if output.Cluster != nil {
		r.populateClusterFromAWS(cluster, output.Cluster)
	}

	return nil
}

// Get retrieves an ECS cluster
func (r *Repository) Get(ctx context.Context, clusterName string) (*ecs.Cluster, error) {
	input := &awsecs.DescribeClustersInput{
		Clusters: []string{clusterName},
		Include:  []types.ClusterField{types.ClusterFieldTags, types.ClusterFieldSettings, types.ClusterFieldConfigurations},
	}

	output, err := r.client.DescribeClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	if len(output.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found")
	}

	cluster := &ecs.Cluster{
		ClusterName: clusterName,
	}
	r.populateClusterFromAWS(cluster, &output.Clusters[0])

	return cluster, nil
}

// Delete deletes an ECS cluster
func (r *Repository) Delete(ctx context.Context, clusterARN string) error {
	input := &awsecs.DeleteClusterInput{
		Cluster: aws.String(clusterARN),
	}

	_, err := r.client.DeleteCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}

// UpdateSettings updates cluster settings
func (r *Repository) UpdateSettings(ctx context.Context, cluster *ecs.Cluster) error {
	var settings []types.ClusterSetting
	for _, setting := range cluster.Settings {
		settings = append(settings, types.ClusterSetting{
			Name:  types.ClusterSettingName(setting.Name),
			Value: aws.String(setting.Value),
		})
	}

	input := &awsecs.UpdateClusterSettingsInput{
		Cluster:  aws.String(cluster.ClusterARN),
		Settings: settings,
	}

	_, err := r.client.UpdateClusterSettings(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update cluster settings: %w", err)
	}

	return nil
}

// UpdateCapacityProviders updates cluster capacity providers
func (r *Repository) UpdateCapacityProviders(ctx context.Context, cluster *ecs.Cluster) error {
	input := &awsecs.PutClusterCapacityProvidersInput{
		Cluster:           aws.String(cluster.ClusterARN),
		CapacityProviders: cluster.CapacityProviders,
	}

	// Add default capacity provider strategy
	if len(cluster.DefaultCapacityProviderStrategy) > 0 {
		var strategy []types.CapacityProviderStrategyItem
		for _, item := range cluster.DefaultCapacityProviderStrategy {
			strategy = append(strategy, types.CapacityProviderStrategyItem{
				CapacityProvider: aws.String(item.CapacityProvider),
				Weight:           item.Weight,
				Base:             item.Base,
			})
		}
		input.DefaultCapacityProviderStrategy = strategy
	}

	_, err := r.client.PutClusterCapacityProviders(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update cluster capacity providers: %w", err)
	}

	return nil
}

// TagResource adds or updates tags on a cluster
func (r *Repository) TagResource(ctx context.Context, clusterARN string, tags map[string]string) error {
	var awsTags []types.Tag
	for key, value := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	input := &awsecs.TagResourceInput{
		ResourceArn: aws.String(clusterARN),
		Tags:        awsTags,
	}

	_, err := r.client.TagResource(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag cluster: %w", err)
	}

	return nil
}

// populateClusterFromAWS populates domain cluster from AWS cluster
func (r *Repository) populateClusterFromAWS(cluster *ecs.Cluster, awsCluster *types.Cluster) {
	cluster.ClusterARN = aws.ToString(awsCluster.ClusterArn)
	cluster.Status = aws.ToString(awsCluster.Status)
	cluster.RegisteredContainerInstancesCount = awsCluster.RegisteredContainerInstancesCount
	cluster.RunningTasksCount = awsCluster.RunningTasksCount
	cluster.PendingTasksCount = awsCluster.PendingTasksCount
	cluster.ActiveServicesCount = awsCluster.ActiveServicesCount

	// Populate capacity providers
	cluster.CapacityProviders = awsCluster.CapacityProviders

	// Populate default capacity provider strategy
	if len(awsCluster.DefaultCapacityProviderStrategy) > 0 {
		var strategy []ecs.CapacityProviderStrategyItem
		for _, item := range awsCluster.DefaultCapacityProviderStrategy {
			strategy = append(strategy, ecs.CapacityProviderStrategyItem{
				CapacityProvider: aws.ToString(item.CapacityProvider),
				Weight:           item.Weight,
				Base:             item.Base,
			})
		}
		cluster.DefaultCapacityProviderStrategy = strategy
	}

	// Populate settings
	if len(awsCluster.Settings) > 0 {
		var settings []ecs.ClusterSetting
		for _, setting := range awsCluster.Settings {
			settings = append(settings, ecs.ClusterSetting{
				Name:  string(setting.Name),
				Value: aws.ToString(setting.Value),
			})
		}
		cluster.Settings = settings
	}

	// Populate tags
	if len(awsCluster.Tags) > 0 {
		if cluster.Tags == nil {
			cluster.Tags = make(map[string]string)
		}
		for _, tag := range awsCluster.Tags {
			cluster.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
}
