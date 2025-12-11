package elasticip_test

import (
	"testing"

	"infra-operator/internal/domain/elasticip"
)

func TestAddress_Validate(t *testing.T) {
	tests := []struct {
		name    string
		addr    *elasticip.Address
		wantErr error
	}{
		{
			name: "valid vpc domain",
			addr: &elasticip.Address{
				Domain: "vpc",
			},
			wantErr: nil,
		},
		{
			name: "valid standard domain",
			addr: &elasticip.Address{
				Domain: "standard",
			},
			wantErr: nil,
		},
		{
			name: "empty domain (default will be set)",
			addr: &elasticip.Address{
				Domain: "",
			},
			wantErr: nil,
		},
		{
			name: "invalid domain",
			addr: &elasticip.Address{
				Domain: "invalid",
			},
			wantErr: elasticip.ErrInvalidDomain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddress_SetDefaults(t *testing.T) {
	addr := &elasticip.Address{}
	addr.SetDefaults()

	if addr.Domain != "vpc" {
		t.Errorf("Domain = %v, want vpc", addr.Domain)
	}
	if addr.DeletionPolicy != "Delete" {
		t.Errorf("DeletionPolicy = %v, want Delete", addr.DeletionPolicy)
	}
	if addr.Tags == nil {
		t.Error("Tags should be initialized")
	}
}

func TestAddress_ShouldDelete(t *testing.T) {
	tests := []struct {
		name   string
		addr   *elasticip.Address
		want   bool
	}{
		{"Delete policy", &elasticip.Address{DeletionPolicy: "Delete"}, true},
		{"Retain policy", &elasticip.Address{DeletionPolicy: "Retain"}, false},
		{"Empty policy", &elasticip.Address{DeletionPolicy: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_IsAllocated(t *testing.T) {
	tests := []struct {
		name   string
		addr   *elasticip.Address
		want   bool
	}{
		{
			"with allocation ID",
			&elasticip.Address{AllocationID: "eipalloc-12345"},
			true,
		},
		{
			"without allocation ID",
			&elasticip.Address{AllocationID: ""},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.IsAllocated(); got != tt.want {
				t.Errorf("IsAllocated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_IsAssociated(t *testing.T) {
	tests := []struct {
		name   string
		addr   *elasticip.Address
		want   bool
	}{
		{
			"with association ID",
			&elasticip.Address{AssociationID: "eipassoc-12345"},
			true,
		},
		{
			"without association ID",
			&elasticip.Address{AssociationID: ""},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.IsAssociated(); got != tt.want {
				t.Errorf("IsAssociated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_IsVPC(t *testing.T) {
	tests := []struct {
		name   string
		addr   *elasticip.Address
		want   bool
	}{
		{"vpc domain", &elasticip.Address{Domain: "vpc"}, true},
		{"standard domain", &elasticip.Address{Domain: "standard"}, false},
		{"empty domain", &elasticip.Address{Domain: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.IsVPC(); got != tt.want {
				t.Errorf("IsVPC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_IsStandard(t *testing.T) {
	tests := []struct {
		name   string
		addr   *elasticip.Address
		want   bool
	}{
		{"standard domain", &elasticip.Address{Domain: "standard"}, true},
		{"vpc domain", &elasticip.Address{Domain: "vpc"}, false},
		{"empty domain", &elasticip.Address{Domain: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.IsStandard(); got != tt.want {
				t.Errorf("IsStandard() = %v, want %v", got, tt.want)
			}
		})
	}
}
