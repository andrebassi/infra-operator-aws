# Development Guide - Infra Operator

This guide covers local development, testing, and deployment workflows for the infra-operator.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Setup](#local-development-setup)
3. [Development Workflows](#development-workflows)
4. [Testing](#testing)
5. [Deployment](#deployment)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools

**Command:**

```bash
# Check if tools are installed
go version          # Go 1.21+
kubectl version     # Kubernetes CLI
docker --version    # Docker or OrbStack
task --version      # Task (go-task) automation tool
```

### Installing Missing Tools

**Command:**

```bash
# macOS (using Homebrew)
brew install go
brew install kubectl
brew install go-task
brew install jq

# Install OrbStack (Docker alternative for macOS)
# Download from: https://orbstack.dev/

# Or use Docker Desktop
# Download from: https://www.docker.com/products/docker-desktop
```

### Kubernetes Cluster

You need a local Kubernetes cluster. Options:

- **OrbStack** (recommended for macOS) - Built-in Kubernetes
- **Docker Desktop** - Enable Kubernetes in settings
- **kind** - `brew install kind && kind create cluster`
- **minikube** - `brew install minikube && minikube start`

**Verify cluster:**

```bash
kubectl cluster-info
kubectl get nodes
```

## Local Development Setup

### 1. Initial Setup

Run the complete setup (only needed once):

**Command:**

```bash
# Clone/navigate to project
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# Run setup (installs tools, starts LocalStack)
task setup
```

This will:
- Verify all required tools are installed
- Start LocalStack for AWS simulation
- Initialize LocalStack with test resources

### 2. Verify LocalStack

**Command:**

```bash
# Check LocalStack health
task localstack:health

# List services
curl http://localhost:4566/_localstack/health | jq

# Test AWS CLI against LocalStack
task localstack:aws -- s3 ls
task localstack:aws -- sqs list-queues
task localstack:aws -- dynamodb list-tables
```

### 3. Environment Variables

Check `.env` file (created from `.env.example`):

```bash
cat .env
```

Key variables:
- `AWS_ENDPOINT_URL=http://localhost:4566` - LocalStack endpoint
- `AWS_REGION=us-east-1` - Default region
- `AWS_ACCESS_KEY_ID=test` - LocalStack credentials
- `AWS_SECRET_ACCESS_KEY=test`

## Development Workflows

### Workflow 1: Local Development (No Kubernetes)

Run the operator locally without deploying to Kubernetes. Useful for quick iterations and debugging.

**Command:**

```bash
# Start development mode
task dev
```

This will:
1. Start LocalStack (if not running)
2. Generate code (if controller-gen is available)
3. Run the operator binary with `--clean-arch=true`

The operator will use your local kubeconfig to connect to the cluster but will create AWS resources in LocalStack.

**Logs will stream in real-time.**

Press `Ctrl+C` to stop.

### Workflow 2: Run Against Cluster (Recommended)

Run the operator locally but reconcile resources in your Kubernetes cluster:

**Command:**

```bash
# Start operator locally, watching cluster resources
task run:local
```

This mode:
- Runs operator on your machine (not in-cluster)
- Watches Kubernetes resources (S3Buckets, AWSProviders, etc.)
- Creates AWS resources in LocalStack
- Logs to `/tmp/log.txt` and console

**In another terminal, apply test resources:**

**Command:**

```bash
# Apply samples
kubectl apply -f config/samples/awsprovider_sample.yaml
kubectl apply -f config/samples/s3bucket_sample.yaml

# Watch status
kubectl get awsproviders -w
kubectl get s3buckets -w

# Check details
kubectl describe s3bucket my-app-bucket
```

### Workflow 3: Full Cluster Deployment

Deploy the operator as a Pod in Kubernetes (production-like):

**Command:**

```bash
# Complete deployment workflow
task dev:full
```

This will:
1. Run unit tests
2. Build Docker image
3. Deploy to Kubernetes
4. Install CRDs
5. Apply sample resources
6. Show status

**Monitor the deployment:**

**Command:**

```bash
# View operator logs
task k8s:logs

# Check status
task k8s:status

# View samples
task samples:status
```

### Workflow 4: Quick Rebuild

After making code changes, quickly rebuild and redeploy:

**Command:**

```bash
# Rebuild image and restart operator
task dev:quick
```

This is faster than `dev:full` because it skips tests and only rebuilds the image.

## Testing

### Unit Tests

Run pure unit tests (no Kubernetes, no AWS):

**Command:**

```bash
# Run unit tests with coverage
task test:unit
```

Tests are in:
- `internal/domain/s3/bucket_test.go` - Domain logic tests
- Add more tests in `*_test.go` files

**Coverage report:**

```bash
# View detailed coverage
go tool cover -html=coverage.out
```

### Integration Tests

Test against LocalStack (AWS simulation):

**Command:**

```bash
# Run integration tests
task test:integration
```

Integration tests:
- Use LocalStack endpoint
- Create real S3 buckets, SQS queues, etc. (locally)
- Test AWS SDK interactions
- Located in `test/integration/`

### End-to-End Tests

Full E2E tests with operator deployed in cluster:

**Command:**

```bash
# Run complete E2E test suite
task test:e2e
```

This will:
1. Deploy operator to cluster
2. Start LocalStack
3. Apply test fixtures from `test/e2e/fixtures/`
4. Verify resources are created
5. Show status

**E2E Test Fixtures:**
- `test/e2e/fixtures/01-awsprovider.yaml` - LocalStack provider
- `test/e2e/fixtures/02-s3bucket.yaml` - Test buckets

### Run All Tests

**Command:**

```bash
# Run unit, integration, and E2E tests
task test:all
```

## Deployment

### Deploy to Local Cluster

**Command:**

```bash
# Install CRDs
task k8s:install-crds

# Deploy operator
task k8s:deploy

# Verify
task k8s:status
```

### Apply Samples

**Command:**

```bash
# Create sample resources (AWSProvider + S3Bucket)
task samples:apply

# Check status
task samples:status

# View detailed status
kubectl describe s3bucket -n default
```

### View Logs

**Command:**

```bash
# Stream operator logs
task k8s:logs

# Or directly with kubectl
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager \
  -f --tail=100
```

### Restart Operator

**Command:**

```bash
# Restart operator deployment
task k8s:restart
```

### Undeploy

**Command:**

```bash
# Remove operator (keeps CRDs)
task k8s:undeploy

# Remove everything including CRDs
task clean:all
```

## Building and Publishing

### Build Binary

**Command:**

```bash
# Build operator binary
task build
```

Binary will be at: `bin/manager`

### Build Docker Image

**Command:**

```bash
# Build image with default tag
task docker:build
```

Image will be tagged as:
- `ttl.sh/infra-operator:dev-YYYYMMDD-HHMMSS`
- `ttl.sh/infra-operator:latest`

### Push to Registry

**Command:**

```bash
# Build and push to ttl.sh (ephemeral registry)
task docker:push
```

**Using ttl.sh:**
- Images auto-expire after 24 hours
- No authentication required
- Perfect for testing
- Format: `ttl.sh/infra-operator:tag`

**For production, use a real registry:**

Update `Taskfile.yaml`:

```yaml
vars:
  DOCKER_REGISTRY: ghcr.io/your-org
  # or: docker.io/your-username
  # or: your-registry.io
```

## LocalStack Management

### Start/Stop LocalStack

**Command:**

```bash
# Start LocalStack
task localstack:start

# Stop LocalStack
task localstack:stop

# Restart LocalStack
task localstack:restart

# View logs
task localstack:logs
```

### Health Check

**Command:**

```bash
# Check LocalStack health
task localstack:health

# Or manually
curl http://localhost:4566/_localstack/health | jq
```

### Run AWS CLI Commands

**Command:**

```bash
# List S3 buckets in LocalStack
task localstack:aws -- s3 ls

# Create a test bucket
task localstack:aws -- s3 mb s3://test-bucket

# List SQS queues
task localstack:aws -- sqs list-queues

# Describe DynamoDB table
task localstack:aws -- dynamodb describe-table --table-name test-table
```

## Code Generation

### Generate DeepCopy and CRDs

**Command:**

```bash
# Generate code (requires controller-gen)
task generate
```

**Install controller-gen:**

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

Then update `Taskfile.yaml` to uncomment the controller-gen line.

### Format Code

**Command:**

```bash
# Format all Go code
task fmt
```

### Lint Code

**Command:**

```bash
# Run linter
task lint
```

**Install golangci-lint:**

```bash
brew install golangci-lint
```

## Troubleshooting

### Operator Not Reconciling

**Check operator is running:**

```bash
task k8s:status
kubectl get pods -n infra-operator-system
```

**Check logs for errors:**

```bash
task k8s:logs
```

**Common issues:**
- AWSProvider not Ready - check credentials
- CRDs not installed - run `task k8s:install-crds`
- RBAC issues - check `config/rbac/`

### LocalStack Not Working

**Check LocalStack is running:**

```bash
docker ps | grep localstack
task localstack:health
```

**Restart LocalStack:**

```bash
task localstack:restart
```

**Check LocalStack logs:**

```bash
task localstack:logs
```

**Common issues:**
- Port 4566 already in use - stop conflicting services
- Docker not running - start Docker/OrbStack
- Init script failed - check `hack/localstack-init.sh`

### S3Bucket Stuck in NotReady

**Check AWSProvider:**

```bash
kubectl get awsproviders
kubectl describe awsprovider localstack
```

**Check S3Bucket events:**

```bash
kubectl describe s3bucket my-app-bucket
```

**Check bucket exists in LocalStack:**

```bash
task localstack:aws -- s3 ls
```

**Common issues:**
- Provider not ready - wait for AWSProvider to reconcile
- Bucket name not unique - change bucket name
- LocalStack not accessible - check endpoint URL

### Resources Not Deleting

**Check finalizers:**

```bash
kubectl get s3bucket my-bucket -o yaml | grep finalizers -A 5
```

**Force delete (if stuck):**

```bash
kubectl patch s3bucket my-bucket \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

kubectl delete s3bucket my-bucket
```

### Unit Tests Failing

**Run tests with verbose output:**

```bash
go test -v -race ./internal/...
```

**Run specific test:**

```bash
go test -v -run TestBucket_Validate ./internal/domain/s3/
```

**Common issues:**
- Import paths wrong - check `go.mod` module name
- Missing dependencies - run `go mod download`

### Integration Tests Failing

**Ensure LocalStack is running:**

```bash
task localstack:start
task localstack:health
```

**Check AWS_ENDPOINT_URL:**

```bash
echo $AWS_ENDPOINT_URL  # Should be http://localhost:4566
```

**Run integration tests with debug:**

```bash
export AWS_SDK_LOG_LEVEL=debug
task test:integration
```

## Development Tips

### Faster Development Loop

1. **Use `task run:local`** instead of full cluster deployment
2. **Keep LocalStack running** between test runs
3. **Use `deletionPolicy: Delete`** in test resources for auto-cleanup
4. **Watch logs in separate terminal** with `task k8s:logs`

### Debugging

**Enable debug logging:**

Update `cmd/main.go`:

```go
opts := zap.Options{
Development: true,
// Add this:
Level: zapcore.DebugLevel,
}
```

Or run with flag:

```bash
go run ./cmd/main.go --zap-log-level=debug
```

**Use delve debugger:**

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug operator
dlv debug ./cmd/main.go -- --clean-arch=true
```

### Clean Start

Remove all resources and start fresh:

**Command:**

```bash
# Clean everything
task clean:all

# Start fresh
task setup
task dev:full
```

## Common Task Commands

Quick reference of most-used commands:

**Command:**

```bash
# Setup
task setup                    # Initial setup
task install-tools            # Verify tools

# Development
task dev                      # Local dev (no K8s)
task run:local                # Run against cluster
task dev:full                 # Complete deployment
task dev:quick                # Quick rebuild

# Testing
task test:unit                # Unit tests
task test:integration         # Integration tests
task test:e2e                 # E2E tests
task test:all                 # All tests

# Build
task build                    # Build binary
task docker:build             # Build image
task docker:push              # Push image

# Kubernetes
task k8s:install-crds         # Install CRDs
task k8s:deploy               # Deploy operator
task k8s:status               # Show status
task k8s:logs                 # Stream logs
task k8s:restart              # Restart operator
task k8s:undeploy             # Remove operator

# Samples
task samples:apply            # Apply samples
task samples:status           # Check samples
task samples:delete           # Remove samples

# LocalStack
task localstack:start         # Start LocalStack
task localstack:stop          # Stop LocalStack
task localstack:health        # Health check
task localstack:logs          # View logs
task localstack:aws -- CMD    # AWS CLI

# Cleanup
task clean                    # Clean temp files
task clean:all                # Clean everything

# Help
task help                     # Show all tasks
task --list                   # List tasks
```

## Next Steps

1. **Implement more controllers** - Use S3 as template for Lambda, DynamoDB, etc.
2. **Add webhooks** - Validation and mutation webhooks
3. **Add metrics** - Prometheus metrics export
4. **Improve tests** - More integration and E2E tests
5. **Production deployment** - Helm chart, IRSA setup

## Resources

- [Kubebuilder Documentation](https://book.kubebuilder.io/)
- [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)
- [LocalStack Documentation](https://docs.localstack.cloud/)
- [Task Documentation](https://taskfile.dev/)
- [Clean Architecture Guide](/advanced/clean-architecture)
