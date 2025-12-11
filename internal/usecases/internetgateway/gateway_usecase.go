package internetgateway

import (
	"context"
	"fmt"
	"infra-operator/internal/domain/internetgateway"
	"infra-operator/internal/ports"
)

type GatewayUseCase struct {
	repo ports.InternetGatewayRepository
}

func NewGatewayUseCase(repo ports.InternetGatewayRepository) *GatewayUseCase {
	return &GatewayUseCase{repo: repo}
}

func (uc *GatewayUseCase) SyncGateway(ctx context.Context, g *internetgateway.Gateway) error {
	g.SetDefaults()
	if err := g.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if g.InternetGatewayID != "" {
		exists, err := uc.repo.Exists(ctx, g.InternetGatewayID)
		if err != nil {
			return err
		}
		if exists {
			current, err := uc.repo.Get(ctx, g.InternetGatewayID)
			if err != nil {
				return err
			}
			g.State = current.State
			return nil
		}
	}

	return uc.repo.Create(ctx, g)
}

func (uc *GatewayUseCase) DeleteGateway(ctx context.Context, g *internetgateway.Gateway) error {
	if !g.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, g.InternetGatewayID, g.VpcID)
}
