package alb

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/alb"
	"infra-operator/internal/ports"
)

type LoadBalancerUseCase struct {
	repo ports.ALBRepository
}

func NewLoadBalancerUseCase(repo ports.ALBRepository) *LoadBalancerUseCase {
	return &LoadBalancerUseCase{
		repo: repo,
	}
}

// SyncLoadBalancer creates or updates a load balancer
func (uc *LoadBalancerUseCase) SyncLoadBalancer(ctx context.Context, lb *alb.LoadBalancer) error {
	// Set defaults
	lb.SetDefaults()

	// Validate
	if err := lb.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if load balancer exists
	exists, err := uc.repo.Exists(ctx, lb.LoadBalancerName)
	if err != nil {
		return fmt.Errorf("failed to check load balancer existence: %w", err)
	}

	if exists {
		// Get current state
		current, err := uc.repo.Get(ctx, lb.LoadBalancerName)
		if err != nil {
			return fmt.Errorf("failed to get load balancer: %w", err)
		}

		// Update domain object with current state
		lb.LoadBalancerARN = current.LoadBalancerARN
		lb.DNSName = current.DNSName
		lb.State = current.State
		lb.VpcID = current.VpcID
		lb.CanonicalHostedZoneID = current.CanonicalHostedZoneID

		// Update attributes if changed
		if err := uc.repo.SetAttributes(ctx, lb); err != nil {
			return fmt.Errorf("failed to update load balancer attributes: %w", err)
		}

		// Update tags if provided
		if len(lb.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, lb.LoadBalancerARN, lb.Tags); err != nil {
				return fmt.Errorf("failed to update load balancer tags: %w", err)
			}
		}

		return nil
	}

	// Create new load balancer
	if err := uc.repo.Create(ctx, lb); err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	return nil
}

// DeleteLoadBalancer deletes a load balancer
func (uc *LoadBalancerUseCase) DeleteLoadBalancer(ctx context.Context, lb *alb.LoadBalancer) error {
	if !lb.ShouldDelete() {
		return nil
	}

	if lb.LoadBalancerARN == "" {
		// Already deleted or never created
		return nil
	}

	if err := uc.repo.Delete(ctx, lb.LoadBalancerARN); err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	return nil
}
