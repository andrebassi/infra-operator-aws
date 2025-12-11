package subnet_test
import ("testing"; "infra-operator/internal/domain/subnet")
func TestSubnet_Validate(t *testing.T) {
	tests := []struct {name string; s *subnet.Subnet; wantErr error}{
		{"valid", &subnet.Subnet{VpcID: "vpc-123", CidrBlock: "10.0.1.0/24"}, nil},
		{"no VPC", &subnet.Subnet{CidrBlock: "10.0.1.0/24"}, subnet.ErrInvalidVpcID},
		{"no CIDR", &subnet.Subnet{VpcID: "vpc-123"}, subnet.ErrInvalidCidrBlock},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.s.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestSubnet_SetDefaults(t *testing.T) {s := &subnet.Subnet{}; s.SetDefaults(); if s.DeletionPolicy != "Delete" || s.Tags == nil {t.Error("failed")}}
func TestSubnet_ShouldDelete(t *testing.T) {if !(&subnet.Subnet{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}}
