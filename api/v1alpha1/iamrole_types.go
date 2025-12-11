package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IAMRoleSpec defines the desired state of IAMRole
type IAMRoleSpec struct {
	// ProviderRef references the AWSProvider to use for this resource
	ProviderRef ProviderReference `json:"providerRef"`

	// RoleName is the name of the IAM role
	RoleName string `json:"roleName"`

	// Description of the IAM role
	// +optional
	Description string `json:"description,omitempty"`

	// AssumeRolePolicyDocument is the trust policy that grants permission to assume this role
	AssumeRolePolicyDocument string `json:"assumeRolePolicyDocument"`

	// ManagedPolicyArns is a list of managed policy ARNs to attach to the role
	// +optional
	ManagedPolicyArns []string `json:"managedPolicyArns,omitempty"`

	// InlinePolicy defines an inline policy to embed in the role
	// +optional
	InlinePolicy *InlinePolicySpec `json:"inlinePolicy,omitempty"`

	// MaxSessionDuration is the maximum session duration (in seconds) for the role
	// Valid values: 3600 (1 hour) to 43200 (12 hours)
	// +optional
	// +kubebuilder:validation:Minimum=3600
	// +kubebuilder:validation:Maximum=43200
	MaxSessionDuration int32 `json:"maxSessionDuration,omitempty"`

	// Path is the path to the role
	// +optional
	Path string `json:"path,omitempty"`

	// PermissionsBoundary is the ARN of the policy that sets the permissions boundary for the role
	// +optional
	PermissionsBoundary string `json:"permissionsBoundary,omitempty"`

	// Tags to apply to the IAM role
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens to the AWS resource when the CR is deleted
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain;Orphan
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// InlinePolicySpec defines an inline policy
type InlinePolicySpec struct {
	// PolicyName is the name of the inline policy
	PolicyName string `json:"policyName"`

	// PolicyDocument is the JSON policy document
	PolicyDocument string `json:"policyDocument"`
}

// IAMRoleStatus defines the observed state of IAMRole
type IAMRoleStatus struct {
	// Ready indicates whether the IAM role is ready
	Ready bool `json:"ready"`

	// RoleArn is the ARN of the IAM role
	// +optional
	RoleArn string `json:"roleArn,omitempty"`

	// RoleId is the stable and unique string identifying the role
	// +optional
	RoleId string `json:"roleId,omitempty"`

	// CreatedAt is when the role was created
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// LastSyncTime is when the role was last synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Message provides additional information about the role status
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.roleName`
// +kubebuilder:printcolumn:name="ARN",type=string,JSONPath=`.status.roleArn`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// IAMRole is the Schema for the iamroles API
type IAMRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMRoleSpec   `json:"spec,omitempty"`
	Status IAMRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IAMRoleList contains a list of IAMRole
type IAMRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IAMRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IAMRole{}, &IAMRoleList{})
}
