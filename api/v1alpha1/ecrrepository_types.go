package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ECRRepositorySpec defines the desired state of ECRRepository
type ECRRepositorySpec struct {
	ProviderRef ProviderReference `json:"providerRef"`

	// RepositoryName is the name of the ECR repository
	RepositoryName string `json:"repositoryName"`

	// ImageTagMutability controls whether image tags can be overwritten
	// Valid values: MUTABLE, IMMUTABLE
	ImageTagMutability string `json:"imageTagMutability,omitempty"`

	// ImageScanningConfiguration enables automatic image scanning
	ScanOnPush bool `json:"scanOnPush,omitempty"`

	// EncryptionConfiguration for images in the repository
	EncryptionConfiguration *EncryptionConfiguration `json:"encryptionConfiguration,omitempty"`

	// LifecyclePolicy defines automatic cleanup of images
	LifecyclePolicy *LifecyclePolicy `json:"lifecyclePolicy,omitempty"`

	// Tags for the ECR repository
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens when CR is deleted
	// Valid values: Delete, Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

type EncryptionConfiguration struct {
	// EncryptionType is AES256 or KMS
	EncryptionType string `json:"encryptionType"`

	// KmsKey is the ARN of the KMS key (required if encryptionType is KMS)
	KmsKey string `json:"kmsKey,omitempty"`
}

type LifecyclePolicy struct {
	// PolicyText is the JSON policy document
	PolicyText string `json:"policyText"`
}

// ECRRepositoryStatus defines the observed state of ECRRepository
type ECRRepositoryStatus struct {
	Ready bool `json:"ready"`

	// RepositoryArn is the ARN of the ECR repository
	RepositoryArn string `json:"repositoryArn,omitempty"`

	// RepositoryUri is the URI to use for pushing/pulling images
	RepositoryUri string `json:"repositoryUri,omitempty"`

	// RegistryId is the AWS account ID that owns the registry
	RegistryId string `json:"registryId,omitempty"`

	// CreatedAt is when the repository was created
	CreatedAt metav1.Time `json:"createdAt,omitempty"`

	// ImageCount is the number of images in the repository
	ImageCount int64 `json:"imageCount,omitempty"`

	// LastSyncTime is when the repository was last synced
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ecr
// +kubebuilder:printcolumn:name="Repository",type=string,JSONPath=`.spec.repositoryName`
// +kubebuilder:printcolumn:name="URI",type=string,JSONPath=`.status.repositoryUri`
// +kubebuilder:printcolumn:name="Images",type=integer,JSONPath=`.status.imageCount`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ECRRepository is the Schema for the ecrrepositories API
type ECRRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ECRRepositorySpec   `json:"spec,omitempty"`
	Status ECRRepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ECRRepositoryList contains a list of ECRRepository
type ECRRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ECRRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ECRRepository{}, &ECRRepositoryList{})
}
