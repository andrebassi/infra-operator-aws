package acm

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/acm"
	"infra-operator/internal/ports"
)

type CertificateUseCase struct {
	repo ports.ACMRepository
}

func NewCertificateUseCase(repo ports.ACMRepository) *CertificateUseCase {
	return &CertificateUseCase{repo: repo}
}

func (uc *CertificateUseCase) SyncCertificate(ctx context.Context, cert *acm.Certificate) error {
	cert.SetDefaults()
	if err := cert.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if cert.CertificateARN != "" {
		current, err := uc.repo.Describe(ctx, cert.CertificateARN)
		if err != nil {
			return fmt.Errorf("failed to describe certificate: %w", err)
		}
		cert.Status = current.Status
		cert.ValidationRecords = current.ValidationRecords
		return nil
	}

	if err := uc.repo.Request(ctx, cert); err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	return nil
}

func (uc *CertificateUseCase) DeleteCertificate(ctx context.Context, cert *acm.Certificate) error {
	if !cert.ShouldDelete() || cert.CertificateARN == "" {
		return nil
	}
	return uc.repo.Delete(ctx, cert.CertificateARN)
}
