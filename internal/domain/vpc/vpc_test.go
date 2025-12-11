package vpc_test

import (
	"testing"
	"infra-operator/internal/domain/vpc"
)

func TestVPC_Validate(t *testing.T) {
	tests := []struct {
		name    string
		v       *vpc.VPC
		wantErr error
	}{
		{"valid VPC", &vpc.VPC{CidrBlock: "10.0.0.0/16"}, nil},
		{"empty CIDR", &vpc.VPC{CidrBlock: ""}, vpc.ErrInvalidCidrBlock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.v.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVPC_SetDefaults(t *testing.T) {
	v := &vpc.VPC{CidrBlock: "10.0.0.0/16"}
	v.SetDefaults()
	if v.DeletionPolicy != "Delete" || v.InstanceTenancy != "default" || v.Tags == nil {
		t.Errorf("SetDefaults() failed")
	}
}

func TestVPC_ShouldDelete(t *testing.T) {
	tests := []struct {
		name string
		v    *vpc.VPC
		want bool
	}{
		{"Delete", &vpc.VPC{DeletionPolicy: "Delete"}, true},
		{"Retain", &vpc.VPC{DeletionPolicy: "Retain"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVPC_IsAvailable(t *testing.T) {
	tests := []struct {
		name string
		v    *vpc.VPC
		want bool
	}{
		{"available", &vpc.VPC{State: "available"}, true},
		{"pending", &vpc.VPC{State: "pending"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
