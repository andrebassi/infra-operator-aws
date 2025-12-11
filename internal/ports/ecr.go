// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/ecr"
)

// ECRRepository defines the interface for ECR operations
type ECRRepository interface {
	Exists(ctx context.Context, repositoryName string) (bool, error)
	Create(ctx context.Context, repository *ecr.Repository) error
	Get(ctx context.Context, repositoryName string) (*ecr.Repository, error)
	UpdateImageScanning(ctx context.Context, repositoryName string, scanOnPush bool) error
	UpdateImageTagMutability(ctx context.Context, repositoryName string, mutability string) error
	PutLifecyclePolicy(ctx context.Context, repositoryName, policyText string) error
	Delete(ctx context.Context, repositoryName string) error
	TagResource(ctx context.Context, arn string, tags map[string]string) error
	GetImageCount(ctx context.Context, repositoryName string) (int64, error)
}

// ECRUseCase defines the use case interface for ECR operations
type ECRUseCase interface {
	SyncRepository(ctx context.Context, repository *ecr.Repository) error
	DeleteRepository(ctx context.Context, repository *ecr.Repository) error
}
