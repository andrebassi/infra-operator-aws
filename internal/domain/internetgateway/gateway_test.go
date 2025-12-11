package internetgateway_test

import (
	"testing"
	"infra-operator/internal/domain/internetgateway"
)

func TestGateway_Validate(t *testing.T) {
	tests := []struct {
		name    string
		g       *internetgateway.Gateway
		wantErr error
	}{
		{"valid", &internetgateway.Gateway{VpcID: "vpc-123"}, nil},
		{"no VPC", &internetgateway.Gateway{}, internetgateway.ErrInvalidVpcID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.g.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGateway_SetDefaults(t *testing.T) {
	g := &internetgateway.Gateway{}
	g.SetDefaults()
	if g.DeletionPolicy != "Delete" || g.Tags == nil {
		t.Errorf("SetDefaults() failed")
	}
}

func TestGateway_ShouldDelete(t *testing.T) {
	if !(&internetgateway.Gateway{DeletionPolicy: "Delete"}).ShouldDelete() {
		t.Error("ShouldDelete() = false, want true")
	}
}
