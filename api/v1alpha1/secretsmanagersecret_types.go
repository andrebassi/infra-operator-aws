package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretsManagerSecretSpec defines the desired state of SecretsManagerSecret
type SecretsManagerSecretSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// SecretName
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// Description
	// +optional
	Description string `json:"description,omitempty"`

	// SecretString or SecretBinary (one must be specified)
	// Reference to a Kubernetes Secret containing the value
	// +optional
	SecretStringRef *SecretKeySelector `json:"secretStringRef,omitempty"`

	// SecretBinary reference (base64 encoded)
	// +optional
	SecretBinaryRef *SecretKeySelector `json:"secretBinaryRef,omitempty"`

	// KmsKeyId for encryption
	// +optional
	KmsKeyId string `json:"kmsKeyId,omitempty"`

	// RotationEnabled
	// +optional
	RotationEnabled bool `json:"rotationEnabled,omitempty"`

	// RotationLambdaARN (required if RotationEnabled=true)
	// +optional
	RotationLambdaARN string `json:"rotationLambdaARN,omitempty"`

	// AutomaticallyAfterDays (rotation interval)
	// +optional
	// +kubebuilder:validation:Minimum=1
	AutomaticallyAfterDays int32 `json:"automaticallyAfterDays,omitempty"`

	// Tags
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// RecoveryWindowInDays (7-30 days, or 0 for immediate deletion)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=30
	RecoveryWindowInDays int32 `json:"recoveryWindowInDays,omitempty"`
}

// SecretsManagerSecretStatus defines the observed state of SecretsManagerSecret
type SecretsManagerSecretStatus struct {
	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ARN
	// +optional
	ARN string `json:"arn,omitempty"`

	// VersionId of current secret value
	// +optional
	VersionId string `json:"versionId,omitempty"`

	// CreatedDate
	// +optional
	CreatedDate *metav1.Time `json:"createdDate,omitempty"`

	// LastChangedDate
	// +optional
	LastChangedDate *metav1.Time `json:"lastChangedDate,omitempty"`

	// LastRotatedDate
	// +optional
	LastRotatedDate *metav1.Time `json:"lastRotatedDate,omitempty"`

	// NextRotationDate
	// +optional
	NextRotationDate *metav1.Time `json:"nextRotationDate,omitempty"`

	// RotationEnabled
	// +optional
	RotationEnabled bool `json:"rotationEnabled,omitempty"`

	// DeletedDate (if in deletion state)
	// +optional
	DeletedDate *metav1.Time `json:"deletedDate,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=awssecret
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.spec.secretName`
// +kubebuilder:printcolumn:name="Rotation",type=boolean,JSONPath=`.status.rotationEnabled`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SecretsManagerSecret is the Schema for the secretsmanagersecrets API
type SecretsManagerSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretsManagerSecretSpec   `json:"spec,omitempty"`
	Status SecretsManagerSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretsManagerSecretList contains a list of SecretsManagerSecret
type SecretsManagerSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretsManagerSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretsManagerSecret{}, &SecretsManagerSecretList{})
}
