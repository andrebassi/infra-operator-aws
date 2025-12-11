package ecr_test
import ("testing"; "infra-operator/internal/domain/ecr")
func TestRepository_Validate(t *testing.T) {
	tests := []struct {name string; r *ecr.Repository; wantErr error}{
		{"valid", &ecr.Repository{RepositoryName: "test"}, nil},
		{"no name", &ecr.Repository{}, ecr.ErrInvalidRepositoryName},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.r.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestRepository_SetDefaults(t *testing.T) {r := &ecr.Repository{}; r.SetDefaults(); if r.DeletionPolicy != "Delete" {t.Error("failed")}}
