package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ALBSpec defines the desired state of ALB
type ALBSpec struct {
	// ProviderRef references the AWSProvider for credentials
	ProviderRef ProviderReference `json:"providerRef"`

	// LoadBalancerName is the name of the Application Load Balancer
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32
	LoadBalancerName string `json:"loadBalancerName"`

	// Scheme defines if the ALB is internet-facing or internal
	// +kubebuilder:validation:Enum=internet-facing;internal
	// +kubebuilder:default=internet-facing
	Scheme string `json:"scheme,omitempty"`

	// Subnets is the list of subnet IDs where the ALB will be deployed
	// Must be at least 2 subnets in different AZs
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=2
	Subnets []string `json:"subnets"`

	// SecurityGroups is the list of security group IDs
	// +kubebuilder:validation:MinItems=1
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// IPAddressType defines the IP address type (ipv4, dualstack)
	// +kubebuilder:validation:Enum=ipv4;dualstack
	// +kubebuilder:default=ipv4
	IPAddressType string `json:"ipAddressType,omitempty"`

	// EnableDeletionProtection enables deletion protection
	EnableDeletionProtection bool `json:"enableDeletionProtection,omitempty"`

	// EnableHttp2 enables HTTP/2
	// +kubebuilder:default=true
	EnableHttp2 bool `json:"enableHttp2,omitempty"`

	// EnableWafFailOpen enables WAF fail open
	EnableWafFailOpen bool `json:"enableWafFailOpen,omitempty"`

	// IdleTimeout is the idle timeout in seconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4000
	// +kubebuilder:default=60
	IdleTimeout int32 `json:"idleTimeout,omitempty"`

	// Tags are custom tags for the ALB
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens when the CR is deleted
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// ALBStatus defines the observed state of ALB
type ALBStatus struct {
	// Ready indicates if the ALB is ready
	Ready bool `json:"ready,omitempty"`

	// LoadBalancerARN is the ARN of the ALB
	LoadBalancerARN string `json:"loadBalancerARN,omitempty"`

	// DNSName is the DNS name of the ALB
	DNSName string `json:"dnsName,omitempty"`

	// CanonicalHostedZoneID is the Route53 hosted zone ID
	CanonicalHostedZoneID string `json:"canonicalHostedZoneID,omitempty"`

	// State is the current state of the ALB
	State string `json:"state,omitempty"`

	// VpcID is the VPC ID where the ALB is deployed
	VpcID string `json:"vpcID,omitempty"`

	// LastSyncTime is the last time the resource was synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=alb
// +kubebuilder:printcolumn:name="DNS",type=string,JSONPath=`.status.dnsName`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ALB is the Schema for the albs API
type ALB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ALBSpec   `json:"spec,omitempty"`
	Status ALBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ALBList contains a list of ALB
type ALBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ALB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ALB{}, &ALBList{})
}
