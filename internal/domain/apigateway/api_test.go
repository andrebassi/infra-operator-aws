package apigateway

import "testing"

func TestAPI_Validate(t *testing.T) {
	tests := []struct {
		name    string
		api     *API
		wantErr bool
	}{
		{
			name: "valid REST API",
			api: &API{
				Name:         "my-api",
				ProtocolType: ProtocolTypeREST,
				EndpointType: EndpointTypeREGIONAL,
			},
			wantErr: false,
		},
		{
			name: "valid HTTP API",
			api: &API{
				Name:         "my-http-api",
				ProtocolType: ProtocolTypeHTTP,
				EndpointType: EndpointTypeEDGE,
			},
			wantErr: false,
		},
		{
			name: "valid WebSocket API",
			api: &API{
				Name:         "my-websocket-api",
				ProtocolType: ProtocolTypeWEBSOCKET,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			api: &API{
				ProtocolType: ProtocolTypeREST,
			},
			wantErr: true,
		},
		{
			name: "invalid protocol type",
			api: &API{
				Name:         "my-api",
				ProtocolType: "INVALID",
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint type",
			api: &API{
				Name:         "my-api",
				EndpointType: "INVALID",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.api.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("API.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPI_SetDefaults(t *testing.T) {
	api := &API{
		Name: "my-api",
	}

	api.SetDefaults()

	if api.ProtocolType != ProtocolTypeREST {
		t.Errorf("expected ProtocolType %s, got %s", ProtocolTypeREST, api.ProtocolType)
	}

	if api.EndpointType != EndpointTypeREGIONAL {
		t.Errorf("expected EndpointType %s, got %s", EndpointTypeREGIONAL, api.EndpointType)
	}

	if api.DeletionPolicy != DeletionPolicyDelete {
		t.Errorf("expected DeletionPolicy %s, got %s", DeletionPolicyDelete, api.DeletionPolicy)
	}
}

func TestAPI_IsReady(t *testing.T) {
	tests := []struct {
		name string
		api  *API
		want bool
	}{
		{
			name: "ready API",
			api: &API{
				APIID:       "abc123",
				APIEndpoint: "https://abc123.execute-api.us-east-1.amazonaws.com",
			},
			want: true,
		},
		{
			name: "not ready - missing ID",
			api: &API{
				APIEndpoint: "https://abc123.execute-api.us-east-1.amazonaws.com",
			},
			want: false,
		},
		{
			name: "not ready - missing endpoint",
			api: &API{
				APIID: "abc123",
			},
			want: false,
		},
		{
			name: "not ready - both missing",
			api:  &API{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.api.IsReady(); got != tt.want {
				t.Errorf("API.IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
