package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/elasticache"
)

func CRToDomainElastiCacheCluster(cr *infrav1alpha1.ElastiCacheCluster, authToken string) *elasticache.Cluster {
	cluster := &elasticache.Cluster{
		ClusterID:                   cr.Spec.ClusterID,
		Engine:                      cr.Spec.Engine,
		EngineVersion:               cr.Spec.EngineVersion,
		NodeType:                    cr.Spec.NodeType,
		NumCacheNodes:               cr.Spec.NumCacheNodes,
		ReplicationGroupDescription: cr.Spec.ReplicationGroupDescription,
		NumNodeGroups:               cr.Spec.NumNodeGroups,
		ReplicasPerNodeGroup:        cr.Spec.ReplicasPerNodeGroup,
		AutomaticFailoverEnabled:    cr.Spec.AutomaticFailoverEnabled,
		MultiAZEnabled:              cr.Spec.MultiAZEnabled,
		SubnetGroupName:             cr.Spec.SubnetGroupName,
		SecurityGroupIds:            cr.Spec.SecurityGroupIds,
		ParameterGroupName:          cr.Spec.ParameterGroupName,
		SnapshotRetentionLimit:      cr.Spec.SnapshotRetentionLimit,
		SnapshotWindow:              cr.Spec.SnapshotWindow,
		PreferredMaintenanceWindow:  cr.Spec.PreferredMaintenanceWindow,
		AtRestEncryptionEnabled:     cr.Spec.AtRestEncryptionEnabled,
		TransitEncryptionEnabled:    cr.Spec.TransitEncryptionEnabled,
		AuthToken:                   authToken,
		KmsKeyId:                    cr.Spec.KmsKeyId,
		NotificationTopicArn:        cr.Spec.NotificationTopicArn,
		Tags:                        cr.Spec.Tags,
		DeletionPolicy:              cr.Spec.DeletionPolicy,
		FinalSnapshotIdentifier:     cr.Spec.FinalSnapshotIdentifier,
	}

	// Copy status fields if available
	if cr.Status.ClusterStatus != "" {
		cluster.ClusterStatus = cr.Status.ClusterStatus
	}
	if cr.Status.CacheClusterARN != "" {
		cluster.CacheClusterARN = cr.Status.CacheClusterARN
	}
	if cr.Status.ConfigurationEndpoint != nil {
		cluster.ConfigurationEndpoint = &elasticache.Endpoint{
			Address: cr.Status.ConfigurationEndpoint.Address,
			Port:    cr.Status.ConfigurationEndpoint.Port,
		}
	}
	if cr.Status.PrimaryEndpoint != nil {
		cluster.PrimaryEndpoint = &elasticache.Endpoint{
			Address: cr.Status.PrimaryEndpoint.Address,
			Port:    cr.Status.PrimaryEndpoint.Port,
		}
	}
	if cr.Status.ReaderEndpoint != nil {
		cluster.ReaderEndpoint = &elasticache.Endpoint{
			Address: cr.Status.ReaderEndpoint.Address,
			Port:    cr.Status.ReaderEndpoint.Port,
		}
	}
	if len(cr.Status.NodeEndpoints) > 0 {
		cluster.NodeEndpoints = make([]elasticache.Endpoint, 0, len(cr.Status.NodeEndpoints))
		for _, ep := range cr.Status.NodeEndpoints {
			cluster.NodeEndpoints = append(cluster.NodeEndpoints, elasticache.Endpoint{
				Address: ep.Address,
				Port:    ep.Port,
			})
		}
	}
	cluster.MemberClusters = cr.Status.MemberClusters

	return cluster
}

func DomainToStatusElastiCacheCluster(cluster *elasticache.Cluster, cr *infrav1alpha1.ElastiCacheCluster) {
	now := metav1.Now()

	cr.Status.Ready = cluster.IsAvailable()
	cr.Status.ClusterStatus = cluster.ClusterStatus
	cr.Status.CacheClusterARN = cluster.CacheClusterARN
	cr.Status.CacheNodeType = cluster.NodeType
	cr.Status.EngineVersion = cluster.EngineVersion
	cr.Status.MemberClusters = cluster.MemberClusters
	cr.Status.LastSyncTime = &now

	if cluster.ConfigurationEndpoint != nil {
		cr.Status.ConfigurationEndpoint = &infrav1alpha1.CacheEndpoint{
			Address: cluster.ConfigurationEndpoint.Address,
			Port:    cluster.ConfigurationEndpoint.Port,
		}
	}

	if cluster.PrimaryEndpoint != nil {
		cr.Status.PrimaryEndpoint = &infrav1alpha1.CacheEndpoint{
			Address: cluster.PrimaryEndpoint.Address,
			Port:    cluster.PrimaryEndpoint.Port,
		}
	}

	if cluster.ReaderEndpoint != nil {
		cr.Status.ReaderEndpoint = &infrav1alpha1.CacheEndpoint{
			Address: cluster.ReaderEndpoint.Address,
			Port:    cluster.ReaderEndpoint.Port,
		}
	}

	if len(cluster.NodeEndpoints) > 0 {
		cr.Status.NodeEndpoints = make([]infrav1alpha1.CacheEndpoint, 0, len(cluster.NodeEndpoints))
		for _, ep := range cluster.NodeEndpoints {
			cr.Status.NodeEndpoints = append(cr.Status.NodeEndpoints, infrav1alpha1.CacheEndpoint{
				Address: ep.Address,
				Port:    ep.Port,
			})
		}
	}

	if cluster.ClusterCreateTime != nil {
		cr.Status.ClusterCreateTime = &metav1.Time{Time: *cluster.ClusterCreateTime}
	}

	// Update conditions
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: cr.Generation,
		LastTransitionTime: now,
		Reason:             "ClusterNotAvailable",
		Message:            "Cluster is not in available state",
	}

	if cluster.IsAvailable() {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "ClusterAvailable"
		condition.Message = "Cluster is available"
	} else if cluster.IsCreating() {
		condition.Reason = "ClusterCreating"
		condition.Message = "Cluster is being created"
	} else if cluster.IsDeleting() {
		condition.Reason = "ClusterDeleting"
		condition.Message = "Cluster is being deleted"
	}

	// Find and update or append condition
	conditionUpdated := false
	for i, c := range cr.Status.Conditions {
		if c.Type == "Ready" {
			// Only update LastTransitionTime if status actually changed
			if c.Status != condition.Status {
				cr.Status.Conditions[i] = condition
			} else {
				condition.LastTransitionTime = c.LastTransitionTime
				cr.Status.Conditions[i] = condition
			}
			conditionUpdated = true
			break
		}
	}
	if !conditionUpdated {
		cr.Status.Conditions = append(cr.Status.Conditions, condition)
	}

	// Set last sync time
	cluster.LastSyncTime = &time.Time{}
	*cluster.LastSyncTime = now.Time
}
