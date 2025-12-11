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

// HostedZoneUseCase handles Route53 hosted zone business logic
type HostedZoneUseCase struct {
	repo ports.Route53Repository
}

// NewHostedZoneUseCase creates a new hosted zone use case
func NewHostedZoneUseCase(repo ports.Route53Repository) *HostedZoneUseCase {
	return &HostedZoneUseCase{
		repo: repo,
	}
}

// SyncHostedZone creates or updates a hosted zone
func (uc *HostedZoneUseCase) SyncHostedZone(ctx context.Context, hz *route53.HostedZone) error {
	// Set defaults
	hz.SetDefaults()

	// Validate
	if err := hz.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// If already created, get current state
	if hz.HostedZoneID != "" {
		exists, err := uc.repo.HostedZoneExists(ctx, hz.HostedZoneID)
		if err != nil {
			return fmt.Errorf("failed to check hosted zone existence: %w", err)
		}

		if exists {
			// Get current hosted zone state
			current, err := uc.repo.GetHostedZone(ctx, hz.HostedZoneID)
			if err != nil {
				return fmt.Errorf("failed to get hosted zone: %w", err)
			}

			// Update hosted zone with current state
			hz.NameServers = current.NameServers
			hz.ResourceRecordSetCount = current.ResourceRecordSetCount

			// Update tags if provided
			if len(hz.Tags) > 0 {
				if err := uc.repo.TagHostedZone(ctx, hz.HostedZoneID, hz.Tags); err != nil {
					return fmt.Errorf("failed to update tags: %w", err)
				}
			}

			return nil
		}
	}

	// Create new hosted zone
	if err := uc.repo.CreateHostedZone(ctx, hz); err != nil {
		return fmt.Errorf("failed to create hosted zone: %w", err)
	}

	return nil
}

// DeleteHostedZone deletes a hosted zone
func (uc *HostedZoneUseCase) DeleteHostedZone(ctx context.Context, hz *route53.HostedZone) error {
	// Check deletion policy
	if !hz.ShouldDelete() {
		return nil // Skip deletion based on policy
	}

	// Skip if not created
	if hz.HostedZoneID == "" {
		return nil
	}

	// Delete the hosted zone
	if err := uc.repo.DeleteHostedZone(ctx, hz.HostedZoneID); err != nil {
		return fmt.Errorf("failed to delete hosted zone: %w", err)
	}

	return nil
}
