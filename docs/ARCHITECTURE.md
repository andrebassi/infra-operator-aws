# Infra Operator - Arquitetura e Design

## Introdução

Este documento explica a arquitetura do **infra-operator**, um Kubernetes Operator que provisiona recursos AWS usando o padrão de Custom Resources (CRs). O operator foi desenvolvido seguindo as melhores práticas da comunidade Kubernetes e utiliza o AWS SDK Go v2 para interação direta com os serviços AWS.

## Visão Geral da Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
│                                                                  │
│  ┌────────────┐                                                 │
│  │   User     │                                                 │
│  └─────┬──────┘                                                 │
│        │ kubectl apply -f s3bucket.yaml                         │
│        ▼                                                         │
│  ┌────────────────────────────────────┐                        │
│  │      Kubernetes API Server         │                        │
│  │  ┌──────────────────────────────┐  │                        │
│  │  │   Custom Resource (CR)        │  │                        │
│  │  │   - AWSProvider               │  │                        │
│  │  │   - S3Bucket                  │  │                        │
│  │  │   - RDSInstance               │  │                        │
│  │  └──────────────┬────────────────┘  │                        │
│  └─────────────────┼────────────────────┘                        │
│                    │ Watch                                       │
│                    ▼                                             │
│  ┌─────────────────────────────────────────────┐                │
│  │       Infra Operator Manager Pod            │                │
│  │  ┌──────────────────────────────────────┐   │                │
│  │  │   Controller Runtime Manager          │   │                │
│  │  │  ┌────────────────────────────────┐   │   │                │
│  │  │  │  AWSProvider Controller        │   │   │                │
│  │  │  └────────────┬───────────────────┘   │   │                │
│  │  │  ┌────────────▼───────────────────┐   │   │                │
│  │  │  │  S3Bucket Controller           │   │   │                │
│  │  │  └────────────┬───────────────────┘   │   │                │
│  │  │  ┌────────────▼───────────────────┐   │   │                │
│  │  │  │  RDSInstance Controller        │   │   │                │
│  │  │  └────────────┬───────────────────┘   │   │                │
│  │  │               │                        │   │                │
│  │  └───────────────┼────────────────────────┘   │                │
│  └──────────────────┼────────────────────────────┘                │
│                     │ AWS SDK Calls                              │
└─────────────────────┼────────────────────────────────────────────┘
                      │
                      ▼
        ┌─────────────────────────────────┐
        │         AWS Cloud               │
        │  ┌───────────┐  ┌────────────┐  │
        │  │    S3     │  │    RDS     │  │
        │  └───────────┘  └────────────┘  │
        │  ┌───────────┐  ┌────────────┐  │
        │  │   EC2     │  │    SQS     │  │
        │  └───────────┘  └────────────┘  │
        └─────────────────────────────────┘
```

## Componentes Principais

### 1. Custom Resource Definitions (CRDs)

As CRDs definem a estrutura dos recursos customizados que o operator gerencia.

#### AWSProvider CRD

**Propósito**: Centralizar credenciais e configurações AWS.

**Campos principais**:
- `region`: Região AWS (obrigatório)
- `roleARN`: ARN da role IAM para IRSA
- `accessKeyIDRef/secretAccessKeyRef`: Referências a Secret para credenciais estáticas
- `defaultTags`: Tags aplicadas a todos os recursos

**Status**:
- `ready`: Indica se as credenciais estão válidas
- `accountID`: ID da conta AWS
- `callerIdentity`: ARN da identidade autenticada
- `conditions`: Condições de status

**Arquivo**: `api/v1alpha1/awsprovider_types.go`

#### S3Bucket CRD

**Propósito**: Gerenciar buckets S3 com configuração completa.

**Campos principais**:
- `providerRef`: Referência ao AWSProvider
- `bucketName`: Nome do bucket (globalmente único)
- `versioning`: Configuração de versionamento
- `encryption`: Configuração de criptografia
- `lifecycleRules`: Regras de ciclo de vida
- `corsRules`: Configuração CORS
- `publicAccessBlock`: Bloqueio de acesso público
- `deletionPolicy`: Política de deleção (Delete, Retain, Orphan)

**Status**:
- `ready`: Bucket criado e configurado
- `arn`: ARN do bucket
- `region`: Região onde o bucket existe
- `bucketDomainName`: Domain name do bucket

**Arquivo**: `api/v1alpha1/s3bucket_types.go`

### 2. Controllers (Reconcilers)

Controllers implementam a lógica de reconciliação - observam o estado desejado (spec) e o estado atual, e trabalham para convergir o estado atual ao desejado.

#### AWSProvider Controller

**Responsabilidades**:
1. Validar credenciais AWS usando STS GetCallerIdentity
2. Construir configuração AWS (aws.Config) baseado no spec
3. Atualizar status com informações da conta
4. Re-validar credenciais periodicamente (a cada 5 minutos)

**Fluxo de reconciliação**:
```
1. Fetch AWSProvider CR
2. Construir aws.Config:
   - Se roleARN: usar IRSA/AssumeRole
   - Se accessKeyIDRef: usar credenciais estáticas
3. Chamar STS GetCallerIdentity
4. Se sucesso:
   - Atualizar status.ready = true
   - Atualizar status.accountID
   - Atualizar status.callerIdentity
5. Se erro:
   - Atualizar status.ready = false
   - Adicionar condition com erro
   - Requeue após 1 minuto
6. Requeue após 5 minutos
```

**Arquivo**: `controllers/awsprovider_controller.go`

#### S3Bucket Controller

**Responsabilidades**:
1. Criar bucket S3 se não existir
2. Configurar versioning, encryption, lifecycle, CORS, tags
3. Configurar public access block
4. Gerenciar ciclo de vida (criação, atualização, deleção)
5. Implementar finalizers para cleanup controlado

**Fluxo de reconciliação**:
```
1. Fetch S3Bucket CR
2. Get AWSProvider e construir aws.Config
3. Verificar se está sendo deletado:
   - Se sim e tem finalizer:
     - Executar cleanup baseado em deletionPolicy
     - Remover finalizer
     - Retornar
4. Adicionar finalizer se não existir
5. Verificar se bucket existe (HeadBucket)
6. Se não existe:
   - Criar bucket com CreateBucket
   - Aplicar LocationConstraint se região != us-east-1
7. Configurar bucket:
   - Versioning (PutBucketVersioning)
   - Encryption (PutBucketEncryption)
   - Lifecycle (PutBucketLifecycleConfiguration)
   - CORS (PutBucketCors)
   - Public Access Block (PutPublicAccessBlock)
   - Tags (PutBucketTagging)
8. Atualizar status:
   - ready = true
   - arn, region, bucketDomainName
   - lastSyncTime
9. Requeue após 5 minutos
```

**Arquivo**: `controllers/s3bucket_controller.go`

### 3. Package AWS Helper

Funções auxiliares para interação com AWS.

**Principais funções**:
- `GetAWSConfigFromProvider()`: Obtém aws.Config a partir de AWSProvider CR
- `buildAWSConfig()`: Constrói configuração AWS com credenciais
- `getSecretValue()`: Lê valores de Secrets Kubernetes

**Arquivo**: `pkg/aws/provider.go`

## Padrões de Design Implementados

### 1. Reconciliation Loop

O padrão fundamental de Kubernetes Operators. O controller observa o estado desejado (spec do CR) e trabalha continuamente para fazer o estado atual convergir ao desejado.

**Características**:
- **Idempotente**: Múltiplas execuções produzem o mesmo resultado
- **Edge-triggered e Level-triggered**: Reage a mudanças e também verifica estado periodicamente
- **Error handling**: Retorna erro para requeue automático

### 2. Finalizers

Mecanismo para executar cleanup antes de deletar um CR.

**Implementação no S3Bucket**:
```go
const s3BucketFinalizer = "aws-infra-operator.runner.codes/s3bucket-finalizer"

// Ao criar/atualizar:
if !controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
    controllerutil.AddFinalizer(bucket, s3BucketFinalizer)
    r.Update(ctx, bucket)
}

// Ao deletar:
if !bucket.ObjectMeta.DeletionTimestamp.IsZero() {
    if controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
        // Executar cleanup
        r.deleteBucket(ctx, s3Client, bucket)

        // Remover finalizer
        controllerutil.RemoveFinalizer(bucket, s3BucketFinalizer)
        r.Update(ctx, bucket)
    }
}
```

### 3. Status Conditions

Seguindo o padrão Kubernetes de conditions para reportar estado.

**Estrutura**:
```go
type Condition struct {
    Type               string      // "Ready"
    Status             string      // "True", "False", "Unknown"
    LastTransitionTime metav1.Time
    Reason             string      // "BucketReady", "CreationFailed"
    Message            string      // Descrição detalhada
}
```

### 4. Provider Pattern

Separação de credenciais (AWSProvider) dos recursos (S3Bucket, RDS, etc).

**Vantagens**:
- Reutilização de credenciais entre múltiplos recursos
- Validação centralizada de autenticação
- Suporte a múltiplos providers (multi-account, multi-region)
- Rotação de credenciais facilitada

### 5. Deletion Policies

Política flexível para controlar o que acontece com recursos AWS quando CR é deletado.

**Opções**:
- **Delete**: Remove o recurso AWS (padrão)
- **Retain**: Mantém o recurso AWS, remove apenas o CR
- **Orphan**: Desanexa do operator mas mantém tudo

## Fluxo de Dados

### Criação de um S3Bucket

```
1. User aplica manifest S3Bucket
   │
   ▼
2. Kubernetes API Server persiste o CR
   │
   ▼
3. Controller Runtime notifica S3BucketController
   │
   ▼
4. S3BucketController.Reconcile() é chamado
   │
   ├─▶ 5. Fetch S3Bucket CR
   │
   ├─▶ 6. Fetch AWSProvider CR
   │    └─▶ Construir aws.Config
   │
   ├─▶ 7. Verificar se bucket existe
   │    └─▶ AWS S3 HeadBucket API
   │
   ├─▶ 8. Criar bucket (se não existe)
   │    └─▶ AWS S3 CreateBucket API
   │
   ├─▶ 9. Configurar bucket
   │    ├─▶ AWS S3 PutBucketVersioning API
   │    ├─▶ AWS S3 PutBucketEncryption API
   │    ├─▶ AWS S3 PutBucketLifecycleConfiguration API
   │    ├─▶ AWS S3 PutBucketCors API
   │    ├─▶ AWS S3 PutPublicAccessBlock API
   │    └─▶ AWS S3 PutBucketTagging API
   │
   └─▶ 10. Atualizar status do CR
        └─▶ Kubernetes API Server
```

### Autenticação com IRSA

```
1. Pod do operator inicia com ServiceAccount anotado
   │
   ▼
2. Kubernetes monta token JWT no pod
   │  Path: /var/run/secrets/eks.amazonaws.com/serviceaccount/token
   │
   ▼
3. AWSProvider controller constrói aws.Config
   │  com roleARN especificado
   │
   ▼
4. AWS SDK automaticamente:
   │  ├─▶ Lê token JWT do filesystem
   │  ├─▶ Chama STS AssumeRoleWithWebIdentity
   │  └─▶ Obtém credenciais temporárias
   │
   ▼
5. Credenciais usadas para chamadas AWS
   │  (S3, RDS, EC2, etc)
   │
   ▼
6. SDK renova credenciais automaticamente
   │  antes de expirar
```

## Estrutura de Arquivos

```
infra-operator/
│
├── api/v1alpha1/              # API Definitions (CRDs)
│   ├── groupversion_info.go  # API group registration
│   ├── awsprovider_types.go  # AWSProvider CRD struct
│   ├── s3bucket_types.go     # S3Bucket CRD struct
│   ├── rdsinstance_types.go  # RDS CRD struct (pending controller)
│   ├── ec2instance_types.go  # EC2 CRD struct (pending controller)
│   └── sqsqueue_types.go     # SQS CRD struct (pending controller)
│
├── controllers/               # Reconciliation Logic
│   ├── awsprovider_controller.go  # Valida credenciais AWS
│   └── s3bucket_controller.go     # Gerencia buckets S3
│
├── pkg/                       # Shared Libraries
│   └── aws/
│       └── provider.go        # Helper functions para AWS config
│
├── config/                    # Kubernetes Manifests
│   ├── crd/bases/             # CRD YAML manifests
│   ├── rbac/                  # RBAC (ServiceAccount, Role, Binding)
│   ├── manager/               # Deployment and Namespace
│   └── samples/               # Example CRs
│
├── docs/                      # Documentation
│   ├── ARCHITECTURE.md        # Este arquivo
│   └── DEPLOYMENT_GUIDE.md    # Guia de implantação
│
├── main.go                    # Operator entry point
├── Dockerfile                 # Container image build
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
├── CLAUDE.md                  # Documentação técnica completa
└── README.md                  # User-facing documentation
```

## Decisões Arquiteturais

### Por que Go SDK v2 ao invés de ACK?

**Decisão**: Usar AWS SDK for Go v2 diretamente.

**Alternativa considerada**: AWS Controllers for Kubernetes (ACK)

**Razões**:

1. **Controle Total**:
   - Lógica customizada de reconciliação
   - Validações específicas de negócio
   - Tratamento de erros customizado

2. **Simplicidade de Deployment**:
   - Um único operator para todos os serviços
   - Sem necessidade de instalar múltiplos ACK controllers
   - Menor overhead de recursos

3. **Aprendizado**:
   - Melhor compreensão dos padrões de operators
   - Conhecimento profundo das APIs AWS
   - Flexibilidade para implementar features customizadas

4. **Dependências**:
   - Menos dependências externas
   - Atualização de SDK mais simples
   - Sem dependência do roadmap ACK

**Trade-offs**:
- Mais código para manter
- Necessidade de implementar features manualmente (paginação, retries)
- Responsabilidade de acompanhar mudanças nas APIs AWS

### Por que Provider Pattern?

**Decisão**: Separar credenciais (AWSProvider) dos recursos (S3, RDS, etc).

**Razões**:

1. **Reutilização**: Uma AWSProvider pode ser usada por múltiplos recursos
2. **Segurança**: Credenciais isoladas em um recurso específico
3. **Multi-tenancy**: Diferentes namespaces podem ter providers diferentes
4. **Multi-account**: Suporta acesso a múltiplas contas AWS
5. **Rotação**: Facilita rotação de credenciais

**Alternativa**: Credenciais diretas em cada recurso (muito repetitivo e inseguro)

### Por que Deletion Policies?

**Decisão**: Permitir ao usuário escolher o que acontece ao deletar um CR.

**Razões**:

1. **Segurança**: Evitar deleção acidental de dados importantes
2. **Flexibilidade**: Diferentes casos de uso (dev vs prod)
3. **Compliance**: Algumas regulações exigem retenção de dados
4. **Migração**: Facilita migração para outras ferramentas

**Políticas implementadas**:
- `Delete`: Remove recurso AWS (padrão para ambientes efêmeros)
- `Retain`: Mantém recurso AWS (padrão para produção)
- `Orphan`: Desanexa mas mantém tudo

## Concurrency e Thread Safety

### Controller Runtime

O controller-runtime gerencia concurrency automaticamente:

- **Work Queue**: Serializa eventos de reconciliação
- **Max Concurrent Reconciles**: Configurável (padrão: 1)
- **Retry Backoff**: Exponential backoff para erros

### Estado Compartilhado

**Sem estado compartilhado entre reconciliações**:
- Cada reconciliação busca estado atual do Kubernetes
- Não há cache compartilhado entre reconciles
- Estado persistido apenas em CRs

### Rate Limiting

**AWS SDK**:
- Implementa retry automático com exponential backoff
- Respeita rate limits da AWS
- Usa context.Context para timeout

**Kubernetes Client**:
- Rate limiting configurável
- Default: 5 QPS, burst 10

## Performance e Escalabilidade

### Otimizações Implementadas

1. **Requeue Inteligente**:
   - Recursos prontos: requeue a cada 5 minutos
   - Recursos com erro: requeue a cada 1 minuto
   - Mudanças imediatas: requeue imediato

2. **Minimizar API Calls**:
   - HeadBucket antes de GetBucket (mais barato)
   - Aplicar configurações apenas se mudaram
   - Batch updates quando possível

3. **Status Subresource**:
   - Atualizações de status não triggam nova reconciliação
   - Reduz reconciliações desnecessárias

### Limites Atuais

1. **Single Region por Provider**: Um AWSProvider = uma região
   - **Workaround**: Criar múltiplos AWSProviders

2. **No Pagination**: Não lista grandes quantidades de recursos
   - **Impact**: Limitado para operators que gerenciam recursos

3. **Sequential Processing**: Um recurso por vez
   - **Impact**: Escalabilidade limitada para muitos recursos

### Melhorias Futuras

1. **Metrics**: Prometheus metrics para observabilidade
2. **Webhooks**: Validação e defaults no admission
3. **Caching**: Cache de informações AWS para reduzir API calls
4. **Parallel Reconciliation**: MaxConcurrentReconciles > 1
5. **Health Checks**: Health checks detalhados para recursos AWS

## Segurança

### Princípios Implementados

1. **Least Privilege**:
   - RBAC mínimo necessário
   - IAM policies específicas por serviço
   - Namespaced resources quando possível

2. **Secrets Management**:
   - Credenciais em Kubernetes Secrets
   - IRSA preferred (sem long-lived credentials)
   - Nunca log credentials

3. **Container Security**:
   - Non-root user (UID 65532)
   - Read-only root filesystem
   - No privileged escalation
   - Distroless base image

4. **Network Security**:
   - HTTPS para AWS APIs
   - Certificate validation
   - No expose de credentials em status

### Threat Model

**Ameaças consideradas**:
1. **Credential Leakage**: Mitigado com IRSA e Secrets
2. **Unauthorized Access**: Mitigado com RBAC
3. **Privilege Escalation**: Mitigado com security context
4. **MITM Attacks**: Mitigado com HTTPS

## Observabilidade

### Logging

**Structured logging** com controller-runtime:
- Contexto automático (namespace, name, kind)
- Log levels: info, error, debug
- JSON format (opcional)

**Exemplo**:
```go
logger := log.FromContext(ctx)
logger.Info("Creating S3 bucket", "bucket", bucketName)
logger.Error(err, "Failed to create bucket")
```

### Status Reporting

**Todos os CRs expõem**:
- `status.ready`: Boolean indicando se recurso está pronto
- `status.conditions`: Array de conditions detalhadas
- `status.lastSyncTime`: Timestamp da última reconciliação

### Health Checks

**Endpoints disponíveis**:
- `/healthz`: Liveness probe
- `/readyz`: Readiness probe
- `/metrics`: Prometheus metrics (futuro)

## Testing Strategy

### Níveis de Teste

1. **Unit Tests** (futuro):
   - Teste de lógica de controllers
   - Mock do AWS SDK
   - Mock do Kubernetes client

2. **Integration Tests** (futuro):
   - Teste com envtest (fake API server)
   - Teste de reconciliação completa
   - Sem dependência de AWS real

3. **E2E Tests** (futuro):
   - Teste em cluster real
   - Teste com AWS real
   - Validação de recursos criados

### Test Coverage Goals

- **Controllers**: > 80% coverage
- **AWS helpers**: > 90% coverage
- **E2E**: Happy paths e principais error cases

## Conclusão

O **infra-operator** foi arquitetado seguindo os padrões estabelecidos pela comunidade Kubernetes, com foco em:

✅ **Simplicidade**: Fácil de entender e manter
✅ **Extensibilidade**: Fácil adicionar novos recursos AWS
✅ **Segurança**: IRSA, RBAC, least privilege
✅ **Confiabilidade**: Finalizers, deletion policies, idempotência
✅ **Observabilidade**: Status conditions, logging estruturado

A arquitetura permite crescimento futuro com adição de novos serviços AWS, implementação de webhooks, e melhorias de performance, mantendo a base sólida e bem estruturada.
