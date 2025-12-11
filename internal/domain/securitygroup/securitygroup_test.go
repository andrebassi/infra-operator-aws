package securitygroup_test

import (
	"testing"

	"infra-operator/internal/domain/securitygroup"
)

func TestSecurityGroup_Validate(t *testing.T) {
	tests := []struct {
		name    string
		sg      *securitygroup.SecurityGroup
		wantErr error
	}{
		{
			name: "valid security group",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-security-group",
				Description: "My security group description",
				VpcID:       "vpc-12345678",
			},
			wantErr: nil,
		},
		{
			name: "empty group name",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "",
				Description: "My security group description",
				VpcID:       "vpc-12345678",
			},
			wantErr: securitygroup.ErrInvalidGroupName,
		},
		{
			name: "empty VPC ID",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-security-group",
				Description: "My security group description",
				VpcID:       "",
			},
			wantErr: securitygroup.ErrInvalidVpcID,
		},
		{
			name: "empty description",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-security-group",
				Description: "",
				VpcID:       "vpc-12345678",
			},
			wantErr: securitygroup.ErrInvalidDescription,
		},
		{
			name: "all fields empty",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "",
				Description: "",
				VpcID:       "",
			},
			wantErr: securitygroup.ErrInvalidGroupName, // First validation error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityGroup_SetDefaults(t *testing.T) {
	tests := []struct {
		name               string
		sg                 *securitygroup.SecurityGroup
		wantDeletionPolicy string
		wantTagsNotNil     bool
		wantIngressNotNil  bool
		wantEgressNotNil   bool
	}{
		{
			name: "empty security group",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-sg",
				Description: "Test SG",
				VpcID:       "vpc-12345678",
			},
			wantDeletionPolicy: "Delete",
			wantTagsNotNil:     true,
			wantIngressNotNil:  true,
			wantEgressNotNil:   true,
		},
		{
			name: "with custom deletion policy",
			sg: &securitygroup.SecurityGroup{
				GroupName:      "my-sg",
				Description:    "Test SG",
				VpcID:          "vpc-12345678",
				DeletionPolicy: "Retain",
			},
			wantDeletionPolicy: "Retain",
			wantTagsNotNil:     true,
			wantIngressNotNil:  true,
			wantEgressNotNil:   true,
		},
		{
			name: "with existing tags",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-sg",
				Description: "Test SG",
				VpcID:       "vpc-12345678",
				Tags: map[string]string{
					"environment": "production",
				},
			},
			wantDeletionPolicy: "Delete",
			wantTagsNotNil:     true,
			wantIngressNotNil:  true,
			wantEgressNotNil:   true,
		},
		{
			name: "with existing rules",
			sg: &securitygroup.SecurityGroup{
				GroupName:   "my-sg",
				Description: "Test SG",
				VpcID:       "vpc-12345678",
				IngressRules: []securitygroup.Rule{
					{
						IpProtocol: "tcp",
						FromPort:   80,
						ToPort:     80,
						CidrBlocks: []string{"0.0.0.0/0"},
					},
				},
				EgressRules: []securitygroup.Rule{
					{
						IpProtocol: "-1",
						CidrBlocks: []string{"0.0.0.0/0"},
					},
				},
			},
			wantDeletionPolicy: "Delete",
			wantTagsNotNil:     true,
			wantIngressNotNil:  true,
			wantEgressNotNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sg.SetDefaults()

			if tt.sg.DeletionPolicy != tt.wantDeletionPolicy {
				t.Errorf("SetDefaults() DeletionPolicy = %v, want %v", tt.sg.DeletionPolicy, tt.wantDeletionPolicy)
			}

			if tt.wantTagsNotNil && tt.sg.Tags == nil {
				t.Errorf("SetDefaults() Tags is nil, want not nil")
			}

			if tt.wantIngressNotNil && tt.sg.IngressRules == nil {
				t.Errorf("SetDefaults() IngressRules is nil, want not nil")
			}

			if tt.wantEgressNotNil && tt.sg.EgressRules == nil {
				t.Errorf("SetDefaults() EgressRules is nil, want not nil")
			}
		})
	}
}

func TestSecurityGroup_ShouldDelete(t *testing.T) {
	tests := []struct {
		name string
		sg   *securitygroup.SecurityGroup
		want bool
	}{
		{
			name: "deletion policy Delete",
			sg: &securitygroup.SecurityGroup{
				DeletionPolicy: "Delete",
			},
			want: true,
		},
		{
			name: "deletion policy Retain",
			sg: &securitygroup.SecurityGroup{
				DeletionPolicy: "Retain",
			},
			want: false,
		},
		{
			name: "deletion policy Orphan",
			sg: &securitygroup.SecurityGroup{
				DeletionPolicy: "Orphan",
			},
			want: false,
		},
		{
			name: "empty deletion policy",
			sg: &securitygroup.SecurityGroup{
				DeletionPolicy: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sg.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityGroup_IngressRules(t *testing.T) {
	tests := []struct {
		name         string
		sg           *securitygroup.SecurityGroup
		wantRulesCnt int
	}{
		{
			name: "no ingress rules",
			sg: &securitygroup.SecurityGroup{
				IngressRules: []securitygroup.Rule{},
			},
			wantRulesCnt: 0,
		},
		{
			name: "single ingress rule",
			sg: &securitygroup.SecurityGroup{
				IngressRules: []securitygroup.Rule{
					{
						IpProtocol:  "tcp",
						FromPort:    80,
						ToPort:      80,
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow HTTP",
					},
				},
			},
			wantRulesCnt: 1,
		},
		{
			name: "multiple ingress rules",
			sg: &securitygroup.SecurityGroup{
				IngressRules: []securitygroup.Rule{
					{
						IpProtocol:  "tcp",
						FromPort:    80,
						ToPort:      80,
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow HTTP",
					},
					{
						IpProtocol:  "tcp",
						FromPort:    443,
						ToPort:      443,
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow HTTPS",
					},
				},
			},
			wantRulesCnt: 2,
		},
		{
			name: "ingress rule with IPv6",
			sg: &securitygroup.SecurityGroup{
				IngressRules: []securitygroup.Rule{
					{
						IpProtocol:     "tcp",
						FromPort:       80,
						ToPort:         80,
						Ipv6CidrBlocks: []string{"::/0"},
						Description:    "Allow HTTP IPv6",
					},
				},
			},
			wantRulesCnt: 1,
		},
		{
			name: "ingress rule with source security group",
			sg: &securitygroup.SecurityGroup{
				IngressRules: []securitygroup.Rule{
					{
						IpProtocol:            "tcp",
						FromPort:              3306,
						ToPort:                3306,
						SourceSecurityGroupID: "sg-87654321",
						Description:           "Allow MySQL from app tier",
					},
				},
			},
			wantRulesCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.sg.IngressRules) != tt.wantRulesCnt {
				t.Errorf("IngressRules count = %d, want %d", len(tt.sg.IngressRules), tt.wantRulesCnt)
			}
		})
	}
}

func TestSecurityGroup_EgressRules(t *testing.T) {
	tests := []struct {
		name         string
		sg           *securitygroup.SecurityGroup
		wantRulesCnt int
	}{
		{
			name: "no egress rules",
			sg: &securitygroup.SecurityGroup{
				EgressRules: []securitygroup.Rule{},
			},
			wantRulesCnt: 0,
		},
		{
			name: "single egress rule - allow all",
			sg: &securitygroup.SecurityGroup{
				EgressRules: []securitygroup.Rule{
					{
						IpProtocol:  "-1",
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow all outbound",
					},
				},
			},
			wantRulesCnt: 1,
		},
		{
			name: "multiple egress rules",
			sg: &securitygroup.SecurityGroup{
				EgressRules: []securitygroup.Rule{
					{
						IpProtocol:  "tcp",
						FromPort:    443,
						ToPort:      443,
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow HTTPS outbound",
					},
					{
						IpProtocol:  "tcp",
						FromPort:    80,
						ToPort:      80,
						CidrBlocks:  []string{"0.0.0.0/0"},
						Description: "Allow HTTP outbound",
					},
				},
			},
			wantRulesCnt: 2,
		},
		{
			name: "egress rule with destination security group",
			sg: &securitygroup.SecurityGroup{
				EgressRules: []securitygroup.Rule{
					{
						IpProtocol:            "tcp",
						FromPort:              3306,
						ToPort:                3306,
						SourceSecurityGroupID: "sg-database",
						Description:           "Allow MySQL to database tier",
					},
				},
			},
			wantRulesCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.sg.EgressRules) != tt.wantRulesCnt {
				t.Errorf("EgressRules count = %d, want %d", len(tt.sg.EgressRules), tt.wantRulesCnt)
			}
		})
	}
}

func TestSecurityGroup_Tags(t *testing.T) {
	tests := []struct {
		name     string
		sg       *securitygroup.SecurityGroup
		wantTags map[string]string
	}{
		{
			name: "no tags",
			sg: &securitygroup.SecurityGroup{
				Tags: map[string]string{},
			},
			wantTags: map[string]string{},
		},
		{
			name: "single tag",
			sg: &securitygroup.SecurityGroup{
				Tags: map[string]string{
					"environment": "production",
				},
			},
			wantTags: map[string]string{
				"environment": "production",
			},
		},
		{
			name: "multiple tags",
			sg: &securitygroup.SecurityGroup{
				Tags: map[string]string{
					"environment": "production",
					"team":        "platform",
					"managed-by":  "infra-operator",
				},
			},
			wantTags: map[string]string{
				"environment": "production",
				"team":        "platform",
				"managed-by":  "infra-operator",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.sg.Tags) != len(tt.wantTags) {
				t.Errorf("Tags count = %d, want %d", len(tt.sg.Tags), len(tt.wantTags))
			}
			for k, v := range tt.wantTags {
				if tt.sg.Tags[k] != v {
					t.Errorf("Tag[%s] = %v, want %v", k, tt.sg.Tags[k], v)
				}
			}
		})
	}
}
