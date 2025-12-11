package secretsmanager_test
import ("testing"; "infra-operator/internal/domain/secretsmanager")
func TestSecret_Validate(t *testing.T) {
	tests := []struct {name string; s *secretsmanager.Secret; wantErr error}{
		{"valid", &secretsmanager.Secret{SecretName: "test", SecretString: "value"}, nil},
		{"no name", &secretsmanager.Secret{SecretString: "value"}, secretsmanager.ErrInvalidSecretName},
		{"no value", &secretsmanager.Secret{SecretName: "test"}, secretsmanager.ErrNoSecretValue},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.s.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestSecret_SetDefaults(t *testing.T) {s := &secretsmanager.Secret{}; s.SetDefaults(); if s.DeletionPolicy != "Delete" || s.Tags == nil {t.Error("failed")}}
func TestSecret_ShouldDelete(t *testing.T) {if !(&secretsmanager.Secret{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}}
