package dynamodb_test
import ("testing"; "infra-operator/internal/domain/dynamodb")
func TestTable_Validate(t *testing.T) {
	tests := []struct {name string; tbl *dynamodb.Table; wantErr error}{
		{"valid", &dynamodb.Table{Name: "test", HashKey: dynamodb.AttributeDefinition{Name: "id", Type: "S"}}, nil},
		{"no name", &dynamodb.Table{HashKey: dynamodb.AttributeDefinition{Name: "id", Type: "S"}}, dynamodb.ErrInvalidTableName},
		{"no hash key", &dynamodb.Table{Name: "test"}, dynamodb.ErrInvalidKey},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.tbl.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
