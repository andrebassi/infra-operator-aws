package eks

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awseks "github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"infra-operator/internal/domain/eks"
)

type Repository struct {
	client *awseks.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awseks.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, clusterName string) (bool, error) {
	_, err := r.client.DescribeCluster(ctx, &awseks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		// Check if error is cluster not found
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to describe cluster: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, cluster *eks.Cluster) error {
	input := &awseks.CreateClusterInput{
		Name:    aws.String(cluster.ClusterName),
		Version: aws.String(cluster.Version),
		RoleArn: aws.String(cluster.RoleARN),
		ResourcesVpcConfig: &types.VpcConfigRequest{
			SubnetIds:             cluster.VpcConfig.SubnetIDs,
			SecurityGroupIds:      cluster.VpcConfig.SecurityGroupIDs,
			EndpointPublicAccess:  aws.Bool(cluster.VpcConfig.EndpointPublicAccess),
			EndpointPrivateAccess: aws.Bool(cluster.VpcConfig.EndpointPrivateAccess),
			PublicAccessCidrs:     cluster.VpcConfig.PublicAccessCidrs,
		},
	}

	// Add logging
	if cluster.Logging != nil && len(cluster.Logging.ClusterLogging) > 0 {
		logSetup := &types.Logging{
			ClusterLogging: []types.LogSetup{},
		}
		for _, logType := range cluster.Logging.ClusterLogging {
			logSetup.ClusterLogging = append(logSetup.ClusterLogging, types.LogSetup{
				Enabled: aws.Bool(true),
				Types:   []types.LogType{types.LogType(logType)},
			})
		}
		input.Logging = logSetup
	}

	// Add encryption
	if cluster.Encryption != nil && cluster.Encryption.ProviderKeyARN != "" {
		input.EncryptionConfig = []types.EncryptionConfig{
			{
				Resources: cluster.Encryption.Resources,
				Provider: &types.Provider{
					KeyArn: aws.String(cluster.Encryption.ProviderKeyARN),
				},
			},
		}
	}

	// Add tags
	if len(cluster.Tags) > 0 {
		input.Tags = cluster.Tags
	}

	output, err := r.client.CreateCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create EKS cluster: %w", err)
	}

	if output.Cluster != nil {
		cluster.ARN = aws.ToString(output.Cluster.Arn)
		cluster.Status = string(output.Cluster.Status)
		if output.Cluster.Endpoint != nil {
			cluster.Endpoint = aws.ToString(output.Cluster.Endpoint)
		}
		if output.Cluster.PlatformVersion != nil {
			cluster.PlatformVersion = aws.ToString(output.Cluster.PlatformVersion)
		}
		if output.Cluster.CertificateAuthority != nil && output.Cluster.CertificateAuthority.Data != nil {
			cluster.CertificateAuthority = aws.ToString(output.Cluster.CertificateAuthority.Data)
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, clusterName string) (*eks.Cluster, error) {
	output, err := r.client.DescribeCluster(ctx, &awseks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	if output.Cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	c := output.Cluster
	cluster := &eks.Cluster{
		ClusterName: aws.ToString(c.Name),
		Version:     aws.ToString(c.Version),
		RoleARN:     aws.ToString(c.RoleArn),
		ARN:         aws.ToString(c.Arn),
		Status:      string(c.Status),
	}

	if c.Endpoint != nil {
		cluster.Endpoint = aws.ToString(c.Endpoint)
	}

	if c.PlatformVersion != nil {
		cluster.PlatformVersion = aws.ToString(c.PlatformVersion)
	}

	if c.CertificateAuthority != nil && c.CertificateAuthority.Data != nil {
		cluster.CertificateAuthority = aws.ToString(c.CertificateAuthority.Data)
	}

	if c.ResourcesVpcConfig != nil {
		cluster.VpcConfig = eks.VpcConfig{
			SubnetIDs:             c.ResourcesVpcConfig.SubnetIds,
			SecurityGroupIDs:      c.ResourcesVpcConfig.SecurityGroupIds,
			EndpointPublicAccess:  c.ResourcesVpcConfig.EndpointPublicAccess,
			EndpointPrivateAccess: c.ResourcesVpcConfig.EndpointPrivateAccess,
			PublicAccessCidrs:     c.ResourcesVpcConfig.PublicAccessCidrs,
		}
	}

	if c.Logging != nil && len(c.Logging.ClusterLogging) > 0 {
		cluster.Logging = &eks.Logging{
			ClusterLogging: []string{},
		}
		for _, logSetup := range c.Logging.ClusterLogging {
			if aws.ToBool(logSetup.Enabled) {
				for _, logType := range logSetup.Types {
					cluster.Logging.ClusterLogging = append(cluster.Logging.ClusterLogging, string(logType))
				}
			}
		}
	}

	if len(c.EncryptionConfig) > 0 {
		cluster.Encryption = &eks.Encryption{
			Resources: c.EncryptionConfig[0].Resources,
		}
		if c.EncryptionConfig[0].Provider != nil && c.EncryptionConfig[0].Provider.KeyArn != nil {
			cluster.Encryption.ProviderKeyARN = aws.ToString(c.EncryptionConfig[0].Provider.KeyArn)
		}
	}

	if len(c.Tags) > 0 {
		cluster.Tags = c.Tags
	}

	now := time.Now()
	cluster.LastSyncTime = &now

	return cluster, nil
}

func (r *Repository) Delete(ctx context.Context, clusterName string) error {
	_, err := r.client.DeleteCluster(ctx, &awseks.DeleteClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete EKS cluster: %w", err)
	}
	return nil
}

func (r *Repository) WaitForActive(ctx context.Context, clusterName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cluster, err := r.Get(ctx, clusterName)
		if err != nil {
			return err
		}
		if cluster.IsActive() {
			return nil
		}
		if cluster.IsFailed() {
			return fmt.Errorf("cluster creation failed")
		}
		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("timeout waiting for cluster to become active")
}

func (r *Repository) UpdateVersion(ctx context.Context, clusterName, version string) error {
	_, err := r.client.UpdateClusterVersion(ctx, &awseks.UpdateClusterVersionInput{
		Name:    aws.String(clusterName),
		Version: aws.String(version),
	})
	if err != nil {
		return fmt.Errorf("failed to update cluster version: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awseks.TagResourceInput{
		ResourceArn: aws.String(arn),
		Tags:        tags,
	})
	if err != nil {
		return fmt.Errorf("failed to tag EKS cluster: %w", err)
	}
	return nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for ResourceNotFoundException
	return err.Error() != "" && (
		err.Error() == "ResourceNotFoundException" ||
		err.Error() == "ClusterNotFoundException" ||
		err.Error() == "cluster not found")
}
