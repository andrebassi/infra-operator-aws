// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/secretsmanager"
)

// SecretsManagerRepository defines the interface for Secrets Manager operations
type SecretsManagerRepository interface {
	Exists(ctx context.Context, secretName string) (bool, error)
	Create(ctx context.Context, secret *secretsmanager.Secret) error
	Get(ctx context.Context, secretName string) (*secretsmanager.Secret, error)
	UpdateSecretValue(ctx context.Context, secretName, secretString string, secretBinary []byte) (string, error)
	UpdateRotation(ctx context.Context, secretName, lambdaARN string, days int32) error
	DisableRotation(ctx context.Context, secretName string) error
	Delete(ctx context.Context, secretName string, recoveryWindowInDays int32, forceDelete bool) error
	TagResource(ctx context.Context, secretARN string, tags map[string]string) error
}

// SecretsManagerUseCase defines the use case interface for Secrets Manager operations
type SecretsManagerUseCase interface {
	SyncSecret(ctx context.Context, secret *secretsmanager.Secret) error
	DeleteSecret(ctx context.Context, secret *secretsmanager.Secret) error
}
