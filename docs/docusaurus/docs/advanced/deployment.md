# Deployment Guide - Infra Operator

This guide describes each step needed to deploy the **infra-operator** in a Kubernetes cluster.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building the Operator](#building-the-operator)
3. [IAM/IRSA Configuration](#iamirsa-configuration)
4. [Cluster Installation](#cluster-installation)
5. [Verification](#verification)
6. [First Resource](#first-resource)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### 1. Kubernetes Cluster

- **Version**: 1.28 or higher
- **Access**: `kubectl` configured and working
- **Type**: Any distribution (EKS, GKE, AKS, K3s, minikube, etc.)

**Verify:**

```bash
kubectl version --short
kubectl cluster-info
```

### 2. Build Tools

- **Go**: 1.21 or higher (for local build)
- **Docker or Podman**: To build the image
- **Make**: For task automation

**Verify:**

```bash
go version
docker --version
make --version
```

### 3. AWS Account

- **AWS Account** with administrator permissions (for initial setup)
- **AWS CLI** configured (optional but recommended)

**Verify:**

```bash
aws sts get-caller-identity
```

---

## Building the Operator

### Option 1: Local Build + Docker

**Command:**

```bash
# 1. Navigate to project directory
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# 2. Download Go dependencies
make mod-download

# 3. Build binary (optional - for local testing)
make build
./bin/manager --help

# 4. Build Docker image
make docker-build IMG=infra-operator:v1.0.0

# 5. Tag for registry (example with ttl.sh - ephemeral registry)
docker tag infra-operator:v1.0.0 ttl.sh/infra-operator:v1.0.0

# 6. Push to registry
docker push ttl.sh/infra-operator:v1.0.0
```

### Option 2: Multi-arch Build with Buildx

**Command:**

```bash
# Build and push for multiple architectures
make docker-buildx REGISTRY=ttl.sh IMG=infra-operator:v1.0.0
```

### Option 3: Build for Private Registry

**Command:**

```bash
# Login to registry
docker login registry.example.com

# Build and push
docker build -t registry.example.com/infra-operator:v1.0.0 .
docker push registry.example.com/infra-operator:v1.0.0

# Update deployment
# Edit config/manager/deployment.yaml:
#   image: registry.example.com/infra-operator:v1.0.0
```

---

## IAM/IRSA Configuration

### For EKS with IRSA (Recommended)

#### 1. Create IAM Policy

**Command:**

```bash
# Create policy.json file
cat > /tmp/infra-operator-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Sid": "STSPermissions",
      "Effect": "Allow",
      "Action": ["sts:GetCallerIdentity"],
      "Resource": "*"
},
{
      "Sid": "S3FullAccess",
      "Effect": "Allow",
      "Action": ["s3:*"],
      "Resource": "*"
},
{
      "Sid": "RDSFullAccess",
      "Effect": "Allow",
      "Action": ["rds:*"],
      "Resource": "*"
},
{
      "Sid": "EC2FullAccess",
      "Effect": "Allow",
      "Action": ["ec2:*"],
      "Resource": "*"
},
{
      "Sid": "SQSFullAccess",
      "Effect": "Allow",
      "Action": ["sqs:*"],
      "Resource": "*"
}
  ]
}
EOF

# Create the policy
aws iam create-policy \
  --policy-name InfraOperatorPolicy \
  --policy-document file:///tmp/infra-operator-policy.json
```

#### 2. Create IAM Role with Trust Policy for IRSA

**Command:**

```bash
# Get EKS cluster OIDC provider
CLUSTER_NAME=your-cluster-name
OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e 's|^https://||')

# Create trust policy
cat > /tmp/trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
}
  ]
}
EOF

# Create role
aws iam create-role \
  --role-name infra-operator-role \
  --assume-role-policy-document file:///tmp/trust-policy.json

# Attach policy to role
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/InfraOperatorPolicy
```

#### 3. Annotate Service Account

**Command:**

```bash
# Edit config/rbac/service_account.yaml
# Uncomment and adjust the annotation:

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

cat > config/rbac/service_account.yaml <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: infra-operator-controller-manager
  namespace: infra-operator-system
  annotations:
eks.amazonaws.com/role-arn: arn:aws:iam::${ACCOUNT_ID}:role/infra-operator-role
EOF
```

### For Non-EKS Kubernetes (Static Credentials)

**Command:**

```bash
# Create Secret with AWS credentials
kubectl create namespace infra-operator-system

kubectl create secret generic aws-credentials \
  -n infra-operator-system \
  --from-literal=access-key-id=AKIAXXXXXXXXXXXXXXXX \
  --from-literal=secret-access-key=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

---

## Cluster Installation

### Complete Installation (Automated Method)

**Command:**

```bash
# Installs CRDs and deploys the operator
make install-complete
```

### Step-by-Step Installation (Manual Method)

#### 1. Create Namespace

**Command:**

```bash
kubectl apply -f config/manager/namespace.yaml
```

#### 2. Install CRDs

**Command:**

```bash
kubectl apply -f config/crd/bases/aws-infra-operator.runner.codes_awsproviders.yaml
kubectl apply -f config/crd/bases/aws-infra-operator.runner.codes_s3buckets.yaml

# Verify installation
kubectl get crds | grep aws-infra-operator.runner.codes
```

#### 3. Configure RBAC

**Command:**

```bash
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
```

#### 4. Deploy the Operator

**Command:**

```bash
# If you pushed to a different registry, edit first:
# vim config/manager/deployment.yaml
# Change the line: image: infra-operator:latest
# To: image: ttl.sh/infra-operator:v1.0.0

kubectl apply -f config/manager/deployment.yaml
```

---

## Verification

### 1. Check Pods

**Command:**

```bash
# Wait for pod to become Running
kubectl get pods -n infra-operator-system

# Expected output example:
# NAME                                              READY   STATUS    RESTARTS   AGE
# infra-operator-controller-manager-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
```

### 2. Check Logs

**Command:**

```bash
# View operator logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager \
  -f

# Look for:
# - "starting manager"
# - No authentication errors
```

### 3. Check CRDs

**Command:**

```bash
# List installed CRDs
kubectl get crds | grep aws-infra-operator.runner.codes

# Test explain (verifies schema)
kubectl explain awsprovider.spec
kubectl explain s3bucket.spec
```

### 4. Health Checks

**Command:**

```bash
# Port-forward to health endpoint
kubectl port-forward -n infra-operator-system \
  deploy/infra-operator-controller-manager 8081:8081

# In another terminal, test endpoints
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Both should return: ok
```

---

## First Resource

### 1. Create AWSProvider

**Command:**

```bash
# For IRSA (EKS)
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-default
  namespace: default
spec:
  region: us-east-1
  roleARN: arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/infra-operator-role
  defaultTags:
    managed-by: infra-operator
    environment: test
EOF
```

OR for static credentials:

**Example:**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  access-key-id: AKIAXXXXXXXXXXXXXXXX
  secret-access-key: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-default
  namespace: default
spec:
  region: us-east-1
  accessKeyIDRef:
    name: aws-credentials
    key: access-key-id
  secretAccessKeyRef:
    name: aws-credentials
    key: secret-access-key
  defaultTags:
    managed-by: infra-operator
EOF
```

### 2. Verify Provider

**Command:**

```bash
# Wait for provider to become Ready
kubectl get awsprovider aws-default -w

# View details
kubectl describe awsprovider aws-default

# Verify status.ready = true and accountID is populated
kubectl get awsprovider aws-default -o jsonpath='{.status.ready}'
kubectl get awsprovider aws-default -o jsonpath='{.status.accountID}'
```

### 3. Create S3 Bucket

**Command:**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: test-bucket
  namespace: default
spec:
  providerRef:
    name: aws-default
  bucketName: infra-operator-test-$(date +%s)
  encryption:
    algorithm: AES256
  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true
  deletionPolicy: Delete
EOF
```

### 4. Verify Bucket

**Command:**

```bash
# Watch bucket status
kubectl get s3bucket test-bucket -w

# View details
kubectl describe s3bucket test-bucket

# Verify in AWS
BUCKET_NAME=$(kubectl get s3bucket test-bucket -o jsonpath='{.spec.bucketName}')
aws s3 ls | grep $BUCKET_NAME
```

### 5. Test Deletion

**Command:**

```bash
# Delete bucket
kubectl delete s3bucket test-bucket

# Verify it was removed from AWS
aws s3 ls | grep $BUCKET_NAME || echo "Bucket deleted successfully"
```

---

## Troubleshooting

### Problem: Pod doesn't start

**Command:**

```bash
# Check events
kubectl describe pod -n infra-operator-system \
  -l control-plane=controller-manager

# Check logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager --previous

# Common causes:
# - Image not found (ImagePullBackOff)
# - Insufficient resources
# - Incorrect RBAC
```

### Problem: AWSProvider not Ready

**Command:**

```bash
# View operator logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager | grep -i error

# View provider status
kubectl describe awsprovider <name>

# Common causes:
# - Misconfigured IRSA (incorrect trust policy)
# - Service account without annotation
# - Invalid credentials
# - Invalid region
```

### Problem: S3Bucket doesn't create

**Command:**

```bash
# View bucket events
kubectl describe s3bucket <name>

# View operator logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager | grep -i s3

# Common causes:
# - Bucket name already exists (must be globally unique)
# - AWSProvider is not Ready
# - Insufficient IAM permissions
# - Incorrect region
```

### Problem: Resource doesn't delete

**Command:**

```bash
# View finalizers
kubectl get s3bucket <name> -o yaml | grep finalizers -A 5

# Force remove finalizer
kubectl patch s3bucket <name> \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

# Delete again
kubectl delete s3bucket <name>
```

### Debug with Port Forward

**Command:**

```bash
# Expose metrics endpoint
kubectl port-forward -n infra-operator-system \
  deploy/infra-operator-controller-manager 8080:8080

# View metrics (if configured)
curl http://localhost:8080/metrics
```

---

## Uninstallation

### Remove Created Resources

**Command:**

```bash
# Delete all buckets
kubectl delete s3buckets --all -A

# Delete all providers
kubectl delete awsproviders --all -A
```

### Remove Operator

**Command:**

```bash
# Automated method
make uninstall-complete

# Or manual:
kubectl delete -f config/manager/deployment.yaml
kubectl delete -f config/rbac/
kubectl delete -f config/crd/bases/
kubectl delete namespace infra-operator-system
```

---

## Next Steps

1. **Production**: Review IAM permissions (least privilege principle)
2. **GitOps**: Integrate with ArgoCD or Flux
3. **Monitoring**: Configure Prometheus metrics
4. **Alerting**: Configure alerts for resources in NotReady state
5. **Backup**: Implement backup of important CRs

For more information, see:
- [Introduction](/) - Project overview
- [Quickstart](/quickstart) - Quick start guide
- [AWS Services](/services/networking/vpc) - Services documentation
