package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ECSClusterSpec defines the desired state of ECSCluster
type ECSClusterSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// ClusterName is the name of the ECS cluster
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	ClusterName string `json:"clusterName"`

	// CapacityProviders is a list of capacity providers to associate with the cluster
	// +optional
	CapacityProviders []string `json:"capacityProviders,omitempty"`

	// DefaultCapacityProviderStrategy defines the default capacity provider strategy for the cluster
	// +optional
	DefaultCapacityProviderStrategy []CapacityProviderStrategyItem `json:"defaultCapacityProviderStrategy,omitempty"`

	// Settings are the cluster settings (e.g., containerInsights)
	// +optional
	Settings []ClusterSetting `json:"settings,omitempty"`

	// Configuration for execute command functionality
	// +optional
	Configuration *ClusterConfiguration `json:"configuration,omitempty"`

	// ServiceConnectDefaults for the cluster
	// +optional
	ServiceConnectDefaults *ServiceConnectDefaults `json:"serviceConnectDefaults,omitempty"`

	// Tags to apply to the cluster
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the cluster when the CR is deleted
	// Valid values: Delete, Retain
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// CapacityProviderStrategyItem represents a capacity provider strategy
type CapacityProviderStrategyItem struct {
	// CapacityProvider is the name of the capacity provider
	CapacityProvider string `json:"capacityProvider"`

	// Weight is the relative percentage of the total number of tasks
	// +optional
	Weight int32 `json:"weight,omitempty"`

	// Base is the number of tasks to use this capacity provider for
	// +optional
	Base int32 `json:"base,omitempty"`
}

// ClusterSetting represents a cluster setting
type ClusterSetting struct {
	// Name of the cluster setting
	// Valid values: containerInsights
	Name string `json:"name"`

	// Value of the cluster setting
	Value string `json:"value"`
}

// ClusterConfiguration represents cluster configuration
type ClusterConfiguration struct {
	// ExecuteCommandConfiguration for the cluster
	// +optional
	ExecuteCommandConfiguration *ExecuteCommandConfiguration `json:"executeCommandConfiguration,omitempty"`
}

// ExecuteCommandConfiguration represents execute command configuration
type ExecuteCommandConfiguration struct {
	// KmsKeyID for encryption
	// +optional
	KmsKeyID string `json:"kmsKeyId,omitempty"`

	// Logging configuration
	// +optional
	Logging string `json:"logging,omitempty"`

	// LogConfiguration for CloudWatch Logs or S3
	// +optional
	LogConfiguration *ExecuteCommandLogConfiguration `json:"logConfiguration,omitempty"`
}

// ExecuteCommandLogConfiguration represents log configuration
type ExecuteCommandLogConfiguration struct {
	// CloudWatchLogGroupName for CloudWatch Logs
	// +optional
	CloudWatchLogGroupName string `json:"cloudWatchLogGroupName,omitempty"`

	// CloudWatchEncryptionEnabled enables encryption
	// +optional
	CloudWatchEncryptionEnabled bool `json:"cloudWatchEncryptionEnabled,omitempty"`

	// S3BucketName for S3 logs
	// +optional
	S3BucketName string `json:"s3BucketName,omitempty"`

	// S3EncryptionEnabled enables S3 encryption
	// +optional
	S3EncryptionEnabled bool `json:"s3EncryptionEnabled,omitempty"`

	// S3KeyPrefix for S3 logs
	// +optional
	S3KeyPrefix string `json:"s3KeyPrefix,omitempty"`
}

// ServiceConnectDefaults represents service connect defaults
type ServiceConnectDefaults struct {
	// Namespace for service connect
	Namespace string `json:"namespace"`
}

// ECSClusterStatus defines the observed state of ECSCluster
type ECSClusterStatus struct {
	// Ready indicates if the cluster is active
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ClusterARN is the ARN of the ECS cluster
	// +optional
	ClusterARN string `json:"clusterARN,omitempty"`

	// Status is the status of the cluster (ACTIVE, PROVISIONING, DEPROVISIONING, FAILED, INACTIVE)
	// +optional
	Status string `json:"status,omitempty"`

	// RegisteredContainerInstancesCount is the number of registered container instances
	// +optional
	RegisteredContainerInstancesCount int32 `json:"registeredContainerInstancesCount,omitempty"`

	// RunningTasksCount is the number of running tasks
	// +optional
	RunningTasksCount int32 `json:"runningTasksCount,omitempty"`

	// PendingTasksCount is the number of pending tasks
	// +optional
	PendingTasksCount int32 `json:"pendingTasksCount,omitempty"`

	// ActiveServicesCount is the number of active services
	// +optional
	ActiveServicesCount int32 `json:"activeServicesCount,omitempty"`

	// LastSyncTime is the last time the cluster was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ecscluster;ecsclusters
// +kubebuilder:printcolumn:name="Cluster Name",type=string,JSONPath=`.spec.clusterName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ECSCluster is the Schema for the ecsclusters API
type ECSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ECSClusterSpec   `json:"spec,omitempty"`
	Status ECSClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ECSClusterList contains a list of ECSCluster
type ECSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ECSCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ECSCluster{}, &ECSClusterList{})
}
