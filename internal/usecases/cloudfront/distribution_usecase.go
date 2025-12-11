package cloudfront

import (
	"context"

	"infra-operator/internal/domain/cloudfront"
	"infra-operator/internal/ports"
)

type DistributionUseCase struct {
	repo ports.CloudFrontRepository
}

func NewDistributionUseCase(repo ports.CloudFrontRepository) *DistributionUseCase {
	return &DistributionUseCase{repo: repo}
}

func (uc *DistributionUseCase) SyncDistribution(ctx context.Context, dist *cloudfront.Distribution) error {
	dist.SetDefaults()

	if err := dist.Validate(); err != nil {
		return err
	}

	if dist.DistributionID == "" {
		return uc.repo.Create(ctx, dist)
	}

	exists, err := uc.repo.Exists(ctx, dist.DistributionID)
	if err != nil {
		return err
	}

	if !exists {
		dist.DistributionID = ""
		return uc.repo.Create(ctx, dist)
	}

	return uc.repo.Update(ctx, dist)
}

func (uc *DistributionUseCase) DeleteDistribution(ctx context.Context, dist *cloudfront.Distribution) error {
	if dist.DeletionPolicy == cloudfront.DeletionPolicyRetain ||
	   dist.DeletionPolicy == cloudfront.DeletionPolicyOrphan {
		return nil
	}

	if dist.DistributionID != "" {
		return uc.repo.Delete(ctx, dist.DistributionID, dist.ETag)
	}

	return nil
}
