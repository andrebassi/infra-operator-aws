package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InternetGatewaySpec defines the desired state of InternetGateway
type InternetGatewaySpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// VpcID is the ID of the VPC to attach to
	// +kubebuilder:validation:Required
	VpcID string `json:"vpcID"`

	// Tags to apply to the internet gateway
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// InternetGatewayStatus defines the observed state of InternetGateway
type InternetGatewayStatus struct {
	// Ready indicates if the internet gateway is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// InternetGatewayID is the ID of the internet gateway
	// +optional
	InternetGatewayID string `json:"internetGatewayID,omitempty"`

	// VpcID is the ID of the attached VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// State is the attachment state
	// +optional
	State string `json:"state,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=igw
// +kubebuilder:printcolumn:name="IGW-ID",type=string,JSONPath=`.status.internetGatewayID`
// +kubebuilder:printcolumn:name="VPC-ID",type=string,JSONPath=`.status.vpcID`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// InternetGateway is the Schema for the internetgateways API
type InternetGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternetGatewaySpec   `json:"spec,omitempty"`
	Status InternetGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InternetGatewayList contains a list of InternetGateway
type InternetGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternetGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternetGateway{}, &InternetGatewayList{})
}
