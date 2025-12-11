---
title: 'AWSProvider'
description: 'Configure AWS credentials and settings for Infra Operator'
sidebar_position: 2
---

# AWSProvider

The `AWSProvider` resource configures AWS credentials and settings that other resources use to interact with AWS.

## Overview

Every AWS resource managed by Infra Operator must reference an AWSProvider. The provider handles:

- AWS credentials (static or IRSA)
- Region configuration
- Custom endpoint (for LocalStack testing)
- Default tags applied to all resources

## Quick Start

### Using IRSA (Recommended for Production)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-production
  namespace: infra-operator
spec:
  region: us-east-1
  # IRSA: No credentials needed, uses ServiceAccount annotations
  defaultTags:
    ManagedBy: infra-operator
    Environment: production
```

### Using Static Credentials

**Example:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: infra-operator
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: "AKIAIOSFODNN7EXAMPLE"
  AWS_SECRET_ACCESS_KEY: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-production
  namespace: infra-operator
spec:
  region: us-east-1
  credentialsSecret:
    name: aws-credentials
    namespace: infra-operator
  defaultTags:
    ManagedBy: infra-operator
```

### Using LocalStack (Development/Testing)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
  namespace: infra-operator
spec:
  region: us-east-1
  endpoint: http://localstack.default.svc.cluster.local:4566
  credentialsSecret:
    name: aws-credentials
    namespace: infra-operator
  defaultTags:
    ManagedBy: infra-operator
    Environment: localstack
```

## Specification

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `spec.region` | string | AWS region (e.g., `us-east-1`, `eu-west-1`) |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `spec.credentialsSecret` | object | - | Reference to Secret with AWS credentials |
| `spec.endpoint` | string | - | Custom AWS endpoint (for LocalStack) |
| `spec.roleARN` | string | - | IAM Role ARN for cross-account access |
| `spec.defaultTags` | object | - | Tags applied to all resources |

### CredentialsSecret

**Example:**

```yaml
spec:
  credentialsSecret:
    name: aws-credentials      # Secret name
    namespace: infra-operator  # Secret namespace
```

The Secret must contain:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

Optionally:
- `AWS_SESSION_TOKEN` (for temporary credentials)

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `status.ready` | boolean | Provider is configured and ready |
| `status.accountID` | string | AWS Account ID |
| `status.region` | string | Configured AWS region |
| `status.lastSyncTime` | string | Last successful AWS API call |

## IRSA Configuration

For production EKS deployments, use IRSA (IAM Roles for Service Accounts):

### 1. Create IAM Role

**Command:**

```bash
# Get OIDC Provider
export CLUSTER_NAME=my-cluster
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

# Create trust policy
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator:infra-operator",
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
  --assume-role-policy-document file://trust-policy.json
```

### 2. Attach Policies

**Command:**

```bash
# Attach managed policies or create custom ones
aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonVPCFullAccess

aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

# Add more policies as needed for your resources
```

### 3. Annotate ServiceAccount

**Command:**

```bash
kubectl annotate serviceaccount infra-operator \
  -n infra-operator \
  eks.amazonaws.com/role-arn=arn:aws:iam::${AWS_ACCOUNT_ID}:role/infra-operator-role
```

## Multi-Account Setup

For managing resources across multiple AWS accounts:

**Example:**

```yaml
# Account A (Primary)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: account-a
spec:
  region: us-east-1
  credentialsSecret:
    name: account-a-credentials
---
# Account B (Secondary)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: account-b
spec:
  region: us-east-1
  roleARN: arn:aws:iam::222222222222:role/infra-operator-cross-account
  credentialsSecret:
    name: account-a-credentials  # Use primary credentials to assume role
```

## Best Practices

:::note Best Practices

- **Use IRSA in production** — Never use static credentials in production, leverage IAM Roles for Service Accounts
- **Apply least-privilege IAM policies** — Only grant permissions that the operator actually needs
- **Rotate credentials regularly** — If using static credentials, implement regular rotation
- **Use separate providers per environment** — Keep dev/staging/prod isolated with different providers
- **One provider per AWS account/region** — Use descriptive names like aws-prod-us-east-1, aws-dev-eu-west-1
- **Apply consistent default tags** — Set ManagedBy, Environment tags at provider level for all resources

:::

## Troubleshooting

### Provider Not Ready

**Command:**

```bash
# Check provider status
kubectl describe awsprovider aws-production

# Check operator logs
kubectl logs -n infra-operator deploy/infra-operator --tail=100

# Verify credentials Secret exists
kubectl get secret aws-credentials -n infra-operator
```

### Invalid Credentials

**Command:**

```bash
# Test credentials manually
export AWS_ACCESS_KEY_ID=$(kubectl get secret aws-credentials -n infra-operator -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
export AWS_SECRET_ACCESS_KEY=$(kubectl get secret aws-credentials -n infra-operator -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)

aws sts get-caller-identity
```

### IRSA Not Working

**Command:**

```bash
# Check ServiceAccount annotations
kubectl get sa infra-operator -n infra-operator -o yaml

# Verify OIDC provider is configured
aws eks describe-cluster --name $CLUSTER_NAME --query "cluster.identity.oidc"

# Check IAM role trust policy
aws iam get-role --role-name infra-operator-role --query "Role.AssumeRolePolicyDocument"
```
