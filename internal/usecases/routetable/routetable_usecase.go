package routetable

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/routetable"
	"infra-operator/internal/ports"
)

type RouteTableUseCase struct {
	repo ports.RouteTableRepository
}

func NewRouteTableUseCase(repo ports.RouteTableRepository) *RouteTableUseCase {
	return &RouteTableUseCase{repo: repo}
}

func (uc *RouteTableUseCase) SyncRouteTable(ctx context.Context, rt *routetable.RouteTable) error {
	rt.SetDefaults()

	if err := rt.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if rt.RouteTableID != "" {
		exists, err := uc.repo.Exists(ctx, rt.RouteTableID)
		if err != nil {
			return fmt.Errorf("failed to check route table existence: %w", err)
		}

		if exists {
			// Get current state
			current, err := uc.repo.Get(ctx, rt.RouteTableID)
			if err != nil {
				return fmt.Errorf("failed to get route table: %w", err)
			}

			// Update with current state
			rt.RouteTableID = current.RouteTableID
			rt.AssociatedSubnets = current.AssociatedSubnets
			rt.LastSyncTime = current.LastSyncTime

			return nil
		}
	}

	// Create new route table
	if err := uc.repo.Create(ctx, rt); err != nil {
		return fmt.Errorf("failed to create route table: %w", err)
	}

	return nil
}

func (uc *RouteTableUseCase) DeleteRouteTable(ctx context.Context, rt *routetable.RouteTable) error {
	if !rt.ShouldDelete() {
		return nil
	}

	if rt.RouteTableID == "" {
		return nil
	}

	exists, err := uc.repo.Exists(ctx, rt.RouteTableID)
	if err != nil {
		return fmt.Errorf("failed to check route table existence: %w", err)
	}

	if !exists {
		return nil
	}

	if err := uc.repo.Delete(ctx, rt.RouteTableID); err != nil {
		return fmt.Errorf("failed to delete route table: %w", err)
	}

	return nil
}
