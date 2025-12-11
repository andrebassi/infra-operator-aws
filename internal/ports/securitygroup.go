// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/securitygroup"
)

// SecurityGroupRepository defines the interface for Security Group operations
type SecurityGroupRepository interface {
	Exists(ctx context.Context, groupID string) (bool, error)
	Create(ctx context.Context, sg *securitygroup.SecurityGroup) error
	Get(ctx context.Context, groupID string) (*securitygroup.SecurityGroup, error)
	Delete(ctx context.Context, groupID string) error
	AuthorizeIngress(ctx context.Context, groupID string, rules []securitygroup.Rule) error
	AuthorizeEgress(ctx context.Context, groupID string, rules []securitygroup.Rule) error
	TagResource(ctx context.Context, groupID string, tags map[string]string) error
}

// SecurityGroupUseCase defines the use case interface for Security Group operations
type SecurityGroupUseCase interface {
	SyncSecurityGroup(ctx context.Context, sg *securitygroup.SecurityGroup) error
	DeleteSecurityGroup(ctx context.Context, sg *securitygroup.SecurityGroup) error
}
