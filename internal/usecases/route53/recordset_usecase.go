// Package route53 implementa os casos de uso da aplicação.
//
// Contém a lógica de negócio que orquestra repositórios e aplica regras de domínio,
// atuando como camada de aplicação na Clean Architecture.
package route53


import (
	"context"
	"fmt"

	"infra-operator/internal/domain/route53"
	"infra-operator/internal/ports"
)

// RecordSetUseCase handles Route53 record set business logic
type RecordSetUseCase struct {
	repo ports.Route53Repository
}

// NewRecordSetUseCase creates a new record set use case
func NewRecordSetUseCase(repo ports.Route53Repository) *RecordSetUseCase {
	return &RecordSetUseCase{
		repo: repo,
	}
}

// SyncRecordSet creates or updates a record set
func (uc *RecordSetUseCase) SyncRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	// Set defaults
	rs.SetDefaults()

	// Validate
	if err := rs.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if record set exists
	exists, err := uc.repo.RecordSetExists(ctx, rs.HostedZoneID, rs.Name, rs.Type)
	if err != nil {
		return fmt.Errorf("failed to check record set existence: %w", err)
	}

	if exists {
		// Update existing record set
		if err := uc.repo.UpdateRecordSet(ctx, rs); err != nil {
			return fmt.Errorf("failed to update record set: %w", err)
		}
	} else {
		// Create new record set
		if err := uc.repo.CreateRecordSet(ctx, rs); err != nil {
			return fmt.Errorf("failed to create record set: %w", err)
		}
	}

	// Check change status if change ID is available
	if rs.ChangeID != "" {
		status, err := uc.repo.GetChangeStatus(ctx, rs.ChangeID)
		if err != nil {
			return fmt.Errorf("failed to get change status: %w", err)
		}
		rs.ChangeStatus = status
	}

	return nil
}

// DeleteRecordSet deletes a record set
func (uc *RecordSetUseCase) DeleteRecordSet(ctx context.Context, rs *route53.RecordSet) error {
	// Check deletion policy
	if !rs.ShouldDelete() {
		return nil // Skip deletion based on policy
	}

	// Check if record set exists
	exists, err := uc.repo.RecordSetExists(ctx, rs.HostedZoneID, rs.Name, rs.Type)
	if err != nil {
		return fmt.Errorf("failed to check record set existence: %w", err)
	}

	if !exists {
		return nil // Already deleted
	}

	// Get current record set to ensure we have all necessary data for deletion
	current, err := uc.repo.GetRecordSet(ctx, rs.HostedZoneID, rs.Name, rs.Type)
	if err != nil {
		return fmt.Errorf("failed to get record set: %w", err)
	}

	// Use current record set data for deletion
	// This ensures we have the exact record set configuration
	if err := uc.repo.DeleteRecordSet(ctx, current); err != nil {
		return fmt.Errorf("failed to delete record set: %w", err)
	}

	return nil
}
