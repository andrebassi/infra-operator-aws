package natgateway

import (
	"context"
	"fmt"
	"infra-operator/internal/domain/natgateway"
	"infra-operator/internal/ports"
)

type GatewayUseCase struct {
	repo ports.NATGatewayRepository
}

func NewGatewayUseCase(repo ports.NATGatewayRepository) *GatewayUseCase {
	return &GatewayUseCase{repo: repo}
}

func (uc *GatewayUseCase) SyncGateway(ctx context.Context, g *natgateway.Gateway) error {
	g.SetDefaults()
	if err := g.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if g.NatGatewayID != "" {
		exists, err := uc.repo.Exists(ctx, g.NatGatewayID)
		if err != nil {
			return err
		}
		if exists {
			current, err := uc.repo.Get(ctx, g.NatGatewayID)
			if err != nil {
				return err
			}
			g.State = current.State
			g.VpcID = current.VpcID
			g.PublicIP = current.PublicIP
			g.PrivateIP = current.PrivateIP

			// Always apply tags (ensures Name tag and any updates are applied)
			if len(g.Tags) > 0 {
				uc.repo.TagResource(ctx, g.NatGatewayID, g.Tags)
			}
			return nil
		}
	}

	return uc.repo.Create(ctx, g)
}

func (uc *GatewayUseCase) DeleteGateway(ctx context.Context, g *natgateway.Gateway) error {
	if !g.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, g.NatGatewayID)
}
