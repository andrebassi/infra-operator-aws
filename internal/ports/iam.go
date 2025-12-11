// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/iam"
)

// IAMRepository defines the interface for IAM operations
type IAMRepository interface {
	Exists(ctx context.Context, roleName string) (bool, error)
	Create(ctx context.Context, role *iam.Role) error
	Get(ctx context.Context, roleName string) (*iam.Role, error)
	Update(ctx context.Context, role *iam.Role) error
	Delete(ctx context.Context, roleName string) error
	AttachManagedPolicy(ctx context.Context, roleName, policyArn string) error
	DetachManagedPolicy(ctx context.Context, roleName, policyArn string) error
	ListAttachedPolicies(ctx context.Context, roleName string) ([]string, error)
	PutInlinePolicy(ctx context.Context, roleName, policyName, policyDocument string) error
	DeleteInlinePolicy(ctx context.Context, roleName, policyName string) error
	TagResource(ctx context.Context, roleName string, tags map[string]string) error
}

// IAMUseCase defines the use case interface for IAM operations
type IAMUseCase interface {
	SyncRole(ctx context.Context, role *iam.Role) error
	DeleteRole(ctx context.Context, role *iam.Role) error
}
