package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NLBSpec defines the desired state of NLB
type NLBSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// LoadBalancerName is the name of the Network Load Balancer
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32
	LoadBalancerName string `json:"loadBalancerName"`

	// Scheme is the load balancer scheme
	// Valid values: internet-facing, internal
	// +kubebuilder:default=internet-facing
	// +kubebuilder:validation:Enum=internet-facing;internal
	// +optional
	Scheme string `json:"scheme,omitempty"`

	// Subnets is the list of subnet IDs (minimum 1)
	// +kubebuilder:validation:MinItems=1
	Subnets []string `json:"subnets"`

	// IPAddressType is the IP address type
	// Valid values: ipv4, dualstack
	// +kubebuilder:default=ipv4
	// +kubebuilder:validation:Enum=ipv4;dualstack
	// +optional
	IPAddressType string `json:"ipAddressType,omitempty"`

	// EnableDeletionProtection enables deletion protection
	// +optional
	EnableDeletionProtection bool `json:"enableDeletionProtection,omitempty"`

	// EnableCrossZoneLoadBalancing enables cross-zone load balancing
	// +kubebuilder:default=false
	// +optional
	EnableCrossZoneLoadBalancing bool `json:"enableCrossZoneLoadBalancing,omitempty"`

	// Tags to apply to the NLB
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the NLB when the CR is deleted
	// Valid values: Delete, Retain
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// NLBStatus defines the observed state of NLB
type NLBStatus struct {
	// Ready indicates if the NLB is active
	// +optional
	Ready bool `json:"ready,omitempty"`

	// LoadBalancerARN is the ARN of the NLB
	// +optional
	LoadBalancerARN string `json:"loadBalancerARN,omitempty"`

	// DNSName is the DNS name of the NLB
	// +optional
	DNSName string `json:"dnsName,omitempty"`

	// State is the state of the NLB
	// +optional
	State string `json:"state,omitempty"`

	// VpcID is the VPC ID
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// CanonicalHostedZoneID for Route53
	// +optional
	CanonicalHostedZoneID string `json:"canonicalHostedZoneID,omitempty"`

	// LastSyncTime is the last time the NLB was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=nlb;nlbs
// +kubebuilder:printcolumn:name="Load Balancer",type=string,JSONPath=`.spec.loadBalancerName`
// +kubebuilder:printcolumn:name="DNS Name",type=string,JSONPath=`.status.dnsName`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NLB is the Schema for the nlbs API
type NLB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NLBSpec   `json:"spec,omitempty"`
	Status NLBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NLBList contains a list of NLB
type NLBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NLB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NLB{}, &NLBList{})
}
