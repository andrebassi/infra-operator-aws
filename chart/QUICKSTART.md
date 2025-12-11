# Infra Operator Helm Chart - Quick Start Guide

## TL;DR

```bash
# Development (LocalStack)
helm install infra-operator ./chart \
  --namespace infra-operator-system \
  --create-namespace \
  --values chart/values-development.yaml

# Production (EKS with IRSA)
helm install infra-operator ./chart \
  --namespace infra-operator-system \
  --create-namespace \
  --values chart/values-production.yaml \
  --set aws.irsa.roleARN="arn:aws:iam::ACCOUNT_ID:role/infra-operator"
```

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+
- AWS Account with IAM permissions
- (Optional) cert-manager for webhooks
- (Optional) Prometheus Operator for monitoring

## Installation Scenarios

### 1. Development (Local with LocalStack)

```bash
# Using Makefile
make helm-install-dev

# Or manually
helm install infra-operator ./chart \
  --namespace infra-operator-system \
  --create-namespace \
  --values chart/values-development.yaml
```

**Features:**
- Single replica
- Debug logging
- No webhooks
- LocalStack endpoint configured
- Fast iteration

### 2. Staging (AWS with Static Credentials)

```bash
# Create secret
kubectl create secret generic aws-credentials \
  --namespace infra-operator-system \
  --from-literal=aws_access_key_id=AKIAIOSFODNN7EXAMPLE \
  --from-literal=aws_secret_access_key=wJalrXUtnFEMI/...

# Install
helm install infra-operator ./chart \
  --namespace infra-operator-system \
  --create-namespace \
  --set aws.staticCredentials.enabled=true \
  --set aws.staticCredentials.secretName=aws-credentials \
  --set aws.defaultRegion=us-east-1
```

### 3. Production (EKS with IRSA) - Recommended

```bash
# Using Makefile
make helm-install-prod

# Or manually
helm install infra-operator ./chart \
  --namespace infra-operator-system \
  --create-namespace \
  --values chart/values-production.yaml \
  --set aws.irsa.roleARN="arn:aws:iam::123456789012:role/infra-operator" \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::123456789012:role/infra-operator"
```

**Features:**
- 2 replicas with HA
- Leader election
- Pod anti-affinity
- Prometheus monitoring
- Admission webhooks
- Network policies
- PodDisruptionBudget
- JSON logging

## Quick Verification

```bash
# 1. Check IMPLANTAÇÃO
kubectl get deployment -n infra-operator-system

# 2. Check pods
kubectl get pods -n infra-operator-system

# 3. Check CRDs
kubectl get crds | grep aws-infra-operator.runner.codes
# Should show 19 CRDs

# 4. View logs
kubectl logs -n infra-operator-system -l control-plane=controller-manager -f
```

## First Resource

```bash
# 1. Create AWS Provider
kubectl apply -f - <<EOF
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: default
spec:
  region: us-east-1
EOF

# 2. Wait for ready
kubectl get awsprovider default -w

# 3. Create VPC
kubectl apply -f - <<EOF
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: my-vpc
spec:
  providerRef:
    name: default
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
EOF

# 4. Check status
kubectl get vpc my-vpc
kubectl describe vpc my-vpc
```

## Common Commands

```bash
# Lint chart
helm lint chart/

# Template (dry-run)
helm template infra-operator chart/ --values chart/values.yaml

# Install
helm install infra-operator chart/ -n infra-operator-system --create-namespace

# Upgrade
helm upgrade infra-operator chart/ -n infra-operator-system

# Uninstall
helm uninstall infra-operator -n infra-operator-system

# Get values
helm get values infra-operator -n infra-operator-system

# Get status
helm status infra-operator -n infra-operator-system

# Run tests
helm test infra-operator -n infra-operator-system
```

## Key Configuration Options

```yaml
# IMPLANTAÇÃO
replicaCount: 2
image.tag: "v1.0.0"

# AWS
aws.defaultRegion: "us-east-1"
aws.irsa.enabled: true
aws.irsa.roleARN: "arn:aws:iam::..."

# Features
webhooks.enabled: true
prometheus.serviceMonitor.enabled: true
driftDetection.enabled: true
driftDetection.checkInterval: "10m"

# Logging
logging.level: "info"  # debug, info, warn, error
logging.encoder: "json"  # json, console

# HA
leaderElection.enabled: true
podDisruptionBudget.enabled: true
```

## Troubleshooting

### Operator Not Starting

```bash
# Check logs
kubectl logs -n infra-operator-system -l control-plane=controller-manager

# Check events
kubectl get events -n infra-operator-system --sort-by='.lastTimestamp'

# Describe pod
kubectl describe pod -n infra-operator-system -l control-plane=controller-manager
```

### AWS Authentication Issues

```bash
# For IRSA
kubectl describe sa infra-operator -n infra-operator-system
# Check for: eks.amazonaws.com/role-arn annotation

# For static credentials
kubectl get secret aws-credentials -n infra-operator-system
```

### Webhook Problems

```bash
# Check cert-manager
kubectl get pods -n cert-manager

# Check certificate
kubectl get certificate -n infra-operator-system
kubectl describe certificate infra-operator-webhook-cert -n infra-operator-system

# Disable WEBHOOKS temporarily
helm upgrade infra-operator chart/ --set webhooks.enabled=false
```

## Getting Help

```bash
# View NOTES (post-install instructions)
helm get notes infra-operator -n infra-operator-system

# View all RECURSOS
helm get manifest infra-operator -n infra-operator-system

# View complete README
cat chart/README.md
```

## Next Steps

1. **Configure AWS Provider** - Set up credentials
2. **Create Infrastructure** - Deploy VPCs, S3, RDS, etc.
3. **Enable Monitoring** - Configure Prometheus
4. **Set Up Webhooks** - Install cert-manager
5. **Configure GitOps** - Integrate with ArgoCD/Flux

For complete documentation, see: `chart/README.md`
