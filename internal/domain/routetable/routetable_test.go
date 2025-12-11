package routetable_test

import (
	"testing"

	"infra-operator/internal/domain/routetable"
)

func TestRouteTable_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rt      *routetable.RouteTable
		wantErr error
	}{
		{
			name: "valid route table",
			rt: &routetable.RouteTable{
				VpcID: "vpc-12345678",
			},
			wantErr: nil,
		},
		{
			name: "empty VPC ID",
			rt: &routetable.RouteTable{
				VpcID: "",
			},
			wantErr: routetable.ErrInvalidVpcID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rt.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRouteTable_SetDefaults(t *testing.T) {
	tests := []struct {
		name               string
		rt                 *routetable.RouteTable
		wantDeletionPolicy string
		wantTagsNotNil     bool
		wantRoutesNotNil   bool
		wantSubnetsNotNil  bool
	}{
		{
			name: "empty route table",
			rt: &routetable.RouteTable{
				VpcID: "vpc-12345678",
			},
			wantDeletionPolicy: "Delete",
			wantTagsNotNil:     true,
			wantRoutesNotNil:   true,
			wantSubnetsNotNil:  true,
		},
		{
			name: "with custom deletion policy",
			rt: &routetable.RouteTable{
				VpcID:          "vpc-12345678",
				DeletionPolicy: "Retain",
			},
			wantDeletionPolicy: "Retain",
			wantTagsNotNil:     true,
			wantRoutesNotNil:   true,
			wantSubnetsNotNil:  true,
		},
		{
			name: "with existing tags",
			rt: &routetable.RouteTable{
				VpcID: "vpc-12345678",
				Tags: map[string]string{
					"environment": "production",
				},
			},
			wantDeletionPolicy: "Delete",
			wantTagsNotNil:     true,
			wantRoutesNotNil:   true,
			wantSubnetsNotNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rt.SetDefaults()

			if tt.rt.DeletionPolicy != tt.wantDeletionPolicy {
				t.Errorf("SetDefaults() DeletionPolicy = %v, want %v", tt.rt.DeletionPolicy, tt.wantDeletionPolicy)
			}

			if tt.wantTagsNotNil && tt.rt.Tags == nil {
				t.Errorf("SetDefaults() Tags is nil, want not nil")
			}

			if tt.wantRoutesNotNil && tt.rt.Routes == nil {
				t.Errorf("SetDefaults() Routes is nil, want not nil")
			}

			if tt.wantSubnetsNotNil && tt.rt.SubnetAssociations == nil {
				t.Errorf("SetDefaults() SubnetAssociations is nil, want not nil")
			}
		})
	}
}

func TestRouteTable_ShouldDelete(t *testing.T) {
	tests := []struct {
		name string
		rt   *routetable.RouteTable
		want bool
	}{
		{
			name: "deletion policy Delete",
			rt: &routetable.RouteTable{
				DeletionPolicy: "Delete",
			},
			want: true,
		},
		{
			name: "deletion policy Retain",
			rt: &routetable.RouteTable{
				DeletionPolicy: "Retain",
			},
			want: false,
		},
		{
			name: "empty deletion policy",
			rt: &routetable.RouteTable{
				DeletionPolicy: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rt.ShouldDelete(); got != tt.want {
				t.Errorf("ShouldDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteTable_Routes(t *testing.T) {
	tests := []struct {
		name          string
		rt            *routetable.RouteTable
		wantRoutesLen int
	}{
		{
			name: "no routes",
			rt: &routetable.RouteTable{
				Routes: []routetable.Route{},
			},
			wantRoutesLen: 0,
		},
		{
			name: "single route to internet gateway",
			rt: &routetable.RouteTable{
				Routes: []routetable.Route{
					{
						DestinationCidrBlock: "0.0.0.0/0",
						GatewayID:            "igw-12345678",
					},
				},
			},
			wantRoutesLen: 1,
		},
		{
			name: "multiple routes",
			rt: &routetable.RouteTable{
				Routes: []routetable.Route{
					{
						DestinationCidrBlock: "0.0.0.0/0",
						GatewayID:            "igw-12345678",
					},
					{
						DestinationCidrBlock: "10.0.0.0/16",
						NatGatewayID:         "nat-12345678",
					},
				},
			},
			wantRoutesLen: 2,
		},
		{
			name: "route to NAT gateway",
			rt: &routetable.RouteTable{
				Routes: []routetable.Route{
					{
						DestinationCidrBlock: "0.0.0.0/0",
						NatGatewayID:         "nat-12345678",
					},
				},
			},
			wantRoutesLen: 1,
		},
		{
			name: "route to VPC peering connection",
			rt: &routetable.RouteTable{
				Routes: []routetable.Route{
					{
						DestinationCidrBlock:   "10.1.0.0/16",
						VpcPeeringConnectionID: "pcx-12345678",
					},
				},
			},
			wantRoutesLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.rt.Routes) != tt.wantRoutesLen {
				t.Errorf("Routes length = %d, want %d", len(tt.rt.Routes), tt.wantRoutesLen)
			}
		})
	}
}

func TestRouteTable_SubnetAssociations(t *testing.T) {
	tests := []struct {
		name           string
		rt             *routetable.RouteTable
		wantSubnetsLen int
	}{
		{
			name: "no subnet associations",
			rt: &routetable.RouteTable{
				SubnetAssociations: []string{},
			},
			wantSubnetsLen: 0,
		},
		{
			name: "single subnet association",
			rt: &routetable.RouteTable{
				SubnetAssociations: []string{"subnet-12345678"},
			},
			wantSubnetsLen: 1,
		},
		{
			name: "multiple subnet associations",
			rt: &routetable.RouteTable{
				SubnetAssociations: []string{
					"subnet-12345678",
					"subnet-87654321",
					"subnet-11111111",
				},
			},
			wantSubnetsLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.rt.SubnetAssociations) != tt.wantSubnetsLen {
				t.Errorf("SubnetAssociations length = %d, want %d", len(tt.rt.SubnetAssociations), tt.wantSubnetsLen)
			}
		})
	}
}

func TestRouteTable_Tags(t *testing.T) {
	tests := []struct {
		name     string
		rt       *routetable.RouteTable
		wantTags map[string]string
	}{
		{
			name: "no tags",
			rt: &routetable.RouteTable{
				Tags: map[string]string{},
			},
			wantTags: map[string]string{},
		},
		{
			name: "single tag",
			rt: &routetable.RouteTable{
				Tags: map[string]string{
					"Name": "my-route-table",
				},
			},
			wantTags: map[string]string{
				"Name": "my-route-table",
			},
		},
		{
			name: "multiple tags",
			rt: &routetable.RouteTable{
				Tags: map[string]string{
					"Name":        "my-route-table",
					"Environment": "production",
					"Terraform":   "true",
				},
			},
			wantTags: map[string]string{
				"Name":        "my-route-table",
				"Environment": "production",
				"Terraform":   "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.rt.Tags) != len(tt.wantTags) {
				t.Errorf("Tags count = %d, want %d", len(tt.rt.Tags), len(tt.wantTags))
			}
			for k, v := range tt.wantTags {
				if tt.rt.Tags[k] != v {
					t.Errorf("Tag[%s] = %v, want %v", k, tt.rt.Tags[k], v)
				}
			}
		})
	}
}
