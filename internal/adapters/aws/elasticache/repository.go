package elasticache

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awselasticache "github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"

	"infra-operator/internal/domain/elasticache"
)

type Repository struct {
	client *awselasticache.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awselasticache.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, clusterID string, isRedisCluster bool) (bool, error) {
	if isRedisCluster {
		// For Redis replication groups
		output, err := r.client.DescribeReplicationGroups(ctx, &awselasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(clusterID),
		})
		if err != nil {
			return false, nil
		}
		return len(output.ReplicationGroups) > 0, nil
	}

	// For single cache clusters (memcached or single Redis node)
	output, err := r.client.DescribeCacheClusters(ctx, &awselasticache.DescribeCacheClustersInput{
		CacheClusterId: aws.String(clusterID),
	})
	if err != nil {
		return false, nil
	}
	return len(output.CacheClusters) > 0, nil
}

func (r *Repository) CreateReplicationGroup(ctx context.Context, cluster *elasticache.Cluster) error {
	input := &awselasticache.CreateReplicationGroupInput{
		ReplicationGroupId:          aws.String(cluster.ClusterID),
		ReplicationGroupDescription: aws.String(cluster.ReplicationGroupDescription),
		Engine:                      aws.String(cluster.Engine),
		CacheNodeType:               aws.String(cluster.NodeType),
		AutomaticFailoverEnabled:    aws.Bool(cluster.AutomaticFailoverEnabled),
		MultiAZEnabled:              aws.Bool(cluster.MultiAZEnabled),
	}

	// Engine version
	if cluster.EngineVersion != "" {
		input.EngineVersion = aws.String(cluster.EngineVersion)
	}

	// Cluster mode (sharded)
	if cluster.NumNodeGroups > 0 {
		input.NumNodeGroups = aws.Int32(cluster.NumNodeGroups)
		input.ReplicasPerNodeGroup = aws.Int32(cluster.ReplicasPerNodeGroup)
	} else {
		// Non-cluster mode
		input.NumCacheClusters = aws.Int32(cluster.NumCacheNodes)
	}

	// Network settings
	if cluster.SubnetGroupName != "" {
		input.CacheSubnetGroupName = aws.String(cluster.SubnetGroupName)
	}
	if len(cluster.SecurityGroupIds) > 0 {
		input.SecurityGroupIds = cluster.SecurityGroupIds
	}

	// Parameter group
	if cluster.ParameterGroupName != "" {
		input.CacheParameterGroupName = aws.String(cluster.ParameterGroupName)
	}

	// Snapshots
	if cluster.SnapshotRetentionLimit > 0 {
		input.SnapshotRetentionLimit = aws.Int32(cluster.SnapshotRetentionLimit)
	}
	if cluster.SnapshotWindow != "" {
		input.SnapshotWindow = aws.String(cluster.SnapshotWindow)
	}

	// Maintenance window
	if cluster.PreferredMaintenanceWindow != "" {
		input.PreferredMaintenanceWindow = aws.String(cluster.PreferredMaintenanceWindow)
	}

	// Encryption
	if cluster.AtRestEncryptionEnabled {
		input.AtRestEncryptionEnabled = aws.Bool(true)
		if cluster.KmsKeyId != "" {
			input.KmsKeyId = aws.String(cluster.KmsKeyId)
		}
	}

	if cluster.TransitEncryptionEnabled {
		input.TransitEncryptionEnabled = aws.Bool(true)
		if cluster.AuthToken != "" {
			input.AuthToken = aws.String(cluster.AuthToken)
		}
	}

	// Notifications
	if cluster.NotificationTopicArn != "" {
		input.NotificationTopicArn = aws.String(cluster.NotificationTopicArn)
	}

	// Tags
	if len(cluster.Tags) > 0 {
		input.Tags = convertTags(cluster.Tags)
	}

	output, err := r.client.CreateReplicationGroup(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create replication group: %w", err)
	}

	// Update cluster with created info
	if output.ReplicationGroup != nil {
		rg := output.ReplicationGroup
		cluster.ClusterStatus = aws.ToString(rg.Status)
		cluster.CacheClusterARN = aws.ToString(rg.ARN)

		if rg.ConfigurationEndpoint != nil {
			cluster.ConfigurationEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(rg.ConfigurationEndpoint.Address),
				Port:    aws.ToInt32(rg.ConfigurationEndpoint.Port),
			}
		}

		if len(rg.NodeGroups) > 0 && rg.NodeGroups[0].PrimaryEndpoint != nil {
			cluster.PrimaryEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(rg.NodeGroups[0].PrimaryEndpoint.Address),
				Port:    aws.ToInt32(rg.NodeGroups[0].PrimaryEndpoint.Port),
			}
		}

		if len(rg.NodeGroups) > 0 && rg.NodeGroups[0].ReaderEndpoint != nil {
			cluster.ReaderEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(rg.NodeGroups[0].ReaderEndpoint.Address),
				Port:    aws.ToInt32(rg.NodeGroups[0].ReaderEndpoint.Port),
			}
		}

		cluster.MemberClusters = rg.MemberClusters
	}

	return nil
}

func (r *Repository) CreateCacheCluster(ctx context.Context, cluster *elasticache.Cluster) error {
	input := &awselasticache.CreateCacheClusterInput{
		CacheClusterId: aws.String(cluster.ClusterID),
		Engine:         aws.String(cluster.Engine),
		CacheNodeType:  aws.String(cluster.NodeType),
		NumCacheNodes:  aws.Int32(cluster.NumCacheNodes),
	}

	if cluster.EngineVersion != "" {
		input.EngineVersion = aws.String(cluster.EngineVersion)
	}

	if cluster.SubnetGroupName != "" {
		input.CacheSubnetGroupName = aws.String(cluster.SubnetGroupName)
	}

	if len(cluster.SecurityGroupIds) > 0 {
		input.SecurityGroupIds = cluster.SecurityGroupIds
	}

	if cluster.ParameterGroupName != "" {
		input.CacheParameterGroupName = aws.String(cluster.ParameterGroupName)
	}

	if cluster.PreferredMaintenanceWindow != "" {
		input.PreferredMaintenanceWindow = aws.String(cluster.PreferredMaintenanceWindow)
	}

	if cluster.NotificationTopicArn != "" {
		input.NotificationTopicArn = aws.String(cluster.NotificationTopicArn)
	}

	if len(cluster.Tags) > 0 {
		input.Tags = convertTags(cluster.Tags)
	}

	output, err := r.client.CreateCacheCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create cache cluster: %w", err)
	}

	if output.CacheCluster != nil {
		cc := output.CacheCluster
		cluster.ClusterStatus = aws.ToString(cc.CacheClusterStatus)
		cluster.CacheClusterARN = aws.ToString(cc.ARN)

		if cc.ConfigurationEndpoint != nil {
			cluster.ConfigurationEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(cc.ConfigurationEndpoint.Address),
				Port:    aws.ToInt32(cc.ConfigurationEndpoint.Port),
			}
		}

		if cc.CacheClusterCreateTime != nil {
			t := *cc.CacheClusterCreateTime
			cluster.ClusterCreateTime = &t
		}
	}

	return nil
}

func (r *Repository) GetReplicationGroup(ctx context.Context, clusterID string) (*elasticache.Cluster, error) {
	output, err := r.client.DescribeReplicationGroups(ctx, &awselasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(clusterID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe replication group: %w", err)
	}

	if len(output.ReplicationGroups) == 0 {
		return nil, fmt.Errorf("replication group not found")
	}

	rg := output.ReplicationGroups[0]
	cluster := &elasticache.Cluster{
		ClusterID:                   aws.ToString(rg.ReplicationGroupId),
		ReplicationGroupDescription: aws.ToString(rg.Description),
		ClusterStatus:               aws.ToString(rg.Status),
		CacheClusterARN:             aws.ToString(rg.ARN),
		AutomaticFailoverEnabled:    rg.AutomaticFailover == types.AutomaticFailoverStatusEnabled,
		MultiAZEnabled:              rg.MultiAZ == types.MultiAZStatusEnabled,
		MemberClusters:              rg.MemberClusters,
	}

	if rg.ConfigurationEndpoint != nil {
		cluster.ConfigurationEndpoint = &elasticache.Endpoint{
			Address: aws.ToString(rg.ConfigurationEndpoint.Address),
			Port:    aws.ToInt32(rg.ConfigurationEndpoint.Port),
		}
	}

	if len(rg.NodeGroups) > 0 {
		if rg.NodeGroups[0].PrimaryEndpoint != nil {
			cluster.PrimaryEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(rg.NodeGroups[0].PrimaryEndpoint.Address),
				Port:    aws.ToInt32(rg.NodeGroups[0].PrimaryEndpoint.Port),
			}
		}
		if rg.NodeGroups[0].ReaderEndpoint != nil {
			cluster.ReaderEndpoint = &elasticache.Endpoint{
				Address: aws.ToString(rg.NodeGroups[0].ReaderEndpoint.Address),
				Port:    aws.ToInt32(rg.NodeGroups[0].ReaderEndpoint.Port),
			}
		}
	}

	return cluster, nil
}

func (r *Repository) GetCacheCluster(ctx context.Context, clusterID string) (*elasticache.Cluster, error) {
	output, err := r.client.DescribeCacheClusters(ctx, &awselasticache.DescribeCacheClustersInput{
		CacheClusterId:    aws.String(clusterID),
		ShowCacheNodeInfo: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cache cluster: %w", err)
	}

	if len(output.CacheClusters) == 0 {
		return nil, fmt.Errorf("cache cluster not found")
	}

	cc := output.CacheClusters[0]
	cluster := &elasticache.Cluster{
		ClusterID:       aws.ToString(cc.CacheClusterId),
		Engine:          aws.ToString(cc.Engine),
		EngineVersion:   aws.ToString(cc.EngineVersion),
		NodeType:        aws.ToString(cc.CacheNodeType),
		NumCacheNodes:   aws.ToInt32(cc.NumCacheNodes),
		ClusterStatus:   aws.ToString(cc.CacheClusterStatus),
		CacheClusterARN: aws.ToString(cc.ARN),
	}

	if cc.ConfigurationEndpoint != nil {
		cluster.ConfigurationEndpoint = &elasticache.Endpoint{
			Address: aws.ToString(cc.ConfigurationEndpoint.Address),
			Port:    aws.ToInt32(cc.ConfigurationEndpoint.Port),
		}
	}

	if cc.CacheClusterCreateTime != nil {
		t := *cc.CacheClusterCreateTime
		cluster.ClusterCreateTime = &t
	}

	// Extract node endpoints
	if len(cc.CacheNodes) > 0 {
		cluster.NodeEndpoints = make([]elasticache.Endpoint, 0, len(cc.CacheNodes))
		for _, node := range cc.CacheNodes {
			if node.Endpoint != nil {
				cluster.NodeEndpoints = append(cluster.NodeEndpoints, elasticache.Endpoint{
					Address: aws.ToString(node.Endpoint.Address),
					Port:    aws.ToInt32(node.Endpoint.Port),
				})
			}
		}
	}

	return cluster, nil
}

func (r *Repository) DeleteReplicationGroup(ctx context.Context, clusterID string, finalSnapshotID string) error {
	input := &awselasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: aws.String(clusterID),
	}

	if finalSnapshotID != "" {
		input.FinalSnapshotIdentifier = aws.String(finalSnapshotID)
	}

	_, err := r.client.DeleteReplicationGroup(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete replication group: %w", err)
	}

	return nil
}

func (r *Repository) DeleteCacheCluster(ctx context.Context, clusterID string, finalSnapshotID string) error {
	input := &awselasticache.DeleteCacheClusterInput{
		CacheClusterId: aws.String(clusterID),
	}

	if finalSnapshotID != "" {
		input.FinalSnapshotIdentifier = aws.String(finalSnapshotID)
	}

	_, err := r.client.DeleteCacheCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete cache cluster: %w", err)
	}

	return nil
}

func convertTags(tags map[string]string) []types.Tag {
	elasticacheTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		elasticacheTags = append(elasticacheTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return elasticacheTags
}
