package lambda

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/lambda"
	"infra-operator/internal/ports"
)

// FunctionUseCase implements business logic for Lambda functions
type FunctionUseCase struct {
	repo ports.LambdaRepository
}

// NewFunctionUseCase creates a new Lambda function use case
func NewFunctionUseCase(repo ports.LambdaRepository) ports.LambdaUseCase {
	return &FunctionUseCase{
		repo: repo,
	}
}

// SyncFunction creates or updates a Lambda function idempotently
func (uc *FunctionUseCase) SyncFunction(ctx context.Context, function *lambda.Function) error {
	// Validate function configuration
	if err := function.Validate(); err != nil {
		return fmt.Errorf("invalid function configuration: %w", err)
	}

	// Set defaults for optional fields
	function.SetDefaults()

	// Check if function exists
	exists, err := uc.repo.Exists(ctx, function.Name)
	if err != nil {
		return fmt.Errorf("failed to check function existence: %w", err)
	}

	if !exists {
		// Create new function
		if err := uc.repo.Create(ctx, function); err != nil {
			return fmt.Errorf("failed to create function: %w", err)
		}

		// Tag the function
		if len(function.Tags) > 0 && function.ARN != "" {
			if err := uc.repo.TagFunction(ctx, function.ARN, function.Tags); err != nil {
				return fmt.Errorf("failed to tag function: %w", err)
			}
		}

		return nil
	}

	// Function exists - get current state
	currentFunction, err := uc.repo.Get(ctx, function.Name)
	if err != nil {
		return fmt.Errorf("failed to get current function: %w", err)
	}

	// Update function ARN from existing function
	function.ARN = currentFunction.ARN

	// Check if code needs updating
	if uc.codeNeedsUpdate(function.Code, currentFunction.Code) {
		if err := uc.repo.UpdateCode(ctx, function); err != nil {
			return fmt.Errorf("failed to update function code: %w", err)
		}
	}

	// Check if configuration needs updating
	if uc.configNeedsUpdate(function, currentFunction) {
		if err := uc.repo.UpdateConfiguration(ctx, function); err != nil {
			return fmt.Errorf("failed to update function configuration: %w", err)
		}
	}

	// Sync tags
	if err := uc.syncTags(ctx, function, currentFunction); err != nil {
		return fmt.Errorf("failed to sync tags: %w", err)
	}

	// Get updated function state
	updatedFunction, err := uc.repo.Get(ctx, function.Name)
	if err != nil {
		return fmt.Errorf("failed to get updated function: %w", err)
	}

	// Update function with latest state
	function.State = updatedFunction.State
	function.StateReason = updatedFunction.StateReason
	function.LastModified = updatedFunction.LastModified
	function.Version = updatedFunction.Version
	function.CodeSize = updatedFunction.CodeSize

	return nil
}

// DeleteFunction deletes a Lambda function
func (uc *FunctionUseCase) DeleteFunction(ctx context.Context, function *lambda.Function) error {
	// Check if function exists
	exists, err := uc.repo.Exists(ctx, function.Name)
	if err != nil {
		return fmt.Errorf("failed to check function existence: %w", err)
	}

	if !exists {
		// Function doesn't exist, nothing to do
		return nil
	}

	// Delete the function
	if err := uc.repo.Delete(ctx, function.Name); err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}

	return nil
}

// Helper methods

func (uc *FunctionUseCase) codeNeedsUpdate(desired, current lambda.Code) bool {
	// Compare code sources - if any source is different, update is needed
	if desired.ZipFile != "" && desired.ZipFile != current.ZipFile {
		return true
	}
	if desired.S3Bucket != "" && (desired.S3Bucket != current.S3Bucket || desired.S3Key != current.S3Key) {
		return true
	}
	if desired.ImageUri != "" && desired.ImageUri != current.ImageUri {
		return true
	}
	return false
}

func (uc *FunctionUseCase) configNeedsUpdate(desired, current *lambda.Function) bool {
	// Check if any configuration field has changed
	if desired.Runtime != current.Runtime {
		return true
	}
	if desired.Handler != current.Handler {
		return true
	}
	if desired.Role != current.Role {
		return true
	}
	if desired.Timeout != current.Timeout {
		return true
	}
	if desired.MemorySize != current.MemorySize {
		return true
	}
	if desired.Description != current.Description {
		return true
	}

	// Check environment variables
	if !mapsEqual(desired.Environment, current.Environment) {
		return true
	}

	// Check layers
	if !slicesEqual(desired.Layers, current.Layers) {
		return true
	}

	// Check VPC config
	if !vpcConfigEqual(desired.VpcConfig, current.VpcConfig) {
		return true
	}

	return false
}

func (uc *FunctionUseCase) syncTags(ctx context.Context, desired, current *lambda.Function) error {
	if desired.ARN == "" {
		return nil
	}

	// Find tags to add or update
	tagsToAdd := make(map[string]string)
	for key, value := range desired.Tags {
		if currentValue, exists := current.Tags[key]; !exists || currentValue != value {
			tagsToAdd[key] = value
		}
	}

	// Find tags to remove
	var tagsToRemove []string
	for key := range current.Tags {
		if _, exists := desired.Tags[key]; !exists {
			tagsToRemove = append(tagsToRemove, key)
		}
	}

	// Add/update tags
	if len(tagsToAdd) > 0 {
		if err := uc.repo.TagFunction(ctx, desired.ARN, tagsToAdd); err != nil {
			return err
		}
	}

	// Remove tags
	if len(tagsToRemove) > 0 {
		if err := uc.repo.UntagFunction(ctx, desired.ARN, tagsToRemove); err != nil {
			return err
		}
	}

	return nil
}

// Utility functions

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func vpcConfigEqual(a, b *lambda.VpcConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return slicesEqual(a.SecurityGroupIds, b.SecurityGroupIds) &&
		slicesEqual(a.SubnetIds, b.SubnetIds)
}
