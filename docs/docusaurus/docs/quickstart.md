# Quick Start - Infra Operator

Get up and running with the infra-operator in under 5 minutes!

## Prerequisites

- Go 1.21+
- kubectl
- Docker or OrbStack
- Task (go-task)

Install missing tools:
```bash
brew install go kubectl go-task
# Install OrbStack from https://orbstack.dev/
```

## 1. Setup Environment

**Command:**

```bash
# Navigate to project
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# Run complete setup (installs tools, starts LocalStack)
task setup
```

## 2. Start Development

Choose your workflow:

### Option A: Local Development (Quick Testing)

Run operator locally without Kubernetes deployment:

```bash
task dev
```

### Option B: Run Against Cluster (Recommended)

Run operator locally, managing resources in your cluster:

```bash
# Terminal 1: Start operator
task run:local

# Terminal 2: Apply resources
kubectl apply -f test/e2e/fixtures/01-awsprovider.yaml
kubectl apply -f test/e2e/fixtures/02-s3bucket.yaml

# Watch resources
kubectl get awsproviders,s3buckets -w
```

### Option C: Full Cluster Deployment (Production-like)

Deploy operator as a Pod in Kubernetes:

```bash
task dev:full
```

## 3. Verify Everything Works

**Command:**

```bash
# Check AWSProvider is ready
kubectl get awsproviders
# NAME         REGION      ACCOUNT         READY
# localstack   us-east-1   000000000000    true

# Check S3Bucket is ready
kubectl get s3buckets
# NAME               BUCKET-NAME                        REGION      READY
# e2e-test-bucket    e2e-test-bucket-infra-operator    us-east-1   true

# Verify bucket exists in LocalStack
task localstack:aws -- s3 ls
# 2025-01-22 10:30:45 e2e-test-bucket-infra-operator

# Check detailed status
kubectl describe s3bucket e2e-test-bucket
```

## 4. View Logs

**Command:**

```bash
# If using dev:full (operator in cluster)
task k8s:logs

# If using run:local (operator on host)
# Logs stream in terminal, also saved to /tmp/log.txt
tail -f /tmp/log.txt
```

## 5. Test Resource Lifecycle

**Command:**

```bash
# Create a new bucket
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: my-test-bucket
  namespace: default
spec:
  providerRef:
    name: localstack
  bucketName: my-test-bucket-$(date +%s)
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true
  tags:
    test: quickstart
  deletionPolicy: Delete
EOF

# Watch it get created
kubectl get s3bucket my-test-bucket -w

# Check it exists in LocalStack
task localstack:aws -- s3 ls | grep my-test-bucket

# Delete the bucket
kubectl delete s3bucket my-test-bucket

# Verify it's cleaned up
task localstack:aws -- s3 ls | grep my-test-bucket
# (should return nothing)
```

## 6. Run Tests

**Command:**

```bash
# Unit tests (fast, no external dependencies)
task test:unit

# Integration tests (uses LocalStack)
task test:integration

# E2E tests (full stack)
task test:e2e

# All tests
task test:all
```

## 7. Clean Up

**Command:**

```bash
# Remove sample resources
task samples:delete

# Stop LocalStack (keeps data)
task localstack:stop

# Remove everything (operator, CRDs, LocalStack)
task clean:all
```

## Next Steps

- **Read the [Development Guide](/advanced/development)** for detailed workflows
- **Read the [Clean Architecture Guide](/advanced/clean-architecture)** to understand the codebase
- **Check [AWS Services](/services/networking/vpc)** for all supported services
- **Implement more controllers** using S3 as a template
- **Add webhooks** for validation and mutation
- **Deploy to production** using IRSA for secure AWS access

## Common Issues

### LocalStack won't start

**Command:**

```bash
# Check Docker is running
docker ps

# Restart LocalStack
task localstack:restart

# View logs
task localstack:logs
```

### Operator not reconciling

**Command:**

```bash
# Check operator is running
kubectl get pods -n infra-operator-system

# Check logs
task k8s:logs

# Restart operator
task k8s:restart
```

### S3Bucket stuck in NotReady

**Command:**

```bash
# Check AWSProvider is ready
kubectl get awsproviders

# Check bucket details
kubectl describe s3bucket <name>

# Check operator logs for errors
task k8s:logs | grep -i error
```

## Quick Reference

**Command:**

```bash
# Development
task dev                  # Local development
task run:local            # Run against cluster
task dev:full             # Full deployment
task dev:quick            # Quick rebuild

# Testing
task test:unit            # Unit tests
task test:integration     # Integration tests
task test:e2e             # E2E tests

# Kubernetes
task k8s:deploy           # Deploy operator
task k8s:logs             # View logs
task k8s:status           # Show status

# LocalStack
task localstack:start     # Start LocalStack
task localstack:aws -- s3 ls  # AWS CLI

# Cleanup
task clean:all            # Clean everything
```

## Getting Help

**Command:**

```bash
# List all available tasks
task --list

# Show detailed help
task help

# Show task source
task --summary <task-name>
```

For more detailed information, see:
- [Development Guide](/advanced/development)
- [Clean Architecture](/advanced/clean-architecture)
- [AWS Services](/services/networking/vpc)
- [Troubleshooting](/advanced/troubleshooting)
