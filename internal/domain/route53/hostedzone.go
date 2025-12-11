// Package route53 contém o modelo de domínio e regras de negócio.
//
// Define as entidades e lógica de negócio independentes de frameworks externos,
// seguindo os princípios de Clean Architecture.
package route53


import (
	"errors"
	"time"
)

var (
	ErrInvalidDomainName         = errors.New("domain name cannot be empty")
	ErrPrivateZoneRequiresVPC    = errors.New("private zone requires VPC ID and region")
	ErrPublicZoneCannotHaveVPC   = errors.New("public zone cannot have VPC configuration")
)

// HostedZone represents a Route53 hosted zone
type HostedZone struct {
	HostedZoneID           string
	Name                   string
	Comment                string
	PrivateZone            bool
	VPCId                  string
	VPCRegion              string
	Tags                   map[string]string
	DeletionPolicy         string
	NameServers            []string
	ResourceRecordSetCount int64
	LastSyncTime           *time.Time
}

// Validate validates the hosted zone configuration
func (hz *HostedZone) Validate() error {
	if hz.Name == "" {
		return ErrInvalidDomainName
	}

	// Private zone validation
	if hz.PrivateZone {
		if hz.VPCId == "" || hz.VPCRegion == "" {
			return ErrPrivateZoneRequiresVPC
		}
	} else {
		// Public zone cannot have VPC configuration
		if hz.VPCId != "" || hz.VPCRegion != "" {
			return ErrPublicZoneCannotHaveVPC
		}
	}

	return nil
}

// SetDefaults sets default values for the hosted zone
func (hz *HostedZone) SetDefaults() {
	if hz.DeletionPolicy == "" {
		hz.DeletionPolicy = "Delete"
	}

	if hz.Tags == nil {
		hz.Tags = make(map[string]string)
	}

	if hz.Comment == "" {
		hz.Comment = "Managed by infra-operator"
	}
}

// ShouldDelete returns true if the hosted zone should be deleted when the CR is deleted
func (hz *HostedZone) ShouldDelete() bool {
	return hz.DeletionPolicy == "Delete"
}

// IsCreated returns true if the hosted zone is created
func (hz *HostedZone) IsCreated() bool {
	return hz.HostedZoneID != ""
}

// IsPrivate returns true if this is a private hosted zone
func (hz *HostedZone) IsPrivate() bool {
	return hz.PrivateZone
}

// IsPublic returns true if this is a public hosted zone
func (hz *HostedZone) IsPublic() bool {
	return !hz.PrivateZone
}
