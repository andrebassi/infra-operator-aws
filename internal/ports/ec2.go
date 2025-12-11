// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports

import (
	"context"
	"infra-operator/internal/domain/ec2"
	"time"
)

// ConsoleOutput representa a saída do console de uma instância EC2
type ConsoleOutput struct {
	// Output contém o texto do console output
	Output string
	// Timestamp é o momento em que o output foi gerado
	Timestamp time.Time
}

// EC2Repository defines the interface for EC2 operations
type EC2Repository interface {
	Exists(ctx context.Context, instanceID string) (bool, error)
	Create(ctx context.Context, instance *ec2.Instance) error
	Get(ctx context.Context, instanceID string) (*ec2.Instance, error)
	StartInstance(ctx context.Context, instanceID string) error
	StopInstance(ctx context.Context, instanceID string) error
	TerminateInstance(ctx context.Context, instanceID string) error
	TagResource(ctx context.Context, instanceID string, tags map[string]string) error
	// GetConsoleOutput obtém os logs do console da instância EC2
	// Retorna as últimas linhas do console output (boot logs, kernel messages, etc)
	GetConsoleOutput(ctx context.Context, instanceID string, maxLines int) (*ConsoleOutput, error)
}

// EC2UseCase defines the use case interface for EC2 operations
type EC2UseCase interface {
	SyncInstance(ctx context.Context, instance *ec2.Instance) error
	DeleteInstance(ctx context.Context, instance *ec2.Instance) error
}
