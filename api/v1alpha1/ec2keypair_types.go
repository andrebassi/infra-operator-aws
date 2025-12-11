package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EC2KeyPairSpec define o estado desejado do EC2KeyPair
type EC2KeyPairSpec struct {
	// ProviderRef referencia o AWSProvider a ser usado
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// KeyName é o nome do par de chaves na AWS
	// Se não especificado, usa metadata.name
	// +optional
	KeyName string `json:"keyName,omitempty"`

	// PublicKeyMaterial é a chave pública para importar (para importar chaves existentes)
	// Se não especificado, a AWS gerará um novo par de chaves
	// +optional
	PublicKeyMaterial string `json:"publicKeyMaterial,omitempty"`

	// Tags são as etiquetas a serem aplicadas ao recurso
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy define o comportamento ao deletar o recurso
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// SecretRef especifica onde armazenar a chave privada (apenas para chaves recém-geradas)
	// +optional
	SecretRef *KeyPairSecretRef `json:"secretRef,omitempty"`
}

// KeyPairSecretRef define onde armazenar a chave privada do EC2KeyPair
type KeyPairSecretRef struct {
	// Name é o nome do secret a ser criado
	Name string `json:"name"`

	// Namespace do secret (padrão é o namespace do KeyPair)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// EC2KeyPairStatus define o estado observado do EC2KeyPair
type EC2KeyPairStatus struct {
	// Conditions são as condições do recurso
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready indica se o recurso está pronto
	// +optional
	Ready bool `json:"ready,omitempty"`

	// KeyPairID é o ID do par de chaves na AWS
	// +optional
	KeyPairID string `json:"keyPairID,omitempty"`

	// KeyFingerprint é a impressão digital da chave
	// +optional
	KeyFingerprint string `json:"keyFingerprint,omitempty"`

	// KeyName é o nome do par de chaves na AWS
	// +optional
	KeyName string `json:"keyName,omitempty"`

	// KeyType é o tipo da chave (rsa, ed25519)
	// +optional
	KeyType string `json:"keyType,omitempty"`

	// SecretCreated indica se o secret com a chave privada foi criado
	// +optional
	SecretCreated bool `json:"secretCreated,omitempty"`

	// LastSyncTime é a última vez que o recurso foi sincronizado
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=keypair
// +kubebuilder:printcolumn:name="KeyName",type=string,JSONPath=`.status.keyName`
// +kubebuilder:printcolumn:name="KeyPairID",type=string,JSONPath=`.status.keyPairID`
// +kubebuilder:printcolumn:name="Fingerprint",type=string,JSONPath=`.status.keyFingerprint`,priority=1
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// EC2KeyPair é o Schema para a API ec2keypairs
type EC2KeyPair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EC2KeyPairSpec   `json:"spec,omitempty"`
	Status EC2KeyPairStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EC2KeyPairList contém uma lista de EC2KeyPair
type EC2KeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EC2KeyPair `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EC2KeyPair{}, &EC2KeyPairList{})
}
