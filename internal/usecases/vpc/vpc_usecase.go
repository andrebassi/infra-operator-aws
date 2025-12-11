package vpc

import (
	"context"
	"fmt"
	"infra-operator/internal/domain/vpc"
	"infra-operator/internal/ports"
)

type VPCUseCase struct {
	repo ports.VPCRepository
}

func NewVPCUseCase(repo ports.VPCRepository) *VPCUseCase {
	return &VPCUseCase{repo: repo}
}

func (uc *VPCUseCase) SyncVPC(ctx context.Context, v *vpc.VPC) error {
	v.SetDefaults()
	if err := v.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if v.VpcID != "" {
		exists, err := uc.repo.Exists(ctx, v.VpcID)
		if err != nil {
			return err
		}
		if exists {
			current, err := uc.repo.Get(ctx, v.VpcID)
			if err != nil {
				return err
			}
			v.State = current.State
			v.IsDefault = current.IsDefault

			// Always apply tags (ensures Name tag and any updates are applied)
			if len(v.Tags) > 0 {
				uc.repo.TagResource(ctx, v.VpcID, v.Tags)
			}
			return nil
		}
	}

	return uc.repo.Create(ctx, v)
}

func (uc *VPCUseCase) DeleteVPC(ctx context.Context, v *vpc.VPC) error {
	if !v.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, v.VpcID)
}
