package acm_test

import (
	"testing"

	"infra-operator/internal/domain/acm"
)

func TestCertificate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cert    *acm.Certificate
		wantErr error
	}{
		{"valid DNS", &acm.Certificate{DomainName: "example.com", ValidationMethod: "DNS"}, nil},
		{"valid EMAIL", &acm.Certificate{DomainName: "example.com", ValidationMethod: "EMAIL"}, nil},
		{"empty domain", &acm.Certificate{DomainName: ""}, acm.ErrInvalidDomainName},
		{"invalid validation", &acm.Certificate{DomainName: "example.com", ValidationMethod: "INVALID"}, acm.ErrInvalidValidationMethod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cert.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCertificate_SetDefaults(t *testing.T) {
	cert := &acm.Certificate{}
	cert.SetDefaults()

	if cert.ValidationMethod != "DNS" {
		t.Errorf("ValidationMethod = %v, want DNS", cert.ValidationMethod)
	}
	if cert.DeletionPolicy != "Delete" {
		t.Errorf("DeletionPolicy = %v, want Delete", cert.DeletionPolicy)
	}
	if cert.Tags == nil {
		t.Error("Tags should be initialized")
	}
}

func TestCertificate_IsIssued(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"issued", "ISSUED", true},
		{"pending", "PENDING_VALIDATION", false},
		{"failed", "FAILED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &acm.Certificate{Status: tt.status}
			if got := cert.IsIssued(); got != tt.want {
				t.Errorf("IsIssued() = %v, want %v", got, tt.want)
			}
		})
	}
}
