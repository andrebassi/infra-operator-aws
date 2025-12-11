package eks_test

import (
	"testing"

	"infra-operator/internal/domain/eks"
)

func TestCluster_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		wantErr error
	}{
		{
			name: "valid cluster",
			cluster: &eks.Cluster{
				ClusterName: "my-eks-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-cluster-role",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{"subnet-123", "subnet-456"},
				},
			},
			wantErr: nil,
		},
		{
			name: "empty cluster name",
			cluster: &eks.Cluster{
				ClusterName: "",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-cluster-role",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{"subnet-123", "subnet-456"},
				},
			},
			wantErr: eks.ErrInvalidClusterName,
		},
		{
			name: "empty version",
			cluster: &eks.Cluster{
				ClusterName: "my-eks-cluster",
				Version:     "",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-cluster-role",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{"subnet-123", "subnet-456"},
				},
			},
			wantErr: eks.ErrInvalidVersion,
		},
		{
			name: "empty role ARN",
			cluster: &eks.Cluster{
				ClusterName: "my-eks-cluster",
				Version:     "1.28",
				RoleARN:     "",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{"subnet-123", "subnet-456"},
				},
			},
			wantErr: eks.ErrInvalidRoleARN,
		},
		{
			name: "insufficient subnets",
			cluster: &eks.Cluster{
				ClusterName: "my-eks-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-cluster-role",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{"subnet-123"},
				},
			},
			wantErr: eks.ErrInvalidSubnets,
		},
		{
			name: "no subnets",
			cluster: &eks.Cluster{
				ClusterName: "my-eks-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-cluster-role",
				VpcConfig: eks.VpcConfig{
					SubnetIDs: []string{},
				},
			},
			wantErr: eks.ErrInvalidSubnets,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCluster_SetDefaults(t *testing.T) {
	tests := []struct {
		name                       string
		cluster                    *eks.Cluster
		wantDeletionPolicy         string
		wantTagsNotNil             bool
		wantPublicAccessCidrsCount int
		wantPublicAccess           bool
	}{
		{
			name: "empty cluster",
			cluster: &eks.Cluster{
				ClusterName: "my-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-role",
			},
			wantDeletionPolicy:         "Delete",
			wantTagsNotNil:             true,
			wantPublicAccessCidrsCount: 1,
			wantPublicAccess:           true,
		},
		{
			name: "with custom deletion policy",
			cluster: &eks.Cluster{
				ClusterName:    "my-cluster",
				Version:        "1.28",
				RoleARN:        "arn:aws:iam::123456789012:role/eks-role",
				DeletionPolicy: "Retain",
			},
			wantDeletionPolicy:         "Retain",
			wantTagsNotNil:             true,
			wantPublicAccessCidrsCount: 1,
			wantPublicAccess:           true,
		},
		{
			name: "with private access enabled",
			cluster: &eks.Cluster{
				ClusterName: "my-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-role",
				VpcConfig: eks.VpcConfig{
					EndpointPrivateAccess: true,
				},
			},
			wantDeletionPolicy:         "Delete",
			wantTagsNotNil:             true,
			wantPublicAccessCidrsCount: 1,
			wantPublicAccess:           false,
		},
		{
			name: "with existing tags",
			cluster: &eks.Cluster{
				ClusterName: "my-cluster",
				Version:     "1.28",
				RoleARN:     "arn:aws:iam::123456789012:role/eks-role",
				Tags: map[string]string{
					"environment": "production",
				},
			},
			wantDeletionPolicy:         "Delete",
			wantTagsNotNil:             true,
			wantPublicAccessCidrsCount: 1,
			wantPublicAccess:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cluster.SetDefaults()

			if tt.cluster.DeletionPolicy != tt.wantDeletionPolicy {
				t.Errorf("SetDefaults() DeletionPolicy = %v, want %v", tt.cluster.DeletionPolicy, tt.wantDeletionPolicy)
			}

			if tt.wantTagsNotNil && tt.cluster.Tags == nil {
				t.Errorf("SetDefaults() Tags is nil, want not nil")
			}

			if len(tt.cluster.VpcConfig.PublicAccessCidrs) != tt.wantPublicAccessCidrsCount {
				t.Errorf("SetDefaults() PublicAccessCidrs count = %d, want %d", len(tt.cluster.VpcConfig.PublicAccessCidrs), tt.wantPublicAccessCidrsCount)
			}

			if tt.cluster.VpcConfig.EndpointPublicAccess != tt.wantPublicAccess {
				t.Errorf("SetDefaults() EndpointPublicAccess = %v, want %v", tt.cluster.VpcConfig.EndpointPublicAccess, tt.wantPublicAccess)
			}
		})
	}
}

func TestCluster_ShouldDelete(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		want    bool
	}{
		{
			name: "deletion policy Delete",
			cluster: &eks.Cluster{
				DeletionPolicy: "Delete",
			},
			want: true,
		},
		{
			name: "deletion policy Retain",
			cluster: &eks.Cluster{
				DeletionPolicy: "Retain",
			},
			want: false,
		},
		{
			name: "empty deletion policy",
			cluster: &eks.Cluster{
				DeletionPolicy: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsActive(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		want    bool
	}{
		{
			name: "status ACTIVE",
			cluster: &eks.Cluster{
				Status: "ACTIVE",
			},
			want: true,
		},
		{
			name: "status CREATING",
			cluster: &eks.Cluster{
				Status: "CREATING",
			},
			want: false,
		},
		{
			name: "status FAILED",
			cluster: &eks.Cluster{
				Status: "FAILED",
			},
			want: false,
		},
		{
			name: "empty status",
			cluster: &eks.Cluster{
				Status: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsCreating(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		want    bool
	}{
		{
			name: "status CREATING",
			cluster: &eks.Cluster{
				Status: "CREATING",
			},
			want: true,
		},
		{
			name: "status ACTIVE",
			cluster: &eks.Cluster{
				Status: "ACTIVE",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.IsCreating(); got != tt.want {
				t.Errorf("IsCreating() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsDeleting(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		want    bool
	}{
		{
			name: "status DELETING",
			cluster: &eks.Cluster{
				Status: "DELETING",
			},
			want: true,
		},
		{
			name: "status ACTIVE",
			cluster: &eks.Cluster{
				Status: "ACTIVE",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.IsDeleting(); got != tt.want {
				t.Errorf("IsDeleting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsFailed(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
		want    bool
	}{
		{
			name: "status FAILED",
			cluster: &eks.Cluster{
				Status: "FAILED",
			},
			want: true,
		},
		{
			name: "status ACTIVE",
			cluster: &eks.Cluster{
				Status: "ACTIVE",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.IsFailed(); got != tt.want {
				t.Errorf("IsFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_VpcConfig(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
	}{
		{
			name: "with security groups",
			cluster: &eks.Cluster{
				VpcConfig: eks.VpcConfig{
					SubnetIDs:        []string{"subnet-123", "subnet-456"},
					SecurityGroupIDs: []string{"sg-123", "sg-456"},
				},
			},
		},
		{
			name: "with public and private access",
			cluster: &eks.Cluster{
				VpcConfig: eks.VpcConfig{
					SubnetIDs:             []string{"subnet-123", "subnet-456"},
					EndpointPublicAccess:  true,
					EndpointPrivateAccess: true,
				},
			},
		},
		{
			name: "with custom public access CIDRs",
			cluster: &eks.Cluster{
				VpcConfig: eks.VpcConfig{
					SubnetIDs:         []string{"subnet-123", "subnet-456"},
					PublicAccessCidrs: []string{"10.0.0.0/8", "192.168.0.0/16"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster.VpcConfig.SubnetIDs == nil {
				t.Errorf("VpcConfig SubnetIDs should not be nil")
			}
		})
	}
}

func TestCluster_Logging(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
	}{
		{
			name: "with logging enabled",
			cluster: &eks.Cluster{
				Logging: &eks.Logging{
					ClusterLogging: []string{"api", "audit", "authenticator"},
				},
			},
		},
		{
			name: "without logging",
			cluster: &eks.Cluster{
				Logging: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster.Logging != nil && tt.cluster.Logging.ClusterLogging == nil {
				t.Errorf("Logging.ClusterLogging should not be nil when Logging is set")
			}
		})
	}
}

func TestCluster_Encryption(t *testing.T) {
	tests := []struct {
		name    string
		cluster *eks.Cluster
	}{
		{
			name: "with encryption",
			cluster: &eks.Cluster{
				Encryption: &eks.Encryption{
					Resources:      []string{"secrets"},
					ProviderKeyARN: "arn:aws:kms:us-east-1:123456789012:key/abc",
				},
			},
		},
		{
			name: "without encryption",
			cluster: &eks.Cluster{
				Encryption: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cluster.Encryption != nil && tt.cluster.Encryption.ProviderKeyARN == "" {
				t.Errorf("Encryption.ProviderKeyARN should not be empty when Encryption is set")
			}
		})
	}
}
