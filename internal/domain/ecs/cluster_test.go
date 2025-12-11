package ecs_test

import (
	"testing"

	"infra-operator/internal/domain/ecs"
)

func TestCluster_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ecs.Cluster
		wantErr error
	}{
		{
			name: "valid cluster",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
			},
			wantErr: nil,
		},
		{
			name: "empty cluster name",
			cluster: &ecs.Cluster{
				ClusterName: "",
			},
			wantErr: ecs.ErrInvalidClusterName,
		},
		{
			name: "valid with container insights enabled",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
				Settings: []ecs.ClusterSetting{
					{Name: "containerInsights", Value: "enabled"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with container insights disabled",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
				Settings: []ecs.ClusterSetting{
					{Name: "containerInsights", Value: "disabled"},
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid container insights value",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
				Settings: []ecs.ClusterSetting{
					{Name: "containerInsights", Value: "invalid"},
				},
			},
			wantErr: ecs.ErrInvalidSettings,
		},
		{
			name: "empty setting name",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
				Settings: []ecs.ClusterSetting{
					{Name: "", Value: "enabled"},
				},
			},
			wantErr: ecs.ErrInvalidSettings,
		},
		{
			name: "empty setting value",
			cluster: &ecs.Cluster{
				ClusterName: "my-cluster",
				Settings: []ecs.ClusterSetting{
					{Name: "containerInsights", Value: ""},
				},
			},
			wantErr: ecs.ErrInvalidSettings,
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
	cluster := &ecs.Cluster{}
	cluster.SetDefaults()

	if cluster.DeletionPolicy != "Delete" {
		t.Errorf("DeletionPolicy = %v, want Delete", cluster.DeletionPolicy)
	}
	if cluster.Tags == nil {
		t.Error("Tags should be initialized")
	}
	if len(cluster.Settings) != 1 {
		t.Errorf("Settings length = %v, want 1", len(cluster.Settings))
	}
	if cluster.Settings[0].Name != "containerInsights" {
		t.Errorf("Settings[0].Name = %v, want containerInsights", cluster.Settings[0].Name)
	}
	if cluster.Settings[0].Value != "enabled" {
		t.Errorf("Settings[0].Value = %v, want enabled", cluster.Settings[0].Value)
	}
}

func TestCluster_ShouldDelete(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ecs.Cluster
		want    bool
	}{
		{"Delete policy", &ecs.Cluster{DeletionPolicy: "Delete"}, true},
		{"Retain policy", &ecs.Cluster{DeletionPolicy: "Retain"}, false},
		{"Empty policy", &ecs.Cluster{DeletionPolicy: ""}, false},
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
		name   string
		status string
		want   bool
	}{
		{"ACTIVE", "ACTIVE", true},
		{"PROVISIONING", "PROVISIONING", false},
		{"DEPROVISIONING", "DEPROVISIONING", false},
		{"FAILED", "FAILED", false},
		{"INACTIVE", "INACTIVE", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &ecs.Cluster{Status: tt.status}
			if got := cluster.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsProvisioning(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"PROVISIONING", "PROVISIONING", true},
		{"ACTIVE", "ACTIVE", false},
		{"FAILED", "FAILED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &ecs.Cluster{Status: tt.status}
			if got := cluster.IsProvisioning(); got != tt.want {
				t.Errorf("IsProvisioning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsFailed(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"FAILED", "FAILED", true},
		{"ACTIVE", "ACTIVE", false},
		{"PROVISIONING", "PROVISIONING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &ecs.Cluster{Status: tt.status}
			if got := cluster.IsFailed(); got != tt.want {
				t.Errorf("IsFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsInactive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"INACTIVE", "INACTIVE", true},
		{"ACTIVE", "ACTIVE", false},
		{"PROVISIONING", "PROVISIONING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &ecs.Cluster{Status: tt.status}
			if got := cluster.IsInactive(); got != tt.want {
				t.Errorf("IsInactive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_IsDeprovisioning(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"DEPROVISIONING", "DEPROVISIONING", true},
		{"ACTIVE", "ACTIVE", false},
		{"PROVISIONING", "PROVISIONING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &ecs.Cluster{Status: tt.status}
			if got := cluster.IsDeprovisioning(); got != tt.want {
				t.Errorf("IsDeprovisioning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_HasTasks(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ecs.Cluster
		want    bool
	}{
		{
			"with running tasks",
			&ecs.Cluster{RunningTasksCount: 5, PendingTasksCount: 0},
			true,
		},
		{
			"with pending tasks",
			&ecs.Cluster{RunningTasksCount: 0, PendingTasksCount: 3},
			true,
		},
		{
			"with both",
			&ecs.Cluster{RunningTasksCount: 2, PendingTasksCount: 1},
			true,
		},
		{
			"no tasks",
			&ecs.Cluster{RunningTasksCount: 0, PendingTasksCount: 0},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.HasTasks(); got != tt.want {
				t.Errorf("HasTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_HasServices(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ecs.Cluster
		want    bool
	}{
		{
			"with services",
			&ecs.Cluster{ActiveServicesCount: 3},
			true,
		},
		{
			"no services",
			&ecs.Cluster{ActiveServicesCount: 0},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.HasServices(); got != tt.want {
				t.Errorf("HasServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_HasContainerInstances(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ecs.Cluster
		want    bool
	}{
		{
			"with instances",
			&ecs.Cluster{RegisteredContainerInstancesCount: 2},
			true,
		},
		{
			"no instances",
			&ecs.Cluster{RegisteredContainerInstancesCount: 0},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cluster.HasContainerInstances(); got != tt.want {
				t.Errorf("HasContainerInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}
