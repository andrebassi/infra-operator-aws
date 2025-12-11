package nlb

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/nlb"
	"infra-operator/internal/ports"
)

// LoadBalancerUseCase handles NLB business logic
type LoadBalancerUseCase struct {
	repo ports.NLBRepository
}

// NewLoadBalancerUseCase creates a new NLB use case
func NewLoadBalancerUseCase(repo ports.NLBRepository) *LoadBalancerUseCase {
	return &LoadBalancerUseCase{repo: repo}
}

// SyncLoadBalancer creates or updates an NLB
func (uc *LoadBalancerUseCase) SyncLoadBalancer(ctx context.Context, lb *nlb.LoadBalancer) error {
	lb.SetDefaults()

	if err := lb.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	exists, err := uc.repo.Exists(ctx, lb.LoadBalancerName)
	if err != nil {
		return fmt.Errorf("failed to check NLB existence: %w", err)
	}

	if exists {
		current, err := uc.repo.Get(ctx, lb.LoadBalancerName)
		if err != nil {
			return fmt.Errorf("failed to get NLB: %w", err)
		}

		lb.LoadBalancerARN = current.LoadBalancerARN
		lb.DNSName = current.DNSName
		lb.State = current.State
		lb.VpcID = current.VpcID
		lb.CanonicalHostedZoneID = current.CanonicalHostedZoneID

		if err := uc.repo.SetAttributes(ctx, lb); err != nil {
			return fmt.Errorf("failed to update NLB attributes: %w", err)
		}

		if len(lb.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, lb.LoadBalancerARN, lb.Tags); err != nil {
				return fmt.Errorf("failed to update NLB tags: %w", err)
			}
		}

		return nil
	}

	if err := uc.repo.Create(ctx, lb); err != nil {
		return fmt.Errorf("failed to create NLB: %w", err)
	}

	return nil
}

// DeleteLoadBalancer deletes an NLB
func (uc *LoadBalancerUseCase) DeleteLoadBalancer(ctx context.Context, lb *nlb.LoadBalancer) error {
	if !lb.ShouldDelete() {
		return nil
	}

	if lb.LoadBalancerARN == "" {
		return nil
	}

	if err := uc.repo.Delete(ctx, lb.LoadBalancerARN); err != nil {
		return fmt.Errorf("failed to delete NLB: %w", err)
	}

	return nil
}
