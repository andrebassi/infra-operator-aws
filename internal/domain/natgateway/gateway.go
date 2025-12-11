package natgateway

import (
	"errors"
	"time"
)

var (
	ErrInvalidSubnetID = errors.New("subnet ID is required")
)

type Gateway struct {
	NatGatewayID     string
	SubnetID         string
	AllocationID     string
	ConnectivityType string
	Tags             map[string]string
	DeletionPolicy   string

	// Status fields
	State        string
	VpcID        string
	PublicIP     string
	PrivateIP    string
	LastSyncTime *time.Time
}

func (g *Gateway) SetDefaults() {
	if g.DeletionPolicy == "" {
		g.DeletionPolicy = "Delete"
	}
	if g.ConnectivityType == "" {
		g.ConnectivityType = "public"
	}
	if g.Tags == nil {
		g.Tags = make(map[string]string)
	}
}

func (g *Gateway) Validate() error {
	if g.SubnetID == "" {
		return ErrInvalidSubnetID
	}
	return nil
}

func (g *Gateway) ShouldDelete() bool {
	return g.DeletionPolicy == "Delete"
}

func (g *Gateway) IsAvailable() bool {
	return g.State == "available"
}
