// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/kms"
)

// KMSRepository defines the interface for KMS operations
type KMSRepository interface {
	Exists(ctx context.Context, keyId string) (bool, error)
	Create(ctx context.Context, key *kms.Key) error
	Get(ctx context.Context, keyId string) (*kms.Key, error)
	Update(ctx context.Context, key *kms.Key) error
	EnableKey(ctx context.Context, keyId string) error
	DisableKey(ctx context.Context, keyId string) error
	EnableKeyRotation(ctx context.Context, keyId string) error
	DisableKeyRotation(ctx context.Context, keyId string) error
	ScheduleKeyDeletion(ctx context.Context, keyId string, pendingWindowInDays int32) error
	CancelKeyDeletion(ctx context.Context, keyId string) error
	TagResource(ctx context.Context, keyId string, tags map[string]string) error
}

// KMSUseCase defines the use case interface for KMS operations
type KMSUseCase interface {
	SyncKey(ctx context.Context, key *kms.Key) error
	DeleteKey(ctx context.Context, key *kms.Key) error
}
