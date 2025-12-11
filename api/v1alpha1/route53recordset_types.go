// Package v1alpha1 define o CRD route53recordset para gerenciamento de route53recordset da AWS.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AliasTarget represents an alias target for Route53
type AliasTarget struct {
	// HostedZoneID is the hosted zone ID of the target
	// +kubebuilder:validation:Required
	HostedZoneID string `json:"hostedZoneID"`

	// DNSName is the DNS name of the target
	// +kubebuilder:validation:Required
	DNSName string `json:"dnsName"`

	// EvaluateTargetHealth determines if Route53 health checks the target
	// +kubebuilder:default=false
	// +optional
	EvaluateTargetHealth bool `json:"evaluateTargetHealth,omitempty"`
}

// GeoLocation represents geographic location for routing policy
type GeoLocation struct {
	// ContinentCode is the two-letter continent code
	// +optional
	ContinentCode string `json:"continentCode,omitempty"`

	// CountryCode is the two-letter country code
	// +optional
	CountryCode string `json:"countryCode,omitempty"`

	// SubdivisionCode is the subdivision code (state/province)
	// +optional
	SubdivisionCode string `json:"subdivisionCode,omitempty"`
}

// Route53RecordSetSpec defines the desired state of Route53RecordSet
type Route53RecordSetSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// HostedZoneID is the ID of the hosted zone containing the record
	// +kubebuilder:validation:Required
	HostedZoneID string `json:"hostedZoneID"`

	// Name is the DNS name of the record (e.g., www.example.com)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the DNS record type (A, AAAA, CNAME, MX, TXT, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=A;AAAA;CNAME;MX;TXT;PTR;SRV;SPF;NS;SOA;CAA;NAPTR
	Type string `json:"type"`

	// TTL is the time-to-live in seconds
	// Required for non-alias records
	// +optional
	TTL *int64 `json:"ttl,omitempty"`

	// ResourceRecords are the resource record values
	// Required for non-alias records
	// +optional
	ResourceRecords []string `json:"resourceRecords,omitempty"`

	// AliasTarget defines alias target for alias records
	// Mutually exclusive with ResourceRecords and TTL
	// +optional
	AliasTarget *AliasTarget `json:"aliasTarget,omitempty"`

	// SetIdentifier is the unique identifier for weighted, latency, geolocation, or failover routing
	// +optional
	SetIdentifier string `json:"setIdentifier,omitempty"`

	// Weight is the weight for weighted routing policy (0-255)
	// +optional
	Weight *int64 `json:"weight,omitempty"`

	// Region is the AWS region for latency-based routing
	// +optional
	Region string `json:"region,omitempty"`

	// GeoLocation is the geographic location for geolocation routing
	// +optional
	GeoLocation *GeoLocation `json:"geoLocation,omitempty"`

	// Failover is the failover type (PRIMARY or SECONDARY)
	// +kubebuilder:validation:Enum=PRIMARY;SECONDARY
	// +optional
	Failover string `json:"failover,omitempty"`

	// MultiValueAnswer enables multivalue answer routing
	// +optional
	MultiValueAnswer bool `json:"multiValueAnswer,omitempty"`

	// HealthCheckID is the health check to associate with the record
	// +optional
	HealthCheckID string `json:"healthCheckID,omitempty"`

	// DeletionPolicy determines what happens to the record when the CR is deleted
	// Valid values: Delete, Retain
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// Route53RecordSetStatus defines the observed state of Route53RecordSet
type Route53RecordSetStatus struct {
	// Ready indicates if the record set is created
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ChangeID is the ID of the last change batch
	// +optional
	ChangeID string `json:"changeID,omitempty"`

	// ChangeStatus is the status of the last change (PENDING or INSYNC)
	// +optional
	ChangeStatus string `json:"changeStatus,omitempty"`

	// LastSyncTime is the last time the record set was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=r53rs;r53record
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="TTL",type=integer,JSONPath=`.spec.ttl`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Route53RecordSet is the Schema for the route53recordsets API
type Route53RecordSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Route53RecordSetSpec   `json:"spec,omitempty"`
	Status Route53RecordSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Route53RecordSetList contains a list of Route53RecordSet
type Route53RecordSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Route53RecordSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Route53RecordSet{}, &Route53RecordSetList{})
}
