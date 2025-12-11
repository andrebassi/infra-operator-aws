package alb_test

import (
	"testing"

	"infra-operator/internal/domain/alb"
)

func TestLoadBalancer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		lb      *alb.LoadBalancer
		wantErr error
	}{
		{
			name: "valid internet-facing",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Scheme:           "internet-facing",
				Subnets:          []string{"subnet-1", "subnet-2"},
			},
			wantErr: nil,
		},
		{
			name: "valid internal",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Scheme:           "internal",
				Subnets:          []string{"subnet-1", "subnet-2"},
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "",
				Subnets:          []string{"subnet-1", "subnet-2"},
			},
			wantErr: alb.ErrInvalidLoadBalancerName,
		},
		{
			name: "invalid scheme",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Scheme:           "invalid",
				Subnets:          []string{"subnet-1", "subnet-2"},
			},
			wantErr: alb.ErrInvalidScheme,
		},
		{
			name: "insufficient subnets",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Subnets:          []string{"subnet-1"},
			},
			wantErr: alb.ErrInvalidSubnets,
		},
		{
			name: "no subnets",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Subnets:          []string{},
			},
			wantErr: alb.ErrInvalidSubnets,
		},
		{
			name: "invalid idle timeout low",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Subnets:          []string{"subnet-1", "subnet-2"},
				IdleTimeout:      0,
			},
			wantErr: nil, // 0 is valid, defaults to 60
		},
		{
			name: "invalid idle timeout high",
			lb: &alb.LoadBalancer{
				LoadBalancerName: "my-alb",
				Subnets:          []string{"subnet-1", "subnet-2"},
				IdleTimeout:      5000,
			},
			wantErr: alb.ErrInvalidIdleTimeout,
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
	lb := &alb.LoadBalancer{}
	lb.SetDefaults()

	if lb.Scheme != "internet-facing" {
		t.Errorf("Scheme = %v, want internet-facing", lb.Scheme)
	}
	if lb.IPAddressType != "ipv4" {
		t.Errorf("IPAddressType = %v, want ipv4", lb.IPAddressType)
	}
	if lb.IdleTimeout != 60 {
		t.Errorf("IdleTimeout = %v, want 60", lb.IdleTimeout)
	}
	if lb.DeletionPolicy != "Delete" {
		t.Errorf("DeletionPolicy = %v, want Delete", lb.DeletionPolicy)
	}
	if lb.Tags == nil {
		t.Error("Tags should be initialized")
	}
	if !lb.EnableHttp2 {
		t.Error("EnableHttp2 should be true by default")
	}
}

func TestLoadBalancer_ShouldDelete(t *testing.T) {
	tests := []struct {
		name string
		lb   *alb.LoadBalancer
		want bool
	}{
		{"Delete policy", &alb.LoadBalancer{DeletionPolicy: "Delete"}, true},
		{"Retain policy", &alb.LoadBalancer{DeletionPolicy: "Retain"}, false},
		{"Empty policy", &alb.LoadBalancer{DeletionPolicy: ""}, false},
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
			lb := &alb.LoadBalancer{State: tt.state}
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
			lb := &alb.LoadBalancer{Scheme: tt.scheme}
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
			lb := &alb.LoadBalancer{Scheme: tt.scheme}
			if got := lb.IsInternal(); got != tt.want {
				t.Errorf("IsInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}
