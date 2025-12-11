package lambda_test
import ("testing"; "infra-operator/internal/domain/lambda")
func TestFunction_Validate(t *testing.T) {
	tests := []struct {name string; f *lambda.Function; wantErr error}{
		{"valid", &lambda.Function{Name: "test", Runtime: "python3.9", Handler: "index.handler", Role: "arn:aws:iam::123:role/test", Code: lambda.Code{ZipFile: "code"}}, nil},
		{"no name", &lambda.Function{Runtime: "python3.9", Handler: "index.handler", Role: "arn:aws:iam::123:role/test", Code: lambda.Code{ZipFile: "code"}}, lambda.ErrInvalidFunctionName},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.f.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestFunction_SetDefaults(t *testing.T) {f := &lambda.Function{}; f.SetDefaults(); if f.DeletionPolicy != "Delete" {t.Error("failed")}}
