package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EKSClusterSpec defines the desired state of EKSCluster
type EKSClusterSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// ClusterName is the name of the EKS cluster
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	ClusterName string `json:"clusterName"`

	// Version is the Kubernetes version
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^1\.(2[4-9]|[3-9][0-9])$`
	Version string `json:"version"`

	// RoleARN is the IAM role ARN for the EKS cluster
	// +kubebuilder:validation:Required
	RoleARN string `json:"roleARN"`

	// VpcConfig defines the VPC configuration
	// +kubebuilder:validation:Required
	VpcConfig EKSVpcConfig `json:"vpcConfig"`

	// Logging configuration
	// +optional
	Logging *EKSLogging `json:"logging,omitempty"`

	// Encryption configuration
	// +optional
	Encryption *EKSEncryption `json:"encryption,omitempty"`

	// Tags to apply to the cluster
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// EKSVpcConfig defines VPC configuration for EKS
type EKSVpcConfig struct {
	// SubnetIDs are the subnet IDs for the cluster
	// +kubebuilder:validation:MinItems=2
	SubnetIDs []string `json:"subnetIDs"`

	// SecurityGroupIDs are additional security groups
	// +optional
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`

	// EndpointPublicAccess enables public API endpoint
	// +optional
	// +kubebuilder:default=true
	EndpointPublicAccess bool `json:"endpointPublicAccess,omitempty"`

	// EndpointPrivateAccess enables private API endpoint
	// +optional
	// +kubebuilder:default=false
	EndpointPrivateAccess bool `json:"endpointPrivateAccess,omitempty"`

	// PublicAccessCidrs are CIDR blocks allowed for public access
	// +optional
	PublicAccessCidrs []string `json:"publicAccessCidrs,omitempty"`
}

// EKSLogging defines logging configuration
type EKSLogging struct {
	// ClusterLogging enables specific log types
	// +optional
	ClusterLogging []EKSLogType `json:"clusterLogging,omitempty"`
}

// EKSLogType represents EKS log types
// +kubebuilder:validation:Enum=api;audit;authenticator;controllerManager;scheduler
type EKSLogType string

const (
	EKSLogTypeAPI               EKSLogType = "api"
	EKSLogTypeAudit             EKSLogType = "audit"
	EKSLogTypeAuthenticator     EKSLogType = "authenticator"
	EKSLogTypeControllerManager EKSLogType = "controllerManager"
	EKSLogTypeScheduler         EKSLogType = "scheduler"
)

// EKSEncryption defines encryption configuration
type EKSEncryption struct {
	// Resources to encrypt
	// +optional
	Resources []string `json:"resources,omitempty"`

	// Provider KMS key ARN
	// +optional
	ProviderKeyARN string `json:"providerKeyARN,omitempty"`
}

// EKSClusterStatus defines the observed state of EKSCluster
type EKSClusterStatus struct {
	// Ready indicates if the cluster is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ClusterName is the name of the cluster
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// ARN of the cluster
	// +optional
	ARN string `json:"arn,omitempty"`

	// Endpoint is the API server endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Status of the cluster
	// +optional
	Status string `json:"status,omitempty"`

	// Version is the current Kubernetes version
	// +optional
	Version string `json:"version,omitempty"`

	// PlatformVersion is the EKS platform version
	// +optional
	PlatformVersion string `json:"platformVersion,omitempty"`

	// CertificateAuthority contains the CA data
	// +optional
	CertificateAuthority string `json:"certificateAuthority,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=eks
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.status.clusterName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// EKSCluster is the Schema for the eksclusters API
type EKSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EKSClusterSpec   `json:"spec,omitempty"`
	Status EKSClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EKSClusterList contains a list of EKSCluster
type EKSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EKSCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EKSCluster{}, &EKSClusterList{})
}
