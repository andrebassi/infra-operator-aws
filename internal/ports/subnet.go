// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/subnet"
)

// SubnetRepository defines the interface for Subnet operations
type SubnetRepository interface {
	Exists(ctx context.Context, subnetID string) (bool, error)
	Create(ctx context.Context, s *subnet.Subnet) error
	Get(ctx context.Context, subnetID string) (*subnet.Subnet, error)
	Delete(ctx context.Context, subnetID string) error
	TagResource(ctx context.Context, subnetID string, tags map[string]string) error
}

// SubnetUseCase defines the use case interface for Subnet operations
type SubnetUseCase interface {
	SyncSubnet(ctx context.Context, s *subnet.Subnet) error
	DeleteSubnet(ctx context.Context, s *subnet.Subnet) error
}
