package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"infra-operator/pkg/core"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "infra-operator/pkg/api/docs" // Swagger docs
)

// ServerConfig configuração do servidor API
type ServerConfig struct {
	Port           int
	Host           string
	Version        string
	StateDir       string
	AllowedOrigins []string
	Auth           AuthConfig
	AWS            core.AWSConfig
}

// Server representa o servidor HTTP da API
type Server struct {
	config   *ServerConfig
	router   *chi.Mux
	engine   *core.Engine
	handlers *Handlers
	server   *http.Server
}

// NewServer cria um novo servidor API
func NewServer(config *ServerConfig) (*Server, error) {
	ctx := context.Background()

	// Cria provider config
	provider := &core.ProviderConfig{
		Name: "api",
		AWS:  config.AWS,
	}

	// Se não tem região configurada, usa env
	if provider.AWS.Region == "" {
		envProvider := core.NewProviderConfigFromEnv()
		provider.AWS = envProvider.AWS
	}

	// Cria engine com output silencioso (API não precisa de output para stdout)
	engine, err := core.NewEngine(ctx, core.EngineConfig{
		StateDir: config.StateDir,
		Provider: provider,
		Output:   &core.SilentOutputWriter{},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar engine: %w", err)
	}

	s := &Server{
		config: config,
		router: chi.NewRouter(),
		engine: engine,
	}

	s.handlers = NewHandlers(engine, config)
	s.setupRoutes()

	return s, nil
}

// setupRoutes configura as rotas da API
func (s *Server) setupRoutes() {
	r := s.router

	// Middlewares globais
	r.Use(RequestID)
	r.Use(Logger)
	r.Use(Recoverer)
	r.Use(ContentTypeJSON)

	// CORS
	if len(s.config.AllowedOrigins) > 0 {
		r.Use(CORS(s.config.AllowedOrigins))
	}

	// Autenticação
	r.Use(APIKeyAuth(s.config.Auth))

	// Rotas públicas (sem auth se habilitado)
	r.Get("/health", s.handlers.Health)
	r.Get("/", s.handleRoot)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Health
		r.Get("/health", s.handlers.Health)

		// Operações principais
		r.Post("/plan", s.handlers.Plan)
		r.Post("/apply", s.handlers.Apply)
		r.Delete("/resources", s.handlers.Delete)
		r.Post("/delete", s.handlers.Delete) // Alternativa POST para clientes que não suportam DELETE com body

		// Listagem de recursos
		r.Get("/resources", s.handlers.Get)
		r.Get("/resources/{kind}", s.handlers.GetByKind)

		// Configuração AWS
		r.Post("/config/aws", s.handlers.ConfigureAWS)
		r.Get("/config/aws", s.handlers.GetAWSConfig)

		// Aliases por tipo de recurso
		r.Get("/vpcs", s.makeKindHandler("VPC"))
		r.Get("/subnets", s.makeKindHandler("Subnet"))
		r.Get("/security-groups", s.makeKindHandler("SecurityGroup"))
		r.Get("/ec2-instances", s.makeKindHandler("EC2Instance"))
		r.Get("/compute-stacks", s.makeKindHandler("ComputeStack"))
		r.Get("/internet-gateways", s.makeKindHandler("InternetGateway"))
		r.Get("/route-tables", s.makeKindHandler("RouteTable"))
	})
}

// handleRoot retorna informações básicas da API
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":    "Infra Operator AWS API",
		"version": s.config.Version,
		"swagger": "/swagger/index.html",
		"health":  "/health",
		"endpoints": map[string]string{
			"plan":      "POST /api/v1/plan",
			"apply":     "POST /api/v1/apply",
			"delete":    "DELETE /api/v1/resources",
			"resources": "GET /api/v1/resources",
		},
	}
	writeJSON(w, http.StatusOK, NewSuccessResponse(info))
}

// makeKindHandler cria um handler para um tipo específico de recurso
func (s *Server) makeKindHandler(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		states, err := s.engine.Get(r.Context(), kind)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, NewErrorResponse("GET_FAILED", err.Error(), ""))
			return
		}
		writeJSON(w, http.StatusOK, NewSuccessResponse(ToGetResponse(states)))
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Iniciando servidor API em http://%s", addr)
	log.Printf("  Swagger UI: http://%s/swagger/index.html", addr)
	log.Printf("  Endpoints:")
	log.Printf("    POST   /api/v1/plan      - Gera plano de execução")
	log.Printf("    POST   /api/v1/apply     - Aplica recursos")
	log.Printf("    DELETE /api/v1/resources - Deleta recursos")
	log.Printf("    GET    /api/v1/resources - Lista recursos")

	return s.server.ListenAndServe()
}

// Shutdown para o servidor graciosamente
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Router retorna o router Chi (para testes)
func (s *Server) Router() *chi.Mux {
	return s.router
}
