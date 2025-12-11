package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"infra-operator/pkg/core"

	"github.com/go-chi/chi/v5"
)

// Nota: 'strings' ainda é usado em parseResources para Content-Type

// Handlers contém os handlers da API
type Handlers struct {
	engine *core.Engine
	config *ServerConfig
}

// NewHandlers cria uma nova instância de handlers
func NewHandlers(engine *core.Engine, config *ServerConfig) *Handlers {
	return &Handlers{
		engine: engine,
		config: config,
	}
}

// Health godoc
// @Summary      Health check
// @Description  Retorna o status de saúde da API
// @Tags         health
// @Produce      json
// @Success      200  {object}  APIResponse{data=HealthResponse}
// @Router       /health [get]
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "healthy",
		Version:   h.config.Version,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, NewSuccessResponse(resp))
}

// Plan godoc
// @Summary      Gerar plano de execução
// @Description  Analisa os recursos e retorna o que será criado, atualizado ou deletado
// @Description
// @Description  **Exemplo de request (VPC simples):**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "VPC",
// @Description      "metadata": {"name": "minha-vpc"},
// @Description      "spec": {"cidrBlock": "10.0.0.0/16"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo ComputeStack (infraestrutura completa):**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "ComputeStack",
// @Description      "metadata": {"name": "meu-ambiente"},
// @Description      "spec": {
// @Description        "vpcCIDR": "10.100.0.0/16",
// @Description        "bastionInstance": {
// @Description          "enabled": true,
// @Description          "instanceType": "t3.micro"
// @Description        }
// @Description      }
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo Subnet:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "Subnet",
// @Description      "metadata": {"name": "public-subnet-1"},
// @Description      "spec": {"vpcId": "vpc-0abc123def456", "cidrBlock": "10.0.1.0/24", "availabilityZone": "us-east-1a", "mapPublicIpOnLaunch": true}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo SecurityGroup:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "SecurityGroup",
// @Description      "metadata": {"name": "web-sg"},
// @Description      "spec": {"vpcId": "vpc-0abc123def456", "description": "Security group para web servers", "ingressRules": [{"protocol": "tcp", "fromPort": 443, "toPort": 443, "cidrIp": "0.0.0.0/0"}]}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo EC2Instance:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "EC2Instance",
// @Description      "metadata": {"name": "web-server"},
// @Description      "spec": {"instanceName": "web-server-1", "instanceType": "t3.micro", "imageId": "ami-0c55b159cbfafe1f0", "subnetId": "subnet-0abc123", "securityGroupIds": ["sg-0abc123"], "keyName": "my-keypair"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo S3Bucket:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "S3Bucket",
// @Description      "metadata": {"name": "my-bucket"},
// @Description      "spec": {"bucketName": "my-company-data-bucket", "versioning": true}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo RDSInstance:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "RDSInstance",
// @Description      "metadata": {"name": "app-postgres"},
// @Description      "spec": {"dbInstanceIdentifier": "app-postgres", "dbInstanceClass": "db.t3.micro", "engine": "postgres", "engineVersion": "15.4", "allocatedStorage": 20, "masterUsername": "admin"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo DynamoDBTable:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "DynamoDBTable",
// @Description      "metadata": {"name": "users-table"},
// @Description      "spec": {"tableName": "Users", "billingMode": "PAY_PER_REQUEST"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo SQSQueue:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "SQSQueue",
// @Description      "metadata": {"name": "orders-queue"},
// @Description      "spec": {"queueName": "orders-queue", "visibilityTimeout": 30, "fifoQueue": false}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo SNSTopic:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "SNSTopic",
// @Description      "metadata": {"name": "app-notifications"},
// @Description      "spec": {"topicName": "app-notifications", "displayName": "App Notifications"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo LambdaFunction:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "LambdaFunction",
// @Description      "metadata": {"name": "process-orders"},
// @Description      "spec": {"functionName": "process-orders", "runtime": "nodejs18.x", "handler": "index.handler", "role": "arn:aws:iam::123456789:role/lambda-role", "memorySize": 256, "timeout": 30}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo EKSCluster:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "EKSCluster",
// @Description      "metadata": {"name": "production"},
// @Description      "spec": {"name": "production", "version": "1.28", "roleArn": "arn:aws:iam::123456789:role/eks-cluster-role"}
// @Description    }]
// @Description  }
// @Description  ```
// @Description
// @Description  **Exemplo IAMRole:**
// @Description  ```json
// @Description  {
// @Description    "resources": [{
// @Description      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
// @Description      "kind": "IAMRole",
// @Description      "metadata": {"name": "lambda-execution-role"},
// @Description      "spec": {"roleName": "lambda-execution-role", "assumeRolePolicyDocument": "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}", "managedPolicyArns": ["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"]}
// @Description    }]
// @Description  }
// @Description  ```
// @Tags         operations
// @Accept       json
// @Produce      json
// @Param        body  body  PlanRequest  true  "Recursos para planejar"
// @Success      200  {object}  APIResponse{data=PlanResponse}
// @Failure      400  {object}  APIResponse
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /plan [post]
func (h *Handlers) Plan(w http.ResponseWriter, r *http.Request) {
	resources, err := h.parseResources(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("INVALID_REQUEST", err.Error(), ""))
		return
	}

	if len(resources) == 0 {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("NO_RESOURCES", "Nenhum recurso fornecido", ""))
		return
	}

	result, err := h.engine.Plan(r.Context(), resources)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("PLAN_FAILED", err.Error(), ""))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(ToPlanResponse(result)))
}

// Apply godoc
// @Summary      Aplicar recursos
// @Description  Cria ou atualiza recursos AWS conforme especificado
// @Tags         operations
// @Accept       json
// @Produce      json
// @Param        body  body  ApplyRequest  true  "Recursos para aplicar"
// @Success      200  {object}  APIResponse{data=ApplyResponse}
// @Failure      400  {object}  APIResponse
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /apply [post]
func (h *Handlers) Apply(w http.ResponseWriter, r *http.Request) {
	resources, err := h.parseResources(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("INVALID_REQUEST", err.Error(), ""))
		return
	}

	if len(resources) == 0 {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("NO_RESOURCES", "Nenhum recurso fornecido", ""))
		return
	}

	result, err := h.engine.Apply(r.Context(), resources)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("APPLY_FAILED", err.Error(), ""))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(ToApplyResponse(result)))
}

// Delete godoc
// @Summary      Deletar recursos
// @Description  Remove recursos AWS especificados
// @Tags         operations
// @Accept       json
// @Produce      json
// @Param        body  body  DeleteRequest  true  "Recursos para deletar"
// @Success      200  {object}  APIResponse{data=DeleteResponse}
// @Failure      400  {object}  APIResponse
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /resources [delete]
// @Router       /delete [post]
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	resources, err := h.parseResources(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("INVALID_REQUEST", err.Error(), ""))
		return
	}

	if len(resources) == 0 {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("NO_RESOURCES", "Nenhum recurso fornecido", ""))
		return
	}

	result, err := h.engine.Delete(r.Context(), resources)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("DELETE_FAILED", err.Error(), ""))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(ToDeleteResponse(result)))
}

// Get godoc
// @Summary      Listar recursos
// @Description  Lista todos os recursos do estado local
// @Tags         resources
// @Produce      json
// @Param        kind  query  string  false  "Filtrar por tipo (ex: VPC, ComputeStack)"
// @Success      200  {object}  APIResponse{data=GetResponse}
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /resources [get]
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if kind == "" {
		kind = r.URL.Query().Get("kind")
	}

	states, err := h.engine.Get(r.Context(), kind)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("GET_FAILED", err.Error(), ""))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(ToGetResponse(states)))
}

// GetByKind godoc
// @Summary      Listar recursos por tipo
// @Description  Lista recursos de um tipo específico
// @Tags         resources
// @Produce      json
// @Param        kind  path  string  true  "Tipo do recurso (ex: VPC, ComputeStack, EC2Instance)"
// @Success      200  {object}  APIResponse{data=GetResponse}
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /resources/{kind} [get]
func (h *Handlers) GetByKind(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")

	states, err := h.engine.Get(r.Context(), kind)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("GET_FAILED", err.Error(), ""))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(ToGetResponse(states)))
}

// parseResources extrai recursos da requisição
func (h *Handlers) parseResources(r *http.Request) ([]core.Resource, error) {
	contentType := r.Header.Get("Content-Type")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Se Content-Type é YAML ou texto, faz parse como YAML
	if strings.Contains(contentType, "yaml") || strings.Contains(contentType, "text/plain") {
		return core.ParseYAML(body)
	}

	// Tenta como JSON primeiro
	if strings.Contains(contentType, "json") || contentType == "" {
		// Tenta deserializar como array de recursos
		var resources []core.Resource
		if err := json.Unmarshal(body, &resources); err == nil && len(resources) > 0 {
			return resources, nil
		}

		// Tenta como objeto com campo resources
		var req struct {
			Resources []core.Resource `json:"resources"`
			YAML      string          `json:"yaml"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			if len(req.Resources) > 0 {
				return req.Resources, nil
			}
			if req.YAML != "" {
				return core.ParseYAML([]byte(req.YAML))
			}
		}

		// Tenta como recurso único
		var resource core.Resource
		if err := json.Unmarshal(body, &resource); err == nil && resource.Kind != "" {
			return []core.Resource{resource}, nil
		}
	}

	// Fallback: tenta como YAML
	resources, err := core.ParseYAML(body)
	if err == nil && len(resources) > 0 {
		return resources, nil
	}

	return nil, err
}

// ConfigureAWS godoc
// @Summary      Configurar credenciais AWS
// @Description  Configura as credenciais AWS para a API. Existem 3 formas de autenticar:
// @Description
// @Description  **1. Credenciais diretas (Access Key + Secret Key):**
// @Description  ```json
// @Description  {
// @Description    "region": "us-east-1",
// @Description    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
// @Description    "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
// @Description  }
// @Description  ```
// @Description
// @Description  **2. Profile do AWS CLI (~/.aws/credentials):**
// @Description  ```json
// @Description  {
// @Description    "region": "us-east-1",
// @Description    "profile": "my-profile"
// @Description  }
// @Description  ```
// @Description
// @Description  **3. LocalStack ou endpoint customizado:**
// @Description  ```json
// @Description  {
// @Description    "region": "us-east-1",
// @Description    "endpoint": "http://localhost:4566",
// @Description    "accessKeyId": "test",
// @Description    "secretAccessKey": "test"
// @Description  }
// @Description  ```
// @Description
// @Description  **4. Credenciais temporárias (STS):**
// @Description  ```json
// @Description  {
// @Description    "region": "us-east-1",
// @Description    "accessKeyId": "ASIAXXX...",
// @Description    "secretAccessKey": "xxx...",
// @Description    "sessionToken": "FwoGZXIvYXdzEBY..."
// @Description  }
// @Description  ```
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        body  body  AWSConfigRequest  true  "Configuração AWS"
// @Success      200  {object}  APIResponse{data=AWSConfigResponse}
// @Failure      400  {object}  APIResponse
// @Failure      500  {object}  APIResponse
// @Security     ApiKeyAuth
// @Router       /config/aws [post]
func (h *Handlers) ConfigureAWS(w http.ResponseWriter, r *http.Request) {
	var req AWSConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("INVALID_REQUEST", "JSON inválido", err.Error()))
		return
	}
	defer r.Body.Close()

	// Valida região
	if req.Region == "" {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse("MISSING_REGION", "Região AWS é obrigatória", ""))
		return
	}

	// Configura o provider
	awsConfig := core.AWSConfig{
		Region:          req.Region,
		AccessKeyID:     req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		SessionToken:    req.SessionToken,
		Endpoint:        req.Endpoint,
		Profile:         req.Profile,
	}

	// Recria o engine com as novas credenciais
	provider := &core.ProviderConfig{
		Name: "api",
		AWS:  awsConfig,
	}

	newEngine, err := core.NewEngine(r.Context(), core.EngineConfig{
		StateDir: h.config.StateDir,
		Provider: provider,
		Output:   &core.SilentOutputWriter{},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse("CONFIG_FAILED", "Falha ao configurar AWS", err.Error()))
		return
	}

	// Atualiza o engine
	h.engine = newEngine

	resp := AWSConfigResponse{
		Region:     req.Region,
		Configured: true,
		Endpoint:   req.Endpoint,
		Profile:    req.Profile,
		Message:    "Credenciais AWS configuradas com sucesso",
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(resp))
}

// GetAWSConfig godoc
// @Summary      Obter configuração AWS atual
// @Description  Retorna a configuração AWS atual (sem expor secrets)
// @Tags         config
// @Produce      json
// @Success      200  {object}  APIResponse{data=AWSConfigResponse}
// @Security     ApiKeyAuth
// @Router       /config/aws [get]
func (h *Handlers) GetAWSConfig(w http.ResponseWriter, r *http.Request) {
	// Obtém config atual do engine (sem expor secrets)
	resp := AWSConfigResponse{
		Region:     h.config.AWS.Region,
		Configured: h.config.AWS.AccessKeyID != "" || h.config.AWS.Profile != "",
		Endpoint:   h.config.AWS.Endpoint,
		Profile:    h.config.AWS.Profile,
		Message:    "Configuração atual",
	}

	if !resp.Configured {
		resp.Message = "Nenhuma credencial configurada. Use POST /api/v1/config/aws para configurar."
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(resp))
}

// ---- Helper Functions ----

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
