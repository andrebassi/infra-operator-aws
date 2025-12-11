package routetable

import (
	"errors"
	"time"
)

var (
	ErrInvalidVpcID = errors.New("VPC ID is required")
)

type RouteTable struct {
	RouteTableID       string
	VpcID              string
	Routes             []Route
	SubnetAssociations []string
	Tags               map[string]string
	DeletionPolicy     string

	// Status fields
	AssociatedSubnets []string
	LastSyncTime      *time.Time
}

type Route struct {
	DestinationCidrBlock   string
	GatewayID              string
	NatGatewayID           string
	InstanceID             string
	NetworkInterfaceID     string
	VpcPeeringConnectionID string
}

func (rt *RouteTable) SetDefaults() {
	if rt.DeletionPolicy == "" {
		rt.DeletionPolicy = "Delete"
	}
	if rt.Tags == nil {
		rt.Tags = make(map[string]string)
	}
	if rt.Routes == nil {
		rt.Routes = []Route{}
	}
	if rt.SubnetAssociations == nil {
		rt.SubnetAssociations = []string{}
	}
}

func (rt *RouteTable) Validate() error {
	if rt.VpcID == "" {
		return ErrInvalidVpcID
	}
	return nil
}

func (rt *RouteTable) ShouldDelete() bool {
	return rt.DeletionPolicy == "Delete"
}
