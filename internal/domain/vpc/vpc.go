package vpc

import (
	"errors"
	"time"
)

var (
	ErrInvalidCidrBlock = errors.New("CIDR block is required and must be valid")
)

type VPC struct {
	VpcID              string
	CidrBlock          string
	EnableDnsSupport   bool
	EnableDnsHostnames bool
	InstanceTenancy    string
	Tags               map[string]string
	DeletionPolicy     string

	// Status fields
	State        string
	IsDefault    bool
	LastSyncTime *time.Time
}

func (v *VPC) SetDefaults() {
	if v.DeletionPolicy == "" {
		v.DeletionPolicy = "Delete"
	}
	if v.InstanceTenancy == "" {
		v.InstanceTenancy = "default"
	}
	if v.Tags == nil {
		v.Tags = make(map[string]string)
	}
}

func (v *VPC) Validate() error {
	if v.CidrBlock == "" {
		return ErrInvalidCidrBlock
	}
	return nil
}

func (v *VPC) ShouldDelete() bool {
	return v.DeletionPolicy == "Delete"
}

func (v *VPC) IsAvailable() bool {
	return v.State == "available"
}
