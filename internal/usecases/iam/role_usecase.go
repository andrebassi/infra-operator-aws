package iam

import (
	"context"
	"fmt"
	"time"

	"infra-operator/internal/domain/iam"
	"infra-operator/internal/ports"
)

type RoleUseCase struct {
	repo ports.IAMRepository
}

func NewRoleUseCase(repo ports.IAMRepository) *RoleUseCase {
	return &RoleUseCase{
		repo: repo,
	}
}

// SyncRole synchronizes an IAM role with AWS
func (uc *RoleUseCase) SyncRole(ctx context.Context, role *iam.Role) error {
	// Set defaults and validate
	role.SetDefaults()
	if err := role.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if role exists
	exists, err := uc.repo.Exists(ctx, role.RoleName)
	if err != nil {
		return fmt.Errorf("failed to check if role exists: %w", err)
	}

	if !exists {
		// Create new role
		if err := uc.repo.Create(ctx, role); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}

		// Attach managed policies
		for _, policyArn := range role.ManagedPolicyArns {
			if err := uc.repo.AttachManagedPolicy(ctx, role.RoleName, policyArn); err != nil {
				return fmt.Errorf("failed to attach managed policy %s: %w", policyArn, err)
			}
		}

		// Add inline policy
		if role.InlinePolicyName != "" && role.InlinePolicyDocument != "" {
			if err := uc.repo.PutInlinePolicy(ctx, role.RoleName, role.InlinePolicyName, role.InlinePolicyDocument); err != nil {
				return fmt.Errorf("failed to put inline policy: %w", err)
			}
		}

		// Tag role
		if len(role.Tags) > 0 {
			if err := uc.repo.TagResource(ctx, role.RoleName, role.Tags); err != nil {
				return fmt.Errorf("failed to tag role: %w", err)
			}
		}
	} else {
		// Update existing role
		if err := uc.repo.Update(ctx, role); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}

		// Sync managed policies
		if err := uc.syncManagedPolicies(ctx, role); err != nil {
			return fmt.Errorf("failed to sync managed policies: %w", err)
		}

		// Update inline policy
		if role.InlinePolicyName != "" && role.InlinePolicyDocument != "" {
			if err := uc.repo.PutInlinePolicy(ctx, role.RoleName, role.InlinePolicyName, role.InlinePolicyDocument); err != nil {
				return fmt.Errorf("failed to update inline policy: %w", err)
			}
		}
	}

	// Get updated role info
	updatedRole, err := uc.repo.Get(ctx, role.RoleName)
	if err != nil {
		return fmt.Errorf("failed to get updated role: %w", err)
	}

	// Update role with AWS values
	role.RoleArn = updatedRole.RoleArn
	role.RoleId = updatedRole.RoleId
	role.CreatedAt = updatedRole.CreatedAt

	now := time.Now()
	role.LastSyncTime = &now

	return nil
}

// syncManagedPolicies synchronizes managed policies
func (uc *RoleUseCase) syncManagedPolicies(ctx context.Context, role *iam.Role) error {
	// Get currently attached policies
	attachedPolicies, err := uc.repo.ListAttachedPolicies(ctx, role.RoleName)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	// Create maps for easy lookup
	desiredPolicies := make(map[string]bool)
	for _, arn := range role.ManagedPolicyArns {
		desiredPolicies[arn] = true
	}

	currentPolicies := make(map[string]bool)
	for _, arn := range attachedPolicies {
		currentPolicies[arn] = true
	}

	// Attach new policies
	for _, arn := range role.ManagedPolicyArns {
		if !currentPolicies[arn] {
			if err := uc.repo.AttachManagedPolicy(ctx, role.RoleName, arn); err != nil {
				return fmt.Errorf("failed to attach policy %s: %w", arn, err)
			}
		}
	}

	// Detach removed policies
	for _, arn := range attachedPolicies {
		if !desiredPolicies[arn] {
			if err := uc.repo.DetachManagedPolicy(ctx, role.RoleName, arn); err != nil {
				return fmt.Errorf("failed to detach policy %s: %w", arn, err)
			}
		}
	}

	return nil
}

// DeleteRole deletes an IAM role and its associated policies
func (uc *RoleUseCase) DeleteRole(ctx context.Context, role *iam.Role) error {
	// Check if role should be deleted
	if !role.ShouldDelete() {
		return nil
	}

	// Check if role exists
	exists, err := uc.repo.Exists(ctx, role.RoleName)
	if err != nil {
		return fmt.Errorf("failed to check if role exists: %w", err)
	}

	if !exists {
		return nil // Already deleted
	}

	// Detach all managed policies
	attachedPolicies, err := uc.repo.ListAttachedPolicies(ctx, role.RoleName)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	for _, policyArn := range attachedPolicies {
		if err := uc.repo.DetachManagedPolicy(ctx, role.RoleName, policyArn); err != nil {
			return fmt.Errorf("failed to detach policy %s: %w", policyArn, err)
		}
	}

	// Delete inline policy if it exists
	if role.InlinePolicyName != "" {
		if err := uc.repo.DeleteInlinePolicy(ctx, role.RoleName, role.InlinePolicyName); err != nil {
			// Ignore error if policy doesn't exist
			return fmt.Errorf("failed to delete inline policy: %w", err)
		}
	}

	// Delete the role
	if err := uc.repo.Delete(ctx, role.RoleName); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}
