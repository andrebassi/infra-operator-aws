# Infra Operator - Complete Installation Guide

This guide covers all installation scenarios for the Infra Operator Helm chart.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Production Installation](#production-installation)
4. [Development Installation](#development-installation)
5. [LocalStack Installation](#localstack-installation)
6. [AWS Configuration](#aws-configuration)
7. [Verification](#verification)
8. [Post-Installation](#post-installation)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Obrigatório

- **Kubernetes**: 1.20 or higher
- **Helm**: 3.8 or higher
- **kubectl**: Configured to access your cluster
- **AWS Account**: With appropriate IAM permissions

### Recommended

- **cert-manager**: For automatic webhook certificate management
- **Prometheus Operator**: For metrics collection
- **ArgoCD**: For GitOps workflows

### Verify Prerequisites

```bash
# Check Kubernetes version
kubectl version --short

# Check Helm version
helm version --short

# Check cluster connectivity
kubectl cluster-info

# Check if cert-manager is installed (opcional)
kubectl get pods -n cert-manager
```

---

## Quick Start

### 1. Add Helm Repository

```bash
helm repo add infra-operator https://your-org.github.io/infra-operator
helm repo update
```

### 2. Install with Default Settings

```bash
# Create namespace
kubectl create namespace infra-operator

# Install chart
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --create-namespace \
  --wait
```

### 3. Verify Installation

```bash
# Check pods
kubectl get pods -n infra-operator

# Check CRDs
kubectl get crds | grep aws-infra-operator.runner.codes

# View logs
kubectl logs -n infra-operator -l app.kubernetes.io/name=infra-operator --tail=50
```

---

## Production Installation

### Recommended Configuration

Create `values-production.yaml`:

```yaml
# High Availability
replicaCount: 2

# Leader Election (required for HA)
operator:
  leaderElection:
    enabled: true

# Production RECURSOS
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

# Pod Anti-AFINIDADE
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app.kubernetes.io/name: infra-operator
        topologyKey: kubernetes.io/hostname
    - weight: 50
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app.kubernetes.io/name: infra-operator
        topologyKey: topology.kubernetes.io/zone

# ORÇAMENTO DE INTERRUPÇÃO DE POD
podDisruptionBudget:
  enabled: true
  minAvailable: 1

# CLASSE DE PRIORIDADE
priorityClassName: system-cluster-critical

# POLÍTICA DE REDE
networkPolicy:
  enabled: true
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 443

# MONITORAMENTO
prometheus:
  serviceMonitor:
    enabled: true
    namespace: monitoring
    labels:
      prometheus: kube-prometheus
    interval: 30s

# WEBHOOKS
webhooks:
  enabled: true
  certManager:
    enabled: true
    issuerRef:
      kind: ClusterIssuer
      name: letsencrypt-prod

# AWS IRSA (EKS)
aws:
  defaultRegion: us-east-1
  irsa:
    enabled: true
    roleARN: "arn:aws:iam::123456789012:role/infra-operator-prod-role"

serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/infra-operator-prod-role"

# Drift Detection
driftDetection:
  enabled: true
  defaultCheckInterval: "10m"
  defaultAutoHeal: true

# Global AWS Tags
globalTags:
  managed-by: infra-operator
  environment: production
  cost-center: platform
  terraform: "false"
```

### Install

```bash
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --create-namespace \
  --values values-production.yaml \
  --wait \
  --timeout 10m
```

---

## Development Installation

### Using Provided Values File

```bash
# Install with development configuration
helm install infra-operator ./chart \
  --namespace infra-operator \
  --create-namespace \
  --values ./chart/values-dev.yaml \
  --wait

# Or use the helper script
./scripts/helm/install-dev.sh
```

### Development Features

- Single replica (no HA overhead)
- Debug logging enabled
- Human-readable console logs
- Faster drift detection (2m interval)
- Lower resource limits
- Webhooks set to Ignore failure policy

---

## LocalStack Installation

### 1. Start LocalStack

```bash
# Using docker-compose
docker-compose up -d localstack

# Or standalone
docker run -d --name localstack \
  -p 4566:4566 \
  localstack/localstack
```

### 2. Install Operator

```bash
# Install with LocalStack configuration
helm install infra-operator ./chart \
  --namespace infra-operator \
  --create-namespace \
  --values ./chart/values-localstack.yaml \
  --wait

# Or use the helper script
./scripts/helm/install-localstack.sh
```

### 3. Test LocalStack Connection

```bash
# Apply sample AWSProvider
kubectl apply -f - <<EOF
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
spec:
  region: us-east-1
  endpoint: http://localstack:4566
EOF

# Check provider status
kubectl get awsprovider localstack -w
```

---

## CONFIGURAÇÃO AWS

### Option 1: IRSA (Recommended for EKS)

#### Create IAM Role

```bash
# Create trust policy
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:sub": "system:serviceaccount:infra-operator:infra-operator"
        }
      }
    }
  ]
}
EOF

# Create IAM role
aws iam create-role \
  --role-name infra-operator-role \
  --assume-role-policy-document file://trust-policy.json

# Attach permissions policy
aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess

# Or create custom policy with least privilege
aws iam create-policy \
  --policy-name InfraOperatorPolicy \
  --policy-document file://iam-policy.json

aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::123456789012:policy/InfraOperatorPolicy
```

#### Install with IRSA

```bash
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --set aws.irsa.enabled=true \
  --set aws.irsa.roleARN="arn:aws:iam::123456789012:role/infra-operator-role" \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::123456789012:role/infra-operator-role"
```

### Option 2: Static Credentials (Not Recommended)

```bash
# Create Kubernetes secret
kubectl create secret generic aws-credentials \
  --namespace infra-operator \
  --from-literal=aws_access_key_id=AKIAIOSFODNN7EXAMPLE \
  --from-literal=aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Install with static credentials
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --set aws.staticCredentials.enabled=true \
  --set aws.staticCredentials.secretName=aws-credentials
```

### Option 3: AssumeRole (Cross-Account)

```bash
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --set aws.assumeRole.enabled=true \
  --set aws.assumeRole.roleARN="arn:aws:iam::987654321098:role/CrossAccountRole" \
  --set aws.assumeRole.externalID="unique-external-id"
```

---

## Verification

### Check Operator Status

```bash
# Get IMPLANTAÇÃO status
kubectl get deployment -n infra-operator

# Get pod status
kubectl get pods -n infra-operator

# Check logs
kubectl logs -n infra-operator -l app.kubernetes.io/name=infra-operator --tail=100 -f
```

### Verify CRDs

```bash
# List all CRDs
kubectl get crds | grep aws-infra-operator.runner.codes

# Expected output: 19-25 CRDs
kubectl get crds | grep aws-infra-operator.runner.codes | wc -l
```

### Run Helm Tests

```bash
# Run built-in tests
helm test infra-operator -n infra-operator

# Check test results
kubectl get pods -n infra-operator -l helm.sh/test=true
kubectl logs -n infra-operator -l helm.sh/test=true
```

### Verify WEBHOOKS (if enabled)

```bash
# Check webhook configuration
kubectl get validatingwebhookconfiguration | grep infra-operator

# Check webhook certificates
kubectl get certificate -n infra-operator
kubectl describe certificate webhook-server-cert -n infra-operator
```

### Check Metrics

```bash
# Port-forward metrics SERVIÇO
kubectl port-forward -n infra-operator svc/infra-operator 8080:8080

# Access metrics
curl http://localhost:8080/metrics

# Check health
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

---

## Post-Installation

### Create AWSProvider

```bash
kubectl apply -f - <<EOF
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: default
spec:
  region: us-east-1
  defaultTags:
    managed-by: infra-operator
    environment: production
EOF
```

### Wait for Provider Ready

```bash
kubectl get awsprovider default -w
```

### Create Your First Resource

```bash
# Create S3 bucket
kubectl apply -f - <<EOF
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: my-app-bucket
spec:
  providerRef:
    name: default
  bucketName: my-company-app-data-prod
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
  deletionPolicy: Retain
  tags:
    Application: my-app
EOF

# Watch status
kubectl get s3bucket my-app-bucket -w
kubectl describe s3bucket my-app-bucket
```

---

## Troubleshooting

### Operator Not Starting

```bash
# Check IMPLANTAÇÃO events
kubectl describe deployment -n infra-operator infra-operator

# Check pod events
kubectl describe pod -n infra-operator -l app.kubernetes.io/name=infra-operator

# Check logs
kubectl logs -n infra-operator -l app.kubernetes.io/name=infra-operator --tail=200
```

### AWS Permission Issues

```bash
# Check IRSA configuration
kubectl get sa infra-operator -n infra-operator -o yaml | grep eks.amazonaws.com

# Check operator logs for AWS errors
kubectl logs -n infra-operator -l app.kubernetes.io/name=infra-operator | grep -i "access denied\|unauthorized\|credential"

# Verify IAM role trust policy
aws iam get-role --role-name infra-operator-role

# Test AWS API access from pod
kubectl exec -n infra-operator deployment/infra-operator -- aws sts get-caller-identity
```

### Webhook Certificate Issues

```bash
# Check cert-manager is running
kubectl get pods -n cert-manager

# Check certificate status
kubectl get certificate -n infra-operator
kubectl describe certificate webhook-server-cert -n infra-operator

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager --tail=100

# Manually approve certificate if needed
kubectl certificate approve webhook-server-cert -n infra-operator
```

### RECURSOS Not Reconciling

```bash
# Check operator logs
kubectl logs -n infra-operator -l app.kubernetes.io/name=infra-operator --tail=100 -f

# Check resource status
kubectl describe <resource-type> <resource-name>

# Check AWSProvider status
kubectl get awsprovider
kubectl describe awsprovider default

# Force reconciliation (delete and recreate)
kubectl delete <resource-type> <resource-name>
kubectl apply -f <resource-manifest>
```

### Cleanup Failed Installation

```bash
# Uninstall chart
helm uninstall infra-operator -n infra-operator

# Delete namespace
kubectl delete namespace infra-operator

# Delete CRDs (⚠️ this deletes all custom RECURSOS)
kubectl delete crds $(kubectl get crds | grep aws-infra-operator.runner.codes | awk '{print $1}')

# Delete webhook configuration
kubectl delete validatingwebhookconfiguration infra-operator-webhook
```

---

## Advanced Configuration

### Custom Values Example

```yaml
# values-custom.yaml

# Image
image:
  registry: my-registry.io
  repository: my-org/infra-operator
  tag: "v1.2.3"

# Variáveis de ambiente extras
env:
- name: HTTP_PROXY
  value: "http://proxy.example.com:8080"
- name: HTTPS_PROXY
  value: "http://proxy.example.com:8080"
- name: NO_PROXY
  value: ".cluster.local,.svc"

# Volumes extras
extraVolumes:
- name: ca-certs
  configMap:
    name: corporate-ca-certs

extraVolumeMounts:
- name: ca-certs
  mountPath: /etc/ssl/certs/corporate
  readOnly: true

# Node AFINIDADE
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: node-role.kubernetes.io/infra
          operator: Exists

# TOLERÂNCIAS
tolerations:
- key: "infra"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"
```

Install with custom values:

```bash
helm install infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --values values-custom.yaml
```

---

## Upgrade Guide

### Upgrade to New Version

```bash
# Update repository
helm repo update

# Check available versions
helm search repo infra-operator

# Upgrade
helm upgrade infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --reuse-values \
  --wait
```

### Update Configuration

```bash
# Change specific value
helm upgrade infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --reuse-values \
  --set aws.defaultRegion=eu-west-1

# Apply new values file
helm upgrade infra-operator infra-operator/infra-operator \
  --namespace infra-operator \
  --values values-updated.yaml
```

---

## Support

- **Documentation**: https://github.com/your-org/infra-operator
- **Issues**: https://github.com/your-org/infra-operator/issues
- **Discussions**: https://github.com/your-org/infra-operator/discussions

---

**Last Updated**: 2025-11-23
