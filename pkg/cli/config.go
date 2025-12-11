package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// AWSConfig contém a configuração AWS para o modo CLI
type AWSConfig struct {
	Region          string
	Endpoint        string // Para LocalStack
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Profile         string
}

// ProviderConfig contém a configuração do recurso AWSProvider ou flags
type ProviderConfig struct {
	Name string
	AWS  AWSConfig
}

// NewProviderConfigFromEnv cria configuração do provider a partir de variáveis de ambiente
func NewProviderConfigFromEnv() *ProviderConfig {
	return &ProviderConfig{
		Name: "env",
		AWS: AWSConfig{
			Region:          getEnvOrDefault("AWS_REGION", "us-east-1"),
			Endpoint:        os.Getenv("AWS_ENDPOINT_URL"),
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
			Profile:         os.Getenv("AWS_PROFILE"),
		},
	}
}

// NewProviderConfigFromResource cria configuração do provider a partir de um recurso AWSProvider YAML
func NewProviderConfigFromResource(r Resource) *ProviderConfig {
	cfg := &ProviderConfig{
		Name: r.Metadata.Name,
		AWS: AWSConfig{
			Region: "us-east-1",
		},
	}

	if region, ok := r.Spec["region"].(string); ok {
		cfg.AWS.Region = region
	}
	if endpoint, ok := r.Spec["endpoint"].(string); ok {
		cfg.AWS.Endpoint = endpoint
	}

	// Verifica credenciais inline (não recomendado, mas suportado)
	if accessKeyID, ok := r.Spec["accessKeyId"].(string); ok {
		cfg.AWS.AccessKeyID = accessKeyID
	}
	if secretAccessKey, ok := r.Spec["secretAccessKey"].(string); ok {
		cfg.AWS.SecretAccessKey = secretAccessKey
	}

	return cfg
}

// GetAWSConfig retorna uma configuração do AWS SDK
func (p *ProviderConfig) GetAWSConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(p.AWS.Region))

	// Usa credenciais estáticas se fornecidas
	if p.AWS.AccessKeyID != "" && p.AWS.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				p.AWS.AccessKeyID,
				p.AWS.SecretAccessKey,
				p.AWS.SessionToken,
			),
		))
	} else if p.AWS.Profile != "" {
		// Usa perfil nomeado
		opts = append(opts, config.WithSharedConfigProfile(p.AWS.Profile))
	}
	// Caso contrário, usa cadeia de credenciais padrão (env vars, IAM role, etc.)

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("falha ao carregar configuração AWS: %w", err)
	}

	// Define endpoint para LocalStack
	if p.AWS.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(p.AWS.Endpoint)
	}

	return awsCfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
