// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/routetable"
)

// RouteTableRepository defines the interface for Route Table operations
type RouteTableRepository interface {
	Exists(ctx context.Context, routeTableID string) (bool, error)
	Create(ctx context.Context, rt *routetable.RouteTable) error
	CreateRoute(ctx context.Context, routeTableID string, route routetable.Route) error
	AssociateSubnet(ctx context.Context, routeTableID, subnetID string) error
	Get(ctx context.Context, routeTableID string) (*routetable.RouteTable, error)
	Delete(ctx context.Context, routeTableID string) error
	TagResource(ctx context.Context, routeTableID string, tags map[string]string) error
}

// RouteTableUseCase defines the use case interface for Route Table operations
type RouteTableUseCase interface {
	SyncRouteTable(ctx context.Context, rt *routetable.RouteTable) error
	DeleteRouteTable(ctx context.Context, rt *routetable.RouteTable) error
}
