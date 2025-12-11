package eks

import (
	"errors"
	"time"
)

var (
	ErrInvalidClusterName = errors.New("cluster name is required")
	ErrInvalidVersion     = errors.New("Kubernetes version is required")
	ErrInvalidRoleARN     = errors.New("role ARN is required")
	ErrInvalidSubnets     = errors.New("at least 2 subnets are required")
)

type Cluster struct {
	ClusterName      string
	Version          string
	RoleARN          string
	VpcConfig        VpcConfig
	Logging          *Logging
	Encryption       *Encryption
	Tags             map[string]string
	DeletionPolicy   string

	// Status fields
	ARN                  string
	Endpoint             string
	Status               string
	PlatformVersion      string
	CertificateAuthority string
	LastSyncTime         *time.Time
}

type VpcConfig struct {
	SubnetIDs             []string
	SecurityGroupIDs      []string
	EndpointPublicAccess  bool
	EndpointPrivateAccess bool
	PublicAccessCidrs     []string
}

type Logging struct {
	ClusterLogging []string
}

type Encryption struct {
	Resources      []string
	ProviderKeyARN string
}

func (c *Cluster) SetDefaults() {
	if c.DeletionPolicy == "" {
		c.DeletionPolicy = "Delete"
	}
	if c.Tags == nil {
		c.Tags = make(map[string]string)
	}
	if c.VpcConfig.PublicAccessCidrs == nil {
		c.VpcConfig.PublicAccessCidrs = []string{"0.0.0.0/0"}
	}
	// Default to public access enabled
	if !c.VpcConfig.EndpointPublicAccess && !c.VpcConfig.EndpointPrivateAccess {
		c.VpcConfig.EndpointPublicAccess = true
	}
}

func (c *Cluster) Validate() error {
	if c.ClusterName == "" {
		return ErrInvalidClusterName
	}
	if c.Version == "" {
		return ErrInvalidVersion
	}
	if c.RoleARN == "" {
		return ErrInvalidRoleARN
	}
	if len(c.VpcConfig.SubnetIDs) < 2 {
		return ErrInvalidSubnets
	}
	return nil
}

func (c *Cluster) ShouldDelete() bool {
	return c.DeletionPolicy == "Delete"
}

func (c *Cluster) IsActive() bool {
	return c.Status == "ACTIVE"
}

func (c *Cluster) IsCreating() bool {
	return c.Status == "CREATING"
}

func (c *Cluster) IsDeleting() bool {
	return c.Status == "DELETING"
}

func (c *Cluster) IsFailed() bool {
	return c.Status == "FAILED"
}
