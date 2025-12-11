package subnet

import (
	"errors"
	"time"
)

var (
	ErrInvalidVpcID     = errors.New("VPC ID is required")
	ErrInvalidCidrBlock = errors.New("CIDR block is required and must be valid")
)

type Subnet struct {
	SubnetID            string
	VpcID               string
	CidrBlock           string
	AvailabilityZone    string
	MapPublicIpOnLaunch bool
	Tags                map[string]string
	DeletionPolicy      string

	// Status fields
	State                   string
	AvailableIpAddressCount int32
	LastSyncTime            *time.Time
}

func (s *Subnet) SetDefaults() {
	if s.DeletionPolicy == "" {
		s.DeletionPolicy = "Delete"
	}
	if s.Tags == nil {
		s.Tags = make(map[string]string)
	}
}

func (s *Subnet) Validate() error {
	if s.VpcID == "" {
		return ErrInvalidVpcID
	}
	if s.CidrBlock == "" {
		return ErrInvalidCidrBlock
	}
	return nil
}

func (s *Subnet) ShouldDelete() bool {
	return s.DeletionPolicy == "Delete"
}

func (s *Subnet) IsAvailable() bool {
	return s.State == "available"
}
