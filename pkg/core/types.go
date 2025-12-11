package core

import (
	"time"
)

// Resource representa um recurso genérico estilo Kubernetes do YAML
type Resource struct {
	APIVersion string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Metadata   Metadata               `yaml:"metadata" json:"metadata"`
	Spec       map[string]interface{} `yaml:"spec" json:"spec"`
	Status     map[string]interface{} `yaml:"status,omitempty" json:"status,omitempty"`
}

// Metadata representa os metadados do recurso
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ResourceState representa o estado de um único recurso
type ResourceState struct {
	APIVersion   string                 `json:"apiVersion"`
	Kind         string                 `json:"kind"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	Spec         map[string]interface{} `json:"spec"`
	Status       map[string]interface{} `json:"status"`
	AWSResources map[string]string      `json:"awsResources"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// AWSConfig contém a configuração AWS
type AWSConfig struct {
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	SessionToken    string `json:"sessionToken,omitempty"`
	Profile         string `json:"profile,omitempty"`
}

// ProviderConfig contém a configuração do provider AWS
type ProviderConfig struct {
	Name string    `json:"name"`
	AWS  AWSConfig `json:"aws"`
}

// PlanResult representa o resultado de um plano de execução
type PlanResult struct {
	ToCreate  []PlanItem `json:"toCreate"`
	ToUpdate  []PlanItem `json:"toUpdate"`
	ToDelete  []PlanItem `json:"toDelete"`
	NoChange  []PlanItem `json:"noChange"`
	Resources []Resource `json:"-"`
}

// PlanItem representa um item do plano
type PlanItem struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Action    string   `json:"action"`
	Changes   []string `json:"changes,omitempty"`
}

// ApplyResult representa o resultado de um apply
type ApplyResult struct {
	Created []ResourceResult `json:"created"`
	Updated []ResourceResult `json:"updated"`
	Failed  []ResourceResult `json:"failed"`
	Skipped []ResourceResult `json:"skipped"`
}

// DeleteResult representa o resultado de um delete
type DeleteResult struct {
	Deleted []ResourceResult `json:"deleted"`
	Failed  []ResourceResult `json:"failed"`
	Skipped []ResourceResult `json:"skipped"`
}

// ResourceResult representa o resultado de uma operação em um recurso
type ResourceResult struct {
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace,omitempty"`
	AWSResources map[string]string `json:"awsResources,omitempty"`
	Error        string            `json:"error,omitempty"`
	Message      string            `json:"message,omitempty"`
}

// StateFromResource cria um ResourceState a partir de um Resource
func StateFromResource(r Resource) *ResourceState {
	return &ResourceState{
		APIVersion:   r.APIVersion,
		Kind:         r.Kind,
		Name:         r.Metadata.Name,
		Namespace:    r.Metadata.Namespace,
		Spec:         r.Spec,
		Status:       r.Status,
		AWSResources: make(map[string]string),
	}
}

// ResourceFromState converte um ResourceState de volta para um Resource
func ResourceFromState(state *ResourceState) Resource {
	return Resource{
		APIVersion: state.APIVersion,
		Kind:       state.Kind,
		Metadata: Metadata{
			Name:      state.Name,
			Namespace: state.Namespace,
		},
		Spec:   state.Spec,
		Status: state.Status,
	}
}
