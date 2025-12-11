package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElasticIPSpec defines the desired state of ElasticIP
type ElasticIPSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// Domain for the Elastic IP address
	// Valid values: vpc (default), standard
	// +kubebuilder:default=vpc
	// +kubebuilder:validation:Enum=vpc;standard
	// +optional
	Domain string `json:"domain,omitempty"`

	// NetworkBorderGroup is the unique set of Availability Zones from which AWS advertises IP addresses
	// +optional
	NetworkBorderGroup string `json:"networkBorderGroup,omitempty"`

	// PublicIpv4Pool is the ID of an address pool to allocate the address from
	// +optional
	PublicIpv4Pool string `json:"publicIpv4Pool,omitempty"`

	// CustomerOwnedIpv4Pool is the ID of a customer-owned address pool
	// +optional
	CustomerOwnedIpv4Pool string `json:"customerOwnedIpv4Pool,omitempty"`

	// Tags to apply to the Elastic IP
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the EIP when the CR is deleted
	// Valid values: Delete, Retain
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// ElasticIPStatus defines the observed state of ElasticIP
type ElasticIPStatus struct {
	// Ready indicates if the Elastic IP is allocated
	// +optional
	Ready bool `json:"ready,omitempty"`

	// AllocationID is the ID for the allocation
	// +optional
	AllocationID string `json:"allocationID,omitempty"`

	// PublicIP is the Elastic IP address
	// +optional
	PublicIP string `json:"publicIP,omitempty"`

	// AssociationID if the EIP is associated with an instance or network interface
	// +optional
	AssociationID string `json:"associationID,omitempty"`

	// InstanceID of the instance the address is associated with
	// +optional
	InstanceID string `json:"instanceID,omitempty"`

	// NetworkInterfaceID of the network interface the address is associated with
	// +optional
	NetworkInterfaceID string `json:"networkInterfaceID,omitempty"`

	// PrivateIPAddress associated with the Elastic IP address
	// +optional
	PrivateIPAddress string `json:"privateIPAddress,omitempty"`

	// Domain (vpc or standard)
	// +optional
	Domain string `json:"domain,omitempty"`

	// LastSyncTime is the last time the EIP was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=eip;eips
// +kubebuilder:printcolumn:name="Public IP",type=string,JSONPath=`.status.publicIP`
// +kubebuilder:printcolumn:name="Allocation ID",type=string,JSONPath=`.status.allocationID`
// +kubebuilder:printcolumn:name="Associated",type=string,JSONPath=`.status.instanceID`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ElasticIP is the Schema for the elasticips API
type ElasticIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticIPSpec   `json:"spec,omitempty"`
	Status ElasticIPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticIPList contains a list of ElasticIP
type ElasticIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticIP{}, &ElasticIPList{})
}
