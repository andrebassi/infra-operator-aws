package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElastiCacheClusterSpec defines the desired state of ElastiCacheCluster
type ElastiCacheClusterSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// ClusterID (also used as replication group ID for Redis cluster mode)
	// +kubebuilder:validation:Required
	ClusterID string `json:"clusterID"`

	// Engine (redis or memcached)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=redis;memcached
	Engine string `json:"engine"`

	// EngineVersion
	// +kubebuilder:validation:Required
	EngineVersion string `json:"engineVersion"`

	// NodeType (e.g., cache.t3.micro, cache.r6g.large)
	// +kubebuilder:validation:Required
	NodeType string `json:"nodeType"`

	// NumCacheNodes (for memcached or Redis without cluster mode)
	// +optional
	// +kubebuilder:validation:Minimum=1
	NumCacheNodes int32 `json:"numCacheNodes,omitempty"`

	// ReplicationGroupDescription (for Redis replication groups)
	// +optional
	ReplicationGroupDescription string `json:"replicationGroupDescription,omitempty"`

	// NumNodeGroups (for Redis cluster mode)
	// +optional
	// +kubebuilder:validation:Minimum=1
	NumNodeGroups int32 `json:"numNodeGroups,omitempty"`

	// ReplicasPerNodeGroup (for Redis cluster mode)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	ReplicasPerNodeGroup int32 `json:"replicasPerNodeGroup,omitempty"`

	// AutomaticFailoverEnabled (for Redis replication groups)
	// +optional
	AutomaticFailoverEnabled bool `json:"automaticFailoverEnabled,omitempty"`

	// MultiAZEnabled
	// +optional
	MultiAZEnabled bool `json:"multiAZEnabled,omitempty"`

	// SubnetGroupName
	// +optional
	SubnetGroupName string `json:"subnetGroupName,omitempty"`

	// SecurityGroupIds
	// +optional
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`

	// ParameterGroupName
	// +optional
	ParameterGroupName string `json:"parameterGroupName,omitempty"`

	// SnapshotRetentionLimit (0-35 days)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=35
	SnapshotRetentionLimit int32 `json:"snapshotRetentionLimit,omitempty"`

	// SnapshotWindow (e.g., "05:00-09:00")
	// +optional
	SnapshotWindow string `json:"snapshotWindow,omitempty"`

	// PreferredMaintenanceWindow
	// +optional
	PreferredMaintenanceWindow string `json:"preferredMaintenanceWindow,omitempty"`

	// AtRestEncryptionEnabled
	// +optional
	AtRestEncryptionEnabled bool `json:"atRestEncryptionEnabled,omitempty"`

	// TransitEncryptionEnabled
	// +optional
	TransitEncryptionEnabled bool `json:"transitEncryptionEnabled,omitempty"`

	// AuthToken (for Redis with transit encryption)
	// +optional
	AuthTokenRef *SecretKeySelector `json:"authTokenRef,omitempty"`

	// KmsKeyId for encryption at rest
	// +optional
	KmsKeyId string `json:"kmsKeyId,omitempty"`

	// NotificationTopicArn
	// +optional
	NotificationTopicArn string `json:"notificationTopicArn,omitempty"`

	// Tags
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain;Snapshot
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// FinalSnapshotIdentifier (required if DeletionPolicy=Snapshot)
	// +optional
	FinalSnapshotIdentifier string `json:"finalSnapshotIdentifier,omitempty"`
}

// ElastiCacheClusterStatus defines the observed state of ElastiCacheCluster
type ElastiCacheClusterStatus struct {
	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ClusterStatus (creating, available, modifying, deleting, etc)
	// +optional
	ClusterStatus string `json:"clusterStatus,omitempty"`

	// CacheClusterARN
	// +optional
	CacheClusterARN string `json:"cacheClusterARN,omitempty"`

	// ConfigurationEndpoint (for cluster mode)
	// +optional
	ConfigurationEndpoint *CacheEndpoint `json:"configurationEndpoint,omitempty"`

	// PrimaryEndpoint
	// +optional
	PrimaryEndpoint *CacheEndpoint `json:"primaryEndpoint,omitempty"`

	// ReaderEndpoint
	// +optional
	ReaderEndpoint *CacheEndpoint `json:"readerEndpoint,omitempty"`

	// NodeEndpoints (individual node endpoints)
	// +optional
	NodeEndpoints []CacheEndpoint `json:"nodeEndpoints,omitempty"`

	// CacheNodeType
	// +optional
	CacheNodeType string `json:"cacheNodeType,omitempty"`

	// EngineVersion
	// +optional
	EngineVersion string `json:"engineVersion,omitempty"`

	// MemberClusters
	// +optional
	MemberClusters []string `json:"memberClusters,omitempty"`

	// ClusterCreateTime
	// +optional
	ClusterCreateTime *metav1.Time `json:"clusterCreateTime,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// CacheEndpoint defines a cache endpoint
type CacheEndpoint struct {
	// Address
	Address string `json:"address"`

	// Port
	Port int32 `json:"port"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=elasticache
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterID`
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="NodeType",type=string,JSONPath=`.spec.nodeType`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.clusterStatus`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.primaryEndpoint.address`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ElastiCacheCluster is the Schema for the elasticacheclusters API
type ElastiCacheCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElastiCacheClusterSpec   `json:"spec,omitempty"`
	Status ElastiCacheClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElastiCacheClusterList contains a list of ElastiCacheCluster
type ElastiCacheClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElastiCacheCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElastiCacheCluster{}, &ElastiCacheClusterList{})
}
