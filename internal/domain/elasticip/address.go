package elasticip

import (
	"errors"
	"time"
)

var (
	ErrInvalidDomain = errors.New("domain must be 'vpc' or 'standard'")
)

// Address represents an Elastic IP address
type Address struct {
	AllocationID          string
	PublicIP              string
	AssociationID         string
	InstanceID            string
	NetworkInterfaceID    string
	PrivateIPAddress      string
	Domain                string
	NetworkBorderGroup    string
	PublicIpv4Pool        string
	CustomerOwnedIpv4Pool string
	Tags                  map[string]string
	DeletionPolicy        string
	LastSyncTime          *time.Time
}

// Validate validates the Elastic IP configuration
func (a *Address) Validate() error {
	if a.Domain != "" {
		if a.Domain != "vpc" && a.Domain != "standard" {
			return ErrInvalidDomain
		}
	}
	return nil
}

// SetDefaults sets default values for the Elastic IP
func (a *Address) SetDefaults() {
	if a.Domain == "" {
		a.Domain = "vpc"
	}

	if a.DeletionPolicy == "" {
		a.DeletionPolicy = "Delete"
	}

	if a.Tags == nil {
		a.Tags = make(map[string]string)
	}
}

// ShouldDelete returns true if the EIP should be deleted when the CR is deleted
func (a *Address) ShouldDelete() bool {
	return a.DeletionPolicy == "Delete"
}

// IsAllocated returns true if the EIP is allocated
func (a *Address) IsAllocated() bool {
	return a.AllocationID != ""
}

// IsAssociated returns true if the EIP is associated with an instance or network interface
func (a *Address) IsAssociated() bool {
	return a.AssociationID != ""
}

// IsVPC returns true if the EIP is in a VPC
func (a *Address) IsVPC() bool {
	return a.Domain == "vpc"
}

// IsStandard returns true if the EIP is EC2-Classic
func (a *Address) IsStandard() bool {
	return a.Domain == "standard"
}
