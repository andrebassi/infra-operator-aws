// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/vpc"
)

// VPCRepository define a interface do repositório para operações de VPC (Virtual Private Cloud).
// Esta interface abstrai as operações de infraestrutura AWS, permitindo diferentes implementações
// (ex: AWS SDK real, LocalStack, mocks para testes).
type VPCRepository interface {
	// Exists verifica se uma VPC existe na AWS pelo ID
	Exists(ctx context.Context, vpcID string) (bool, error)

	// Create cria uma nova VPC na AWS
	Create(ctx context.Context, v *vpc.VPC) error

	// Get obtém os detalhes de uma VPC existente
	Get(ctx context.Context, vpcID string) (*vpc.VPC, error)

	// Delete remove uma VPC da AWS
	Delete(ctx context.Context, vpcID string) error

	// TagResource adiciona ou atualiza tags em uma VPC
	TagResource(ctx context.Context, vpcID string, tags map[string]string) error
}

// VPCUseCase define a interface de caso de uso para operações de VPC.
// Implementa a lógica de negócio para sincronizar e gerenciar VPCs.
type VPCUseCase interface {
	// SyncVPC sincroniza o estado desejado da VPC com o estado real na AWS
	SyncVPC(ctx context.Context, v *vpc.VPC) error

	// DeleteVPC remove uma VPC seguindo as políticas de deleção configuradas
	DeleteVPC(ctx context.Context, v *vpc.VPC) error
}
