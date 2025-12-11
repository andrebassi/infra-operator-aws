package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RouteTableSpec defines the desired state of RouteTable
type RouteTableSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// VpcID is the ID of the VPC
	// +kubebuilder:validation:Required
	VpcID string `json:"vpcID"`

	// Routes to add to the route table
	// +optional
	Routes []Route `json:"routes,omitempty"`

	// SubnetAssociations are the subnets to associate with this route table
	// +optional
	SubnetAssociations []string `json:"subnetAssociations,omitempty"`

	// Tags to apply to the route table
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// Route defines a route in the route table
type Route struct {
	// DestinationCidrBlock is the IPv4 CIDR block destination
	// +optional
	DestinationCidrBlock string `json:"destinationCidrBlock,omitempty"`

	// GatewayID is the ID of an internet gateway or virtual private gateway
	// +optional
	GatewayID string `json:"gatewayID,omitempty"`

	// NatGatewayID is the ID of a NAT gateway
	// +optional
	NatGatewayID string `json:"natGatewayID,omitempty"`

	// InstanceID is the ID of a NAT instance
	// +optional
	InstanceID string `json:"instanceID,omitempty"`

	// NetworkInterfaceID is the ID of a network interface
	// +optional
	NetworkInterfaceID string `json:"networkInterfaceID,omitempty"`

	// VpcPeeringConnectionID is the ID of a VPC peering connection
	// +optional
	VpcPeeringConnectionID string `json:"vpcPeeringConnectionID,omitempty"`
}

// RouteTableStatus defines the observed state of RouteTable
type RouteTableStatus struct {
	// Ready indicates if the route table is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// RouteTableID is the ID of the route table
	// +optional
	RouteTableID string `json:"routeTableID,omitempty"`

	// VpcID is the ID of the VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// AssociatedSubnets lists the associated subnet IDs
	// +optional
	AssociatedSubnets []string `json:"associatedSubnets,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=rt
// +kubebuilder:printcolumn:name="RT-ID",type=string,JSONPath=`.status.routeTableID`
// +kubebuilder:printcolumn:name="VPC-ID",type=string,JSONPath=`.status.vpcID`
// +kubebuilder:printcolumn:name="Subnets",type=integer,JSONPath=`.status.associatedSubnets[*]`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RouteTable is the Schema for the routetables API
type RouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouteTableSpec   `json:"spec,omitempty"`
	Status RouteTableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RouteTableList contains a list of RouteTable
type RouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RouteTable{}, &RouteTableList{})
}
