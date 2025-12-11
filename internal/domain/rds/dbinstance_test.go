package rds_test
import ("testing"; "infra-operator/internal/domain/rds")
func TestDBInstance_Validate(t *testing.T) {
	tests := []struct {name string; db *rds.DBInstance; wantErr error}{
		{"valid", &rds.DBInstance{DBInstanceIdentifier: "test", Engine: "postgres", DBInstanceClass: "db.t3.micro", AllocatedStorage: 20, MasterUsername: "admin", MasterPassword: "password"}, nil},
		{"no identifier", &rds.DBInstance{Engine: "postgres", DBInstanceClass: "db.t3.micro", AllocatedStorage: 20, MasterUsername: "admin", MasterPassword: "password"}, rds.ErrInvalidDBIdentifier},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.db.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestDBInstance_SetDefaults(t *testing.T) {db := &rds.DBInstance{Engine: "postgres"}; db.SetDefaults(); if db.DeletionPolicy != "Delete" {t.Error("failed")}}
