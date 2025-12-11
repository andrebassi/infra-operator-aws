package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsiam "github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"infra-operator/internal/domain/iam"
)

type Repository struct {
	client *awsiam.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsiam.NewFromConfig(cfg),
	}
}

// Exists checks if a role exists
func (r *Repository) Exists(ctx context.Context, roleName string) (bool, error) {
	_, err := r.client.GetRole(ctx, &awsiam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		var nfe *types.NoSuchEntityException
		if errors.As(err, &nfe) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if role exists: %w", err)
	}
	return true, nil
}

// Create creates a new IAM role
func (r *Repository) Create(ctx context.Context, role *iam.Role) error {
	input := &awsiam.CreateRoleInput{
		RoleName:                 aws.String(role.RoleName),
		AssumeRolePolicyDocument: aws.String(role.AssumeRolePolicyDocument),
		Path:                     aws.String(role.Path),
	}

	if role.Description != "" {
		input.Description = aws.String(role.Description)
	}

	if role.MaxSessionDuration > 0 {
		input.MaxSessionDuration = aws.Int32(role.MaxSessionDuration)
	}

	if role.PermissionsBoundary != "" {
		input.PermissionsBoundary = aws.String(role.PermissionsBoundary)
	}

	if len(role.Tags) > 0 {
		input.Tags = convertTags(role.Tags)
	}

	output, err := r.client.CreateRole(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// Update role with AWS-assigned values
	if output.Role != nil {
		role.RoleArn = aws.ToString(output.Role.Arn)
		role.RoleId = aws.ToString(output.Role.RoleId)
		if output.Role.CreateDate != nil {
			t := *output.Role.CreateDate
			role.CreatedAt = &t
		}
	}

	return nil
}

// Get retrieves a role
func (r *Repository) Get(ctx context.Context, roleName string) (*iam.Role, error) {
	output, err := r.client.GetRole(ctx, &awsiam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	role := &iam.Role{
		RoleName:                 aws.ToString(output.Role.RoleName),
		RoleArn:                  aws.ToString(output.Role.Arn),
		RoleId:                   aws.ToString(output.Role.RoleId),
		Description:              aws.ToString(output.Role.Description),
		AssumeRolePolicyDocument: aws.ToString(output.Role.AssumeRolePolicyDocument),
		MaxSessionDuration:       aws.ToInt32(output.Role.MaxSessionDuration),
		Path:                     aws.ToString(output.Role.Path),
	}

	if output.Role.PermissionsBoundary != nil {
		role.PermissionsBoundary = aws.ToString(output.Role.PermissionsBoundary.PermissionsBoundaryArn)
	}

	if output.Role.CreateDate != nil {
		t := *output.Role.CreateDate
		role.CreatedAt = &t
	}

	return role, nil
}

// Update updates a role
func (r *Repository) Update(ctx context.Context, role *iam.Role) error {
	// Update description
	if role.Description != "" {
		_, err := r.client.UpdateRole(ctx, &awsiam.UpdateRoleInput{
			RoleName:    aws.String(role.RoleName),
			Description: aws.String(role.Description),
		})
		if err != nil {
			return fmt.Errorf("failed to update role description: %w", err)
		}
	}

	// Update max session duration
	if role.MaxSessionDuration > 0 {
		_, err := r.client.UpdateRole(ctx, &awsiam.UpdateRoleInput{
			RoleName:           aws.String(role.RoleName),
			MaxSessionDuration: aws.Int32(role.MaxSessionDuration),
		})
		if err != nil {
			return fmt.Errorf("failed to update max session duration: %w", err)
		}
	}

	// Update assume role policy
	_, err := r.client.UpdateAssumeRolePolicy(ctx, &awsiam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(role.RoleName),
		PolicyDocument: aws.String(role.AssumeRolePolicyDocument),
	})
	if err != nil {
		return fmt.Errorf("failed to update assume role policy: %w", err)
	}

	return nil
}

// Delete deletes a role
func (r *Repository) Delete(ctx context.Context, roleName string) error {
	_, err := r.client.DeleteRole(ctx, &awsiam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}

// AttachManagedPolicy attaches a managed policy to the role
func (r *Repository) AttachManagedPolicy(ctx context.Context, roleName, policyArn string) error {
	_, err := r.client.AttachRolePolicy(ctx, &awsiam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return fmt.Errorf("failed to attach managed policy: %w", err)
	}
	return nil
}

// DetachManagedPolicy detaches a managed policy from the role
func (r *Repository) DetachManagedPolicy(ctx context.Context, roleName, policyArn string) error {
	_, err := r.client.DetachRolePolicy(ctx, &awsiam.DetachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return fmt.Errorf("failed to detach managed policy: %w", err)
	}
	return nil
}

// ListAttachedPolicies lists all attached managed policies
func (r *Repository) ListAttachedPolicies(ctx context.Context, roleName string) ([]string, error) {
	output, err := r.client.ListAttachedRolePolicies(ctx, &awsiam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list attached policies: %w", err)
	}

	policyArns := make([]string, 0, len(output.AttachedPolicies))
	for _, policy := range output.AttachedPolicies {
		policyArns = append(policyArns, aws.ToString(policy.PolicyArn))
	}

	return policyArns, nil
}

// PutInlinePolicy creates or updates an inline policy
func (r *Repository) PutInlinePolicy(ctx context.Context, roleName, policyName, policyDocument string) error {
	_, err := r.client.PutRolePolicy(ctx, &awsiam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(policyDocument),
	})
	if err != nil {
		return fmt.Errorf("failed to put inline policy: %w", err)
	}
	return nil
}

// DeleteInlinePolicy deletes an inline policy
func (r *Repository) DeleteInlinePolicy(ctx context.Context, roleName, policyName string) error {
	_, err := r.client.DeleteRolePolicy(ctx, &awsiam.DeleteRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete inline policy: %w", err)
	}
	return nil
}

// TagResource tags a role
func (r *Repository) TagResource(ctx context.Context, roleName string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagRole(ctx, &awsiam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag role: %w", err)
	}
	return nil
}

// convertTags converts a map of tags to AWS IAM tags
func convertTags(tags map[string]string) []types.Tag {
	iamTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		iamTags = append(iamTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return iamTags
}
