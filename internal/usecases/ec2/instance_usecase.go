package ec2

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/ec2"
	"infra-operator/internal/ports"
)

type InstanceUseCase struct {
	repo ports.EC2Repository
}

func NewInstanceUseCase(repo ports.EC2Repository) *InstanceUseCase {
	return &InstanceUseCase{repo: repo}
}

func (uc *InstanceUseCase) SyncInstance(ctx context.Context, instance *ec2.Instance) error {
	// Set defaults
	instance.SetDefaults()

	// Validate
	if err := instance.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if instance exists
	if instance.InstanceID != "" {
		exists, err := uc.repo.Exists(ctx, instance.InstanceID)
		if err != nil {
			return fmt.Errorf("failed to check if instance exists: %w", err)
		}

		if exists {
			// Get current state
			current, err := uc.repo.Get(ctx, instance.InstanceID)
			if err != nil {
				return fmt.Errorf("failed to get instance: %w", err)
			}

			// Update tags if changed
			if len(instance.Tags) > 0 {
				if err := uc.repo.TagResource(ctx, instance.InstanceID, instance.Tags); err != nil {
					return fmt.Errorf("failed to update tags: %w", err)
				}
			}

			// Copy state from current
			instance.InstanceState = current.InstanceState
			instance.PrivateIP = current.PrivateIP
			instance.PublicIP = current.PublicIP
			instance.PrivateDNS = current.PrivateDNS
			instance.PublicDNS = current.PublicDNS
			instance.AvailabilityZone = current.AvailabilityZone
			instance.LaunchTime = current.LaunchTime

			// Update last sync time
			now := time.Now()
			instance.LastSyncTime = &now

			return nil
		}
	}

	// Create new instance
	if err := uc.repo.Create(ctx, instance); err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	// Update last sync time
	now := time.Now()
	instance.LastSyncTime = &now

	return nil
}

func (uc *InstanceUseCase) DeleteInstance(ctx context.Context, instance *ec2.Instance) error {
	if instance.ShouldStop() {
		// Stop instance instead of terminating
		if !instance.IsStopped() {
			if err := uc.repo.StopInstance(ctx, instance.InstanceID); err != nil {
				return fmt.Errorf("failed to stop instance: %w", err)
			}
		}
		return nil
	}

	if !instance.ShouldDelete() {
		// Retain policy - don't delete
		return nil
	}

	// Terminate instance
	if err := uc.repo.TerminateInstance(ctx, instance.InstanceID); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}
