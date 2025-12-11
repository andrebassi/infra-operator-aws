package cloudfront

import "testing"

func TestDistribution_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dist    *Distribution
		wantErr bool
	}{
		{
			name: "valid distribution",
			dist: &Distribution{
				Origins: []Origin{{ID: "origin1", DomainName: "example.com"}},
				DefaultCacheBehavior: CacheBehavior{TargetOriginID: "origin1"},
			},
			wantErr: false,
		},
		{
			name:    "no origins",
			dist:    &Distribution{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dist.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Distribution.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDistribution_SetDefaults(t *testing.T) {
	dist := &Distribution{}
	dist.SetDefaults()

	if dist.PriceClass != PriceClass100 {
		t.Errorf("expected PriceClass %s, got %s", PriceClass100, dist.PriceClass)
	}

	if dist.DeletionPolicy != DeletionPolicyDelete {
		t.Errorf("expected DeletionPolicy %s, got %s", DeletionPolicyDelete, dist.DeletionPolicy)
	}
}

func TestDistribution_IsReady(t *testing.T) {
	tests := []struct {
		name string
		dist *Distribution
		want bool
	}{
		{
			name: "ready distribution",
			dist: &Distribution{DistributionID: "E123", Status: "Deployed"},
			want: true,
		},
		{
			name: "not deployed",
			dist: &Distribution{DistributionID: "E123", Status: "InProgress"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dist.IsReady(); got != tt.want {
				t.Errorf("Distribution.IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
