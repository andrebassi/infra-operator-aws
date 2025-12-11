package nlb

import (
	"errors"
	"time"
)

var (
	ErrInvalidLoadBalancerName = errors.New("load balancer name cannot be empty")
	ErrInvalidScheme           = errors.New("scheme must be 'internet-facing' or 'internal'")
	ErrInvalidSubnets          = errors.New("at least 1 subnet is required")
)

// LoadBalancer represents a Network Load Balancer
type LoadBalancer struct {
	LoadBalancerName             string
	LoadBalancerARN              string
	DNSName                      string
	Scheme                       string
	Subnets                      []string
	IPAddressType                string
	EnableDeletionProtection     bool
	EnableCrossZoneLoadBalancing bool
	Tags                         map[string]string
	DeletionPolicy               string
	State                        string
	VpcID                        string
	CanonicalHostedZoneID        string
	LastSyncTime                 *time.Time
}

// Validate validates the NLB configuration
func (lb *LoadBalancer) Validate() error {
	if lb.LoadBalancerName == "" {
		return ErrInvalidLoadBalancerName
	}

	if lb.Scheme != "" {
		if lb.Scheme != "internet-facing" && lb.Scheme != "internal" {
			return ErrInvalidScheme
		}
	}

	if len(lb.Subnets) < 1 {
		return ErrInvalidSubnets
	}

	return nil
}

// SetDefaults sets default values for the NLB
func (lb *LoadBalancer) SetDefaults() {
	if lb.Scheme == "" {
		lb.Scheme = "internet-facing"
	}

	if lb.IPAddressType == "" {
		lb.IPAddressType = "ipv4"
	}

	if lb.DeletionPolicy == "" {
		lb.DeletionPolicy = "Delete"
	}

	if lb.Tags == nil {
		lb.Tags = make(map[string]string)
	}
}

// ShouldDelete returns true if the NLB should be deleted when the CR is deleted
func (lb *LoadBalancer) ShouldDelete() bool {
	return lb.DeletionPolicy == "Delete"
}

// IsActive returns true if the NLB is active
func (lb *LoadBalancer) IsActive() bool {
	return lb.State == "active"
}

// IsProvisioning returns true if the NLB is being provisioned
func (lb *LoadBalancer) IsProvisioning() bool {
	return lb.State == "provisioning"
}

// IsFailed returns true if the NLB failed to provision
func (lb *LoadBalancer) IsFailed() bool {
	return lb.State == "failed"
}

// IsInternetFacing returns true if the NLB is internet-facing
func (lb *LoadBalancer) IsInternetFacing() bool {
	return lb.Scheme == "internet-facing"
}

// IsInternal returns true if the NLB is internal
func (lb *LoadBalancer) IsInternal() bool {
	return lb.Scheme == "internal"
}
