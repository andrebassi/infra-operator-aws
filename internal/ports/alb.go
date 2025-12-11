// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/alb"
)

// ALBRepository defines the interface for Application Load Balancer operations
type ALBRepository interface {
	Exists(ctx context.Context, lbName string) (bool, error)
	Create(ctx context.Context, lb *alb.LoadBalancer) error
	Get(ctx context.Context, lbName string) (*alb.LoadBalancer, error)
	Delete(ctx context.Context, lbARN string) error
	SetAttributes(ctx context.Context, lb *alb.LoadBalancer) error
	GetAttributes(ctx context.Context, lb *alb.LoadBalancer) error
	TagResource(ctx context.Context, lbARN string, tags map[string]string) error
}

// ALBUseCase defines the use case interface for ALB operations
type ALBUseCase interface {
	SyncLoadBalancer(ctx context.Context, lb *alb.LoadBalancer) error
	DeleteLoadBalancer(ctx context.Context, lb *alb.LoadBalancer) error
}
