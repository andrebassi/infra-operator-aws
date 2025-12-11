package natgateway_test
import ("testing"; "infra-operator/internal/domain/natgateway")
func TestGateway_Validate(t *testing.T) {
	tests := []struct {name string; g *natgateway.Gateway; wantErr error}{
		{"valid", &natgateway.Gateway{SubnetID: "subnet-123"}, nil},
		{"no subnet", &natgateway.Gateway{}, natgateway.ErrInvalidSubnetID},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.g.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestGateway_SetDefaults(t *testing.T) {g := &natgateway.Gateway{}; g.SetDefaults(); if g.DeletionPolicy != "Delete" || g.Tags == nil {t.Error("failed")}}
func TestGateway_ShouldDelete(t *testing.T) {if !(&natgateway.Gateway{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}}
