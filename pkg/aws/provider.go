package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

// GetAWSConfigFromProvider retrieves AWS configuration from an AWSProvider resource
func GetAWSConfigFromProvider(ctx context.Context, k8sClient client.Client, namespace string, providerRef infrav1alpha1.ProviderReference) (aws.Config, *infrav1alpha1.AWSProvider, error) {
	// Determine the provider namespace
	providerNamespace := providerRef.Namespace
	if providerNamespace == "" {
		providerNamespace = namespace
	}

	// Fetch the AWSProvider
	provider := &infrav1alpha1.AWSProvider{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      providerRef.Name,
		Namespace: providerNamespace,
	}, provider); err != nil {
		return aws.Config{}, nil, fmt.Errorf("failed to get AWSProvider: %w", err)
	}

	// Check if provider is ready
	if !provider.Status.Ready {
		return aws.Config{}, nil, fmt.Errorf("AWSProvider %s is not ready", providerRef.Name)
	}

	// Build AWS config
	awsConfig, err := buildAWSConfig(ctx, k8sClient, provider)
	if err != nil {
		return aws.Config{}, nil, err
	}

	return awsConfig, provider, nil
}

func buildAWSConfig(ctx context.Context, k8sClient client.Client, provider *infrav1alpha1.AWSProvider) (aws.Config, error) {
	var cfg aws.Config
	var err error

	// Base config with region
	configOptions := []func(*config.LoadOptions) error{
		config.WithRegion(provider.Spec.Region),
	}

	// If CredentialsSecret is provided (new preferred method)
	if provider.Spec.CredentialsSecret != nil {
		namespace := provider.Spec.CredentialsSecret.Namespace
		if namespace == "" {
			namespace = provider.Namespace
		}

		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      provider.Spec.CredentialsSecret.Name,
			Namespace: namespace,
		}, secret); err != nil {
			return cfg, fmt.Errorf("failed to get credentials secret: %w", err)
		}

		accessKey := string(secret.Data["AWS_ACCESS_KEY_ID"])
		secretKey := string(secret.Data["AWS_SECRET_ACCESS_KEY"])

		if accessKey == "" || secretKey == "" {
			return cfg, fmt.Errorf("AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY not found in secret")
		}

		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	} else if provider.Spec.AccessKeyIDRef != nil && provider.Spec.SecretAccessKeyRef != nil {
		// If AccessKey/SecretKey are provided via secret references (deprecated)
		accessKey, err := getSecretValue(ctx, k8sClient, provider, provider.Spec.AccessKeyIDRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get access key: %w", err)
		}

		secretKey, err := getSecretValue(ctx, k8sClient, provider, provider.Spec.SecretAccessKeyRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get secret key: %w", err)
		}

		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	// If RoleARN is provided, use it (IRSA or assume role)
	if provider.Spec.RoleARN != "" {
		// The AWS SDK will automatically use IRSA if running in EKS with proper service account
		// credentials are loaded automatically from pod service account
	}

	// Load config with all options
	cfg, err = config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return cfg, err
	}

	// If custom endpoint is provided (for LocalStack or custom AWS endpoint)
	if provider.Spec.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(provider.Spec.Endpoint)
	}

	return cfg, nil
}

func getSecretValue(ctx context.Context, k8sClient client.Client, provider *infrav1alpha1.AWSProvider, selector *infrav1alpha1.SecretKeySelector) (string, error) {
	namespace := selector.Namespace
	if namespace == "" {
		namespace = provider.Namespace
	}

	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: namespace,
	}, secret); err != nil {
		return "", err
	}

	value, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)
	}

	return string(value), nil
}
