package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AWSProviderSpec defines the desired state of AWSProvider
type AWSProviderSpec struct {
	// AWS Region
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// Endpoint overrides the default AWS endpoint (useful for LocalStack)
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// RoleARN for AWS authentication via IRSA (IAM Roles for Service Accounts)
	// +optional
	RoleARN string `json:"roleARN,omitempty"`

	// CredentialsSecret reference to a Secret containing AWS credentials
	// +optional
	CredentialsSecret *CredentialsSecretRef `json:"credentialsSecret,omitempty"`

	// AccessKeyID reference to a Secret containing AWS credentials (deprecated, use CredentialsSecret)
	// +optional
	AccessKeyIDRef *SecretKeySelector `json:"accessKeyIDRef,omitempty"`

	// SecretAccessKey reference to a Secret containing AWS credentials (deprecated, use CredentialsSecret)
	// +optional
	SecretAccessKeyRef *SecretKeySelector `json:"secretAccessKeyRef,omitempty"`

	// AssumeRoleARN for cross-account access
	// +optional
	AssumeRoleARN string `json:"assumeRoleARN,omitempty"`

	// ExternalID for assume role
	// +optional
	ExternalID string `json:"externalID,omitempty"`

	// Tags to apply to all resources created by this provider
	// +optional
	DefaultTags map[string]string `json:"defaultTags,omitempty"`
}

// CredentialsSecretRef references a Secret containing AWS credentials
type CredentialsSecretRef struct {
	// Name of the Secret
	Name string `json:"name"`

	// Namespace of the Secret
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	// Name of the Secret
	Name string `json:"name"`

	// Namespace of the Secret
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key within the Secret
	Key string `json:"key"`
}

// AWSProviderStatus defines the observed state of AWSProvider
type AWSProviderStatus struct {
	// Conditions represent the latest available observations of the provider's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready indicates if the provider is ready to provision resources
	// +optional
	Ready bool `json:"ready,omitempty"`

	// LastAuthenticationTime is the timestamp of the last successful authentication
	// +optional
	LastAuthenticationTime *metav1.Time `json:"lastAuthenticationTime,omitempty"`

	// AccountID is the AWS account ID being used
	// +optional
	AccountID string `json:"accountID,omitempty"`

	// CallerIdentity contains information about the AWS caller identity
	// +optional
	CallerIdentity string `json:"callerIdentity,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=awsp
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Account",type=string,JSONPath=`.status.accountID`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AWSProvider is the Schema for the awsproviders API
type AWSProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSProviderSpec   `json:"spec,omitempty"`
	Status AWSProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AWSProviderList contains a list of AWSProvider
type AWSProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWSProvider{}, &AWSProviderList{})
}
