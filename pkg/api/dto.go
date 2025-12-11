package api

import "infra-operator/pkg/core"

// ---- Request DTOs ----

// PlanRequest representa uma requisição de plan
// @Description Request para gerar plano de execução
type PlanRequest struct {
	// Lista de recursos AWS para planejar
	// Example: [{"apiVersion":"aws-infra-operator.runner.codes/v1alpha1","kind":"VPC","metadata":{"name":"my-vpc"},"spec":{"cidrBlock":"10.0.0.0/16"}}]
	Resources []core.Resource `json:"resources,omitempty"`
	// YAML dos recursos (alternativa a resources)
	YAML string `json:"yaml,omitempty" example:"apiVersion: aws-infra-operator.runner.codes/v1alpha1\nkind: VPC\nmetadata:\n  name: my-vpc\nspec:\n  cidrBlock: 10.0.0.0/16"`
}

// ApplyRequest representa uma requisição de apply
type ApplyRequest struct {
	Resources []core.Resource `json:"resources,omitempty"`
	YAML      string          `json:"yaml,omitempty"`
	DryRun    bool            `json:"dryRun,omitempty"`
}

// DeleteRequest representa uma requisição de delete
type DeleteRequest struct {
	Resources []core.Resource `json:"resources,omitempty"`
	YAML      string          `json:"yaml,omitempty"`
	DryRun    bool            `json:"dryRun,omitempty"`
}

// ---- Response DTOs ----

// APIResponse é a resposta padrão da API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError representa um erro da API
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// PlanResponse representa a resposta de um plan
type PlanResponse struct {
	ToCreate  []PlanItemResponse `json:"toCreate"`
	ToUpdate  []PlanItemResponse `json:"toUpdate"`
	ToDelete  []PlanItemResponse `json:"toDelete"`
	NoChange  []PlanItemResponse `json:"noChange"`
	Summary   PlanSummary        `json:"summary"`
}

// PlanItemResponse representa um item do plano na resposta
type PlanItemResponse struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Action    string   `json:"action"`
	Changes   []string `json:"changes,omitempty"`
}

// PlanSummary resume o plano
type PlanSummary struct {
	Create   int `json:"create"`
	Update   int `json:"update"`
	Delete   int `json:"delete"`
	NoChange int `json:"noChange"`
}

// ApplyResponse representa a resposta de um apply
type ApplyResponse struct {
	Created []ResourceResponse `json:"created"`
	Updated []ResourceResponse `json:"updated"`
	Failed  []ResourceResponse `json:"failed"`
	Skipped []ResourceResponse `json:"skipped"`
	Summary ApplySummary       `json:"summary"`
}

// ApplySummary resume o apply
type ApplySummary struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// DeleteResponse representa a resposta de um delete
type DeleteResponse struct {
	Deleted []ResourceResponse `json:"deleted"`
	Failed  []ResourceResponse `json:"failed"`
	Skipped []ResourceResponse `json:"skipped"`
	Summary DeleteSummary      `json:"summary"`
}

// DeleteSummary resume o delete
type DeleteSummary struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// ResourceResponse representa um recurso na resposta
type ResourceResponse struct {
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace,omitempty"`
	AWSResources map[string]string `json:"awsResources,omitempty"`
	Error        string            `json:"error,omitempty"`
	Message      string            `json:"message,omitempty"`
}

// GetResponse representa a resposta de um get
type GetResponse struct {
	Resources []ResourceStateResponse `json:"resources"`
	Count     int                     `json:"count"`
}

// ResourceStateResponse representa o estado de um recurso na resposta
type ResourceStateResponse struct {
	APIVersion   string                 `json:"apiVersion"`
	Kind         string                 `json:"kind"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	AWSResources map[string]string      `json:"awsResources"`
	Status       map[string]interface{} `json:"status"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

// HealthResponse representa a resposta do health check
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// AWSConfigRequest representa uma requisição para configurar credenciais AWS
// @Description Request para configurar credenciais AWS
type AWSConfigRequest struct {
	// Região AWS (ex: us-east-1, sa-east-1)
	Region string `json:"region" example:"us-east-1"`
	// Access Key ID da AWS
	AccessKeyID string `json:"accessKeyId" example:"AKIAIOSFODNN7EXAMPLE"`
	// Secret Access Key da AWS
	SecretAccessKey string `json:"secretAccessKey" example:"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`
	// Session Token (opcional, para credenciais temporárias)
	SessionToken string `json:"sessionToken,omitempty" example:""`
	// Endpoint customizado (opcional, para LocalStack ou outros)
	Endpoint string `json:"endpoint,omitempty" example:"http://localhost:4566"`
	// Profile do AWS CLI (opcional, usa credenciais do ~/.aws/credentials)
	Profile string `json:"profile,omitempty" example:"default"`
}

// AWSConfigResponse representa a resposta da configuração AWS
type AWSConfigResponse struct {
	// Região configurada
	Region string `json:"region"`
	// Indica se as credenciais foram configuradas
	Configured bool `json:"configured"`
	// Endpoint customizado (se configurado)
	Endpoint string `json:"endpoint,omitempty"`
	// Profile usado (se configurado)
	Profile string `json:"profile,omitempty"`
	// Mensagem de status
	Message string `json:"message"`
}

// ---- Helper Functions ----

// NewSuccessResponse cria uma resposta de sucesso
func NewSuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse cria uma resposta de erro
func NewErrorResponse(code, message, details string) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// ToPlanResponse converte PlanResult para PlanResponse
func ToPlanResponse(result *core.PlanResult) PlanResponse {
	resp := PlanResponse{
		ToCreate: make([]PlanItemResponse, 0),
		ToUpdate: make([]PlanItemResponse, 0),
		ToDelete: make([]PlanItemResponse, 0),
		NoChange: make([]PlanItemResponse, 0),
	}

	for _, item := range result.ToCreate {
		resp.ToCreate = append(resp.ToCreate, PlanItemResponse{
			Kind:      item.Kind,
			Name:      item.Name,
			Namespace: item.Namespace,
			Action:    item.Action,
			Changes:   item.Changes,
		})
	}

	for _, item := range result.ToUpdate {
		resp.ToUpdate = append(resp.ToUpdate, PlanItemResponse{
			Kind:      item.Kind,
			Name:      item.Name,
			Namespace: item.Namespace,
			Action:    item.Action,
			Changes:   item.Changes,
		})
	}

	for _, item := range result.ToDelete {
		resp.ToDelete = append(resp.ToDelete, PlanItemResponse{
			Kind:      item.Kind,
			Name:      item.Name,
			Namespace: item.Namespace,
			Action:    item.Action,
		})
	}

	for _, item := range result.NoChange {
		resp.NoChange = append(resp.NoChange, PlanItemResponse{
			Kind:      item.Kind,
			Name:      item.Name,
			Namespace: item.Namespace,
			Action:    item.Action,
		})
	}

	resp.Summary = PlanSummary{
		Create:   len(resp.ToCreate),
		Update:   len(resp.ToUpdate),
		Delete:   len(resp.ToDelete),
		NoChange: len(resp.NoChange),
	}

	return resp
}

// ToApplyResponse converte ApplyResult para ApplyResponse
func ToApplyResponse(result *core.ApplyResult) ApplyResponse {
	resp := ApplyResponse{
		Created: make([]ResourceResponse, 0),
		Updated: make([]ResourceResponse, 0),
		Failed:  make([]ResourceResponse, 0),
		Skipped: make([]ResourceResponse, 0),
	}

	for _, r := range result.Created {
		resp.Created = append(resp.Created, ResourceResponse{
			Kind:         r.Kind,
			Name:         r.Name,
			Namespace:    r.Namespace,
			AWSResources: r.AWSResources,
		})
	}

	for _, r := range result.Updated {
		resp.Updated = append(resp.Updated, ResourceResponse{
			Kind:         r.Kind,
			Name:         r.Name,
			Namespace:    r.Namespace,
			AWSResources: r.AWSResources,
		})
	}

	for _, r := range result.Failed {
		resp.Failed = append(resp.Failed, ResourceResponse{
			Kind:  r.Kind,
			Name:  r.Name,
			Error: r.Error,
		})
	}

	for _, r := range result.Skipped {
		resp.Skipped = append(resp.Skipped, ResourceResponse{
			Kind:    r.Kind,
			Name:    r.Name,
			Message: r.Message,
		})
	}

	resp.Summary = ApplySummary{
		Created: len(resp.Created),
		Updated: len(resp.Updated),
		Failed:  len(resp.Failed),
		Skipped: len(resp.Skipped),
	}

	return resp
}

// ToDeleteResponse converte DeleteResult para DeleteResponse
func ToDeleteResponse(result *core.DeleteResult) DeleteResponse {
	resp := DeleteResponse{
		Deleted: make([]ResourceResponse, 0),
		Failed:  make([]ResourceResponse, 0),
		Skipped: make([]ResourceResponse, 0),
	}

	for _, r := range result.Deleted {
		resp.Deleted = append(resp.Deleted, ResourceResponse{
			Kind:      r.Kind,
			Name:      r.Name,
			Namespace: r.Namespace,
		})
	}

	for _, r := range result.Failed {
		resp.Failed = append(resp.Failed, ResourceResponse{
			Kind:  r.Kind,
			Name:  r.Name,
			Error: r.Error,
		})
	}

	for _, r := range result.Skipped {
		resp.Skipped = append(resp.Skipped, ResourceResponse{
			Kind:    r.Kind,
			Name:    r.Name,
			Message: r.Message,
		})
	}

	resp.Summary = DeleteSummary{
		Deleted: len(resp.Deleted),
		Failed:  len(resp.Failed),
		Skipped: len(resp.Skipped),
	}

	return resp
}

// ToGetResponse converte lista de ResourceState para GetResponse
func ToGetResponse(states []*core.ResourceState) GetResponse {
	resp := GetResponse{
		Resources: make([]ResourceStateResponse, 0, len(states)),
		Count:     len(states),
	}

	for _, s := range states {
		resp.Resources = append(resp.Resources, ResourceStateResponse{
			APIVersion:   s.APIVersion,
			Kind:         s.Kind,
			Name:         s.Name,
			Namespace:    s.Namespace,
			AWSResources: s.AWSResources,
			Status:       s.Status,
			CreatedAt:    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return resp
}
