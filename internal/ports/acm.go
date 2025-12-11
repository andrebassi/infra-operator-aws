// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/acm"
)

// ACMRepository defines the interface for ACM Certificate operations
type ACMRepository interface {
	Request(ctx context.Context, cert *acm.Certificate) error
	Describe(ctx context.Context, certARN string) (*acm.Certificate, error)
	Delete(ctx context.Context, certARN string) error
}

// ACMUseCase defines the use case interface for ACM operations
type ACMUseCase interface {
	SyncCertificate(ctx context.Context, cert *acm.Certificate) error
	DeleteCertificate(ctx context.Context, cert *acm.Certificate) error
}
