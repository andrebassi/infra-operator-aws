package elasticache

import (
	"errors"
	"time"
)

var (
	ErrInvalidClusterID     = errors.New("cluster ID is required")
	ErrInvalidEngine        = errors.New("invalid engine (must be redis or memcached)")
	ErrInvalidEngineVersion = errors.New("engine version is required")
	ErrInvalidNodeType      = errors.New("node type is required")
	ErrInvalidNodeCount     = errors.New("num cache nodes must be >= 1")
	ErrInvalidNumNodeGroups = errors.New("num node groups must be >= 1 for cluster mode")
)

type Cluster struct {
	ClusterID                   string
	Engine                      string
	EngineVersion               string
	NodeType                    string
	NumCacheNodes               int32
	ReplicationGroupDescription string
	NumNodeGroups               int32
	ReplicasPerNodeGroup        int32
	AutomaticFailoverEnabled    bool
	MultiAZEnabled              bool
	SubnetGroupName             string
	SecurityGroupIds            []string
	ParameterGroupName          string
	SnapshotRetentionLimit      int32
	SnapshotWindow              string
	PreferredMaintenanceWindow  string
	AtRestEncryptionEnabled     bool
	TransitEncryptionEnabled    bool
	AuthToken                   string
	KmsKeyId                    string
	NotificationTopicArn        string
	Tags                        map[string]string
	DeletionPolicy              string
	FinalSnapshotIdentifier     string

	// Status fields
	ClusterStatus         string
	CacheClusterARN       string
	ConfigurationEndpoint *Endpoint
	PrimaryEndpoint       *Endpoint
	ReaderEndpoint        *Endpoint
	NodeEndpoints         []Endpoint
	CacheNodeType         string
	MemberClusters        []string
	ClusterCreateTime     *time.Time
	LastSyncTime          *time.Time
}

type Endpoint struct {
	Address string
	Port    int32
}

func (c *Cluster) SetDefaults() {
	if c.DeletionPolicy == "" {
		c.DeletionPolicy = "Delete"
	}
	if c.Tags == nil {
		c.Tags = make(map[string]string)
	}
	if c.Engine == "redis" && c.ReplicationGroupDescription == "" {
		c.ReplicationGroupDescription = "Managed by infra-operator"
	}
	// Default to single node if not specified
	if c.NumCacheNodes == 0 && c.NumNodeGroups == 0 {
		c.NumCacheNodes = 1
	}
}

func (c *Cluster) Validate() error {
	if c.ClusterID == "" {
		return ErrInvalidClusterID
	}
	if c.Engine != "redis" && c.Engine != "memcached" {
		return ErrInvalidEngine
	}
	if c.EngineVersion == "" {
		return ErrInvalidEngineVersion
	}
	if c.NodeType == "" {
		return ErrInvalidNodeType
	}

	// Validate node counts
	if c.Engine == "memcached" {
		if c.NumCacheNodes < 1 {
			return ErrInvalidNodeCount
		}
	} else if c.Engine == "redis" {
		// Redis cluster mode
		if c.NumNodeGroups > 0 {
			if c.NumNodeGroups < 1 {
				return ErrInvalidNumNodeGroups
			}
		} else if c.NumCacheNodes < 1 {
			// Non-cluster mode
			return ErrInvalidNodeCount
		}
	}

	return nil
}

func (c *Cluster) ShouldDelete() bool {
	return c.DeletionPolicy == "Delete"
}

func (c *Cluster) ShouldSnapshot() bool {
	return c.DeletionPolicy == "Snapshot"
}

func (c *Cluster) IsRedis() bool {
	return c.Engine == "redis"
}

func (c *Cluster) IsMemcached() bool {
	return c.Engine == "memcached"
}

func (c *Cluster) IsClusterMode() bool {
	return c.IsRedis() && c.NumNodeGroups > 0
}

func (c *Cluster) IsAvailable() bool {
	return c.ClusterStatus == "available"
}

func (c *Cluster) IsCreating() bool {
	return c.ClusterStatus == "creating"
}

func (c *Cluster) IsDeleting() bool {
	return c.ClusterStatus == "deleting"
}
