// Package v1alpha1 define o CRD route53hostedzone para gerenciamento de route53hostedzone da AWS.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Route53HostedZoneSpec defines the desired state of Route53HostedZone
type Route53HostedZoneSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// Name is the domain name of the hosted zone (e.g., example.com)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Comment is a comment about the hosted zone
	// +optional
	Comment string `json:"comment,omitempty"`

	// PrivateZone indicates if this is a private hosted zone
	// +kubebuilder:default=false
	// +optional
	PrivateZone bool `json:"privateZone,omitempty"`

	// VPCId is the ID of the VPC to associate with a private hosted zone
	// Required if PrivateZone is true
	// +optional
	VPCId string `json:"vpcId,omitempty"`

	// VPCRegion is the region of the VPC to associate with a private hosted zone
	// Required if PrivateZone is true
	// +optional
	VPCRegion string `json:"vpcRegion,omitempty"`

	// Tags to apply to the hosted zone
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the hosted zone when the CR is deleted
	// Valid values: Delete, Retain
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// Route53HostedZoneStatus defines the observed state of Route53HostedZone
type Route53HostedZoneStatus struct {
	// Ready indicates if the hosted zone is created
	// +optional
	Ready bool `json:"ready,omitempty"`

	// HostedZoneID is the Route53 hosted zone ID
	// +optional
	HostedZoneID string `json:"hostedZoneID,omitempty"`

	// NameServers are the authoritative name servers for the hosted zone
	// +optional
	NameServers []string `json:"nameServers,omitempty"`

	// ResourceRecordSetCount is the number of resource record sets in the hosted zone
	// +optional
	ResourceRecordSetCount int64 `json:"resourceRecordSetCount,omitempty"`

	// LastSyncTime is the last time the hosted zone was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=r53hz;r53hostedzone
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Hosted Zone ID",type=string,JSONPath=`.status.hostedZoneID`
// +kubebuilder:printcolumn:name="Private",type=boolean,JSONPath=`.spec.privateZone`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Route53HostedZone is the Schema for the route53hostedzones API
type Route53HostedZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Route53HostedZoneSpec   `json:"spec,omitempty"`
	Status Route53HostedZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Route53HostedZoneList contains a list of Route53HostedZone
type Route53HostedZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Route53HostedZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Route53HostedZone{}, &Route53HostedZoneList{})
}
