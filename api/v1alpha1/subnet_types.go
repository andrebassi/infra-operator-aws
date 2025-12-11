package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubnetSpec defines the desired state of Subnet
type SubnetSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// VpcID is the ID of the VPC
	// +kubebuilder:validation:Required
	VpcID string `json:"vpcID"`

	// CidrBlock is the IPv4 CIDR block for the subnet
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	CidrBlock string `json:"cidrBlock"`

	// AvailabilityZone for the subnet
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// MapPublicIpOnLaunch indicates whether instances launched in this subnet receive a public IP
	// +optional
	MapPublicIpOnLaunch bool `json:"mapPublicIpOnLaunch,omitempty"`

	// Tags to apply to the subnet
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// SubnetStatus defines the observed state of Subnet
type SubnetStatus struct {
	// Ready indicates if the subnet is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// SubnetID is the ID of the subnet
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// State is the current state of the subnet
	// +optional
	State string `json:"state,omitempty"`

	// VpcID is the ID of the VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// CidrBlock is the IPv4 CIDR block
	// +optional
	CidrBlock string `json:"cidrBlock,omitempty"`

	// AvailabilityZone
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// AvailableIpAddressCount
	// +optional
	AvailableIpAddressCount int32 `json:"availableIpAddressCount,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=subnet
// +kubebuilder:printcolumn:name="Subnet-ID",type=string,JSONPath=`.status.subnetID`
// +kubebuilder:printcolumn:name="VPC-ID",type=string,JSONPath=`.status.vpcID`
// +kubebuilder:printcolumn:name="CIDR",type=string,JSONPath=`.status.cidrBlock`
// +kubebuilder:printcolumn:name="AZ",type=string,JSONPath=`.status.availabilityZone`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Subnet is the Schema for the subnets API
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubnetList contains a list of Subnet
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
