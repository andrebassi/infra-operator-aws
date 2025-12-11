# Clean Architecture - Infra Operator

## Hexagonal Architecture (Ports & Adapters)

O **infra-operator** segue os princípios da **Hexagonal Architecture** (também conhecida como Ports and Adapters), que é mais adequada para Go do que a Clean Architecture tradicional.

### Por que Hexagonal Architecture?

Baseado em pesquisas sobre melhores práticas ([Clean Architecture in Go](https://pkritiotis.io/clean-architecture-in-golang/), [Kubernetes Operator Best Practices](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)):

1. **Simplicidade**: Menos camadas do que a Clean Architecture tradicional
2. **Testabilidade**: Fácil criar mocks de interfaces
3. **Flexibilidade**: Trocar implementações (AWS, GCP, Azure) sem mudar lógica de negócio
4. **Idempotência**: Controllers Kubernetes precisam ser idempotentes - a arquitetura ajuda nisso

## Estrutura de Camadas

![Clean Architecture Layers](/img/diagrams/clean-architecture-layers.svg)

## Camadas Explicadas

### 1. Domain Layer (Core)

**Localização**: `internal/domain/{service}/`

**Responsabilidade**: Entidades de negócio puras, sem dependências externas.

**Exemplo** (`internal/domain/s3/bucket.go`):

```go
package s3

import "time"

// Bucket é a entidade de domínio - representa o conceito de negócio
type Bucket struct {
Name              string
Region            string
Versioning        *VersioningConfig
Encryption        *EncryptionConfig
LifecycleRules    []LifecycleRule
PublicAccessBlock *PublicAccessBlockConfig
Tags              map[string]string
DeletionPolicy    DeletionPolicy
}

// Métodos de negócio (lógica de domínio)
func (b *Bucket) Validate() error {
if b.Name == "" {
        return ErrBucketNameRequired
}
if len(b.Name) < 3 || len(b.Name) > 63 {
        return ErrInvalidBucketNameLength
}
return nil
}

func (b *Bucket) IsEncrypted() bool {
return b.Encryption != nil && b.Encryption.Algorithm != ""
}

func (b *Bucket) HasPublicAccessBlocked() bool {
return b.PublicAccessBlock != nil &&
        b.PublicAccessBlock.BlockPublicAcls &&
        b.PublicAccessBlock.IgnorePublicAcls &&
        b.PublicAccessBlock.BlockPublicPolicy &&
        b.PublicAccessBlock.RestrictPublicBuckets
}
```

**Características**:
- ✅ Sem dependências externas (AWS SDK, Kubernetes, etc)
- ✅ Regras de negócio puras
- ✅ Facilmente testável
- ✅ Reutilizável em qualquer contexto

### 2. Ports Layer (Interfaces)

**Localização**: `internal/ports/`

**Responsabilidade**: Define contratos (interfaces) que os adapters devem implementar.

**Exemplo** (`internal/ports/s3_repository.go`):

```go
package ports

import (
"context"
"infra-operator/internal/domain/s3"
)

// S3Repository define O QUE precisamos, não COMO
type S3Repository interface {
Create(ctx context.Context, bucket *s3.Bucket) error
Get(ctx context.Context, name, region string) (*s3.Bucket, error)
Update(ctx context.Context, bucket *s3.Bucket) error
Delete(ctx context.Context, name, region string) error
Exists(ctx context.Context, name, region string) (bool, error)
Configure(ctx context.Context, bucket *s3.Bucket) error
}

// S3UseCase define operações de negócio
type S3UseCase interface {
CreateBucket(ctx context.Context, bucket *s3.Bucket) error
GetBucket(ctx context.Context, name, region string) (*s3.Bucket, error)
SyncBucket(ctx context.Context, bucket *s3.Bucket) error
DeleteBucket(ctx context.Context, bucket *s3.Bucket) error
}
```

**Características**:
- ✅ Define o contrato
- ✅ Não depende da implementação
- ✅ Permite múltiplas implementações (AWS, GCP, Mock)

### 3. Adapters Layer (Implementações)

**Localização**: `internal/adapters/aws/{service}/`

**Responsabilidade**: Implementa interfaces usando tecnologias específicas (AWS SDK v2).

**Exemplo** (`internal/adapters/aws/s3/repository.go`):

```go
package s3

import (
"context"
"github.com/aws/aws-sdk-go-v2/aws"
awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
"infra-operator/internal/domain/s3"
"infra-operator/internal/ports"
)

// Repository implementa ports.S3Repository usando AWS SDK v2
type Repository struct {
client *awss3.Client
}

func NewRepository(awsConfig aws.Config) ports.S3Repository {
return &Repository{
        client: awss3.NewFromConfig(awsConfig),
}
}

func (r *Repository) Create(ctx context.Context, bucket *s3.Bucket) error {
if err := bucket.Validate(); err != nil {
        return err
}

input := &awss3.CreateBucketInput{
        Bucket: aws.String(bucket.Name),
}

if bucket.Region != "us-east-1" {
        input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
            LocationConstraint: types.BucketLocationConstraint(bucket.Region),
        }
}

_, err := r.client.CreateBucket(ctx, input)
return err
}

// ... outras implementações
```

**Características**:
- ✅ Usa AWS SDK v2 ([documentação oficial](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/go_s3_code_examples.html))
- ✅ Converte entre tipos de domínio e tipos AWS
- ✅ Trata erros específicos da AWS
- ✅ Pode ser substituído por mock em testes

### 4. Use Cases Layer (Lógica de Aplicação)

**Localização**: `internal/usecases/{service}/`

**Responsabilidade**: Orquestra operações, implementa regras de negócio complexas.

**Exemplo** (`internal/usecases/s3/bucket_usecase.go`):

```go
package s3

import (
"context"
"fmt"
"infra-operator/internal/domain/s3"
"infra-operator/internal/ports"
)

type BucketUseCase struct {
repo ports.S3Repository
}

func NewBucketUseCase(repo ports.S3Repository) ports.S3UseCase {
return &BucketUseCase{repo: repo}
}

func (uc *BucketUseCase) CreateBucket(ctx context.Context, bucket *s3.Bucket) error {
// Validação de negócio
if err := bucket.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
}

// Verificar se já existe
exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
if err != nil {
        return err
}

if exists {
        return s3.ErrBucketAlreadyExists
}

// Criar bucket
if err := uc.repo.Create(ctx, bucket); err != nil {
        return err
}

// Configurar após criação
return uc.repo.Configure(ctx, bucket)
}

func (uc *BucketUseCase) SyncBucket(ctx context.Context, bucket *s3.Bucket) error {
// Lógica de sincronização - garante idempotência
exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
if err != nil {
        return err
}

if !exists {
        return uc.CreateBucket(ctx, bucket)
}

// Atualizar configuração existente
return uc.repo.Configure(ctx, bucket)
}

func (uc *BucketUseCase) DeleteBucket(ctx context.Context, bucket *s3.Bucket) error {
// Respeitar política de deleção
if bucket.DeletionPolicy == s3.DeletionPolicyRetain ||
       bucket.DeletionPolicy == s3.DeletionPolicyOrphan {
        return nil // Não deletar
}

return uc.repo.Delete(ctx, bucket.Name, bucket.Region)
}
```

**Características**:
- ✅ Orquestra múltiplas operações
- ✅ Implementa regras de negócio complexas
- ✅ Garante idempotência (crucial para Kubernetes)
- ✅ Depende apenas de interfaces (ports)

### 5. Controller Layer (Kubernetes)

**Localização**: `controllers/`

**Responsabilidade**: Loop de reconciliação, observar CRDs, atualizar status.

**Exemplo** (`controllers/s3bucket_controller.go` refatorado):

```go
package controllers

import (
"context"
"time"

ctrl "sigs.k8s.io/controller-runtime"
"sigs.k8s.io/controller-runtime/pkg/client"

infrav1alpha1 "infra-operator/api/v1alpha1"
"infra-operator/internal/domain/s3"
"infra-operator/internal/ports"
"infra-operator/pkg/mapper"
)

type S3BucketReconciler struct {
client.Client
Scheme      *runtime.Scheme
S3UseCase   ports.S3UseCase  // Dependência injetada
}

func (r *S3BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
// 1. Buscar CR
bucketCR := &infrav1alpha1.S3Bucket{}
if err := r.Get(ctx, req.NamespacedName, bucketCR); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
}

// 2. Converter CR para modelo de domínio
domainBucket := mapper.CRToDomainBucket(bucketCR)

// 3. Executar use case (lógica de negócio)
if err := r.S3UseCase.SyncBucket(ctx, domainBucket); err != nil {
        return r.updateStatus(ctx, bucketCR, false, err.Error())
}

// 4. Atualizar status
return r.updateStatus(ctx, bucketCR, true, "Bucket ready")
}
```

**Características**:
- ✅ Lógica mínima - apenas orquestração Kubernetes
- ✅ Depende de use cases (não diretamente de adapters)
- ✅ Fácil de testar (mockar use case)

## Estrutura de Diretórios

```
infra-operator/
├── api/v1alpha1/                    # CRDs (API Kubernetes)
│   ├── s3bucket_types.go
│   └── awsprovider_types.go
│
├── internal/                        # Código privado (não exportável)
│   │
│   ├── domain/                      # 🟢 CORE - Entidades de negócio
│   │   ├── s3/
│   │   │   ├── bucket.go           # Entidade Bucket
│   │   │   └── errors.go           # Erros de domínio
│   │   ├── lambda/
│   │   │   └── function.go
│   │   └── dynamodb/
│   │       └── table.go
│   │
│   ├── ports/                       # 🔵 Interfaces (contratos)
│   │   ├── s3_repository.go        # Interface para S3
│   │   ├── s3_usecase.go           # Interface para use cases
│   │   ├── lambda_repository.go
│   │   └── dynamodb_repository.go
│   │
│   ├── adapters/                    # 🟡 Implementações externas
│   │   └── aws/                    # Adapter para AWS
│   │       ├── s3/
│   │       │   └── repository.go   # Implementa ports.S3Repository
│   │       ├── lambda/
│   │       │   └── repository.go
│   │       └── dynamodb/
│   │           └── repository.go
│   │
│   └── usecases/                    # 🟣 Lógica de aplicação
│       ├── s3/
│       │   └── bucket_usecase.go   # Implementa ports.S3UseCase
│       ├── lambda/
│       │   └── function_usecase.go
│       └── dynamodb/
│           └── table_usecase.go
│
├── controllers/                     # 🔴 Controllers Kubernetes
│   ├── s3bucket_controller.go      # Usa ports.S3UseCase
│   └── awsprovider_controller.go
│
├── pkg/                             # Código público (exportável)
│   ├── mapper/                     # Conversão CR ↔ Domínio
│   │   ├── s3_mapper.go
│   │   └── lambda_mapper.go
│   └── clients/                    # Factories para clientes
│       └── aws_client.go
│
└── cmd/
└── main.go                     # Conectar dependências
```

## Fluxo de Dados

### Criação de Bucket

```
1. Usuário aplica CR S3Bucket
   │
   ▼
2. API Server Kubernetes persiste CR
   │
   ▼
3. S3BucketController.Reconcile() disparado
   │
   ├─▶ Buscar CR do Kubernetes
   │
   ├─▶ mapper.CRToDomainBucket(cr)
   │   └─▶ Converte infrav1alpha1.S3Bucket → domain/s3.Bucket
   │
   ├─▶ s3UseCase.SyncBucket(domainBucket)
   │   │
   │   ├─▶ bucket.Validate() (lógica de domínio)
   │   │
   │   ├─▶ s3Repo.Exists(name)
   │   │   └─▶ AWS SDK: HeadBucket()
   │   │
   │   ├─▶ s3Repo.Create(bucket)
   │   │   └─▶ AWS SDK: CreateBucket()
   │   │
   │   └─▶ s3Repo.Configure(bucket)
   │       ├─▶ AWS SDK: PutBucketVersioning()
   │       ├─▶ AWS SDK: PutBucketEncryption()
   │       └─▶ AWS SDK: PutPublicAccessBlock()
   │
   └─▶ Atualizar Status do CR
```

## Injeção de Dependência

**Localização**: `cmd/main.go`:

```go
func main() {
// ... setup manager ...

// Construir configuração AWS
awsConfig, _ := config.LoadDefaultConfig(context.Background())

// Criar adapters
s3Repo := s3adapter.NewRepository(awsConfig)
lambdaRepo := lambdaadapter.NewRepository(awsConfig)

// Criar use cases
s3UseCase := s3usecase.NewBucketUseCase(s3Repo)
lambdaUseCase := lambdausecase.NewFunctionUseCase(lambdaRepo)

// Criar controllers com dependências injetadas
s3Controller := &controllers.S3BucketReconciler{
        Client:    mgr.GetClient(),
        Scheme:    mgr.GetScheme(),
        S3UseCase: s3UseCase,  // ← Injetado
}

s3Controller.SetupWithManager(mgr)

mgr.Start(ctrl.SetupSignalHandler())
}
```

## Testabilidade

### 1. Testes de Domínio (Puros)

**Código:**

```go
func TestBucket_Validate(t *testing.T) {
tests := []struct {
        name    string
        bucket  *s3.Bucket
        wantErr error
}{
        {
            name: "valid bucket",
            bucket: &s3.Bucket{
                Name:   "my-bucket",
                Region: "us-east-1",
            },
            wantErr: nil,
        },
        {
            name: "empty name",
            bucket: &s3.Bucket{
                Name:   "",
                Region: "us-east-1",
            },
            wantErr: s3.ErrBucketNameRequired,
        },
}

for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.bucket.Validate()
            if err != tt.wantErr {
                t.Errorf("got %v, want %v", err, tt.wantErr)
            }
        })
}
}
```

### 2. Testes de Use Case (com Mock de Repository)

**Código:**

```go
type mockS3Repository struct {
mock.Mock
}

func (m *mockS3Repository) Create(ctx context.Context, bucket *s3.Bucket) error {
args := m.Called(ctx, bucket)
return args.Error(0)
}

func TestBucketUseCase_CreateBucket(t *testing.T) {
repo := new(mockS3Repository)
usecase := s3usecase.NewBucketUseCase(repo)

bucket := &s3.Bucket{
        Name:   "test-bucket",
        Region: "us-east-1",
}

// Expectativas de mock
repo.On("Exists", mock.Anything, "test-bucket", "us-east-1").
        Return(false, nil)
repo.On("Create", mock.Anything, bucket).
        Return(nil)
repo.On("Configure", mock.Anything, bucket).
        Return(nil)

// Executar
err := usecase.CreateBucket(context.Background(), bucket)

// Validar
assert.NoError(t, err)
repo.AssertExpectations(t)
}
```

### 3. Testes de Controller (com Mock de UseCase)

**Código:**

```go
type mockS3UseCase struct {
mock.Mock
}

func (m *mockS3UseCase) SyncBucket(ctx context.Context, bucket *s3.Bucket) error {
args := m.Called(ctx, bucket)
return args.Error(0)
}

func TestS3BucketReconciler_Reconcile(t *testing.T) {
usecase := new(mockS3UseCase)
reconciler := &S3BucketReconciler{
        S3UseCase: usecase,
}

// ... implementação do teste
}
```

## Vantagens desta Arquitetura

### ✅ Para Operators Kubernetes

1. **Idempotência**: Use cases garantem operações idempotentes
2. **Reconcile Simples**: Controller apenas orquestra, sem lógica
3. **Relatório de Status**: Fácil atualizar status baseado no estado do domínio

### ✅ Para Testes

1. **Domínio**: Testes puros, sem mocks
2. **Use Cases**: Mockar repositories
3. **Controllers**: Mockar use cases
4. **Adapters**: Testes de integração com AWS (opcional)

### ✅ Para Manutenção

1. **Separação Clara**: Cada camada tem uma responsabilidade única
2. **Baixo Acoplamento**: Mudanças na AWS não afetam domínio
3. **Alta Coesão**: Código relacionado fica junto

### ✅ Para Extensibilidade

1. **Múltiplas Clouds**: Trocar adapter AWS por GCP/Azure
2. **Múltiplos Backends**: Adicionar adapter para Terraform/Pulumi
3. **Evolução**: Mudar AWS SDK v2 → v3 sem afetar domínio

## Referências

- [Clean Architecture in Go](https://pkritiotis.io/clean-architecture-in-golang/)
- [AWS SDK for Go v2 Developer Guide](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Amazon S3 examples using SDK for Go V2](https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html)
- [Kubernetes Operator SDK Tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)
- [Go Clean Architecture](https://github.com/bxcodec/go-clean-arch)
