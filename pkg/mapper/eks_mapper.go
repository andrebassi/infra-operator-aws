package mapper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/eks"
)

// EKS Cluster Mappers

func CRToDomainEKSCluster(cr *infrav1alpha1.EKSCluster) *eks.Cluster {
	cluster := &eks.Cluster{
		ClusterName: cr.Spec.ClusterName,
		Version:     cr.Spec.Version,
		RoleARN:     cr.Spec.RoleARN,
		VpcConfig: eks.VpcConfig{
			SubnetIDs:             cr.Spec.VpcConfig.SubnetIDs,
			SecurityGroupIDs:      cr.Spec.VpcConfig.SecurityGroupIDs,
			EndpointPublicAccess:  cr.Spec.VpcConfig.EndpointPublicAccess,
			EndpointPrivateAccess: cr.Spec.VpcConfig.EndpointPrivateAccess,
			PublicAccessCidrs:     cr.Spec.VpcConfig.PublicAccessCidrs,
		},
		Tags:           cr.Spec.Tags,
		DeletionPolicy: cr.Spec.DeletionPolicy,
	}

	// Logging
	if cr.Spec.Logging != nil && len(cr.Spec.Logging.ClusterLogging) > 0 {
		cluster.Logging = &eks.Logging{
			ClusterLogging: make([]string, len(cr.Spec.Logging.ClusterLogging)),
		}
		for i, logType := range cr.Spec.Logging.ClusterLogging {
			cluster.Logging.ClusterLogging[i] = string(logType)
		}
	}

	// Encryption
	if cr.Spec.Encryption != nil {
		cluster.Encryption = &eks.Encryption{
			Resources:      cr.Spec.Encryption.Resources,
			ProviderKeyARN: cr.Spec.Encryption.ProviderKeyARN,
		}
	}

	// Status fields
	if cr.Status.ClusterName != "" {
		cluster.ClusterName = cr.Status.ClusterName
	}
	if cr.Status.ARN != "" {
		cluster.ARN = cr.Status.ARN
	}
	if cr.Status.Endpoint != "" {
		cluster.Endpoint = cr.Status.Endpoint
	}
	if cr.Status.Status != "" {
		cluster.Status = cr.Status.Status
	}
	if cr.Status.PlatformVersion != "" {
		cluster.PlatformVersion = cr.Status.PlatformVersion
	}
	if cr.Status.CertificateAuthority != "" {
		cluster.CertificateAuthority = cr.Status.CertificateAuthority
	}

	return cluster
}

func DomainToStatusEKSCluster(cluster *eks.Cluster, cr *infrav1alpha1.EKSCluster) {
	now := metav1.Now()

	cr.Status.ClusterName = cluster.ClusterName
	cr.Status.ARN = cluster.ARN
	cr.Status.Endpoint = cluster.Endpoint
	cr.Status.Status = cluster.Status
	cr.Status.Version = cluster.Version
	cr.Status.PlatformVersion = cluster.PlatformVersion
	cr.Status.CertificateAuthority = cluster.CertificateAuthority
	cr.Status.LastSyncTime = &now

	// Set Ready status based on cluster status
	cr.Status.Ready = cluster.IsActive()
}
