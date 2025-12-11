package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LambdaFunctionSpec defines the desired state of LambdaFunction
type LambdaFunctionSpec struct {
	// ProviderRef references the AWSProvider for authentication
	ProviderRef ProviderReference `json:"providerRef"`

	// FunctionName is the name of the Lambda function
	FunctionName string `json:"functionName"`

	// Runtime is the Lambda runtime (e.g., python3.12, nodejs20.x, go1.x)
	Runtime string `json:"runtime"`

	// Handler is the function entry point (e.g., index.handler, main)
	Handler string `json:"handler"`

	// Code defines the function code source
	Code LambdaCode `json:"code"`

	// Role is the ARN of the IAM role for Lambda execution
	Role string `json:"role"`

	// Description is an optional description of the function
	Description string `json:"description,omitempty"`

	// Timeout in seconds (1-900, default 3)
	Timeout int32 `json:"timeout,omitempty"`

	// MemorySize in MB (128-10240, default 128)
	MemorySize int32 `json:"memorySize,omitempty"`

	// Environment variables for the function
	Environment *LambdaEnvironment `json:"environment,omitempty"`

	// VpcConfig for running Lambda in VPC
	VpcConfig *LambdaVpcConfig `json:"vpcConfig,omitempty"`

	// Layers are ARNs of Lambda layers
	Layers []string `json:"layers,omitempty"`

	// Tags for the Lambda function
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens when the CR is deleted
	// Valid values: Delete (default), Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// LambdaCode defines how the function code is provided
type LambdaCode struct {
	// ZipFile for inline code (base64 encoded zip)
	ZipFile string `json:"zipFile,omitempty"`

	// S3Bucket for code stored in S3
	S3Bucket string `json:"s3Bucket,omitempty"`

	// S3Key is the object key in S3
	S3Key string `json:"s3Key,omitempty"`

	// S3ObjectVersion for versioned S3 objects
	S3ObjectVersion string `json:"s3ObjectVersion,omitempty"`

	// ImageUri for container image (e.g., ECR)
	ImageUri string `json:"imageUri,omitempty"`
}

// LambdaEnvironment defines environment variables
type LambdaEnvironment struct {
	Variables map[string]string `json:"variables,omitempty"`
}

// LambdaVpcConfig defines VPC configuration
type LambdaVpcConfig struct {
	SecurityGroupIds []string `json:"securityGroupIds"`
	SubnetIds        []string `json:"subnetIds"`
}

// LambdaFunctionStatus defines the observed state of LambdaFunction
type LambdaFunctionStatus struct {
	// Ready indicates if the function is ready
	Ready bool `json:"ready"`

	// FunctionArn is the ARN of the Lambda function
	FunctionArn string `json:"functionArn,omitempty"`

	// Version is the published version
	Version string `json:"version,omitempty"`

	// LastModified timestamp
	LastModified string `json:"lastModified,omitempty"`

	// CodeSize in bytes
	CodeSize int64 `json:"codeSize,omitempty"`

	// State is the current state (Pending, Active, Inactive, Failed)
	State string `json:"state,omitempty"`

	// StateReason provides details about the state
	StateReason string `json:"stateReason,omitempty"`

	// LastSyncTime is when the function was last synced
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=lambda
// +kubebuilder:printcolumn:name="Function",type=string,JSONPath=`.spec.functionName`
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LambdaFunction is the Schema for the lambdafunctions API
type LambdaFunction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LambdaFunctionSpec   `json:"spec,omitempty"`
	Status LambdaFunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LambdaFunctionList contains a list of LambdaFunction
type LambdaFunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LambdaFunction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LambdaFunction{}, &LambdaFunctionList{})
}
