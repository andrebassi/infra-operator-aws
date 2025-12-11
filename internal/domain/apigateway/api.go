package apigateway

import "errors"

var (
	ErrInvalidName         = errors.New("invalid API name: cannot be empty")
	ErrInvalidProtocolType = errors.New("invalid protocol type: must be REST, HTTP, or WEBSOCKET")
	ErrInvalidEndpointType = errors.New("invalid endpoint type: must be REGIONAL, EDGE, or PRIVATE")
	ErrInvalidDeletionPolicy = errors.New("invalid deletion policy: must be Delete, Retain, or Orphan")
)

const (
	ProtocolTypeREST      = "REST"
	ProtocolTypeHTTP      = "HTTP"
	ProtocolTypeWEBSOCKET = "WEBSOCKET"

	EndpointTypeREGIONAL = "REGIONAL"
	EndpointTypeEDGE     = "EDGE"
	EndpointTypePRIVATE  = "PRIVATE"

	DeletionPolicyDelete = "Delete"
	DeletionPolicyRetain = "Retain"
	DeletionPolicyOrphan = "Orphan"
)

// API represents an AWS API Gateway domain model
type API struct {
	Name                      string
	Description               string
	ProtocolType              string
	EndpointType              string
	DisableExecuteApiEndpoint bool
	CorsConfiguration         *CorsConfiguration
	Tags                      map[string]string
	DeletionPolicy            string

	// Output fields from AWS
	APIID       string
	APIEndpoint string
}

// CorsConfiguration represents CORS settings for HTTP APIs
type CorsConfiguration struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           int32
	AllowCredentials bool
}

// Validate validates the API fields
func (a *API) Validate() error {
	if a.Name == "" {
		return ErrInvalidName
	}

	if a.ProtocolType != "" {
		if a.ProtocolType != ProtocolTypeREST &&
		   a.ProtocolType != ProtocolTypeHTTP &&
		   a.ProtocolType != ProtocolTypeWEBSOCKET {
			return ErrInvalidProtocolType
		}
	}

	if a.EndpointType != "" {
		if a.EndpointType != EndpointTypeREGIONAL &&
		   a.EndpointType != EndpointTypeEDGE &&
		   a.EndpointType != EndpointTypePRIVATE {
			return ErrInvalidEndpointType
		}
	}

	if a.DeletionPolicy != "" {
		if a.DeletionPolicy != DeletionPolicyDelete &&
		   a.DeletionPolicy != DeletionPolicyRetain &&
		   a.DeletionPolicy != DeletionPolicyOrphan {
			return ErrInvalidDeletionPolicy
		}
	}

	return nil
}

// SetDefaults sets default values for the API
func (a *API) SetDefaults() {
	if a.ProtocolType == "" {
		a.ProtocolType = ProtocolTypeREST
	}

	if a.EndpointType == "" {
		a.EndpointType = EndpointTypeREGIONAL
	}

	if a.DeletionPolicy == "" {
		a.DeletionPolicy = DeletionPolicyDelete
	}
}

// IsReady returns true if the API is ready for use
func (a *API) IsReady() bool {
	return a.APIID != "" && a.APIEndpoint != ""
}
