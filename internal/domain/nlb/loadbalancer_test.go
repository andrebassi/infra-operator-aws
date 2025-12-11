package nlb_test

import (
	"testing"

	"infra-operator/internal/domain/nlb"
)

func TestLoadBalancer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		lb      *nlb.LoadBalancer
		wantErr error
	}{
		{
			name: "valid internet-facing",
			lb: &nlb.LoadBalancer{
				LoadBalancerName: "my-nlb",
				Scheme:           "internet-facing",
				Subnets:          []string{"subnet-1"},
			},
			wantErr: nil,
		},
		{
			name: "valid internal",
			lb: &nlb.LoadBalancer{
				LoadBalancerName: "my-nlb",
				Scheme:           "internal",
				Subnets:          []string{"subnet-1"},
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			lb: &nlb.LoadBalancer{
				LoadBalancerName: "",
				Subnets:          []string{"subnet-1"},
			},
			wantErr: nlb.ErrInvalidLoadBalancerName,
		},
		{
			name: "invalid scheme",
			lb: &nlb.LoadBalancer{
				LoadBalancerName: "my-nlb",
				Scheme:           "invalid",
				Subnets:          []string{"subnet-1"},
			},
			wantErr: nlb.ErrInvalidScheme,
		},
		{
			name: "no subnets",
			lb: &nlb.LoadBalancer{
				LoadBalancerName: "my-nlb",
				Subnets:          []string{},
			},
			wantErr: nlb.ErrInvalidSubnets,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lb.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadBalancer_SetDefaults(t *testing.T) {
	lb := &nlb.LoadBalancer{}
	lb.SetDefaults()

	if lb.Scheme != "internet-facing" {
		t.Errorf("Scheme = %v, want internet-facing", lb.Scheme)
	}
	if lb.IPAddressType != "ipv4" {
		t.Errorf("IPAddressType = %v, want ipv4", lb.IPAddressType)
	}
	if lb.DeletionPolicy != "Delete" {
		t.Errorf("DeletionPolicy = %v, want Delete", lb.DeletionPolicy)
	}
	if lb.Tags == nil {
		t.Error("Tags should be initialized")
	}
}

func TestLoadBalancer_ShouldDelete(t *testing.T) {
	tests := []struct {
		name string
		lb   *nlb.LoadBalancer
		want bool
	}{
		{"Delete policy", &nlb.LoadBalancer{DeletionPolicy: "Delete"}, true},
		{"Retain policy", &nlb.LoadBalancer{DeletionPolicy: "Retain"}, false},
		{"Empty policy", &nlb.LoadBalancer{DeletionPolicy: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lb.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadBalancer_IsActive(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{"active", "active", true},
		{"provisioning", "provisioning", false},
		{"failed", "failed", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &nlb.LoadBalancer{State: tt.state}
			if got := lb.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadBalancer_IsInternetFacing(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		want   bool
	}{
		{"internet-facing", "internet-facing", true},
		{"internal", "internal", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &nlb.LoadBalancer{Scheme: tt.scheme}
			if got := lb.IsInternetFacing(); got != tt.want {
				t.Errorf("IsInternetFacing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadBalancer_IsInternal(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		want   bool
	}{
		{"internal", "internal", true},
		{"internet-facing", "internet-facing", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &nlb.LoadBalancer{Scheme: tt.scheme}
			if got := lb.IsInternal(); got != tt.want {
				t.Errorf("IsInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}
