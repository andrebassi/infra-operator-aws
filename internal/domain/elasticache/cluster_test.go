package elasticache_test
import ("testing"; "infra-operator/internal/domain/elasticache")
func TestCluster_Validate(t *testing.T) {
	tests := []struct {name string; c *elasticache.Cluster; wantErr error}{
		{"valid redis", &elasticache.Cluster{ClusterID: "test", Engine: "redis", EngineVersion: "6.2", NodeType: "cache.t3.micro", NumCacheNodes: 1}, nil},
		{"no cluster ID", &elasticache.Cluster{Engine: "redis", EngineVersion: "6.2", NodeType: "cache.t3.micro", NumCacheNodes: 1}, elasticache.ErrInvalidClusterID},
		{"no engine version", &elasticache.Cluster{ClusterID: "test", Engine: "redis", NodeType: "cache.t3.micro", NumCacheNodes: 1}, elasticache.ErrInvalidEngineVersion},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.c.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
func TestCluster_SetDefaults(t *testing.T) {c := &elasticache.Cluster{}; c.SetDefaults(); if c.DeletionPolicy != "Delete" || c.Tags == nil {t.Error("failed")}}
func TestCluster_ShouldDelete(t *testing.T) {if !(&elasticache.Cluster{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}}
