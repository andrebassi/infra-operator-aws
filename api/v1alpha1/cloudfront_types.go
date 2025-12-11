package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudFrontSpec defines the desired state of CloudFront
type CloudFrontSpec struct {
	// ProviderRef references the AWSProvider for credentials
	ProviderRef ProviderReference `json:"providerRef"`

	// Comment describes the distribution
	Comment string `json:"comment,omitempty"`

	// DefaultRootObject is the default page (e.g., index.html)
	DefaultRootObject string `json:"defaultRootObject,omitempty"`

	// Origins defines the origin servers
	// +kubebuilder:validation:MinItems=1
	Origins []CloudFrontOrigin `json:"origins"`

	// DefaultCacheBehavior defines cache behavior for all paths
	DefaultCacheBehavior CloudFrontCacheBehavior `json:"defaultCacheBehavior"`

	// CacheBehaviors defines path-specific cache behaviors
	CacheBehaviors []CloudFrontCacheBehavior `json:"cacheBehaviors,omitempty"`

	// Enabled indicates if the distribution is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// PriceClass determines distribution coverage
	// +kubebuilder:validation:Enum=PriceClass_All;PriceClass_100;PriceClass_200
	// +kubebuilder:default=PriceClass_100
	PriceClass string `json:"priceClass,omitempty"`

	// ViewerCertificate configures SSL/TLS
	ViewerCertificate *ViewerCertificate `json:"viewerCertificate,omitempty"`

	// Aliases are alternate domain names (CNAMEs)
	Aliases []string `json:"aliases,omitempty"`

	// Tags to apply to the distribution
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines how to handle the distribution on CR deletion
	// +kubebuilder:validation:Enum=Delete;Retain;Orphan
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// CloudFrontOrigin represents an origin server
type CloudFrontOrigin struct {
	// ID uniquely identifies this origin
	ID string `json:"id"`

	// DomainName is the DNS name of the origin
	DomainName string `json:"domainName"`

	// OriginPath is the path to append to origin requests
	OriginPath string `json:"originPath,omitempty"`

	// CustomHeaders are headers to add to origin requests
	CustomHeaders map[string]string `json:"customHeaders,omitempty"`

	// S3OriginConfig for S3 origins
	S3OriginConfig *S3OriginConfig `json:"s3OriginConfig,omitempty"`

	// CustomOriginConfig for custom origins
	CustomOriginConfig *CustomOriginConfig `json:"customOriginConfig,omitempty"`
}

// S3OriginConfig configures S3 origin access
type S3OriginConfig struct {
	// OriginAccessIdentity restricts S3 access
	OriginAccessIdentity string `json:"originAccessIdentity,omitempty"`
}

// CustomOriginConfig configures custom origin settings
type CustomOriginConfig struct {
	// HTTPPort for HTTP connections
	// +kubebuilder:default=80
	HTTPPort int32 `json:"httpPort,omitempty"`

	// HTTPSPort for HTTPS connections
	// +kubebuilder:default=443
	HTTPSPort int32 `json:"httpsPort,omitempty"`

	// OriginProtocolPolicy determines connection protocol
	// +kubebuilder:validation:Enum=http-only;https-only;match-viewer
	// +kubebuilder:default=https-only
	OriginProtocolPolicy string `json:"originProtocolPolicy,omitempty"`
}

// CloudFrontCacheBehavior defines caching behavior
type CloudFrontCacheBehavior struct {
	// PathPattern for this cache behavior (omit for default)
	PathPattern string `json:"pathPattern,omitempty"`

	// TargetOriginID references an origin
	TargetOriginID string `json:"targetOriginId"`

	// ViewerProtocolPolicy determines viewer connection policy
	// +kubebuilder:validation:Enum=allow-all;redirect-to-https;https-only
	// +kubebuilder:default=redirect-to-https
	ViewerProtocolPolicy string `json:"viewerProtocolPolicy,omitempty"`

	// AllowedMethods lists allowed HTTP methods
	AllowedMethods []string `json:"allowedMethods,omitempty"`

	// CachedMethods lists methods to cache
	CachedMethods []string `json:"cachedMethods,omitempty"`

	// Compress enables automatic compression
	Compress bool `json:"compress,omitempty"`

	// MinTTL is minimum cache time in seconds
	MinTTL int64 `json:"minTTL,omitempty"`

	// MaxTTL is maximum cache time in seconds
	MaxTTL int64 `json:"maxTTL,omitempty"`

	// DefaultTTL is default cache time in seconds
	DefaultTTL int64 `json:"defaultTTL,omitempty"`
}

// ViewerCertificate configures SSL/TLS certificate
type ViewerCertificate struct {
	// ACMCertificateARN for ACM certificate
	ACMCertificateARN string `json:"acmCertificateArn,omitempty"`

	// CloudFrontDefaultCertificate uses CloudFront certificate
	CloudFrontDefaultCertificate bool `json:"cloudFrontDefaultCertificate,omitempty"`

	// MinimumProtocolVersion for SSL/TLS
	// +kubebuilder:default=TLSv1.2_2021
	MinimumProtocolVersion string `json:"minimumProtocolVersion,omitempty"`

	// SSLSupportMethod determines how CloudFront serves HTTPS
	// +kubebuilder:validation:Enum=sni-only;vip
	// +kubebuilder:default=sni-only
	SSLSupportMethod string `json:"sslSupportMethod,omitempty"`
}

// CloudFrontStatus defines the observed state of CloudFront
type CloudFrontStatus struct {
	// Ready indicates if the distribution is ready
	Ready bool `json:"ready,omitempty"`

	// DistributionID is the CloudFront distribution ID
	DistributionID string `json:"distributionId,omitempty"`

	// DomainName is the CloudFront domain name
	DomainName string `json:"domainName,omitempty"`

	// Status is the distribution status (Deployed, InProgress)
	Status string `json:"status,omitempty"`

	// LastSyncTime is the last time the resource was synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Distribution ID",type=string,JSONPath=`.status.distributionId`
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.status.domainName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`

// CloudFront is the Schema for the cloudfronts API
type CloudFront struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudFrontSpec   `json:"spec,omitempty"`
	Status CloudFrontStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudFrontList contains a list of CloudFront
type CloudFrontList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudFront `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudFront{}, &CloudFrontList{})
}
