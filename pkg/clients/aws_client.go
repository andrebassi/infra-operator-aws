package clients

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
	awsacm "infra-operator/internal/adapters/aws/acm"
	awsalb "infra-operator/internal/adapters/aws/alb"
	awsapigw "infra-operator/internal/adapters/aws/apigateway"
	awscf "infra-operator/internal/adapters/aws/cloudfront"
	awsec2 "infra-operator/internal/adapters/aws/ec2"
	awsecr "infra-operator/internal/adapters/aws/ecr"
	awsecs "infra-operator/internal/adapters/aws/ecs"
	awseks "infra-operator/internal/adapters/aws/eks"
	awselasticache "infra-operator/internal/adapters/aws/elasticache"
	awselasticip "infra-operator/internal/adapters/aws/elasticip"
	awsiam "infra-operator/internal/adapters/aws/iam"
	awsigw "infra-operator/internal/adapters/aws/internetgateway"
	awskms "infra-operator/internal/adapters/aws/kms"
	awsnat "infra-operator/internal/adapters/aws/natgateway"
	awsnlb "infra-operator/internal/adapters/aws/nlb"
	awsrds "infra-operator/internal/adapters/aws/rds"
	awsroutetable "infra-operator/internal/adapters/aws/routetable"
	awssm "infra-operator/internal/adapters/aws/secretsmanager"
	awssecuritygroup "infra-operator/internal/adapters/aws/securitygroup"
	awssubnet "infra-operator/internal/adapters/aws/subnet"
	awsvpc "infra-operator/internal/adapters/aws/vpc"
	"infra-operator/internal/ports"
	acmuc "infra-operator/internal/usecases/acm"
	albuc "infra-operator/internal/usecases/alb"
	apigwuc "infra-operator/internal/usecases/apigateway"
	cfuc "infra-operator/internal/usecases/cloudfront"
	ec2uc "infra-operator/internal/usecases/ec2"
	ecruc "infra-operator/internal/usecases/ecr"
	ecsuc "infra-operator/internal/usecases/ecs"
	eksuc "infra-operator/internal/usecases/eks"
	elasticacheuc "infra-operator/internal/usecases/elasticache"
	elasticipuc "infra-operator/internal/usecases/elasticip"
	iamuc "infra-operator/internal/usecases/iam"
	igwuc "infra-operator/internal/usecases/internetgateway"
	kmsuc "infra-operator/internal/usecases/kms"
	natuc "infra-operator/internal/usecases/natgateway"
	nlbuc "infra-operator/internal/usecases/nlb"
	rdsuc "infra-operator/internal/usecases/rds"
	routetableuc "infra-operator/internal/usecases/routetable"
	securitygroupuc "infra-operator/internal/usecases/securitygroup"
	smuc "infra-operator/internal/usecases/secretsmanager"
	subnetuc "infra-operator/internal/usecases/subnet"
	vpcuc "infra-operator/internal/usecases/vpc"
)

// AWSClientFactory creates AWS SDK clients from AWSProvider config
type AWSClientFactory struct {
	k8sClient client.Client
}

// NewAWSClientFactory creates a new factory
func NewAWSClientFactory(k8sClient client.Client) *AWSClientFactory {
	return &AWSClientFactory{
		k8sClient: k8sClient,
	}
}

// GetAWSConfig creates AWS config from AWSProvider
func (f *AWSClientFactory) GetAWSConfig(ctx context.Context, provider *infrav1alpha1.AWSProvider) (aws.Config, error) {
	// Check if provider is ready
	if !provider.Status.Ready {
		return aws.Config{}, fmt.Errorf("AWSProvider %s is not ready", provider.Name)
	}

	// Build config based on credential source
	return f.buildAWSConfig(ctx, provider)
}

// GetAWSConfigFromProviderRef gets AWS config from a provider reference
func (f *AWSClientFactory) GetAWSConfigFromProviderRef(ctx context.Context, namespace string, providerRef infrav1alpha1.ProviderReference) (aws.Config, *infrav1alpha1.AWSProvider, error) {
	// Determine provider namespace
	providerNamespace := providerRef.Namespace
	if providerNamespace == "" {
		providerNamespace = namespace
	}

	// Fetch AWSProvider
	provider := &infrav1alpha1.AWSProvider{}
	if err := f.k8sClient.Get(ctx, types.NamespacedName{
		Name:      providerRef.Name,
		Namespace: providerNamespace,
	}, provider); err != nil {
		return aws.Config{}, nil, fmt.Errorf("failed to get AWSProvider: %w", err)
	}

	// Get AWS config
	awsConfig, err := f.GetAWSConfig(ctx, provider)
	if err != nil {
		return aws.Config{}, nil, err
	}

	return awsConfig, provider, nil
}

// buildAWSConfig builds AWS SDK config from provider spec
func (f *AWSClientFactory) buildAWSConfig(ctx context.Context, provider *infrav1alpha1.AWSProvider) (aws.Config, error) {
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
		if err := f.k8sClient.Get(ctx, types.NamespacedName{
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
		// If static credentials are provided via Secret (deprecated)
		accessKey, err := f.getSecretValue(ctx, provider, provider.Spec.AccessKeyIDRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get access key: %w", err)
		}

		secretKey, err := f.getSecretValue(ctx, provider, provider.Spec.SecretAccessKeyRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get secret key: %w", err)
		}

		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	// If RoleARN is provided (IRSA or AssumeRole)
	if provider.Spec.RoleARN != "" {
		// The AWS SDK will automatically use IRSA if running in EKS with proper service account
		// credentials are loaded automatically from pod service account
	}

	// If AssumeRoleARN is provided for cross-account access
	if provider.Spec.AssumeRoleARN != "" {
		// Implement AssumeRole logic here
		// This would require using STS client to assume the role
		// For now, returning error as it needs custom implementation
		return cfg, fmt.Errorf("AssumeRole not yet implemented")
	}

	// Load config with all options
	cfg, err = config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return cfg, fmt.Errorf("failed to load config: %w", err)
	}

	// If custom endpoint is provided (for LocalStack or custom AWS endpoint)
	if provider.Spec.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(provider.Spec.Endpoint)
	}

	return cfg, nil
}

// getSecretValue retrieves a value from a Kubernetes Secret
func (f *AWSClientFactory) getSecretValue(ctx context.Context, provider *infrav1alpha1.AWSProvider, selector *infrav1alpha1.SecretKeySelector) (string, error) {
	namespace := selector.Namespace
	if namespace == "" {
		namespace = provider.Namespace
	}

	secret := &corev1.Secret{}
	if err := f.k8sClient.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: namespace,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, selector.Name, err)
	}

	value, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", selector.Key, namespace, selector.Name)
	}

	return string(value), nil
}

// GetRDSUseCase creates RDS use case from provider reference
func (f *AWSClientFactory) GetRDSUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.RDSUseCase, error) {
	// Get AWS config from provider
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Create RDS repository
	rdsRepo := awsrds.NewRepository(awsConfig)

	// Create and return RDS use case
	return rdsuc.NewInstanceUseCase(rdsRepo), nil
}

// GetECRUseCase creates ECR use case from provider reference
func (f *AWSClientFactory) GetECRUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ECRUseCase, error) {
	// Get AWS config from provider
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Create ECR repository
	ecrRepo := awsecr.NewRepository(awsConfig)

	// Create and return ECR use case
	return ecruc.NewRepositoryUseCase(ecrRepo), nil
}

// GetIAMUseCase creates IAM use case from provider reference
func (f *AWSClientFactory) GetIAMUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.IAMUseCase, error) {
	// Get AWS config from provider
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Create IAM repository
	iamRepo := awsiam.NewRepository(awsConfig)

	// Create and return IAM use case
	return iamuc.NewRoleUseCase(iamRepo), nil
}

// GetSecretsManagerUseCase creates Secrets Manager use case from provider reference
func (f *AWSClientFactory) GetSecretsManagerUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.SecretsManagerUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	smRepo := awssm.NewRepository(awsConfig)
	return smuc.NewSecretUseCase(smRepo), nil
}

// GetKMSUseCase creates KMS use case from provider reference
func (f *AWSClientFactory) GetKMSUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.KMSUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	kmsRepo := awskms.NewRepository(awsConfig)
	return kmsuc.NewKeyUseCase(kmsRepo), nil
}

// GetEC2UseCase creates EC2 use case from provider reference
func (f *AWSClientFactory) GetEC2UseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.EC2UseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	ec2Repo := awsec2.NewRepository(awsConfig)
	return ec2uc.NewInstanceUseCase(ec2Repo), nil
}

// GetEC2Repository creates EC2 repository from provider reference
// Usado para acessar métodos do repositório que não estão no use case (ex: GetConsoleOutput)
func (f *AWSClientFactory) GetEC2Repository(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.EC2Repository, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	return awsec2.NewRepository(awsConfig), nil
}

// GetElastiCacheUseCase creates ElastiCache use case from provider reference
func (f *AWSClientFactory) GetElastiCacheUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ElastiCacheUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	elasticacheRepo := awselasticache.NewRepository(awsConfig)
	return elasticacheuc.NewClusterUseCase(elasticacheRepo), nil
}

// GetVPCUseCase creates VPC use case
func (f *AWSClientFactory) GetVPCUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.VPCUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsvpc.NewRepository(awsConfig)
	return vpcuc.NewVPCUseCase(repo), nil
}

// GetSubnetUseCase creates Subnet use case
func (f *AWSClientFactory) GetSubnetUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.SubnetUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awssubnet.NewRepository(awsConfig)
	return subnetuc.NewSubnetUseCase(repo), nil
}

// GetInternetGatewayUseCase creates Internet Gateway use case
func (f *AWSClientFactory) GetInternetGatewayUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.InternetGatewayUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsigw.NewRepository(awsConfig)
	return igwuc.NewGatewayUseCase(repo), nil
}

// GetNATGatewayUseCase creates NAT Gateway use case
func (f *AWSClientFactory) GetNATGatewayUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.NATGatewayUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsnat.NewRepository(awsConfig)
	return natuc.NewGatewayUseCase(repo), nil
}

// GetSecurityGroupUseCase creates Security Group use case
func (f *AWSClientFactory) GetSecurityGroupUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.SecurityGroupUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awssecuritygroup.NewRepository(awsConfig)
	return securitygroupuc.NewSecurityGroupUseCase(repo), nil
}

// GetEKSUseCase creates EKS Cluster use case
func (f *AWSClientFactory) GetEKSUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.EKSUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awseks.NewRepository(awsConfig)
	return eksuc.NewClusterUseCase(repo), nil
}

// GetRouteTableUseCase creates RouteTable use case
func (f *AWSClientFactory) GetRouteTableUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.RouteTableUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsroutetable.NewRepository(awsConfig)
	return routetableuc.NewRouteTableUseCase(repo), nil
}

// GetALBUseCase creates ALB use case
func (f *AWSClientFactory) GetALBUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ALBUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsalb.NewRepository(awsConfig)
	return albuc.NewLoadBalancerUseCase(repo), nil
}

// GetECSUseCase creates ECS use case
func (f *AWSClientFactory) GetECSUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ECSUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsecs.NewRepository(awsConfig)
	return ecsuc.NewClusterUseCase(repo), nil
}

// GetElasticIPUseCase creates Elastic IP use case
func (f *AWSClientFactory) GetElasticIPUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ElasticIPUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awselasticip.NewRepository(awsConfig)
	return elasticipuc.NewAddressUseCase(repo), nil
}

// GetNLBUseCase creates NLB use case
func (f *AWSClientFactory) GetNLBUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.NLBUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsnlb.NewRepository(awsConfig)
	return nlbuc.NewLoadBalancerUseCase(repo), nil
}

// GetACMUseCase creates ACM use case
func (f *AWSClientFactory) GetACMUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.ACMUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsacm.NewRepository(awsConfig)
	return acmuc.NewCertificateUseCase(repo), nil
}

// GetAPIGatewayUseCase creates API Gateway use case
func (f *AWSClientFactory) GetAPIGatewayUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.APIGatewayUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awsapigw.NewRepository(awsConfig)
	return apigwuc.NewAPIUseCase(repo), nil
}

// GetCloudFrontUseCase creates CloudFront use case
func (f *AWSClientFactory) GetCloudFrontUseCase(ctx context.Context, providerRef infrav1alpha1.ProviderReference, namespace string) (ports.CloudFrontUseCase, error) {
	awsConfig, _, err := f.GetAWSConfigFromProviderRef(ctx, namespace, providerRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}
	repo := awscf.NewRepository(awsConfig)
	return cfuc.NewDistributionUseCase(repo), nil
}
