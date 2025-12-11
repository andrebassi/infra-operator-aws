package securitygroup

import (
	"errors"
	"time"
)

var (
	ErrInvalidGroupName   = errors.New("group name is required")
	ErrInvalidVpcID       = errors.New("VPC ID is required")
	ErrInvalidDescription = errors.New("description is required")
)

type SecurityGroup struct {
	GroupID        string
	GroupName      string
	Description    string
	VpcID          string
	IngressRules   []Rule
	EgressRules    []Rule
	Tags           map[string]string
	DeletionPolicy string

	// Status fields
	LastSyncTime *time.Time
}

type Rule struct {
	IpProtocol            string
	FromPort              int32
	ToPort                int32
	CidrBlocks            []string
	Ipv6CidrBlocks        []string
	SourceSecurityGroupID string
	Description           string
}

func (sg *SecurityGroup) SetDefaults() {
	if sg.DeletionPolicy == "" {
		sg.DeletionPolicy = "Delete"
	}
	if sg.Tags == nil {
		sg.Tags = make(map[string]string)
	}
	if sg.IngressRules == nil {
		sg.IngressRules = []Rule{}
	}
	if sg.EgressRules == nil {
		sg.EgressRules = []Rule{}
	}
}

func (sg *SecurityGroup) Validate() error {
	if sg.GroupName == "" {
		return ErrInvalidGroupName
	}
	if sg.VpcID == "" {
		return ErrInvalidVpcID
	}
	if sg.Description == "" {
		return ErrInvalidDescription
	}
	return nil
}

func (sg *SecurityGroup) ShouldDelete() bool {
	return sg.DeletionPolicy == "Delete"
}
