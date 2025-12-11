package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NATGatewaySpec defines the desired state of NATGateway
type NATGatewaySpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// SubnetID is the ID of the subnet for the NAT gateway
	// +kubebuilder:validation:Required
	SubnetID string `json:"subnetID"`

	// AllocationID is the Elastic IP allocation ID (optional, will be created if not provided)
	// +optional
	AllocationID string `json:"allocationID,omitempty"`

	// ConnectivityType specifies if the NAT gateway is public or private
	// +optional
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:default=public
	ConnectivityType string `json:"connectivityType,omitempty"`

	// Tags to apply to the NAT gateway
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// NATGatewayStatus defines the observed state of NATGateway
type NATGatewayStatus struct {
	// Ready indicates if the NAT gateway is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// NatGatewayID is the ID of the NAT gateway
	// +optional
	NatGatewayID string `json:"natGatewayID,omitempty"`

	// State is the current state of the NAT gateway
	// +optional
	State string `json:"state,omitempty"`

	// SubnetID is the ID of the subnet
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// VpcID is the ID of the VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// PublicIP is the Elastic IP address
	// +optional
	PublicIP string `json:"publicIP,omitempty"`

	// PrivateIP is the private IP address
	// +optional
	PrivateIP string `json:"privateIP,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=natgw
// +kubebuilder:printcolumn:name="NAT-ID",type=string,JSONPath=`.status.natGatewayID`
// +kubebuilder:printcolumn:name="Subnet-ID",type=string,JSONPath=`.status.subnetID`
// +kubebuilder:printcolumn:name="PublicIP",type=string,JSONPath=`.status.publicIP`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NATGateway is the Schema for the natgateways API
type NATGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NATGatewaySpec   `json:"spec,omitempty"`
	Status NATGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NATGatewayList contains a list of NATGateway
type NATGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NATGateway{}, &NATGatewayList{})
}
