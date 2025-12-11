package subnet

import (
	"context"
	"fmt"
	"infra-operator/internal/domain/subnet"
	"infra-operator/internal/ports"
)

type SubnetUseCase struct {
	repo ports.SubnetRepository
}

func NewSubnetUseCase(repo ports.SubnetRepository) *SubnetUseCase {
	return &SubnetUseCase{repo: repo}
}

func (uc *SubnetUseCase) SyncSubnet(ctx context.Context, s *subnet.Subnet) error {
	s.SetDefaults()
	if err := s.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if s.SubnetID != "" {
		exists, err := uc.repo.Exists(ctx, s.SubnetID)
		if err != nil {
			return err
		}
		if exists {
			current, err := uc.repo.Get(ctx, s.SubnetID)
			if err != nil {
				return err
			}
			s.State = current.State
			s.AvailabilityZone = current.AvailabilityZone
			s.AvailableIpAddressCount = current.AvailableIpAddressCount

			// Always apply tags (ensures Name tag and any updates are applied)
			if len(s.Tags) > 0 {
				uc.repo.TagResource(ctx, s.SubnetID, s.Tags)
			}
			return nil
		}
	}

	return uc.repo.Create(ctx, s)
}

func (uc *SubnetUseCase) DeleteSubnet(ctx context.Context, s *subnet.Subnet) error {
	if !s.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, s.SubnetID)
}
