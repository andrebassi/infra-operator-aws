// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/rds"
)

// RDSRepository defines the interface for RDS operations
type RDSRepository interface {
	Exists(ctx context.Context, dbInstanceIdentifier string) (bool, error)
	Create(ctx context.Context, instance *rds.DBInstance) error
	Get(ctx context.Context, dbInstanceIdentifier string) (*rds.DBInstance, error)
	Update(ctx context.Context, instance *rds.DBInstance) error
	Delete(ctx context.Context, dbInstanceIdentifier string, skipFinalSnapshot bool) error
	TagResource(ctx context.Context, arn string, tags map[string]string) error
}

// RDSUseCase defines the use case interface for RDS operations
type RDSUseCase interface {
	SyncDBInstance(ctx context.Context, instance *rds.DBInstance) error
	DeleteDBInstance(ctx context.Context, instance *rds.DBInstance) error
}
