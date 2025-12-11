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
	ErrInvalidRecordName          = errors.New("record name cannot be empty")
	ErrInvalidRecordType          = errors.New("record type cannot be empty")
	ErrInvalidHostedZoneID        = errors.New("hosted zone ID cannot be empty")
	ErrMissingTTL                 = errors.New("TTL is required for non-alias records")
	ErrMissingResourceRecords     = errors.New("resource records are required for non-alias records")
	ErrMissingAliasTarget         = errors.New("alias target is required for alias records")
	ErrConflictingRecordConfig    = errors.New("cannot specify both alias target and resource records")
	ErrInvalidWeight              = errors.New("weight must be between 0 and 255")
	ErrInvalidAliasTarget         = errors.New("alias target must have hosted zone ID and DNS name")
)

// AliasTarget represents an alias target for Route53
type AliasTarget struct {
	HostedZoneID         string
	DNSName              string
	EvaluateTargetHealth bool
}

// GeoLocation represents geographic location for routing policy
type GeoLocation struct {
	ContinentCode   string
	CountryCode     string
	SubdivisionCode string
}

// RecordSet represents a Route53 resource record set
type RecordSet struct {
	HostedZoneID     string
	Name             string
	Type             string
	TTL              *int64
	ResourceRecords  []string
	AliasTarget      *AliasTarget
	SetIdentifier    string
	Weight           *int64
	Region           string
	GeoLocation      *GeoLocation
	Failover         string
	MultiValueAnswer bool
	HealthCheckID    string
	DeletionPolicy   string
	ChangeID         string
	ChangeStatus     string
	LastSyncTime     *time.Time
}

// Validate validates the record set configuration
func (rs *RecordSet) Validate() error {
	if rs.HostedZoneID == "" {
		return ErrInvalidHostedZoneID
	}

	if rs.Name == "" {
		return ErrInvalidRecordName
	}

	if rs.Type == "" {
		return ErrInvalidRecordType
	}

	// Alias vs Resource Records validation
	if rs.AliasTarget != nil {
		// Alias record
		if len(rs.ResourceRecords) > 0 || rs.TTL != nil {
			return ErrConflictingRecordConfig
		}
		if rs.AliasTarget.HostedZoneID == "" || rs.AliasTarget.DNSName == "" {
			return ErrInvalidAliasTarget
		}
	} else {
		// Regular record
		if rs.TTL == nil {
			return ErrMissingTTL
		}
		if len(rs.ResourceRecords) == 0 {
			return ErrMissingResourceRecords
		}
	}

	// Weight validation
	if rs.Weight != nil {
		if *rs.Weight < 0 || *rs.Weight > 255 {
			return ErrInvalidWeight
		}
	}

	return nil
}

// SetDefaults sets default values for the record set
func (rs *RecordSet) SetDefaults() {
	if rs.DeletionPolicy == "" {
		rs.DeletionPolicy = "Delete"
	}
}

// ShouldDelete returns true if the record set should be deleted when the CR is deleted
func (rs *RecordSet) ShouldDelete() bool {
	return rs.DeletionPolicy == "Delete"
}

// IsAlias returns true if this is an alias record
func (rs *RecordSet) IsAlias() bool {
	return rs.AliasTarget != nil
}

// IsWeighted returns true if this record uses weighted routing
func (rs *RecordSet) IsWeighted() bool {
	return rs.Weight != nil && rs.SetIdentifier != ""
}

// IsLatencyBased returns true if this record uses latency-based routing
func (rs *RecordSet) IsLatencyBased() bool {
	return rs.Region != "" && rs.SetIdentifier != ""
}

// IsGeolocation returns true if this record uses geolocation routing
func (rs *RecordSet) IsGeolocation() bool {
	return rs.GeoLocation != nil && rs.SetIdentifier != ""
}

// IsFailover returns true if this record uses failover routing
func (rs *RecordSet) IsFailover() bool {
	return rs.Failover != "" && rs.SetIdentifier != ""
}

// IsMultiValue returns true if this record uses multivalue answer routing
func (rs *RecordSet) IsMultiValue() bool {
	return rs.MultiValueAnswer
}

// HasHealthCheck returns true if this record has a health check
func (rs *RecordSet) HasHealthCheck() bool {
	return rs.HealthCheckID != ""
}
