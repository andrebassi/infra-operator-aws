package apigateway

import (
	"context"

	"infra-operator/internal/domain/apigateway"
	"infra-operator/internal/ports"
)

type APIUseCase struct {
	repo ports.APIGatewayRepository
}

func NewAPIUseCase(repo ports.APIGatewayRepository) *APIUseCase {
	return &APIUseCase{repo: repo}
}

func (uc *APIUseCase) SyncAPI(ctx context.Context, api *apigateway.API) error {
	api.SetDefaults()

	if err := api.Validate(); err != nil {
		return err
	}

	if api.APIID == "" {
		// Create new API
		return uc.repo.Create(ctx, api)
	}

	// Check if API exists
	exists, err := uc.repo.Exists(ctx, api.APIID)
	if err != nil {
		return err
	}

	if !exists {
		// API was deleted outside of operator, recreate
		api.APIID = ""
		return uc.repo.Create(ctx, api)
	}

	// Update existing API
	return uc.repo.Update(ctx, api)
}

func (uc *APIUseCase) DeleteAPI(ctx context.Context, api *apigateway.API) error {
	if api.DeletionPolicy == apigateway.DeletionPolicyRetain ||
	   api.DeletionPolicy == apigateway.DeletionPolicyOrphan {
		return nil
	}

	if api.APIID != "" {
		return uc.repo.Delete(ctx, api.APIID)
	}

	return nil
}
