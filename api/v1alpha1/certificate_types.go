package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertificateSpec defines the desired state of Certificate
type CertificateSpec struct {
	ProviderRef ProviderReference `json:"providerRef"`

	// DomainName is the fully qualified domain name
	// +kubebuilder:validation:Required
	DomainName string `json:"domainName"`

	// SubjectAlternativeNames are additional FQDNs
	// +optional
	SubjectAlternativeNames []string `json:"subjectAlternativeNames,omitempty"`

	// ValidationMethod for domain ownership
	// +kubebuilder:default=DNS
	// +kubebuilder:validation:Enum=DNS;EMAIL
	// +optional
	ValidationMethod string `json:"validationMethod,omitempty"`

	// Tags to apply
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// CertificateStatus defines the observed state
type CertificateStatus struct {
	Ready            bool                        `json:"ready,omitempty"`
	CertificateARN   string                      `json:"certificateARN,omitempty"`
	Status           string                      `json:"status,omitempty"`
	ValidationRecords []CertificateValidationRecord `json:"validationRecords,omitempty"`
	LastSyncTime     *metav1.Time                `json:"lastSyncTime,omitempty"`
}

type CertificateValidationRecord struct {
	DomainName       string `json:"domainName,omitempty"`
	ResourceRecordName string `json:"resourceRecordName,omitempty"`
	ResourceRecordType string `json:"resourceRecordType,omitempty"`
	ResourceRecordValue string `json:"resourceRecordValue,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=cert;certs
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.domainName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CertificateSpec   `json:"spec,omitempty"`
	Status            CertificateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Certificate{}, &CertificateList{})
}
