# Guia de Desenvolvimento - Infra Operator

Este guia cobre desenvolvimento local, testes e fluxos de trabalho de deployment para o infra-operator.

## Índice

1. [Pré-requisitos](#pré-requisitos)
2. [Configuração de Desenvolvimento Local](#configuração-de-desenvolvimento-local)
3. [Fluxos de Trabalho de Desenvolvimento](#fluxos-de-trabalho-de-desenvolvimento)
4. [Testes](#testes)
5. [Deployment](#deployment)
6. [Troubleshooting](#troubleshooting)

## Pré-requisitos

### Ferramentas Necessárias

**Comando:**

```bash
# Verificar se as ferramentas estão instaladas
go version          # Go 1.21+
kubectl version     # Kubernetes CLI
docker --version    # Docker ou OrbStack
task --version      # Task (go-task) ferramenta de automação
```

### Instalando Ferramentas Faltantes

**Comando:**

```bash
# macOS (usando Homebrew)
brew install go
brew install kubectl
brew install go-task
brew install jq

# Instalar OrbStack (alternativa ao Docker para macOS)
# Download de: https://orbstack.dev/

# Ou usar Docker Desktop
# Download de: https://www.docker.com/products/docker-desktop
```

### Cluster Kubernetes

Você precisa de um cluster Kubernetes local. Opções:

- **OrbStack** (recomendado para macOS) - Kubernetes integrado
- **Docker Desktop** - Habilitar Kubernetes nas configurações
- **kind** - `brew install kind && kind create cluster`
- **minikube** - `brew install minikube && minikube start`

**Verificar cluster:**

```bash
kubectl cluster-info
kubectl get nodes
```

## Configuração de Desenvolvimento Local

### 1. Configuração Inicial

Execute a configuração completa (necessário apenas uma vez):

**Comando:**

```bash
# Clonar/navegar para o projeto
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# Executar setup (instala ferramentas, inicia LocalStack)
task setup
```

Isso irá:
- Verificar se todas as ferramentas necessárias estão instaladas
- Iniciar LocalStack para simulação AWS
- Inicializar LocalStack com recursos de teste

### 2. Verificar LocalStack

**Comando:**

```bash
# Verificar saúde do LocalStack
task localstack:health

# Listar serviços
curl http://localhost:4566/_localstack/health | jq

# Testar AWS CLI contra LocalStack
task localstack:aws -- s3 ls
task localstack:aws -- sqs list-queues
task localstack:aws -- dynamodb list-tables
```

### 3. Variáveis de Ambiente

Verifique o arquivo `.env` (criado a partir de `.env.example`):

```bash
cat .env
```

Variáveis principais:
- `AWS_ENDPOINT_URL=http://localhost:4566` - Endpoint do LocalStack
- `AWS_REGION=us-east-1` - Região padrão
- `AWS_ACCESS_KEY_ID=test` - Credenciais do LocalStack
- `AWS_SECRET_ACCESS_KEY=test`

## Fluxos de Trabalho de Desenvolvimento

### Fluxo 1: Desenvolvimento Local (Sem Kubernetes)

Execute o operator localmente sem fazer deploy no Kubernetes. Útil para iterações rápidas e debugging.

**Comando:**

```bash
# Iniciar modo de desenvolvimento
task dev
```

Isso irá:
1. Iniciar LocalStack (se não estiver rodando)
2. Gerar código (se controller-gen estiver disponível)
3. Executar o binário do operator com `--clean-arch=true`

O operator usará seu kubeconfig local para conectar ao cluster mas criará recursos AWS no LocalStack.

**Logs serão transmitidos em tempo real.**

Pressione `Ctrl+C` para parar.

### Fluxo 2: Executar Contra o Cluster (Recomendado)

Execute o operator localmente mas reconcilie recursos no seu cluster Kubernetes:

**Comando:**

```bash
# Iniciar operator localmente, observando recursos do cluster
task run:local
```

Este modo:
- Executa operator na sua máquina (não in-cluster)
- Observa recursos Kubernetes (S3Buckets, AWSProviders, etc.)
- Cria recursos AWS no LocalStack
- Loga em `/tmp/log.txt` e console

**Em outro terminal, aplicar recursos de teste:**

**Comando:**

```bash
# Aplicar samples
kubectl apply -f config/samples/awsprovider_sample.yaml
kubectl apply -f config/samples/s3bucket_sample.yaml

# Observar status
kubectl get awsproviders -w
kubectl get s3buckets -w

# Verificar detalhes
kubectl describe s3bucket my-app-bucket
```

### Fluxo 3: Deployment Completo no Cluster

Faça deploy do operator como um Pod no Kubernetes (similar à produção):

**Comando:**

```bash
# Fluxo de deployment completo
task dev:full
```

Isso irá:
1. Executar testes unitários
2. Construir imagem Docker
3. Deploy no Kubernetes
4. Instalar CRDs
5. Aplicar recursos de exemplo
6. Mostrar status

**Monitorar o deployment:**

**Comando:**

```bash
# Visualizar logs do operator
task k8s:logs

# Verificar status
task k8s:status

# Visualizar samples
task samples:status
```

### Fluxo 4: Rebuild Rápido

Após fazer mudanças no código, reconstruir e fazer redeploy rapidamente:

**Comando:**

```bash
# Reconstruir imagem e reiniciar operator
task dev:quick
```

Isso é mais rápido que `dev:full` porque pula os testes e apenas reconstrói a imagem.

## Testes

### Testes Unitários

Execute testes unitários puros (sem Kubernetes, sem AWS):

**Comando:**

```bash
# Executar testes unitários com cobertura
task test:unit
```

Os testes estão em:
- `internal/domain/s3/bucket_test.go` - Testes de lógica de domínio
- Adicione mais testes em arquivos `*_test.go`

**Relatório de cobertura:**

```bash
# Visualizar cobertura detalhada
go tool cover -html=coverage.out
```

### Testes de Integração

Testar contra LocalStack (simulação AWS):

**Comando:**

```bash
# Executar testes de integração
task test:integration
```

Testes de integração:
- Usam endpoint LocalStack
- Criam buckets S3 reais, filas SQS, etc. (localmente)
- Testam interações com AWS SDK
- Localizados em `test/integration/`

### Testes End-to-End

Testes E2E completos com operator deployado no cluster:

**Comando:**

```bash
# Executar suite completa de testes E2E
task test:e2e
```

Isso irá:
1. Deploy do operator no cluster
2. Iniciar LocalStack
3. Aplicar fixtures de teste de `test/e2e/fixtures/`
4. Verificar que recursos são criados
5. Mostrar status

**Fixtures de Teste E2E:**
- `test/e2e/fixtures/01-awsprovider.yaml` - Provider LocalStack
- `test/e2e/fixtures/02-s3bucket.yaml` - Buckets de teste

### Executar Todos os Testes

**Comando:**

```bash
# Executar testes unitários, de integração e E2E
task test:all
```

## Deployment

### Deploy em Cluster Local

**Comando:**

```bash
# Instalar CRDs
task k8s:install-crds

# Deploy do operator
task k8s:deploy

# Verificar
task k8s:status
```

### Aplicar Samples

**Comando:**

```bash
# Criar recursos de exemplo (AWSProvider + S3Bucket)
task samples:apply

# Verificar status
task samples:status

# Visualizar status detalhado
kubectl describe s3bucket -n default
```

### Visualizar Logs

**Comando:**

```bash
# Stream de logs do operator
task k8s:logs

# Ou diretamente com kubectl
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager \
  -f --tail=100
```

### Reiniciar Operator

**Comando:**

```bash
# Reiniciar deployment do operator
task k8s:restart
```

### Fazer Undeploy

**Comando:**

```bash
# Remover operator (mantém CRDs)
task k8s:undeploy

# Remover tudo incluindo CRDs
task clean:all
```

## Construindo e Publicando

### Construir Binário

**Comando:**

```bash
# Construir binário do operator
task build
```

O binário estará em: `bin/manager`

### Construir Imagem Docker

**Comando:**

```bash
# Construir imagem com tag padrão
task docker:build
```

A imagem será tagueada como:
- `ttl.sh/infra-operator:dev-YYYYMMDD-HHMMSS`
- `ttl.sh/infra-operator:latest`

### Push para Registry

**Comando:**

```bash
# Construir e fazer push para ttl.sh (registry efêmero)
task docker:push
```

**Usando ttl.sh:**
- Imagens expiram automaticamente após 24 horas
- Sem autenticação necessária
- Perfeito para testes
- Formato: `ttl.sh/infra-operator:tag`

**Para produção, use um registry real:**

Atualizar `Taskfile.yaml`:

```yaml
vars:
  DOCKER_REGISTRY: ghcr.io/your-org
  # ou: docker.io/your-username
  # ou: your-registry.io
```

## Gerenciamento do LocalStack

### Iniciar/Parar LocalStack

**Comando:**

```bash
# Iniciar LocalStack
task localstack:start

# Parar LocalStack
task localstack:stop

# Reiniciar LocalStack
task localstack:restart

# Visualizar logs
task localstack:logs
```

### Health Check

**Comando:**

```bash
# Verificar saúde do LocalStack
task localstack:health

# Ou manualmente
curl http://localhost:4566/_localstack/health | jq
```

### Executar Comandos AWS CLI

**Comando:**

```bash
# Listar buckets S3 no LocalStack
task localstack:aws -- s3 ls

# Criar um bucket de teste
task localstack:aws -- s3 mb s3://test-bucket

# Listar filas SQS
task localstack:aws -- sqs list-queues

# Descrever tabela DynamoDB
task localstack:aws -- dynamodb describe-table --table-name test-table
```

## Geração de Código

### Gerar DeepCopy e CRDs

**Comando:**

```bash
# Gerar código (requer controller-gen)
task generate
```

**Instalar controller-gen:**

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

Depois atualize `Taskfile.yaml` para descomentar a linha controller-gen.

### Formatar Código

**Comando:**

```bash
# Formatar todo código Go
task fmt
```

### Fazer Lint do Código

**Comando:**

```bash
# Executar linter
task lint
```

**Instalar golangci-lint:**

```bash
brew install golangci-lint
```

## Troubleshooting

### Operator Não Está Reconciliando

**Verificar se operator está rodando:**

```bash
task k8s:status
kubectl get pods -n infra-operator-system
```

**Verificar logs por erros:**

```bash
task k8s:logs
```

**Problemas comuns:**
- AWSProvider não Ready - verificar credenciais
- CRDs não instalados - executar `task k8s:install-crds`
- Problemas de RBAC - verificar `config/rbac/`

### LocalStack Não Está Funcionando

**Verificar se LocalStack está rodando:**

```bash
docker ps | grep localstack
task localstack:health
```

**Reiniciar LocalStack:**

```bash
task localstack:restart
```

**Verificar logs do LocalStack:**

```bash
task localstack:logs
```

**Problemas comuns:**
- Porta 4566 já em uso - parar serviços conflitantes
- Docker não está rodando - iniciar Docker/OrbStack
- Script de init falhou - verificar `hack/localstack-init.sh`

### S3Bucket Preso em NotReady

**Verificar AWSProvider:**

```bash
kubectl get awsproviders
kubectl describe awsprovider localstack
```

**Verificar eventos do S3Bucket:**

```bash
kubectl describe s3bucket my-app-bucket
```

**Verificar se bucket existe no LocalStack:**

```bash
task localstack:aws -- s3 ls
```

**Problemas comuns:**
- Provider não está ready - aguardar AWSProvider reconciliar
- Nome do bucket não é único - alterar nome do bucket
- LocalStack não acessível - verificar URL do endpoint

### Recursos Não Estão Deletando

**Verificar finalizers:**

```bash
kubectl get s3bucket my-bucket -o yaml | grep finalizers -A 5
```

**Forçar deleção (se preso):**

```bash
kubectl patch s3bucket my-bucket \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

kubectl delete s3bucket my-bucket
```

### Testes Unitários Falhando

**Executar testes com output verbose:**

```bash
go test -v -race ./internal/...
```

**Executar teste específico:**

```bash
go test -v -run TestBucket_Validate ./internal/domain/s3/
```

**Problemas comuns:**
- Caminhos de import errados - verificar nome do módulo em `go.mod`
- Dependências faltando - executar `go mod download`

### Testes de Integração Falhando

**Garantir que LocalStack está rodando:**

```bash
task localstack:start
task localstack:health
```

**Verificar AWS_ENDPOINT_URL:**

```bash
echo $AWS_ENDPOINT_URL  # Deve ser http://localhost:4566
```

**Executar testes de integração com debug:**

```bash
export AWS_SDK_LOG_LEVEL=debug
task test:integration
```

## Dicas de Desenvolvimento

### Loop de Desenvolvimento Mais Rápido

1. **Use `task run:local`** ao invés de deployment completo no cluster
2. **Mantenha LocalStack rodando** entre execuções de teste
3. **Use `deletionPolicy: Delete`** em recursos de teste para auto-cleanup
4. **Observe logs em terminal separado** com `task k8s:logs`

### Debugging

**Habilitar logging de debug:**

Atualizar `cmd/main.go`:

```go
opts := zap.Options{
Development: true,
// Adicionar:
Level: zapcore.DebugLevel,
}
```

Ou executar com flag:

```bash
go run ./cmd/main.go --zap-log-level=debug
```

**Usar debugger delve:**

```bash
# Instalar delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debugar operator
dlv debug ./cmd/main.go -- --clean-arch=true
```

### Início Limpo

Remover todos os recursos e começar do zero:

**Comando:**

```bash
# Limpar tudo
task clean:all

# Começar do zero
task setup
task dev:full
```

## Comandos Task Comuns

Referência rápida dos comandos mais usados:

**Comando:**

```bash
# Setup
task setup                    # Configuração inicial
task install-tools            # Verificar ferramentas

# Desenvolvimento
task dev                      # Dev local (sem K8s)
task run:local                # Executar contra cluster
task dev:full                 # Deployment completo
task dev:quick                # Rebuild rápido

# Testes
task test:unit                # Testes unitários
task test:integration         # Testes de integração
task test:e2e                 # Testes E2E
task test:all                 # Todos os testes

# Build
task build                    # Construir binário
task docker:build             # Construir imagem
task docker:push              # Push da imagem

# Kubernetes
task k8s:install-crds         # Instalar CRDs
task k8s:deploy               # Deploy do operator
task k8s:status               # Mostrar status
task k8s:logs                 # Stream de logs
task k8s:restart              # Reiniciar operator
task k8s:undeploy             # Remover operator

# Samples
task samples:apply            # Aplicar samples
task samples:status           # Verificar samples
task samples:delete           # Remover samples

# LocalStack
task localstack:start         # Iniciar LocalStack
task localstack:stop          # Parar LocalStack
task localstack:health        # Health check
task localstack:logs          # Visualizar logs
task localstack:aws -- CMD    # AWS CLI

# Cleanup
task clean                    # Limpar arquivos temporários
task clean:all                # Limpar tudo

# Ajuda
task help                     # Mostrar todas as tasks
task --list                   # Listar tasks
```

## Próximos Passos

1. **Implementar mais controllers** - Usar S3 como template para Lambda, DynamoDB, etc.
2. **Adicionar webhooks** - Webhooks de validação e mutação
3. **Adicionar métricas** - Export de métricas Prometheus
4. **Melhorar testes** - Mais testes de integração e E2E
5. **Deployment em produção** - Helm chart, configuração IRSA

## Recursos

- [Documentação Kubebuilder](https://book.kubebuilder.io/)
- [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Documentação LocalStack](https://docs.localstack.cloud/)
- [Documentação Task](https://taskfile.dev/)
- [Guia de Clean Architecture](/advanced/clean-architecture)
