package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KMSKeySpec defines the desired state of KMSKey
type KMSKeySpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// Description
	// +optional
	Description string `json:"description,omitempty"`

	// KeyUsage (ENCRYPT_DECRYPT, SIGN_VERIFY, GENERATE_VERIFY_MAC)
	// +optional
	// +kubebuilder:validation:Enum=ENCRYPT_DECRYPT;SIGN_VERIFY;GENERATE_VERIFY_MAC
	// +kubebuilder:default=ENCRYPT_DECRYPT
	KeyUsage string `json:"keyUsage,omitempty"`

	// KeySpec (SYMMETRIC_DEFAULT, RSA_2048, RSA_3072, RSA_4096, ECC_NIST_P256, etc)
	// +optional
	// +kubebuilder:default=SYMMETRIC_DEFAULT
	KeySpec string `json:"keySpec,omitempty"`

	// MultiRegion
	// +optional
	MultiRegion bool `json:"multiRegion,omitempty"`

	// EnableKeyRotation (only for symmetric keys)
	// +optional
	EnableKeyRotation bool `json:"enableKeyRotation,omitempty"`

	// Enabled
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// KeyPolicy in JSON format
	// +optional
	KeyPolicy string `json:"keyPolicy,omitempty"`

	// Tags
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// PendingWindowInDays for key deletion (7-30 days)
	// +optional
	// +kubebuilder:validation:Minimum=7
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=30
	PendingWindowInDays int32 `json:"pendingWindowInDays,omitempty"`
}

// KMSKeyStatus defines the observed state of KMSKey
type KMSKeyStatus struct {
	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// KeyId
	// +optional
	KeyId string `json:"keyId,omitempty"`

	// Arn
	// +optional
	Arn string `json:"arn,omitempty"`

	// KeyState (Enabled, Disabled, PendingDeletion, PendingImport, Unavailable)
	// +optional
	KeyState string `json:"keyState,omitempty"`

	// CreationDate
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// DeletionDate (if PendingDeletion)
	// +optional
	DeletionDate *metav1.Time `json:"deletionDate,omitempty"`

	// KeyManager (AWS or CUSTOMER)
	// +optional
	KeyManager string `json:"keyManager,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=kms
// +kubebuilder:printcolumn:name="KeyId",type=string,JSONPath=`.status.keyId`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.keyState`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KMSKey is the Schema for the kmskeys API
type KMSKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KMSKeySpec   `json:"spec,omitempty"`
	Status KMSKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KMSKeyList contains a list of KMSKey
type KMSKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KMSKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KMSKey{}, &KMSKeyList{})
}
