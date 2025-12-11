# Clean Architecture - Infra Operator

## Hexagonal Architecture (Ports & Adapters)

The **infra-operator** follows the principles of **Hexagonal Architecture** (also known as Ports and Adapters), which is more suitable for Go than traditional Clean Architecture.

### Why Hexagonal Architecture?

Based on research about best practices ([Clean Architecture in Go](https://pkritiotis.io/clean-architecture-in-golang/), [Kubernetes Operator Best Practices](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)):

1. **Simplicity**: Fewer layers than traditional Clean Architecture
2. **Testability**: Easy to create mocks of interfaces
3. **Flexibility**: Swap implementations (AWS, GCP, Azure) without changing business logic
4. **Idempotency**: Kubernetes controllers need to be idempotent - architecture helps with this

## Layer Structure

![Clean Architecture Layers](/img/diagrams/clean-architecture-layers.svg)

## Layers Explained

### 1. Domain Layer (Core)

**Location**: `internal/domain/{service}/`

**Responsibility**: Pure business entities, without external dependencies.

**Example** (`internal/domain/s3/bucket.go`):

```go
package s3

import "time"

// Bucket is the domain entity - represents the business concept
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

// Business methods (domain logic)
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

**Characteristics**:
- ✅ No external dependencies (AWS SDK, Kubernetes, etc)
- ✅ Pure business rules
- ✅ Easily testable
- ✅ Reusable in any context

### 2. Ports Layer (Interfaces)

**Location**: `internal/ports/`

**Responsibility**: Defines contracts (interfaces) that adapters must implement.

**Example** (`internal/ports/s3_repository.go`):

```go
package ports

import (
"context"
"infra-operator/internal/domain/s3"
)

// S3Repository defines WHAT we need, not HOW
type S3Repository interface {
Create(ctx context.Context, bucket *s3.Bucket) error
Get(ctx context.Context, name, region string) (*s3.Bucket, error)
Update(ctx context.Context, bucket *s3.Bucket) error
Delete(ctx context.Context, name, region string) error
Exists(ctx context.Context, name, region string) (bool, error)
Configure(ctx context.Context, bucket *s3.Bucket) error
}

// S3UseCase defines business operations
type S3UseCase interface {
CreateBucket(ctx context.Context, bucket *s3.Bucket) error
GetBucket(ctx context.Context, name, region string) (*s3.Bucket, error)
SyncBucket(ctx context.Context, bucket *s3.Bucket) error
DeleteBucket(ctx context.Context, bucket *s3.Bucket) error
}
```

**Characteristics**:
- ✅ Defines the contract
- ✅ Does not depend on implementation
- ✅ Allows multiple implementations (AWS, GCP, Mock)

### 3. Adapters Layer (Implementations)

**Location**: `internal/adapters/aws/{service}/`

**Responsibility**: Implements interfaces using specific technologies (AWS SDK v2).

**Example** (`internal/adapters/aws/s3/repository.go`):

```go
package s3

import (
"context"
"github.com/aws/aws-sdk-go-v2/aws"
awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
"infra-operator/internal/domain/s3"
"infra-operator/internal/ports"
)

// Repository implements ports.S3Repository using AWS SDK v2
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

// ... other implementations
```

**Characteristics**:
- ✅ Uses AWS SDK v2 ([official documentation](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/go_s3_code_examples.html))
- ✅ Converts between domain types and AWS types
- ✅ Handles AWS-specific errors
- ✅ Can be replaced by mock in tests

### 4. Use Cases Layer (Application Logic)

**Location**: `internal/usecases/{service}/`

**Responsibility**: Orchestrates operations, implements complex business rules.

**Example** (`internal/usecases/s3/bucket_usecase.go`):

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
// Business validation
if err := bucket.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
}

// Check if already exists
exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
if err != nil {
        return err
}

if exists {
        return s3.ErrBucketAlreadyExists
}

// Create bucket
if err := uc.repo.Create(ctx, bucket); err != nil {
        return err
}

// Configure after creation
return uc.repo.Configure(ctx, bucket)
}

func (uc *BucketUseCase) SyncBucket(ctx context.Context, bucket *s3.Bucket) error {
// Synchronization logic - ensures idempotency
exists, err := uc.repo.Exists(ctx, bucket.Name, bucket.Region)
if err != nil {
        return err
}

if !exists {
        return uc.CreateBucket(ctx, bucket)
}

// Update existing configuration
return uc.repo.Configure(ctx, bucket)
}

func (uc *BucketUseCase) DeleteBucket(ctx context.Context, bucket *s3.Bucket) error {
// Respect deletion policy
if bucket.DeletionPolicy == s3.DeletionPolicyRetain ||
       bucket.DeletionPolicy == s3.DeletionPolicyOrphan {
        return nil // Don't delete
}

return uc.repo.Delete(ctx, bucket.Name, bucket.Region)
}
```

**Characteristics**:
- ✅ Orchestrates multiple operations
- ✅ Implements complex business rules
- ✅ Ensures idempotency (crucial for Kubernetes)
- ✅ Depends only on interfaces (ports)

### 5. Controller Layer (Kubernetes)

**Location**: `controllers/`

**Responsibility**: Reconcile loop, watch CRDs, update status.

**Example** (`controllers/s3bucket_controller.go` refactored):

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
S3UseCase   ports.S3UseCase  // Injected dependency
}

func (r *S3BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
// 1. Fetch CR
bucketCR := &infrav1alpha1.S3Bucket{}
if err := r.Get(ctx, req.NamespacedName, bucketCR); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
}

// 2. Convert CR to domain model
domainBucket := mapper.CRToDomainBucket(bucketCR)

// 3. Execute use case (business logic)
if err := r.S3UseCase.SyncBucket(ctx, domainBucket); err != nil {
        return r.updateStatus(ctx, bucketCR, false, err.Error())
}

// 4. Update status
return r.updateStatus(ctx, bucketCR, true, "Bucket ready")
}
```

**Characteristics**:
- ✅ Minimal logic - only Kubernetes orchestration
- ✅ Depends on use cases (not directly on adapters)
- ✅ Easy to test (mock use case)

## Directory Structure

```
infra-operator/
├── api/v1alpha1/                    # CRDs (Kubernetes API)
│   ├── s3bucket_types.go
│   └── awsprovider_types.go
│
├── internal/                        # Private code (not exportable)
│   │
│   ├── domain/                      # 🟢 CORE - Business entities
│   │   ├── s3/
│   │   │   ├── bucket.go           # Bucket entity
│   │   │   └── errors.go           # Domain errors
│   │   ├── lambda/
│   │   │   └── function.go
│   │   └── dynamodb/
│   │       └── table.go
│   │
│   ├── ports/                       # 🔵 Interfaces (contracts)
│   │   ├── s3_repository.go        # Interface for S3
│   │   ├── s3_usecase.go           # Interface for use cases
│   │   ├── lambda_repository.go
│   │   └── dynamodb_repository.go
│   │
│   ├── adapters/                    # 🟡 External implementations
│   │   └── aws/                    # Adapter for AWS
│   │       ├── s3/
│   │       │   └── repository.go   # Implements ports.S3Repository
│   │       ├── lambda/
│   │       │   └── repository.go
│   │       └── dynamodb/
│   │           └── repository.go
│   │
│   └── usecases/                    # 🟣 Application logic
│       ├── s3/
│       │   └── bucket_usecase.go   # Implements ports.S3UseCase
│       ├── lambda/
│       │   └── function_usecase.go
│       └── dynamodb/
│           └── table_usecase.go
│
├── controllers/                     # 🔴 Kubernetes Controllers
│   ├── s3bucket_controller.go      # Uses ports.S3UseCase
│   └── awsprovider_controller.go
│
├── pkg/                             # Public code (exportable)
│   ├── mapper/                     # Conversion CR ↔ Domain
│   │   ├── s3_mapper.go
│   │   └── lambda_mapper.go
│   └── clients/                    # Factories for clients
│       └── aws_client.go
│
└── cmd/
└── main.go                     # Wire dependencies
```

## Data Flow

### Bucket Creation

```
1. User applies S3Bucket CR
   │
   ▼
2. Kubernetes API Server persists CR
   │
   ▼
3. S3BucketController.Reconcile() triggered
   │
   ├─▶ Fetch CR from Kubernetes
   │
   ├─▶ mapper.CRToDomainBucket(cr)
   │   └─▶ Converts infrav1alpha1.S3Bucket → domain/s3.Bucket
   │
   ├─▶ s3UseCase.SyncBucket(domainBucket)
   │   │
   │   ├─▶ bucket.Validate() (domain logic)
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
   └─▶ Update CR Status
```

## Dependency Injection

**Location**: `cmd/main.go`:

```go
func main() {
// ... setup manager ...

// Build AWS config
awsConfig, _ := config.LoadDefaultConfig(context.Background())

// Create adapters
s3Repo := s3adapter.NewRepository(awsConfig)
lambdaRepo := lambdaadapter.NewRepository(awsConfig)

// Create use cases
s3UseCase := s3usecase.NewBucketUseCase(s3Repo)
lambdaUseCase := lambdausecase.NewFunctionUseCase(lambdaRepo)

// Create controllers with injected dependencies
s3Controller := &controllers.S3BucketReconciler{
        Client:    mgr.GetClient(),
        Scheme:    mgr.GetScheme(),
        S3UseCase: s3UseCase,  // ← Injected
}

s3Controller.SetupWithManager(mgr)

mgr.Start(ctrl.SetupSignalHandler())
}
```

## Testability

### 1. Domain Tests (Pure)

**Code:**

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

### 2. Use Case Tests (with Mock Repository)

**Code:**

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

// Mock expectations
repo.On("Exists", mock.Anything, "test-bucket", "us-east-1").
        Return(false, nil)
repo.On("Create", mock.Anything, bucket).
        Return(nil)
repo.On("Configure", mock.Anything, bucket).
        Return(nil)

// Execute
err := usecase.CreateBucket(context.Background(), bucket)

// Assert
assert.NoError(t, err)
repo.AssertExpectations(t)
}
```

### 3. Controller Tests (with Mock UseCase)

**Code:**

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

// ... test implementation
}
```

## Advantages of This Architecture

### ✅ For Kubernetes Operators

1. **Idempotency**: Use cases ensure idempotent operations
2. **Simple Reconcile**: Controller only orchestrates, no logic
3. **Status Reporting**: Easy to update status based on domain state

### ✅ For Testing

1. **Domain**: Pure tests, no mocks
2. **Use Cases**: Mock repositories
3. **Controllers**: Mock use cases
4. **Adapters**: Integration tests with AWS (optional)

### ✅ For Maintenance

1. **Clear Separation**: Each layer has a single responsibility
2. **Low Coupling**: Changes in AWS don't affect domain
3. **High Cohesion**: Related code stays together

### ✅ For Extensibility

1. **Multiple Clouds**: Swap AWS adapter for GCP/Azure
2. **Multiple Backends**: Add adapter for Terraform/Pulumi
3. **Evolution**: Change AWS SDK v2 → v3 without affecting domain

## References

- [Clean Architecture in Go](https://pkritiotis.io/clean-architecture-in-golang/)
- [AWS SDK for Go v2 Developer Guide](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Amazon S3 examples using SDK for Go V2](https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html)
- [Kubernetes Operator SDK Tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)
- [Go Clean Architecture](https://github.com/bxcodec/go-clean-arch)
