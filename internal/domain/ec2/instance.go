package ec2

import (
	"errors"
	"time"
)

var (
	ErrInvalidInstanceName = errors.New("instance name is required")
	ErrInvalidInstanceType = errors.New("instance type is required")
	ErrInvalidImageID      = errors.New("image ID is required")
)

type Instance struct {
	InstanceID            string
	InstanceName          string
	InstanceType          string
	ImageID               string
	KeyName               string
	SubnetID              string
	SecurityGroupIDs      []string
	IAMInstanceProfile    string
	UserData              string
	BlockDeviceMappings   []BlockDeviceMapping
	Monitoring            bool
	DisableAPITermination bool
	EBSOptimized          bool
	Tags                  map[string]string
	DeletionPolicy        string
	InstanceState         string
	PrivateIP             string
	PublicIP              string
	PrivateDNS            string
	PublicDNS             string
	AvailabilityZone      string
	LaunchTime            *time.Time
	LastSyncTime          *time.Time
}

type BlockDeviceMapping struct {
	DeviceName string
	EBS        *EBSBlockDevice
}

type EBSBlockDevice struct {
	VolumeSize          int32
	VolumeType          string
	IOPS                int32
	DeleteOnTermination bool
	Encrypted           bool
	KMSKeyID            string
}

func (i *Instance) SetDefaults() {
	if i.DeletionPolicy == "" {
		i.DeletionPolicy = "Delete"
	}
	if i.Tags == nil {
		i.Tags = make(map[string]string)
	}
	// Set default EBS volume type if block devices exist
	for idx := range i.BlockDeviceMappings {
		if i.BlockDeviceMappings[idx].EBS != nil {
			if i.BlockDeviceMappings[idx].EBS.VolumeType == "" {
				i.BlockDeviceMappings[idx].EBS.VolumeType = "gp3"
			}
		}
	}
}

func (i *Instance) Validate() error {
	if i.InstanceName == "" {
		return ErrInvalidInstanceName
	}
	if i.InstanceType == "" {
		return ErrInvalidInstanceType
	}
	if i.ImageID == "" {
		return ErrInvalidImageID
	}
	return nil
}

func (i *Instance) ShouldDelete() bool {
	return i.DeletionPolicy == "Delete"
}

func (i *Instance) ShouldStop() bool {
	return i.DeletionPolicy == "Stop"
}

func (i *Instance) IsRunning() bool {
	return i.InstanceState == "running"
}

func (i *Instance) IsStopped() bool {
	return i.InstanceState == "stopped"
}

func (i *Instance) IsTerminated() bool {
	return i.InstanceState == "terminated"
}
