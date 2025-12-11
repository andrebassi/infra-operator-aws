# Infra Operator - Architecture and Design

## Introduction

This document explains the architecture of **infra-operator**, a Kubernetes Operator that provisions AWS resources using the Custom Resources (CRs) pattern. The operator was developed following Kubernetes community best practices and uses AWS SDK Go v2 for direct interaction with AWS services.

## Architecture Overview

![Operator Architecture](/img/diagrams/operator-architecture.svg)

## Main Components

### 1. Custom Resource Definitions (CRDs)

CRDs define the structure of custom resources that the operator manages.

#### AWSProvider CRD

**Purpose**: Centralize AWS credentials and configurations.

**Main fields**:
- `region`: AWS region (required)
- `roleARN`: IAM role ARN for IRSA
- `accessKeyIDRef/secretAccessKeyRef`: Secret references for static credentials
- `defaultTags`: Tags applied to all resources

**Status**:
- `ready`: Indicates if credentials are valid
- `accountID`: AWS account ID
- `callerIdentity`: Authenticated identity ARN
- `conditions`: Status conditions

**File**: `api/v1alpha1/awsprovider_types.go`

#### S3Bucket CRD

**Purpose**: Manage S3 buckets with complete configuration.

**Main fields**:
- `providerRef`: Reference to AWSProvider
- `bucketName`: Bucket name (globally unique)
- `versioning`: Versioning configuration
- `encryption`: Encryption configuration
- `lifecycleRules`: Lifecycle rules
- `corsRules`: CORS configuration
- `publicAccessBlock`: Public access blocking
- `deletionPolicy`: Deletion policy (Delete, Retain, Orphan)

**Status**:
- `ready`: Bucket created and configured
- `arn`: Bucket ARN
- `region`: Region where bucket exists
- `bucketDomainName`: Bucket domain name

**File**: `api/v1alpha1/s3bucket_types.go`

### 2. Controllers (Reconcilers)

Controllers implement reconciliation logic - they observe the desired state (spec) and current state, and work to converge current state to desired.

#### AWSProvider Controller

**Responsibilities**:
1. Validate AWS credentials using STS GetCallerIdentity
2. Build AWS configuration (aws.Config) based on spec
3. Update status with account information
4. Re-validate credentials periodically (every 5 minutes)

**Reconciliation flow**:

![AWSProvider Reconciliation Flow](/img/diagrams/awsprovider-reconciliation-flow.svg)

**File**: `controllers/awsprovider_controller.go`

#### S3Bucket Controller

**Responsibilities**:
1. Create S3 bucket if it doesn't exist
2. Configure versioning, encryption, lifecycle, CORS, tags
3. Configure public access block
4. Manage lifecycle (creation, update, deletion)
5. Implement finalizers for controlled cleanup

**Reconciliation flow**:

![S3Bucket Reconciliation Flow](/img/diagrams/s3bucket-reconciliation-flow.svg)

**File**: `controllers/s3bucket_controller.go`

### 3. AWS Helper Package

Helper functions for AWS interaction.

**Main functions**:
- `GetAWSConfigFromProvider()`: Gets aws.Config from AWSProvider CR
- `buildAWSConfig()`: Builds AWS configuration with credentials
- `getSecretValue()`: Reads values from Kubernetes Secrets

**File**: `pkg/aws/provider.go`

## Implemented Design Patterns

### 1. Reconciliation Loop

The fundamental pattern of Kubernetes Operators. The controller observes the desired state (CR spec) and works continuously to make current state converge to desired.

**Characteristics**:
- **Idempotent**: Multiple executions produce the same result
- **Edge-triggered and Level-triggered**: Reacts to changes and also periodically checks state
- **Error handling**: Returns error for automatic requeue

### 2. Finalizers

Mechanism to execute cleanup before deleting a CR.

**Implementation in S3Bucket**:
```go
const s3BucketFinalizer = "aws-infra-operator.runner.codes/s3bucket-finalizer"

// When creating/updating:
if !controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
controllerutil.AddFinalizer(bucket, s3BucketFinalizer)
r.Update(ctx, bucket)
}

// When deleting:
if !bucket.ObjectMeta.DeletionTimestamp.IsZero() {
if controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
        // Execute cleanup
        r.deleteBucket(ctx, s3Client, bucket)

        // Remove finalizer
        controllerutil.RemoveFinalizer(bucket, s3BucketFinalizer)
        r.Update(ctx, bucket)
}
}
```

### 3. Status Conditions

Following the Kubernetes pattern of conditions to report state.

**Structure**:
```go
type Condition struct {
Type               string      // "Ready"
Status             string      // "True", "False", "Unknown"
LastTransitionTime metav1.Time
Reason             string      // "BucketReady", "CreationFailed"
Message            string      // Detailed description
}
```

### 4. Provider Pattern

Separation of credentials (AWSProvider) from resources (S3Bucket, RDS, etc).

**Advantages**:
- Reuse of credentials across multiple resources
- Centralized authentication validation
- Support for multiple providers (multi-account, multi-region)
- Easier credential rotation

### 5. Deletion Policies

Flexible policy to control what happens to AWS resources when CR is deleted.

**Options**:
- **Delete**: Removes AWS resource (default)
- **Retain**: Keeps AWS resource, removes only the CR
- **Orphan**: Detaches from operator but keeps everything

## Data Flow

### Creating an S3Bucket

![S3Bucket Creation Flow](/img/diagrams/s3bucket-creation-flow.svg)

### IRSA Authentication

![IRSA Authentication Flow](/img/diagrams/irsa-authentication-flow.svg)

## File Structure

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
│   ├── awsprovider_controller.go  # Validates AWS credentials
│   └── s3bucket_controller.go     # Manages S3 buckets
│
├── pkg/                       # Shared Libraries
│   └── aws/
│       └── provider.go        # Helper functions for AWS config
│
├── config/                    # Kubernetes Manifests
│   ├── crd/bases/             # CRD YAML manifests
│   ├── rbac/                  # RBAC (ServiceAccount, Role, Binding)
│   ├── manager/               # Deployment and Namespace
│   └── samples/               # Example CRs
│
├── docs/                      # Documentation
│   ├── ARCHITECTURE.md        # This file
│   └── DEPLOYMENT_GUIDE.md    # Deployment guide
│
├── main.go                    # Operator entry point
├── Dockerfile                 # Container image build
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
├── CLAUDE.md                  # Complete technical documentation
└── README.md                  # User-facing documentation
```

## Architectural Decisions

### Why Go SDK v2 instead of ACK?

**Decision**: Use AWS SDK for Go v2 directly.

**Alternative considered**: AWS Controllers for Kubernetes (ACK)

**Reasons**:

1. **Full Control**:
   - Custom reconciliation logic
   - Business-specific validations
   - Custom error handling

2. **Deployment Simplicity**:
   - Single operator for all services
   - No need to install multiple ACK controllers
   - Lower resource overhead

3. **Learning**:
   - Better understanding of operator patterns
   - Deep knowledge of AWS APIs
   - Flexibility to implement custom features

4. **Dependencies**:
   - Fewer external dependencies
   - Simpler SDK updates
   - No dependency on ACK roadmap

**Trade-offs**:
- More code to maintain
- Need to implement features manually (pagination, retries)
- Responsibility to track AWS API changes

### Why Provider Pattern?

**Decision**: Separate credentials (AWSProvider) from resources (S3, RDS, etc).

**Reasons**:

1. **Reuse**: One AWSProvider can be used by multiple resources
2. **Security**: Credentials isolated in a specific resource
3. **Multi-tenancy**: Different namespaces can have different providers
4. **Multi-account**: Supports access to multiple AWS accounts
5. **Rotation**: Facilitates credential rotation

**Alternative**: Direct credentials in each resource (very repetitive and insecure)

### Why Deletion Policies?

**Decision**: Allow users to choose what happens when deleting a CR.

**Reasons**:

1. **Safety**: Avoid accidental deletion of important data
2. **Flexibility**: Different use cases (dev vs prod)
3. **Compliance**: Some regulations require data retention
4. **Migration**: Facilitates migration to other tools

**Implemented policies**:
- `Delete`: Removes AWS resource (default for ephemeral environments)
- `Retain`: Keeps AWS resource (default for production)
- `Orphan`: Detaches but keeps everything

## Concurrency and Thread Safety

### Controller Runtime

Controller-runtime manages concurrency automatically:

- **Work Queue**: Serializes reconciliation events
- **Max Concurrent Reconciles**: Configurable (default: 1)
- **Retry Backoff**: Exponential backoff for errors

### Shared State

**No shared state between reconciliations**:
- Each reconciliation fetches current state from Kubernetes
- No shared cache between reconciles
- State persisted only in CRs

### Rate Limiting

**AWS SDK**:
- Implements automatic retry with exponential backoff
- Respects AWS rate limits
- Uses context.Context for timeout

**Kubernetes Client**:
- Configurable rate limiting
- Default: 5 QPS, burst 10

## Performance and Scalability

### Implemented Optimizations

1. **Smart Requeue**:
   - Ready resources: requeue every 5 minutes
   - Resources with error: requeue every 1 minute
   - Immediate changes: immediate requeue

2. **Minimize API Calls**:
   - HeadBucket before GetBucket (cheaper)
   - Apply configurations only if changed
   - Batch updates when possible

3. **Status Subresource**:
   - Status updates don't trigger new reconciliation
   - Reduces unnecessary reconciliations

### Current Limits

1. **Single Region per Provider**: One AWSProvider = one region
   - **Workaround**: Create multiple AWSProviders

2. **No Pagination**: Doesn't list large quantities of resources
   - **Impact**: Limited for operators managing resources

3. **Sequential Processing**: One resource at a time
   - **Impact**: Limited scalability for many resources

### Future Improvements

1. **Metrics**: Prometheus metrics for observability
2. **Webhooks**: Validation and defaults in admission
3. **Caching**: AWS information cache to reduce API calls
4. **Parallel Reconciliation**: MaxConcurrentReconciles > 1
5. **Health Checks**: Detailed health checks for AWS resources

## Security

### Implemented Principles

1. **Least Privilege**:
   - Minimum necessary RBAC
   - Service-specific IAM policies
   - Namespaced resources when possible

2. **Secrets Management**:
   - Credentials in Kubernetes Secrets
   - IRSA preferred (no long-lived credentials)
   - Never log credentials

3. **Container Security**:
   - Non-root user (UID 65532)
   - Read-only root filesystem
   - No privileged escalation
   - Distroless base image

4. **Network Security**:
   - HTTPS for AWS APIs
   - Certificate validation
   - No credential exposure in status

### Threat Model

**Considered threats**:
1. **Credential Leakage**: Mitigated with IRSA and Secrets
2. **Unauthorized Access**: Mitigated with RBAC
3. **Privilege Escalation**: Mitigated with security context
4. **MITM Attacks**: Mitigated with HTTPS

## Observability

### Logging

**Structured logging** with controller-runtime:
- Automatic context (namespace, name, kind)
- Log levels: info, error, debug
- JSON format (optional)

**Example**:
```go
logger := log.FromContext(ctx)
logger.Info("Creating S3 bucket", "bucket", bucketName)
logger.Error(err, "Failed to create bucket")
```

### Status Reporting

**All CRs expose**:
- `status.ready`: Boolean indicating if resource is ready
- `status.conditions`: Array of detailed conditions
- `status.lastSyncTime`: Timestamp of last reconciliation

### Health Checks

**Available endpoints**:
- `/healthz`: Liveness probe
- `/readyz`: Readiness probe
- `/metrics`: Prometheus metrics (future)

## Testing Strategy

### Test Levels

1. **Unit Tests** (future):
   - Controller logic testing
   - AWS SDK mock
   - Kubernetes client mock

2. **Integration Tests** (future):
   - Test with envtest (fake API server)
   - Complete reconciliation testing
   - No real AWS dependency

3. **E2E Tests** (future):
   - Test on real cluster
   - Test with real AWS
   - Validation of created resources

### Test Coverage Goals

- **Controllers**: > 80% coverage
- **AWS helpers**: > 90% coverage
- **E2E**: Happy paths and main error cases

## Conclusion

The **infra-operator** was architected following patterns established by the Kubernetes community, focusing on:

- **Simplicity**: Easy to understand and maintain
- **Extensibility**: Easy to add new AWS resources
- **Security**: IRSA, RBAC, least privilege
- **Reliability**: Finalizers, deletion policies, idempotency
- **Observability**: Status conditions, structured logging

The architecture allows future growth with addition of new AWS services, webhook implementation, and performance improvements, while maintaining a solid and well-structured foundation.
