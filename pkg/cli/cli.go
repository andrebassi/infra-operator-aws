package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// CLI representa a interface de linha de comando
type CLI struct {
	stateDir string
	region   string
	endpoint string
	dryRun   bool
	verbose  bool
}

// NewCLI cria uma nova instância do CLI
func NewCLI(stateDir, region, endpoint string, dryRun, verbose bool) *CLI {
	return &CLI{
		stateDir: stateDir,
		region:   region,
		endpoint: endpoint,
		dryRun:   dryRun,
		verbose:  verbose,
	}
}

// RunApply executa o comando apply
func (c *CLI) RunApply(ctx context.Context, files []string) error {
	resources, provider, err := c.loadResources(files)
	if err != nil {
		return err
	}

	executor, err := NewExecutor(c.stateDir, provider, c.dryRun, c.verbose)
	if err != nil {
		return fmt.Errorf("falha ao criar executor: %w", err)
	}

	fmt.Printf("\n=== Aplicando recursos de %d arquivo(s) ===\n\n", len(files))
	return executor.Apply(ctx, resources)
}

// RunPlan executa o comando plan
func (c *CLI) RunPlan(ctx context.Context, files []string) error {
	resources, provider, err := c.loadResources(files)
	if err != nil {
		return err
	}

	executor, err := NewExecutor(c.stateDir, provider, true, c.verbose)
	if err != nil {
		return fmt.Errorf("falha ao criar executor: %w", err)
	}

	return executor.Plan(ctx, resources)
}

// RunDelete executa o comando delete
func (c *CLI) RunDelete(ctx context.Context, files []string) error {
	resources, provider, err := c.loadResources(files)
	if err != nil {
		return err
	}

	executor, err := NewExecutor(c.stateDir, provider, c.dryRun, c.verbose)
	if err != nil {
		return fmt.Errorf("falha ao criar executor: %w", err)
	}

	fmt.Printf("\n=== Deletando recursos de %d arquivo(s) ===\n\n", len(files))
	return executor.Delete(ctx, resources)
}

// RunGet lista recursos
func (c *CLI) RunGet(ctx context.Context, kind string) error {
	stateManager := NewStateManager(c.stateDir)

	var states []*ResourceState
	var err error

	if kind == "" || kind == "all" {
		states, err = stateManager.ListAllStates()
	} else {
		states, err = stateManager.ListStates(kind)
	}

	if err != nil {
		return fmt.Errorf("falha ao listar recursos: %w", err)
	}

	if len(states) == 0 {
		fmt.Println("Nenhum recurso encontrado no estado")
		return nil
	}

	fmt.Printf("\n%-20s %-30s %-40s %s\n", "KIND", "NAME", "AWS ID", "STATUS")
	fmt.Println(strings.Repeat("-", 100))

	for _, s := range states {
		awsID := ""
		if len(s.AWSResources) > 0 {
			// Obtém o primeiro/principal AWS ID
			for _, v := range s.AWSResources {
				awsID = v
				break
			}
		}
		status := "unknown"
		if ready, ok := s.Status["ready"].(bool); ok && ready {
			status = "Ready"
		} else if phase, ok := s.Status["phase"].(string); ok {
			status = phase
		}
		fmt.Printf("%-20s %-30s %-40s %s\n", s.Kind, s.Name, awsID, status)
	}

	return nil
}

// loadResources carrega recursos dos arquivos e extrai configuração do provider
func (c *CLI) loadResources(files []string) ([]Resource, *ProviderConfig, error) {
	var allResources []Resource

	for _, file := range files {
		resources, err := ParseFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("falha ao fazer parse de %s: %w", file, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(allResources) == 0 {
		return nil, nil, fmt.Errorf("nenhum recurso encontrado nos arquivos")
	}

	// Encontra configuração do provider
	provider := NewProviderConfigFromEnv()

	// Sobrescreve com flags do CLI
	if c.region != "" {
		provider.AWS.Region = c.region
	}
	if c.endpoint != "" {
		provider.AWS.Endpoint = c.endpoint
	}

	// Verifica recurso AWSProvider nos arquivos
	for _, r := range allResources {
		if r.Kind == "AWSProvider" {
			fileProvider := NewProviderConfigFromResource(r)
			// Merge - provider do arquivo tem precedência, mas flags do CLI sobrescrevem
			if provider.AWS.Region == "" || provider.AWS.Region == "us-east-1" {
				provider.AWS.Region = fileProvider.AWS.Region
			}
			if provider.AWS.Endpoint == "" {
				provider.AWS.Endpoint = fileProvider.AWS.Endpoint
			}
			break
		}
	}

	return allResources, provider, nil
}

// PrintUsage imprime o uso do CLI
func PrintUsage() {
	fmt.Println(`
Infra Operator AWS - Modo CLI

Uso:
  infra-operator apply  -f <arquivo.yaml>  [flags]    Aplica recursos do YAML
  infra-operator plan   -f <arquivo.yaml>  [flags]    Mostra plano de execução
  infra-operator delete -f <arquivo.yaml>  [flags]    Deleta recursos
  infra-operator get    [kind]                        Lista recursos no estado

Flags:
  -f, --file string        Caminho para arquivo de manifesto YAML (pode ser repetido)
  --region string          Região AWS (padrão: us-east-1 ou env AWS_REGION)
  --endpoint string        URL do endpoint AWS (para LocalStack)
  --state-dir string       Diretório de estado (padrão: ~/.infra-operator/state)
  --dry-run                Mostra o que seria feito sem fazer mudanças
  -v, --verbose            Saída detalhada

Exemplos:
  # Aplica um ComputeStack dos samples
  infra-operator apply -f samples/29-computestack.yaml

  # Planeja mudanças (dry-run)
  infra-operator plan -f samples/29-computestack.yaml

  # Aplica com endpoint LocalStack
  infra-operator apply -f samples/29-computestack.yaml --endpoint http://localhost:4566

  # Lista todos os recursos no estado
  infra-operator get

  # Lista apenas ComputeStacks
  infra-operator get ComputeStack

  # Deleta recursos
  infra-operator delete -f samples/29-computestack.yaml

Variáveis de Ambiente:
  AWS_REGION              Região AWS
  AWS_ACCESS_KEY_ID       Chave de acesso AWS
  AWS_SECRET_ACCESS_KEY   Chave secreta AWS
  AWS_ENDPOINT_URL        Endpoint AWS (para LocalStack)
  AWS_PROFILE             Nome do perfil AWS
`)
}

// IsCliCommand verifica se os argumentos indicam modo CLI
func IsCliCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}
	cmd := args[1]
	return cmd == "apply" || cmd == "plan" || cmd == "delete" || cmd == "get" || cmd == "help" || cmd == "--help" || cmd == "-h"
}

// Run executa o CLI baseado nos argumentos da linha de comando
func Run(args []string) error {
	if len(args) < 2 {
		PrintUsage()
		return nil
	}

	cmd := args[1]
	ctx := context.Background()

	// Parse das flags
	var files []string
	var region, endpoint, stateDir string
	var dryRun, verbose bool
	kind := ""

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-f" || arg == "--file":
			if i+1 < len(args) {
				files = append(files, args[i+1])
				i++
			}
		case arg == "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case arg == "--endpoint":
			if i+1 < len(args) {
				endpoint = args[i+1]
				i++
			}
		case arg == "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case arg == "--dry-run":
			dryRun = true
		case arg == "-v" || arg == "--verbose":
			verbose = true
		case !strings.HasPrefix(arg, "-"):
			kind = arg
		}
	}

	cli := NewCLI(stateDir, region, endpoint, dryRun, verbose)

	switch cmd {
	case "apply":
		if len(files) == 0 {
			return fmt.Errorf("nenhum arquivo especificado. Use -f <arquivo.yaml>")
		}
		return cli.RunApply(ctx, files)

	case "plan":
		if len(files) == 0 {
			return fmt.Errorf("nenhum arquivo especificado. Use -f <arquivo.yaml>")
		}
		return cli.RunPlan(ctx, files)

	case "delete":
		if len(files) == 0 {
			return fmt.Errorf("nenhum arquivo especificado. Use -f <arquivo.yaml>")
		}
		return cli.RunDelete(ctx, files)

	case "get":
		return cli.RunGet(ctx, kind)

	case "help", "--help", "-h":
		PrintUsage()
		return nil

	default:
		// Não é um comando CLI, retorna para modo operator
		return fmt.Errorf("comando desconhecido: %s", cmd)
	}
}

// Main é o ponto de entrada do CLI
func Main() {
	if err := Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
