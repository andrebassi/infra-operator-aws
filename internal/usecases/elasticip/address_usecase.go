package elasticip

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/elasticip"
	"infra-operator/internal/ports"
)

// AddressUseCase handles Elastic IP business logic
type AddressUseCase struct {
	repo ports.ElasticIPRepository
}

// NewAddressUseCase creates a new Elastic IP use case
func NewAddressUseCase(repo ports.ElasticIPRepository) *AddressUseCase {
	return &AddressUseCase{
		repo: repo,
	}
}

// SyncAddress allocates or updates an Elastic IP
func (uc *AddressUseCase) SyncAddress(ctx context.Context, addr *elasticip.Address) error {
	// Set defaults
	addr.SetDefaults()

	// Validate
	if err := addr.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// If already allocated, get current state
	if addr.AllocationID != "" {
		exists, err := uc.repo.Exists(ctx, addr.AllocationID)
		if err != nil {
			return fmt.Errorf("failed to check EIP existence: %w", err)
		}

		if exists {
			// Get current EIP state
			current, err := uc.repo.Get(ctx, addr.AllocationID)
			if err != nil {
				return fmt.Errorf("failed to get EIP: %w", err)
			}

			// Update address with current state
			addr.PublicIP = current.PublicIP
			addr.AssociationID = current.AssociationID
			addr.InstanceID = current.InstanceID
			addr.NetworkInterfaceID = current.NetworkInterfaceID
			addr.PrivateIPAddress = current.PrivateIPAddress

			// Update tags if provided
			if len(addr.Tags) > 0 {
				if err := uc.repo.TagResource(ctx, addr.AllocationID, addr.Tags); err != nil {
					return fmt.Errorf("failed to update tags: %w", err)
				}
			}

			return nil
		}
	}

	// Allocate new Elastic IP
	if err := uc.repo.Allocate(ctx, addr); err != nil {
		return fmt.Errorf("failed to allocate EIP: %w", err)
	}

	return nil
}

// ReleaseAddress releases an Elastic IP
func (uc *AddressUseCase) ReleaseAddress(ctx context.Context, addr *elasticip.Address) error {
	// Check deletion policy
	if !addr.ShouldDelete() {
		return nil // Skip deletion based on policy
	}

	// Skip if not allocated
	if addr.AllocationID == "" {
		return nil
	}

	// Release the Elastic IP
	if err := uc.repo.Release(ctx, addr.AllocationID); err != nil {
		return fmt.Errorf("failed to release EIP: %w", err)
	}

	return nil
}
