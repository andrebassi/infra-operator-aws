# Refactoring Summary - Clean Architecture Implementation

## 🎯 Objetivos Alcançados

O **infra-operator** foi refatorado seguindo os princípios de **Hexagonal Architecture** (Ports & Adapters), resultando em um código:

✅ **Modular** - Cada serviço AWS em seu próprio package
✅ **Testável** - Domain logic sem dependências externas
✅ **Manutenível** - Separação clara de responsabilidades
✅ **Extensível** - Fácil adicionar novos serviços ou trocar cloud provider
✅ **Idempotente** - Use cases garantem operações seguras para Kubernetes

## 📐 Arquitetura Implementada

```
┌──────────────────────────────────────────────────────────────┐
│                  KUBERNETES CONTROLLER                        │
│              controllers/s3bucket_controller_clean.go         │
│  - Reconcile loop                                            │
│  - CR watch                                                  │
│  - Status update                                             │
└─────────────────┬────────────────────────────────────────────┘
                  │ uses
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                      MAPPER LAYER                             │
│                   pkg/mapper/s3_mapper.go                     │
│  - CR ↔ Domain conversion                                    │
└─────────────────┬────────────────────────────────────────────┘
                  │ converts to
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                      USE CASE LAYER                           │
│             internal/usecases/s3/bucket_usecase.go            │
│  - Business logic                                             │
│  - Orchestration                                              │
│  - Idempotency                                                │
└─────────────────┬────────────────────────────────────────────┘
                  │ depends on (interface)
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                    PORTS (INTERFACES)                         │
│              internal/ports/s3_repository.go                  │
│  - S3Repository interface                                     │
│  - S3UseCase interface                                        │
└─────────────────┬────────────────────────────────────────────┘
                  │ implemented by
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                   ADAPTER LAYER                               │
│          internal/adapters/aws/s3/repository.go               │
│  - AWS SDK v2 implementation                                  │
│  - Type conversion                                            │
│  - Error handling                                             │
└─────────────────┬────────────────────────────────────────────┘
                  │ calls
                  ▼
┌──────────────────────────────────────────────────────────────┐
│                    DOMAIN LAYER                               │
│            internal/domain/s3/bucket.go                       │
│  - Pure business entities                                     │
│  - Domain logic                                               │
│  - No external dependencies                                   │
└──────────────────────────────────────────────────────────────┘
```

## 📁 Nova Estrutura de Arquivos

### Arquivos Criados

```
infra-operator/
├── internal/                         # 🆕 Código privado modular
│   ├── domain/                       # 🆕 Entidades de negócio
│   │   └── s3/
│   │       ├── bucket.go             # Entidade Bucket
│   │       └── errors.go             # Erros de domínio
│   │
│   ├── ports/                        # 🆕 Interfaces (contratos)
│   │   └── s3_repository.go          # Interface S3
│   │
│   ├── adapters/                     # 🆕 Implementações AWS SDK
│   │   └── aws/s3/
│   │       └── repository.go         # Adapter S3
│   │
│   └── usecases/                     # 🆕 Lógica de aplicação
│       └── s3/
│           └── bucket_usecase.go     # Business logic S3
│
├── pkg/                              # 🆕 Código público
│   ├── mapper/
│   │   └── s3_mapper.go             # CR ↔ Domain conversion
│   └── clients/
│       └── aws_client.go            # AWS client factory
│
├── controllers/
│   ├── s3bucket_controller.go       # Versão original (legacy)
│   └── s3bucket_controller_clean.go # 🆕 Clean Architecture
│
├── cmd/
│   └── main.go                      # 🆕 Dependency Injection
│
├── docs/
│   ├── CLEAN_ARCHITECTURE.md        # 🆕 Arquitetura detalhada
│   └── AWS_SERVICES_REFERENCE.md
│
└── internal/domain/s3/
    └── bucket_test.go               # 🆕 Testes unitários
```

## 🔄 Comparação: Antes vs Depois

### ❌ Antes (Código Monolítico)

```go
// controllers/s3bucket_controller.go (original)
func (r *S3BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch CR
    bucket := &infrav1alpha1.S3Bucket{}
    r.Get(ctx, req.NamespacedName, bucket)

    // 2. Get AWS config inline
    awsConfig := buildAWSConfig(provider) // ❌ Acoplado

    // 3. Create AWS client inline
    s3Client := s3.NewFromConfig(awsConfig) // ❌ Acoplado ao AWS SDK

    // 4. Business logic misturado
    exists, _ := s3Client.HeadBucket(...)  // ❌ Lógica AWS no controller
    if !exists {
        s3Client.CreateBucket(...)          // ❌ Difícil testar
    }
    s3Client.PutBucketVersioning(...)       // ❌ Sem reutilização
    s3Client.PutBucketEncryption(...)

    // 5. Update status
    bucket.Status.Ready = true
    r.Status().Update(ctx, bucket)
}
```

**Problemas**:
- ❌ Controller conhece detalhes do AWS SDK
- ❌ Impossível testar sem AWS real
- ❌ Lógica de negócio espalhada
- ❌ Difícil adicionar outros clouds (GCP, Azure)

### ✅ Depois (Clean Architecture)

```go
// controllers/s3bucket_controller_clean.go
func (r *S3BucketReconcilerClean) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch CR
    bucketCR := &infrav1alpha1.S3Bucket{}
    r.Get(ctx, req.NamespacedName, bucketCR)

    // 2. Get AWS config via factory
    awsConfig, provider, _ := r.AWSClientFactory.GetAWSConfigFromProviderRef(...)

    // 3. Create repository (adapter)
    s3Repo := awsadapter.NewRepository(awsConfig)

    // 4. Convert to domain
    domainBucket := mapper.CRToDomainBucket(bucketCR)

    // 5. Execute business logic via use case
    s3UseCase := s3usecase.NewBucketUseCase(s3Repo)
    s3UseCase.SyncBucket(ctx, domainBucket) // ✅ Idempotente, testável

    // 6. Update status
    mapper.DomainBucketToCRStatus(domainBucket, bucketCR)
    r.Status().Update(ctx, bucketCR)
}
```

**Vantagens**:
- ✅ Controller só orquestra (thin controller)
- ✅ Use case testável com mocks
- ✅ Domain logic reutilizável
- ✅ Fácil trocar AWS por GCP/Azure

## 🧪 Testabilidade

### Domain Tests (Puros - Sem Mocks)

```go
// internal/domain/s3/bucket_test.go
func TestBucket_Validate(t *testing.T) {
    bucket := &s3.Bucket{
        Name:   "my-bucket",
        Region: "us-east-1",
    }

    err := bucket.Validate() // ✅ Teste puro, sem dependências
    assert.NoError(t, err)
}
```

### Use Case Tests (Com Mock Repository)

```go
// internal/usecases/s3/bucket_usecase_test.go
func TestBucketUseCase_CreateBucket(t *testing.T) {
    mockRepo := new(mockS3Repository)
    usecase := s3usecase.NewBucketUseCase(mockRepo)

    bucket := &s3.Bucket{Name: "test", Region: "us-east-1"}

    mockRepo.On("Exists", mock.Anything, "test", "us-east-1").
        Return(false, nil)
    mockRepo.On("Create", mock.Anything, bucket).
        Return(nil)

    err := usecase.CreateBucket(context.Background(), bucket)

    assert.NoError(t, err)
    mockRepo.AssertExpectations(t) // ✅ Verifica se foi chamado corretamente
}
```

### Controller Tests (Com Mock UseCase)

```go
// controllers/s3bucket_controller_clean_test.go
func TestS3BucketReconciler_Reconcile(t *testing.T) {
    mockUseCase := new(mockS3UseCase)
    reconciler := &S3BucketReconcilerClean{
        S3UseCase: mockUseCase,
    }

    mockUseCase.On("SyncBucket", mock.Anything, mock.Anything).
        Return(nil)

    result, err := reconciler.Reconcile(ctx, req)

    assert.NoError(t, err)
    mockUseCase.AssertExpectations(t)
}
```

## 🔌 Dependency Injection

### main.go (Wire Dependencies)

```go
func main() {
    mgr, _ := ctrl.NewManager(...)

    // 1. Create shared factories
    awsClientFactory := clients.NewAWSClientFactory(mgr.GetClient())

    // 2. Create repositories (adapters)
    // Note: Repositories are created per-request in controller
    // based on the AWSProvider referenced by each CR

    // 3. Create controllers with injected dependencies
    s3Controller := &controllers.S3BucketReconcilerClean{
        Client:           mgr.GetClient(),
        Scheme:           mgr.GetScheme(),
        AWSClientFactory: awsClientFactory, // ← Injected
    }

    s3Controller.SetupWithManager(mgr)

    mgr.Start(ctrl.SetupSignalHandler())
}
```

## 📊 Benefícios por Camada

### 1. Domain Layer

**Benefícios**:
- ✅ Testável sem mocks
- ✅ Reutilizável em qualquer contexto
- ✅ Sem dependências externas
- ✅ Evolui independentemente

**Exemplo**:
```go
// Lógica de negócio pura
func (b *Bucket) HasPublicAccessBlocked() bool {
    return b.PublicAccessBlock != nil &&
        b.PublicAccessBlock.BlockPublicAcls &&
        b.PublicAccessBlock.IgnorePublicAcls
}
```

### 2. Ports Layer

**Benefícios**:
- ✅ Define contrato claro
- ✅ Permite múltiplas implementações
- ✅ Facilita testes com mocks
- ✅ Inverte dependências

**Exemplo**:
```go
type S3Repository interface {
    Create(ctx context.Context, bucket *s3.Bucket) error
    Get(ctx context.Context, name, region string) (*s3.Bucket, error)
    // ... pode ter implementação AWS, GCP, Mock, etc
}
```

### 3. Adapters Layer

**Benefícios**:
- ✅ Isola dependências externas
- ✅ Fácil trocar implementação
- ✅ Converte tipos externos para domain
- ✅ Trata erros específicos do provider

**Exemplo**:
```go
// Adapter para AWS SDK v2
type Repository struct {
    client *awss3.Client
}

func (r *Repository) Create(ctx context.Context, bucket *s3.Bucket) error {
    // Converte domain.Bucket → AWS SDK types
    input := &awss3.CreateBucketInput{
        Bucket: aws.String(bucket.Name),
    }
    _, err := r.client.CreateBucket(ctx, input)
    return err
}
```

### 4. Use Cases Layer

**Benefícios**:
- ✅ Orquestra operações complexas
- ✅ Garante idempotência (crucial para K8s)
- ✅ Implementa regras de negócio
- ✅ Testável com repository mock

**Exemplo**:
```go
func (uc *BucketUseCase) SyncBucket(ctx context.Context, bucket *s3.Bucket) error {
    exists, _ := uc.repo.Exists(ctx, bucket.Name, bucket.Region)

    if !exists {
        return uc.CreateBucket(ctx, bucket) // Cria
    }

    return uc.repo.Configure(ctx, bucket) // Atualiza (idempotente)
}
```

### 5. Controllers Layer

**Benefícios**:
- ✅ Thin controller - apenas orchestração K8s
- ✅ Fácil testar (mock use case)
- ✅ Não conhece AWS SDK
- ✅ Reutiliza lógica via use cases

**Exemplo**:
```go
func (r *S3BucketReconcilerClean) Reconcile(ctx, req) (ctrl.Result, error) {
    bucketCR := &infrav1alpha1.S3Bucket{}
    r.Get(ctx, req.NamespacedName, bucketCR)

    domainBucket := mapper.CRToDomainBucket(bucketCR)

    s3UseCase.SyncBucket(ctx, domainBucket) // ← Toda lógica aqui

    mapper.DomainBucketToCRStatus(domainBucket, bucketCR)
    r.Status().Update(ctx, bucketCR)

    return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}
```

## 🚀 Como Estender para Outros Serviços AWS

### Template para Novo Serviço (Lambda)

**1. Domain** (`internal/domain/lambda/function.go`):
```go
package lambda

type Function struct {
    Name    string
    Runtime string
    Handler string
    Code    *Code
    // ... domain fields
}

func (f *Function) Validate() error {
    // Domain validation
}
```

**2. Port** (`internal/ports/lambda_repository.go`):
```go
type LambdaRepository interface {
    Create(ctx context.Context, fn *lambda.Function) error
    Update(ctx context.Context, fn *lambda.Function) error
    Delete(ctx context.Context, name, region string) error
}
```

**3. Adapter** (`internal/adapters/aws/lambda/repository.go`):
```go
type Repository struct {
    client *awslambda.Client
}

func NewRepository(cfg aws.Config) ports.LambdaRepository {
    return &Repository{client: awslambda.NewFromConfig(cfg)}
}

func (r *Repository) Create(ctx context.Context, fn *lambda.Function) error {
    // AWS SDK calls
}
```

**4. Use Case** (`internal/usecases/lambda/function_usecase.go`):
```go
type FunctionUseCase struct {
    repo ports.LambdaRepository
}

func (uc *FunctionUseCase) SyncFunction(ctx context.Context, fn *lambda.Function) error {
    // Business logic
}
```

**5. Controller** (`controllers/lambdafunction_controller.go`):
```go
type LambdaFunctionReconciler struct {
    Client           client.Client
    AWSClientFactory *clients.AWSClientFactory
}

func (r *LambdaFunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Similar ao S3BucketReconcilerClean
}
```

## 📚 Referências

### Documentação Oficial

- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Amazon S3 Examples](https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html)
- [Kubernetes Operator SDK](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)

### Clean Architecture

- [Clean Architecture in Go](https://pkritiotis.io/clean-architecture-in-golang/)
- [Hexagonal Architecture](https://medium.com/@omidahn/clean-architecture-in-go-golang-a-comprehensive-guide-f8e422b7bfae)
- [Go Clean Architecture](https://github.com/bxcodec/go-clean-arch)

## ✅ Checklist de Migração

Para migrar outros controllers para Clean Architecture:

- [ ] Criar domain entities (`internal/domain/{service}/`)
- [ ] Definir ports/interfaces (`internal/ports/{service}_repository.go`)
- [ ] Implementar adapter AWS SDK (`internal/adapters/aws/{service}/repository.go`)
- [ ] Implementar use case (`internal/usecases/{service}/`)
- [ ] Criar mapper CR ↔ Domain (`pkg/mapper/{service}_mapper.go`)
- [ ] Refatorar controller (`controllers/{service}_controller_clean.go`)
- [ ] Adicionar testes unitários (`*_test.go`)
- [ ] Atualizar `main.go` com DI
- [ ] Documentar no `CLEAN_ARCHITECTURE.md`

## 🎯 Status Final

### S3 Bucket - ✅ 100% Completo

- ✅ Domain layer
- ✅ Ports layer
- ✅ Adapter layer
- ✅ Use case layer
- ✅ Controller refatorado
- ✅ Mapper implementado
- ✅ Testes unitários
- ✅ Dependency injection
- ✅ Documentação

### Próximos Serviços (Template Pronto)

- 🟡 Lambda Function
- 🟡 DynamoDB Table
- 🟡 SQS Queue
- 🟡 SNS Topic
- 🟡 ElastiCache
- 🟡 RDS Instance

Cada um pode seguir exatamente o mesmo padrão estabelecido para S3! 🎉

---

**Data**: 2025-01-22
**Status**: ✅ **REFATORAÇÃO COMPLETA - CLEAN ARCHITECTURE IMPLEMENTADA**
**Cobertura**: S3 Bucket (100%) | Template para outros serviços (Pronto)
