package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIGatewaySpec defines the desired state of APIGateway
type APIGatewaySpec struct {
	// ProviderRef references the AWSProvider for credentials
	ProviderRef ProviderReference `json:"providerRef"`

	// Name of the API Gateway
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the API
	Description string `json:"description,omitempty"`

	// ProtocolType specifies the API protocol
	// +kubebuilder:validation:Enum=REST;HTTP;WEBSOCKET
	// +kubebuilder:default=REST
	ProtocolType string `json:"protocolType,omitempty"`

	// EndpointType specifies the endpoint type
	// +kubebuilder:validation:Enum=REGIONAL;EDGE;PRIVATE
	// +kubebuilder:default=REGIONAL
	EndpointType string `json:"endpointType,omitempty"`

	// DisableExecuteApiEndpoint disables the default execute-api endpoint
	DisableExecuteApiEndpoint bool `json:"disableExecuteApiEndpoint,omitempty"`

	// CorsConfiguration for HTTP APIs
	CorsConfiguration *CorsConfiguration `json:"corsConfiguration,omitempty"`

	// Tags to apply to the API Gateway
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines how to handle the API on CR deletion
	// +kubebuilder:validation:Enum=Delete;Retain;Orphan
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// CorsConfiguration defines CORS settings for HTTP APIs
type CorsConfiguration struct {
	// AllowOrigins is a list of allowed origins
	AllowOrigins []string `json:"allowOrigins,omitempty"`

	// AllowMethods is a list of allowed HTTP methods
	AllowMethods []string `json:"allowMethods,omitempty"`

	// AllowHeaders is a list of allowed HTTP headers
	AllowHeaders []string `json:"allowHeaders,omitempty"`

	// ExposeHeaders is a list of exposed HTTP headers
	ExposeHeaders []string `json:"exposeHeaders,omitempty"`

	// MaxAge is the max age for CORS in seconds
	MaxAge int32 `json:"maxAge,omitempty"`

	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool `json:"allowCredentials,omitempty"`
}

// APIGatewayStatus defines the observed state of APIGateway
type APIGatewayStatus struct {
	// Ready indicates if the API Gateway is ready
	Ready bool `json:"ready,omitempty"`

	// APIID is the AWS API Gateway ID
	APIID string `json:"apiId,omitempty"`

	// APIEndpoint is the invoke URL
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// ProtocolType is the protocol type
	ProtocolType string `json:"protocolType,omitempty"`

	// LastSyncTime is the last time the resource was synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="API ID",type=string,JSONPath=`.status.apiId`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.status.protocolType`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.apiEndpoint`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`

// APIGateway is the Schema for the apigateways API
type APIGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIGatewaySpec   `json:"spec,omitempty"`
	Status APIGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// APIGatewayList contains a list of APIGateway
type APIGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIGateway{}, &APIGatewayList{})
}
