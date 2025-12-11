// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/nlb"
)

// NLBRepository defines the interface for Network Load Balancer operations
type NLBRepository interface {
	Exists(ctx context.Context, lbName string) (bool, error)
	Create(ctx context.Context, lb *nlb.LoadBalancer) error
	Get(ctx context.Context, lbName string) (*nlb.LoadBalancer, error)
	Delete(ctx context.Context, lbARN string) error
	SetAttributes(ctx context.Context, lb *nlb.LoadBalancer) error
	GetAttributes(ctx context.Context, lb *nlb.LoadBalancer) error
	TagResource(ctx context.Context, lbARN string, tags map[string]string) error
}

// NLBUseCase defines the use case interface for NLB operations
type NLBUseCase interface {
	SyncLoadBalancer(ctx context.Context, lb *nlb.LoadBalancer) error
	DeleteLoadBalancer(ctx context.Context, lb *nlb.LoadBalancer) error
}
