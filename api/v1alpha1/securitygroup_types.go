package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityGroupSpec defines the desired state of SecurityGroup
type SecurityGroupSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// GroupName is the name of the security group
	// +kubebuilder:validation:Required
	GroupName string `json:"groupName"`

	// Description of the security group
	// +kubebuilder:validation:Required
	Description string `json:"description"`

	// VpcID is the ID of the VPC
	// +kubebuilder:validation:Required
	VpcID string `json:"vpcID"`

	// IngressRules are the inbound rules
	// +optional
	IngressRules []SecurityGroupRule `json:"ingressRules,omitempty"`

	// EgressRules are the outbound rules
	// +optional
	EgressRules []SecurityGroupRule `json:"egressRules,omitempty"`

	// Tags to apply to the security group
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// SecurityGroupRule defines a security group rule
type SecurityGroupRule struct {
	// IpProtocol is the IP protocol (tcp, udp, icmp, or -1 for all)
	// +kubebuilder:validation:Required
	IpProtocol string `json:"ipProtocol"`

	// FromPort is the start of port range (-1 for all)
	// +optional
	FromPort int32 `json:"fromPort,omitempty"`

	// ToPort is the end of port range (-1 for all)
	// +optional
	ToPort int32 `json:"toPort,omitempty"`

	// CidrBlocks are the IPv4 CIDR ranges
	// +optional
	CidrBlocks []string `json:"cidrBlocks,omitempty"`

	// Ipv6CidrBlocks are the IPv6 CIDR ranges
	// +optional
	Ipv6CidrBlocks []string `json:"ipv6CidrBlocks,omitempty"`

	// SourceSecurityGroupID is the source security group ID
	// +optional
	SourceSecurityGroupID string `json:"sourceSecurityGroupID,omitempty"`

	// Description of the rule
	// +optional
	Description string `json:"description,omitempty"`
}

// SecurityGroupStatus defines the observed state of SecurityGroup
type SecurityGroupStatus struct {
	// Ready indicates if the security group is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// GroupID is the ID of the security group
	// +optional
	GroupID string `json:"groupID,omitempty"`

	// GroupName is the name of the security group
	// +optional
	GroupName string `json:"groupName,omitempty"`

	// VpcID is the ID of the VPC
	// +optional
	VpcID string `json:"vpcID,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=sg
// +kubebuilder:printcolumn:name="Group-ID",type=string,JSONPath=`.status.groupID`
// +kubebuilder:printcolumn:name="Group-Name",type=string,JSONPath=`.status.groupName`
// +kubebuilder:printcolumn:name="VPC-ID",type=string,JSONPath=`.status.vpcID`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SecurityGroup is the Schema for the securitygroups API
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupSpec   `json:"spec,omitempty"`
	Status SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityGroupList contains a list of SecurityGroup
type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
