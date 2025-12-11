package alb

import (
	"errors"
	"time"
)

var (
	ErrInvalidLoadBalancerName = errors.New("load balancer name cannot be empty")
	ErrInvalidScheme           = errors.New("scheme must be 'internet-facing' or 'internal'")
	ErrInvalidSubnets          = errors.New("at least 2 subnets in different AZs are required")
	ErrInvalidIdleTimeout      = errors.New("idle timeout must be between 1 and 4000 seconds")
)

// LoadBalancer represents an Application Load Balancer in the domain model
type LoadBalancer struct {
	// Identification
	LoadBalancerName string
	LoadBalancerARN  string
	DNSName          string

	// Configuration
	Scheme                   string   // internet-facing or internal
	Subnets                  []string // Minimum 2 subnets
	SecurityGroups           []string
	IPAddressType            string // ipv4 or dualstack
	EnableDeletionProtection bool
	EnableHttp2              bool
	EnableWafFailOpen        bool
	IdleTimeout              int32

	// Tags
	Tags map[string]string

	// Deletion
	DeletionPolicy string

	// State
	State                 string
	VpcID                 string
	CanonicalHostedZoneID string
	LastSyncTime          *time.Time
}

// Validate checks if the load balancer configuration is valid
func (lb *LoadBalancer) Validate() error {
	if lb.LoadBalancerName == "" {
		return ErrInvalidLoadBalancerName
	}

	// Validate scheme
	if lb.Scheme != "" {
		if lb.Scheme != "internet-facing" && lb.Scheme != "internal" {
			return ErrInvalidScheme
		}
	}

	// Validate subnets (minimum 2)
	if len(lb.Subnets) < 2 {
		return ErrInvalidSubnets
	}

	// Validate idle timeout
	if lb.IdleTimeout != 0 && (lb.IdleTimeout < 1 || lb.IdleTimeout > 4000) {
		return ErrInvalidIdleTimeout
	}

	return nil
}

// SetDefaults sets default values for optional fields
func (lb *LoadBalancer) SetDefaults() {
	if lb.Scheme == "" {
		lb.Scheme = "internet-facing"
	}

	if lb.IPAddressType == "" {
		lb.IPAddressType = "ipv4"
	}

	if lb.IdleTimeout == 0 {
		lb.IdleTimeout = 60
	}

	if lb.DeletionPolicy == "" {
		lb.DeletionPolicy = "Delete"
	}

	if lb.Tags == nil {
		lb.Tags = make(map[string]string)
	}

	// Enable HTTP/2 by default
	if !lb.EnableHttp2 {
		lb.EnableHttp2 = true
	}
}

// ShouldDelete returns true if the load balancer should be deleted when CR is deleted
func (lb *LoadBalancer) ShouldDelete() bool {
	return lb.DeletionPolicy == "Delete"
}

// IsActive checks if the load balancer is in active state
func (lb *LoadBalancer) IsActive() bool {
	return lb.State == "active"
}

// IsProvisioning checks if the load balancer is being provisioned
func (lb *LoadBalancer) IsProvisioning() bool {
	return lb.State == "provisioning"
}

// IsFailed checks if the load balancer is in failed state
func (lb *LoadBalancer) IsFailed() bool {
	return lb.State == "failed"
}

// IsInternetFacing returns true if the load balancer is internet-facing
func (lb *LoadBalancer) IsInternetFacing() bool {
	return lb.Scheme == "internet-facing"
}

// IsInternal returns true if the load balancer is internal
func (lb *LoadBalancer) IsInternal() bool {
	return lb.Scheme == "internal"
}
