package ecs

import (
	"errors"
	"time"
)

var (
	ErrInvalidClusterName = errors.New("cluster name cannot be empty")
	ErrInvalidSettings    = errors.New("invalid cluster settings")
)

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterName                       string
	ClusterARN                        string
	Status                            string
	CapacityProviders                 []string
	DefaultCapacityProviderStrategy   []CapacityProviderStrategyItem
	Settings                          []ClusterSetting
	Configuration                     *ClusterConfiguration
	ServiceConnectDefaults            *ServiceConnectDefaults
	Tags                              map[string]string
	DeletionPolicy                    string
	RegisteredContainerInstancesCount int32
	RunningTasksCount                 int32
	PendingTasksCount                 int32
	ActiveServicesCount               int32
	LastSyncTime                      *time.Time
}

// CapacityProviderStrategyItem represents a capacity provider strategy
type CapacityProviderStrategyItem struct {
	CapacityProvider string
	Weight           int32
	Base             int32
}

// ClusterSetting represents a cluster setting
type ClusterSetting struct {
	Name  string
	Value string
}

// ClusterConfiguration represents cluster configuration
type ClusterConfiguration struct {
	ExecuteCommandConfiguration *ExecuteCommandConfiguration
}

// ExecuteCommandConfiguration represents execute command configuration
type ExecuteCommandConfiguration struct {
	KmsKeyID         string
	Logging          string
	LogConfiguration *ExecuteCommandLogConfiguration
}

// ExecuteCommandLogConfiguration represents log configuration
type ExecuteCommandLogConfiguration struct {
	CloudWatchLogGroupName      string
	CloudWatchEncryptionEnabled bool
	S3BucketName                string
	S3EncryptionEnabled         bool
	S3KeyPrefix                 string
}

// ServiceConnectDefaults represents service connect defaults
type ServiceConnectDefaults struct {
	Namespace string
}

// Validate validates the cluster configuration
func (c *Cluster) Validate() error {
	if c.ClusterName == "" {
		return ErrInvalidClusterName
	}

	// Validate settings
	for _, setting := range c.Settings {
		if setting.Name == "" || setting.Value == "" {
			return ErrInvalidSettings
		}
		// containerInsights can be "enabled" or "disabled"
		if setting.Name == "containerInsights" {
			if setting.Value != "enabled" && setting.Value != "disabled" {
				return ErrInvalidSettings
			}
		}
	}

	return nil
}

// SetDefaults sets default values for the cluster
func (c *Cluster) SetDefaults() {
	if c.DeletionPolicy == "" {
		c.DeletionPolicy = "Delete"
	}

	if c.Tags == nil {
		c.Tags = make(map[string]string)
	}

	// Enable container insights by default
	if len(c.Settings) == 0 {
		c.Settings = []ClusterSetting{
			{
				Name:  "containerInsights",
				Value: "enabled",
			},
		}
	}
}

// ShouldDelete returns true if the cluster should be deleted when the CR is deleted
func (c *Cluster) ShouldDelete() bool {
	return c.DeletionPolicy == "Delete"
}

// IsActive returns true if the cluster is active
func (c *Cluster) IsActive() bool {
	return c.Status == "ACTIVE"
}

// IsProvisioning returns true if the cluster is being provisioned
func (c *Cluster) IsProvisioning() bool {
	return c.Status == "PROVISIONING"
}

// IsFailed returns true if the cluster failed to provision
func (c *Cluster) IsFailed() bool {
	return c.Status == "FAILED"
}

// IsInactive returns true if the cluster is inactive
func (c *Cluster) IsInactive() bool {
	return c.Status == "INACTIVE"
}

// IsDeprovisioning returns true if the cluster is being deprovisioned
func (c *Cluster) IsDeprovisioning() bool {
	return c.Status == "DEPROVISIONING"
}

// HasTasks returns true if the cluster has running or pending tasks
func (c *Cluster) HasTasks() bool {
	return c.RunningTasksCount > 0 || c.PendingTasksCount > 0
}

// HasServices returns true if the cluster has active services
func (c *Cluster) HasServices() bool {
	return c.ActiveServicesCount > 0
}

// HasContainerInstances returns true if the cluster has registered container instances
func (c *Cluster) HasContainerInstances() bool {
	return c.RegisteredContainerInstancesCount > 0
}
