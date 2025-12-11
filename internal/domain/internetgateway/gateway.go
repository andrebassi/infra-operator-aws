package internetgateway

import (
	"errors"
	"time"
)

var (
	ErrInvalidVpcID = errors.New("VPC ID is required")
)

type Gateway struct {
	InternetGatewayID string
	VpcID             string
	Tags              map[string]string
	DeletionPolicy    string

	// Status fields
	State        string
	LastSyncTime *time.Time
}

func (g *Gateway) SetDefaults() {
	if g.DeletionPolicy == "" {
		g.DeletionPolicy = "Delete"
	}
	if g.Tags == nil {
		g.Tags = make(map[string]string)
	}
}

func (g *Gateway) Validate() error {
	if g.VpcID == "" {
		return ErrInvalidVpcID
	}
	return nil
}

func (g *Gateway) ShouldDelete() bool {
	return g.DeletionPolicy == "Delete"
}

func (g *Gateway) IsAttached() bool {
	return g.State == "available"
}
