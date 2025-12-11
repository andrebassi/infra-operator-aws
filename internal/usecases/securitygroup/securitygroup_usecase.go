package securitygroup

import (
	"context"
	"fmt"
	"infra-operator/internal/domain/securitygroup"
	"infra-operator/internal/ports"
)

type SecurityGroupUseCase struct {
	repo ports.SecurityGroupRepository
}

func NewSecurityGroupUseCase(repo ports.SecurityGroupRepository) *SecurityGroupUseCase {
	return &SecurityGroupUseCase{repo: repo}
}

func (uc *SecurityGroupUseCase) SyncSecurityGroup(ctx context.Context, sg *securitygroup.SecurityGroup) error {
	sg.SetDefaults()
	if err := sg.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if security group already exists
	if sg.GroupID != "" {
		exists, err := uc.repo.Exists(ctx, sg.GroupID)
		if err != nil {
			return err
		}
		if exists {
			current, err := uc.repo.Get(ctx, sg.GroupID)
			if err != nil {
				return err
			}
			// Update status with current state
			sg.GroupID = current.GroupID
			sg.GroupName = current.GroupName
			return nil
		}
	}

	// Create new security group
	return uc.repo.Create(ctx, sg)
}

func (uc *SecurityGroupUseCase) DeleteSecurityGroup(ctx context.Context, sg *securitygroup.SecurityGroup) error {
	if !sg.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, sg.GroupID)
}
