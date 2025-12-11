package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VPCSpec defines the desired state of VPC
type VPCSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// CidrBlock is the IPv4 CIDR block for the VPC
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	CidrBlock string `json:"cidrBlock"`

	// EnableDnsSupport indicates whether DNS resolution is supported
	// +optional
	// +kubebuilder:default=true
	EnableDnsSupport bool `json:"enableDnsSupport,omitempty"`

	// EnableDnsHostnames indicates whether instances launched in the VPC get DNS hostnames
	// +optional
	// +kubebuilder:default=true
	EnableDnsHostnames bool `json:"enableDnsHostnames,omitempty"`

	// InstanceTenancy is the tenancy option for instances launched into the VPC
	// +optional
	// +kubebuilder:validation:Enum=default;dedicated;host
	// +kubebuilder:default=default
	InstanceTenancy string `json:"instanceTenancy,omitempty"`

	// Tags to apply to the VPC
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// VPCStatus defines the observed state of VPC
type VPCStatus struct {
	// Ready indicates if the VPC is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// VpcID is the ID of the VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// State is the current state of the VPC
	// +optional
	State string `json:"state,omitempty"`

	// CidrBlock is the primary IPv4 CIDR block
	// +optional
	CidrBlock string `json:"cidrBlock,omitempty"`

	// IsDefault indicates if this is the default VPC
	// +optional
	IsDefault bool `json:"isDefault,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=vpc
// +kubebuilder:printcolumn:name="VPC-ID",type=string,JSONPath=`.status.vpcID`
// +kubebuilder:printcolumn:name="CIDR",type=string,JSONPath=`.status.cidrBlock`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// VPC is the Schema for the vpcs API
type VPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCSpec   `json:"spec,omitempty"`
	Status VPCStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VPCList contains a list of VPC
type VPCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VPC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VPC{}, &VPCList{})
}
