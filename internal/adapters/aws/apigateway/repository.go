package apigateway

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsapigw "github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"

	"infra-operator/internal/domain/apigateway"
)

type Repository struct {
	client *awsapigw.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsapigw.NewFromConfig(cfg)}
}

func (r *Repository) Exists(ctx context.Context, apiID string) (bool, error) {
	_, err := r.client.GetApi(ctx, &awsapigw.GetApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		// Check if error is "not found"
		return false, nil
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, api *apigateway.API) error {
	input := &awsapigw.CreateApiInput{
		Name:                      aws.String(api.Name),
		ProtocolType:              types.ProtocolType(api.ProtocolType),
		DisableExecuteApiEndpoint: aws.Bool(api.DisableExecuteApiEndpoint),
	}

	if api.Description != "" {
		input.Description = aws.String(api.Description)
	}

	// CORS configuration only applies to HTTP APIs
	if api.ProtocolType == apigateway.ProtocolTypeHTTP && api.CorsConfiguration != nil {
		cors := &types.Cors{
			AllowCredentials: aws.Bool(api.CorsConfiguration.AllowCredentials),
		}

		if len(api.CorsConfiguration.AllowOrigins) > 0 {
			cors.AllowOrigins = api.CorsConfiguration.AllowOrigins
		}

		if len(api.CorsConfiguration.AllowMethods) > 0 {
			cors.AllowMethods = api.CorsConfiguration.AllowMethods
		}

		if len(api.CorsConfiguration.AllowHeaders) > 0 {
			cors.AllowHeaders = api.CorsConfiguration.AllowHeaders
		}

		if len(api.CorsConfiguration.ExposeHeaders) > 0 {
			cors.ExposeHeaders = api.CorsConfiguration.ExposeHeaders
		}

		if api.CorsConfiguration.MaxAge > 0 {
			cors.MaxAge = aws.Int32(api.CorsConfiguration.MaxAge)
		}

		input.CorsConfiguration = cors
	}

	if len(api.Tags) > 0 {
		input.Tags = api.Tags
	}

	output, err := r.client.CreateApi(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create API: %w", err)
	}

	api.APIID = aws.ToString(output.ApiId)
	api.APIEndpoint = aws.ToString(output.ApiEndpoint)

	return nil
}

func (r *Repository) Get(ctx context.Context, apiID string) (*apigateway.API, error) {
	output, err := r.client.GetApi(ctx, &awsapigw.GetApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get API: %w", err)
	}

	api := &apigateway.API{
		APIID:                     aws.ToString(output.ApiId),
		Name:                      aws.ToString(output.Name),
		Description:               aws.ToString(output.Description),
		ProtocolType:              string(output.ProtocolType),
		APIEndpoint:               aws.ToString(output.ApiEndpoint),
		DisableExecuteApiEndpoint: aws.ToBool(output.DisableExecuteApiEndpoint),
	}

	// Parse CORS configuration if present
	if output.CorsConfiguration != nil {
		api.CorsConfiguration = &apigateway.CorsConfiguration{
			AllowOrigins:     output.CorsConfiguration.AllowOrigins,
			AllowMethods:     output.CorsConfiguration.AllowMethods,
			AllowHeaders:     output.CorsConfiguration.AllowHeaders,
			ExposeHeaders:    output.CorsConfiguration.ExposeHeaders,
			AllowCredentials: aws.ToBool(output.CorsConfiguration.AllowCredentials),
		}
		if output.CorsConfiguration.MaxAge != nil {
			api.CorsConfiguration.MaxAge = aws.ToInt32(output.CorsConfiguration.MaxAge)
		}
	}

	return api, nil
}

func (r *Repository) Update(ctx context.Context, api *apigateway.API) error {
	input := &awsapigw.UpdateApiInput{
		ApiId: aws.String(api.APIID),
	}

	if api.Name != "" {
		input.Name = aws.String(api.Name)
	}

	if api.Description != "" {
		input.Description = aws.String(api.Description)
	}

	input.DisableExecuteApiEndpoint = aws.Bool(api.DisableExecuteApiEndpoint)

	// Update CORS if provided (HTTP APIs only)
	if api.ProtocolType == apigateway.ProtocolTypeHTTP && api.CorsConfiguration != nil {
		cors := &types.Cors{
			AllowOrigins:     api.CorsConfiguration.AllowOrigins,
			AllowMethods:     api.CorsConfiguration.AllowMethods,
			AllowHeaders:     api.CorsConfiguration.AllowHeaders,
			ExposeHeaders:    api.CorsConfiguration.ExposeHeaders,
			AllowCredentials: aws.Bool(api.CorsConfiguration.AllowCredentials),
		}
		if api.CorsConfiguration.MaxAge > 0 {
			cors.MaxAge = aws.Int32(api.CorsConfiguration.MaxAge)
		}
		input.CorsConfiguration = cors
	}

	_, err := r.client.UpdateApi(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update API: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, apiID string) error {
	_, err := r.client.DeleteApi(ctx, &awsapigw.DeleteApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete API: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, apiARN string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awsapigw.TagResourceInput{
		ResourceArn: aws.String(apiARN),
		Tags:        tags,
	})
	if err != nil {
		return fmt.Errorf("failed to tag API: %w", err)
	}
	return nil
}
