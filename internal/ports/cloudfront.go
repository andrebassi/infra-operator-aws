// Package ports define as interfaces de portas seguindo Clean Architecture.
//
// Este package contém as abstrações que desacoplam a lógica de negócio das
// implementações concretas, permitindo testabilidade e flexibilidade.
package ports


import (
	"context"
	"infra-operator/internal/domain/cloudfront"
)

// CloudFrontRepository defines the interface for CloudFront operations
type CloudFrontRepository interface {
	Exists(ctx context.Context, distributionID string) (bool, error)
	Create(ctx context.Context, dist *cloudfront.Distribution) error
	Get(ctx context.Context, distributionID string) (*cloudfront.Distribution, error)
	Update(ctx context.Context, dist *cloudfront.Distribution) error
	Delete(ctx context.Context, distributionID, etag string) error
}

// CloudFrontUseCase defines the use case interface for CloudFront operations
type CloudFrontUseCase interface {
	SyncDistribution(ctx context.Context, dist *cloudfront.Distribution) error
	DeleteDistribution(ctx context.Context, dist *cloudfront.Distribution) error
}
