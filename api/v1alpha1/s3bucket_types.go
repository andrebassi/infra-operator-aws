package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3BucketSpec defines the desired state of S3Bucket
type S3BucketSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// BucketName is the name of the S3 bucket
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=63
	BucketName string `json:"bucketName"`

	// ACL is the canned ACL to apply to the bucket
	// +optional
	// +kubebuilder:validation:Enum=private;public-read;public-read-write;authenticated-read
	ACL string `json:"acl,omitempty"`

	// Versioning enables versioning for the bucket
	// +optional
	Versioning *VersioningConfiguration `json:"versioning,omitempty"`

	// Encryption configuration for the bucket
	// +optional
	Encryption *ServerSideEncryptionConfiguration `json:"encryption,omitempty"`

	// LifecycleConfiguration rules
	// +optional
	LifecycleRules []LifecycleRule `json:"lifecycleRules,omitempty"`

	// CORS configuration
	// +optional
	CORSRules []CORSRule `json:"corsRules,omitempty"`

	// Public access block configuration
	// +optional
	PublicAccessBlock *PublicAccessBlockConfiguration `json:"publicAccessBlock,omitempty"`

	// Tags for the bucket
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the bucket when the CR is deleted
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain;Orphan
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// ProviderReference references an AWSProvider resource
type ProviderReference struct {
	// Name of the AWSProvider
	Name string `json:"name"`

	// Namespace of the AWSProvider (if different from current namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// VersioningConfiguration defines versioning settings
type VersioningConfiguration struct {
	// Enabled indicates if versioning is enabled
	Enabled bool `json:"enabled"`
}

// ServerSideEncryptionConfiguration defines encryption settings
type ServerSideEncryptionConfiguration struct {
	// Algorithm to use for encryption (AES256 or aws_kms)
	// +kubebuilder:validation:Enum=AES256;aws_kms
	Algorithm string `json:"algorithm"`

	// KMSKeyID for KMS encryption
	// +optional
	KMSKeyID string `json:"kmsKeyID,omitempty"`
}

// LifecycleRule defines a lifecycle rule
type LifecycleRule struct {
	// ID of the rule
	ID string `json:"id"`

	// Enabled indicates if the rule is enabled
	Enabled bool `json:"enabled"`

	// Prefix filter
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Expiration configuration
	// +optional
	Expiration *LifecycleExpiration `json:"expiration,omitempty"`

	// Transition rules
	// +optional
	Transitions []LifecycleTransition `json:"transitions,omitempty"`
}

// LifecycleExpiration defines expiration settings
type LifecycleExpiration struct {
	// Days after which objects expire
	Days int32 `json:"days"`
}

// LifecycleTransition defines transition settings
type LifecycleTransition struct {
	// Days after which to transition
	Days int32 `json:"days"`

	// StorageClass to transition to
	// +kubebuilder:validation:Enum=GLACIER;DEEP_ARCHIVE;INTELLIGENT_TIERING;STANDARD_IA;ONEZONE_IA
	StorageClass string `json:"storageClass"`
}

// CORSRule defines a CORS rule
type CORSRule struct {
	// AllowedOrigins
	AllowedOrigins []string `json:"allowedOrigins"`

	// AllowedMethods
	AllowedMethods []string `json:"allowedMethods"`

	// AllowedHeaders
	// +optional
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`

	// ExposeHeaders
	// +optional
	ExposeHeaders []string `json:"exposeHeaders,omitempty"`

	// MaxAgeSeconds
	// +optional
	MaxAgeSeconds int32 `json:"maxAgeSeconds,omitempty"`
}

// PublicAccessBlockConfiguration defines public access block settings
type PublicAccessBlockConfiguration struct {
	// BlockPublicAcls
	// +optional
	BlockPublicAcls bool `json:"blockPublicAcls,omitempty"`

	// IgnorePublicAcls
	// +optional
	IgnorePublicAcls bool `json:"ignorePublicAcls,omitempty"`

	// BlockPublicPolicy
	// +optional
	BlockPublicPolicy bool `json:"blockPublicPolicy,omitempty"`

	// RestrictPublicBuckets
	// +optional
	RestrictPublicBuckets bool `json:"restrictPublicBuckets,omitempty"`
}

// S3BucketStatus defines the observed state of S3Bucket
type S3BucketStatus struct {
	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready indicates if the bucket is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ARN of the bucket
	// +optional
	ARN string `json:"arn,omitempty"`

	// Region where the bucket exists
	// +optional
	Region string `json:"region,omitempty"`

	// BucketDomainName
	// +optional
	BucketDomainName string `json:"bucketDomainName,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=s3
// +kubebuilder:printcolumn:name="Bucket",type=string,JSONPath=`.spec.bucketName`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.status.region`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// S3Bucket is the Schema for the s3buckets API
type S3Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   S3BucketSpec   `json:"spec,omitempty"`
	Status S3BucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// S3BucketList contains a list of S3Bucket
type S3BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []S3Bucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&S3Bucket{}, &S3BucketList{})
}
