---
title: 'REST API'
description: 'Manage AWS infrastructure via HTTP API'
sidebar_position: 1
---

# REST API

Infra Operator AWS can be run as a REST API server, allowing you to manage AWS infrastructure via HTTP requests.

## Starting the Server

**Command:**

```bash
# Start on default port (8080)
./infra-operator serve

# Custom port
./infra-operator serve --port 3000

# With API key authentication
./infra-operator serve --api-keys "my-secret-key"

# With LocalStack endpoint
./infra-operator serve --endpoint http://localhost:4566
```

## Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--port`, `-p` | Server port | 8080 |
| `--host` | Server host | 0.0.0.0 |
| `--state-dir` | State directory | ~/.infra-operator/state |
| `--region` | AWS region | us-east-1 |
| `--endpoint` | AWS endpoint (LocalStack) | - |
| `--api-keys` | API keys (comma-separated) | - |
| `--cors-origins` | Allowed CORS origins | * |

## Authentication

When `--api-keys` is provided, all requests must include authentication:

```bash
# Via X-API-Key header
curl -H "X-API-Key: my-key" http://localhost:8080/api/v1/resources

# Via Bearer token
curl -H "Authorization: Bearer my-key" http://localhost:8080/api/v1/resources
```

## Endpoints

### Health Check

**Command:**

```bash
GET /health
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.1",
    "timestamp": "2025-11-26T12:00:00Z"
  }
}
```

### Plan (Execution Plan)

Generates a plan showing what would be created, updated, or deleted.

**Command:**

```bash
POST /api/v1/plan
Content-Type: application/yaml
```

**Request (YAML):**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: my-vpc
spec:
  cidrBlock: "10.0.0.0/16"
```

**Request (JSON):**
```json
{
  "resources": [
    {
      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
      "kind": "VPC",
      "metadata": { "name": "my-vpc" },
      "spec": { "cidrBlock": "10.0.0.0/16" }
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "toCreate": [
      { "kind": "VPC", "name": "my-vpc", "action": "create" }
    ],
    "toUpdate": [],
    "toDelete": [],
    "noChange": [],
    "summary": {
      "create": 1,
      "update": 0,
      "delete": 0,
      "noChange": 0
    }
  }
}
```

### Apply (Apply Resources)

Creates or updates resources in AWS.

**Command:**

```bash
POST /api/v1/apply
Content-Type: application/yaml
```

**Request:**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: my-environment
spec:
  vpcCIDR: "10.100.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.micro
```

**Response:**
```json
{
  "success": true,
  "data": {
    "created": [
      {
        "kind": "ComputeStack",
        "name": "my-environment",
        "awsResources": {
          "vpcId": "vpc-0abc123",
          "publicSubnetId": "subnet-0def456",
          "internetGatewayId": "igw-0ghi789"
        }
      }
    ],
    "updated": [],
    "failed": [],
    "skipped": [],
    "summary": {
      "created": 1,
      "updated": 0,
      "failed": 0,
      "skipped": 0
    }
  }
}
```

### Delete (Delete Resources)

Removes resources from AWS.

**Command:**

```bash
DELETE /api/v1/resources
Content-Type: application/yaml
```

or

**Command:**

```bash
POST /api/v1/delete
Content-Type: application/yaml
```

**Request:**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: my-environment
spec:
  vpcCIDR: "10.100.0.0/16"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "deleted": [
      { "kind": "ComputeStack", "name": "my-environment" }
    ],
    "failed": [],
    "skipped": [],
    "summary": {
      "deleted": 1,
      "failed": 0,
      "skipped": 0
    }
  }
}
```

### Get (List Resources)

Lists resources from local state.

**Command:**

```bash
# All resources
GET /api/v1/resources

# By type
GET /api/v1/resources/VPC
GET /api/v1/resources/ComputeStack

# Shortcuts
GET /api/v1/vpcs
GET /api/v1/compute-stacks
GET /api/v1/ec2-instances
```

**Response:**
```json
{
  "success": true,
  "data": {
    "resources": [
      {
        "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
        "kind": "VPC",
        "name": "my-vpc",
        "awsResources": {
          "vpcId": "vpc-0abc123"
        },
        "status": {
          "ready": true,
          "state": "available"
        },
        "createdAt": "2025-11-26T12:00:00Z",
        "updatedAt": "2025-11-26T12:00:00Z"
      }
    ],
    "count": 1
  }
}
```

## Practical Examples

### 1. Create Complete Infrastructure via API

**Command:**

```bash
# Create ComputeStack
curl -X POST http://localhost:8080/api/v1/apply \
  -H "Content-Type: application/yaml" \
  --data-binary @- << 'EOF'
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: production
  namespace: infra
spec:
  vpcCIDR: "10.50.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.micro
    userData: |
      #!/bin/bash
      yum update -y
      yum install -y docker
EOF
```

### 2. Integrate with CI/CD (GitLab/GitHub Actions)

**Example:**

```yaml
# .gitlab-ci.yml
deploy:
  script:
    - |
      curl -X POST $INFRA_API_URL/api/v1/apply \
        -H "X-API-Key: $INFRA_API_KEY" \
        -H "Content-Type: application/yaml" \
        --data-binary @infrastructure.yaml
```

### 3. Automation Script

**Command:**

```bash
#!/bin/bash
API_URL="http://localhost:8080"

# Plan
echo "Generating plan..."
curl -s -X POST "$API_URL/api/v1/plan" \
  -H "Content-Type: application/yaml" \
  --data-binary @infra.yaml | jq .

# Confirm and apply
read -p "Apply? (y/n) " confirm
if [ "$confirm" = "y" ]; then
  curl -s -X POST "$API_URL/api/v1/apply" \
-H "Content-Type: application/yaml" \
--data-binary @infra.yaml | jq .
fi
```

### 4. Python SDK

**Example:**

```python
import requests
import yaml

API_URL = "http://localhost:8080"
API_KEY = "my-key"

def plan(resources):
resp = requests.post(
        f"{API_URL}/api/v1/plan",
        headers={
            "X-API-Key": API_KEY,
            "Content-Type": "application/yaml"
        },
        data=yaml.dump_all(resources)
)
return resp.json()

def apply(resources):
resp = requests.post(
        f"{API_URL}/api/v1/apply",
        headers={
            "X-API-Key": API_KEY,
            "Content-Type": "application/yaml"
        },
        data=yaml.dump_all(resources)
)
return resp.json()

# Usage
vpc = {
"apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
"kind": "VPC",
"metadata": {"name": "my-vpc"},
"spec": {"cidrBlock": "10.0.0.0/16"}
}

result = plan([vpc])
print(f"Resources to create: {result['data']['summary']['create']}")
```

## Response Structure

### Success Response

**JSON:**

```json
{
  "success": true,
  "data": { ... }
}
```

### Error Response

**JSON:**

```json
{
  "success": false,
  "error": {
"code": "INVALID_REQUEST",
"message": "Error description",
"details": "Additional details"
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Malformed request |
| `NO_RESOURCES` | No resources provided |
| `UNAUTHORIZED` | API key not provided |
| `INVALID_API_KEY` | Invalid API key |
| `PLAN_FAILED` | Error generating plan |
| `APPLY_FAILED` | Error applying resources |
| `DELETE_FAILED` | Error deleting resources |
| `GET_FAILED` | Error listing resources |
| `RATE_LIMITED` | Too many requests |
| `INTERNAL_ERROR` | Internal server error |

## Comparison: K8s vs CLI vs API

| Feature | Kubernetes | CLI | API |
|---------|------------|-----|-----|
| Dependency | K8s Cluster | None | None |
| Authentication | RBAC | AWS Credentials | API Key + AWS |
| State | etcd (CRDs) | Local file | Local file |
| Integration | kubectl, ArgoCD | Shell scripts | HTTP clients |
| Multi-tenant | Namespaces | Directories | API Keys |
| Use Case | GitOps, K8s-native IaC | Local automation | CI/CD, SDKs |

## Troubleshooting

### Server won't start

**Command:**

```bash
# Check if port is in use
lsof -i :8080

# Check logs
./infra-operator serve 2>&1 | tee server.log
```

### AWS authentication error

**Command:**

```bash
# Verify credentials
aws sts get-caller-identity

# Use LocalStack for testing
./infra-operator serve --endpoint http://localhost:4566
```

### CORS blocked

**Command:**

```bash
# Allow specific origins
./infra-operator serve --cors-origins "http://localhost:3000,https://myapp.com"
```

## Next Steps

- [CLI Mode](/features/cli) - Command-line management
- [Drift Detection](/features/drift-detection) - Detect configuration changes
- [ComputeStack](/resources/computestack) - Complete infrastructure in one resource