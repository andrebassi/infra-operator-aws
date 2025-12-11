---
title: 'ECR Repository - Container Registry'
description: 'Private Docker image registry managed by AWS'
sidebar_position: 1
---

Create and manage private Docker (container) image registries on AWS with security, vulnerability scanning, and automatic lifecycle policies.

## Prerequisite: AWSProvider Configuration

Before creating any AWS resource, you need to configure an **AWSProvider** that manages credentials and authentication with AWS.

**IRSA:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: production-aws
  namespace: default
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role
  defaultTags:
    managed-by: infra-operator
    environment: production
```

**Static Credentials:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  access-key-id: test
  secret-access-key: test
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
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
    environment: test
```

**Check Status:**

```bash
kubectl get awsprovider
kubectl describe awsprovider production-aws
```
:::warning

For production, always use **IRSA** (IAM Roles for Service Accounts) instead of static credentials.

:::

### Create IAM Role for IRSA

To use IRSA in production, you need to create an IAM Role with the required permissions:

**Trust Policy (trust-policy.json):**

```json
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
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:aud": "sts.amazonaws.com"
        }
      }
}
  ]
}
```

**IAM Policy - ECR (ecr-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ecr:CreateRepository",
        "ecr:DeleteRepository",
        "ecr:DescribeRepositories",
        "ecr:PutImageScanningConfiguration",
        "ecr:PutImageTagMutability",
        "ecr:PutLifecyclePolicy",
        "ecr:GetLifecyclePolicy",
        "ecr:TagResource",
        "ecr:UntagResource",
        "ecr:ListTagsForResource"
      ],
      "Resource": "*"
}
  ]
}
```

**Create Role with AWS CLI:**

```bash
# 1. Get OIDC Provider from EKS cluster
export CLUSTER_NAME=my-cluster
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

# 2. Update trust-policy.json with correct values
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
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
}
  ]
}
EOF

# 3. Create IAM Role
aws iam create-role \
  --role-name infra-operator-ecr-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator ECR management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-ecr-role \
  --policy-name ECRManagement \
  --policy-document file://ecr-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-ecr-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-ecr-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon ECR (Elastic Container Registry) is a fully managed private Docker container registry service that makes it easy to store, manage, and deploy Docker/container images. It integrates seamlessly with ECS, EKS, Lambda, and CI/CD pipelines.

**Features:**
- **Secure Private Registry**: Store private Docker images on AWS
- **No Infrastructure Management**: Fully managed by AWS
- **High Availability**: Automatically replicated across multiple AZs
- **IAM Integration**: Granular access control with IAM policies
- **Image Scanning**: Automatically detect vulnerabilities in images
- **Encryption at Rest**: AES-256 encryption or custom KMS keys
- **Image Tag Mutability**: Prevent tag overwrites (recommended for production)
- **Lifecycle Policies**: Automatically delete old images
- **Cross-Account Access**: Share images with other AWS accounts
- **Repository Policies**: Granular access control per repository
- **Audit Integration**: Audit all operations
- **Replication Rules**: Replicate images across regions
- **Cost Effective**: Pay only for storage used
- **Docker Push/Pull Native**: Standard docker commands work natively

**Status**: ⚠️ Requires LocalStack Pro or Real AWS

## Quick Start

**Basic ECR Repository:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: e2e-app-images
  namespace: default
spec:
  providerRef:
    name: localstack
  repositoryName: e2e-app-images
  imageTagMutability: MUTABLE
  scanOnPush: true
  encryptionConfiguration:
    encryptionType: AES256
  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing
  deletionPolicy: Delete
```

**ECR Repository with Lifecycle Policy:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: e2e-production-images
  namespace: default
spec:
  providerRef:
    name: localstack
  repositoryName: e2e-production-images
  imageTagMutability: IMMUTABLE
  scanOnPush: true
  encryptionConfiguration:
    encryptionType: AES256
  lifecyclePolicy:
    policyText: |
      {
        "rules": [
          {
            "rulePriority": 1,
            "description": "Keep last 10 images",
            "selection": {
              "tagStatus": "any",
              "countType": "imageCountMoreThan",
              "countNumber": 10
            },
            "action": {
              "type": "expire"
            }
          }
        ]
      }
  tags:
    environment: production
    managed-by: infra-operator
    purpose: e2e-testing
  deletionPolicy: Delete
```

**Complete ECR Repository:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: app-backend
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Repository name (optional prefix)
  repositoryName: app/backend

  # Tag mutability (IMMUTABLE recommended for production)
  imageTagMutability: IMMUTABLE

  # Vulnerability scanning
  scanOnPush: true

  # KMS encryption (optional)
  encryptionConfiguration:
    encryptionType: KMS
    kmsKey: alias/ecr-encryption

  # Lifecycle policy
  lifecyclePolicy:
    policyText: |
      {
        "rules": [{
          "rulePriority": 1,
          "description": "Keep last 10 images",
          "selection": {
            "tagStatus": "any",
            "countType": "imageCountMoreThan",
            "countNumber": 10
          },
          "action": {
            "type": "expire"
          }
        }]
      }

  # Tags for organization
  tags:
    Environment: production
    Application: backend
    ManagedBy: infra-operator

  # Keep repository if CR is deleted
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f ecr-repository.yaml
```

**Check Status:**

```bash
kubectl get ecrrepositories
kubectl describe ecrrepository e2e-app-images
kubectl get ecrrepository e2e-app-images -o yaml
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource for authentication

  AWSProvider resource name

ECR repository name. Can include prefix with `/` (e.g., `app/backend`)

  **Rules:**
  - 2 to 256 characters
  - Lowercase letters, numbers, hyphens, underscores, slashes (`/`)
  - Must start with letter or number
  - No spaces

  **Example:**

  ```yaml
  repositoryName: myapp/backend
  # or without prefix
  repositoryName: my-backend-service
  ```

### Optional Fields - Scanning

Enable automatic vulnerability scanning when image is pushed

  **Example:**

  ```yaml
  scanOnPush: true
  ```

  **Options:**
  - `true`: Automatic scan (recommended for production)
  - `false`: Manual scan only (default)

  **Details:**
  - Uses AWS vulnerability database
  - Detects CVEs (Common Vulnerabilities and Exposures)
  - Results available in DescribeImages
  - No additional cost for scanning

### Optional Fields - Tag Mutability

Allow image tag overwrites

  **Options:**
  - `MUTABLE`: Tags can be overwritten (default, less secure)
  - `IMMUTABLE`: Tags cannot be overwritten (recommended for production)

  **Example:**

  ```yaml
  imageTagMutability: IMMUTABLE
  ```

  **Recommendation:** Use IMMUTABLE in production to ensure versioned tags are not accidentally overwritten

### Optional Fields - Encryption

Encryption configuration for stored images

  **Example:**

  ```yaml
  encryptionConfiguration:
    encryptionType: KMS
    kmsKey: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012
  ```

  **encryptionType Options:**
  - `AES256`: AWS-managed encryption (default, no additional cost)
  - `KMS`: AWS KMS encryption (requires CMK, additional cost)

  **If using KMS:**
  - `kmsKey`: ARN or alias of KMS key
  - Example: `arn:aws:kms:us-east-1:123456789012:key/12345678...`
  - Or alias: `alias/ecr-encryption`
  - KMS key MUST exist and have ECR permissions

### Optional Fields - Lifecycle Policy

Policy to automatically manage images (delete old ones)

  **Example:**

  ```yaml
  lifecyclePolicy:
    policyText: |
      {
        "rules": [
          {
            "rulePriority": 1,
            "description": "Keep last 10 images",
            "selection": {
              "tagStatus": "any",
              "countType": "imageCountMoreThan",
              "countNumber": 10
            },
            "action": {
              "type": "expire"
            }
          },
          {
            "rulePriority": 2,
            "description": "Expire untagged images after 30 days",
            "selection": {
              "tagStatus": "untagged",
              "countType": "sinceImagePushed",
              "countUnit": "days",
              "countNumber": 30
            },
            "action": {
              "type": "expire"
            }
          }
        ]
      }
  ```

  JSON document of lifecycle policy

**Structure:**
  - `rules[]`: Array of lifecycle rules
  - `rulePriority`: Execution order (lower first)
  - `description`: Rule description
  - `selection`: Criteria for which images the rule applies to
  - `action`: What to do (only `expire` supported)

  **Selection - tagStatus:**
  - `tagged`: Applies only to tagged images
  - `untagged`: Applies only to untagged images
  - `any`: Applies to all images

  **Selection - countType:**
  - `imageCountMoreThan`: If there are more than N images
  - `sinceImagePushed`: If pushed more than N days/months/years ago

  **Selection - countUnit:** (for sinceImagePushed)
  - `days`
  - `months`
  - `years`

### Optional Fields - Policies and Control

JSON policy for repository access control (similar to bucket policies)

  **Example:**

  ```yaml
  repositoryPolicyText: |
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::123456789012:role/ECS-Task-Execution-Role"
          },
          "Action": [
            "ecr:GetDownloadUrlForLayer",
            "ecr:BatchGetImage"
          ]
        }
      ]
    }
  ```

  **Common ECR Actions:**
  - `ecr:GetDownloadUrlForLayer`: Download image layers
  - `ecr:BatchGetImage`: Download images (pull)
  - `ecr:PutImage`: Push images (push)
  - `ecr:InitiateLayerUpload`: Initiate layer upload
  - `ecr:UploadLayerPart`: Partial layer upload
  - `ecr:CompleteLayerUpload`: Complete layer upload

  **Usage:** Cross-account access, IP restrictions, etc.

### Optional Fields - Tags and Deletion

Key-value pairs for organization and billing

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: backend
    Team: platform
    CostCenter: engineering
    ManagedBy: infra-operator
  ```

What happens to the repository when the CR is deleted

  **Options:**
  - `Delete`: Repository is deleted from AWS (⚠️ WARNING: images will be lost)
  - `Retain`: Repository remains in AWS but unmanaged
  - `Orphan`: Remove management only

  **Example:**

  ```yaml
  deletionPolicy: Retain
  ```

  **Recommendation:** Use `Retain` in production to avoid accidental loss of images

## Status Fields

After the ECR Repository is created, the following status fields are populated:

`true` when the repository is created and ready for use

Full ARN of the ECR repository

  ```
  arn:aws:ecr:us-east-1:123456789012:repository/app/backend
  ```

Repository URI for `docker push/pull`

  ```
  123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend
  ```

AWS registry ID where the repository exists (usually the account ID)

Number of images stored in the repository

Repository creation date/time (ISO 8601 format)

Timestamp of last synchronization with AWS (ISO 8601 format)

Additional status message (errors, warnings, etc.)

## Examples

### Basic ECR Repository

Simple repository to get started:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: my-app
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: my-company/my-app

  # Basic: without scanning, mutable tags
  imageTagMutability: MUTABLE
  scanOnPush: false

  encryptionConfiguration:
    encryptionType: AES256

  tags:
    Environment: development
    Application: my-app

  deletionPolicy: Delete
```

### ECR Repository with Automatic Scan

Repository that automatically detects vulnerabilities:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: secure-backend
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: app/backend-service

  # Scan each push
  scanOnPush: true

  # Tags cannot be overwritten
  imageTagMutability: IMMUTABLE

  # Default encryption
  encryptionConfiguration:
    encryptionType: AES256

  tags:
    Environment: production
    Application: backend
    SecurityScanned: "true"

  deletionPolicy: Retain
```

### ECR Repository with Lifecycle Policy

Repository with automatic cleanup of old images:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: continuous-build
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: ci-cd/app-builder

  scanOnPush: true
  imageTagMutability: IMMUTABLE

  encryptionConfiguration:
    encryptionType: AES256

  # Automatic cleanup
  lifecyclePolicy:
    policyText: |
      {
        "rules": [
          {
            "rulePriority": 1,
            "description": "Keep last 30 tagged images",
            "selection": {
              "tagStatus": "tagged",
              "countType": "imageCountMoreThan",
              "countNumber": 30
            },
            "action": {
              "type": "expire"
            }
          },
          {
            "rulePriority": 2,
            "description": "Delete untagged images after 7 days",
            "selection": {
              "tagStatus": "untagged",
              "countType": "sinceImagePushed",
              "countUnit": "days",
              "countNumber": 7
            },
            "action": {
              "type": "expire"
            }
          }
        ]
      }

  tags:
    Environment: development
    Type: build-cache

  deletionPolicy: Delete
```

### ECR Repository with KMS Encryption

Repository with custom encryption via KMS:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: encrypted-production
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: app/production-images

  scanOnPush: true
  imageTagMutability: IMMUTABLE

  # Custom KMS encryption
  encryptionConfiguration:
encryptionType: KMS
kmsKey: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012

  lifecyclePolicy:
policyText: |
      {
        "rules": [{
          "rulePriority": 1,
          "description": "Keep last 20 images",
          "selection": {
            "tagStatus": "any",
            "countType": "imageCountMoreThan",
            "countNumber": 20
          },
          "action": {"type": "expire"}
        }]
      }

  tags:
    Environment: production
    Compliance: required
    DataClassification: confidential

  deletionPolicy: Retain
```

### ECR Repository with Cross-Account Access

Repository shared with another AWS account:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: shared-images
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: shared/base-images

  scanOnPush: true
  imageTagMutability: IMMUTABLE

  encryptionConfiguration:
encryptionType: AES256

  # Policy to allow cross-account access
  repositoryPolicyText: |
{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Sid": "AllowPullFromOtherAccount",
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::999888777666:role/ECS-Task-Execution-Role"
          },
          "Action": [
            "ecr:GetDownloadUrlForLayer",
            "ecr:BatchGetImage",
            "ecr:DescribeImages"
          ]
        },
        {
          "Sid": "AllowPushFromCI",
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::123456789012:role/GitLab-Runner"
          },
          "Action": [
            "ecr:PutImage",
            "ecr:InitiateLayerUpload",
            "ecr:UploadLayerPart",
            "ecr:CompleteLayerUpload"
          ]
        }
      ]
}

  lifecyclePolicy:
policyText: |
      {
        "rules": [{
          "rulePriority": 1,
          "description": "Keep last 50 images",
          "selection": {
            "tagStatus": "any",
            "countType": "imageCountMoreThan",
            "countNumber": 50
          },
          "action": {"type": "expire"}
        }]
      }

  tags:
    Environment: shared
    Type: base-images

  deletionPolicy: Retain
```

## Verification

### Check Status via kubectl

**Command:**

```bash
# List all repositories
kubectl get ecrrepositories

# Get detailed information
kubectl get ecrrepository secure-backend -o yaml

# Follow creation in real-time
kubectl get ecrrepository secure-backend -w

# View events and status
kubectl describe ecrrepository secure-backend
```

### Verify on AWS

**AWS CLI:**

```bash
# List repositories
aws ecr describe-repositories

# Get specific details
aws ecr describe-repositories \
      --repository-names app/backend-service

# View images in repository
aws ecr describe-images \
      --repository-name app/backend-service

# View image details
aws ecr describe-images \
      --repository-name app/backend-service \
      --image-ids imageTag=latest

# View scanning findings
aws ecr describe-image-scan-findings \
      --repository-name app/backend-service \
      --image-id imageTag=latest

# View lifecycle policy
aws ecr get-lifecycle-policy \
      --repository-name app/backend-service

# View repository policy
aws ecr get-repository-policy \
      --repository-name app/backend-service

# Get authentication token for docker
aws ecr get-authorization-token

# List all images with tags
aws ecr list-images \
      --repository-name app/backend-service
```
  
**Docker CLI:**

```bash
# Login to ECR
aws ecr get-authorization-token --output text --query 'authorizationData[].authorizationToken' | base64 -d | cut -d: -f2 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Or using helper script (easier)
aws ecr get-authorization-token --output text --query 'authorizationData[].authorizationToken' | base64 -d | docker login --username AWS --password-stdin https://123456789012.dkr.ecr.us-east-1.amazonaws.com

# Local image tagging
docker tag my-app:v1.0.0 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend:v1.0.0

# Push to ECR
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend:v1.0.0

# Pull from ECR
docker pull 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend:v1.0.0

# List local images
docker images | grep ecr
```
  
**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws ecr describe-repositories

aws ecr list-images \
      --repository-name app/backend-service

# Docker login for LocalStack
aws ecr get-authorization-token | jq -r '.authorizationData[0].authorizationToken' | base64 -d | cut -d: -f2 | docker login --username AWS --password-stdin localhost:4566
```
  
### Expected Output

**Example:**

```yaml
status:
  repositoryArn: arn:aws:ecr:us-east-1:123456789012:repository/app/backend-service
  repositoryUri: 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend-service
  registryId: "123456789012"
  creationDate: "2025-11-22T20:15:30Z"
  imageTagMutability: IMMUTABLE
  encryptionType: AES256
  imageScanningConfiguration:
scanOnPush: true
  ready: true
  lastSyncTime: "2025-11-22T20:15:45Z"
```

## Troubleshooting

### Docker push denied: requested access to the resource is denied

**Symptoms:** Error pushing Docker image

**Common causes:**
1. Not authenticated with ECR (missing docker login)
2. Expired AWS credentials
3. Insufficient IAM permissions
4. Repository does not exist

**Solutions:**
```bash
# Check if repository exists
aws ecr describe-repositories --repository-names app/backend

# Re-authenticate
aws ecr get-authorization-token --output text --query 'authorizationData[].authorizationToken' | base64 -d | cut -d: -f2 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Check AWS credentials
aws sts get-caller-identity

# Verify IAM policy has ecr:PutImage
aws iam get-user-policy --user-name <username> --policy-name <policy>

# Tag the image correctly
docker tag myapp:latest 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend:latest

# Try push again
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/app/backend:latest

# If still failing, force logout and login
docker logout 123456789012.dkr.ecr.us-east-1.amazonaws.com
aws ecr get-authorization-token --output text --query 'authorizationData[].authorizationToken' | base64 -d | cut -d: -f2 | docker login --username AWS --password-stdin https://123456789012.dkr.ecr.us-east-1.amazonaws.com
```
  
### Image scan finds critical vulnerabilities (CVE)

**Symptoms:** Scan findings show HIGH or CRITICAL vulnerabilities

**Cause:** Image dependencies have known CVEs

**Solutions:**
```bash
# View vulnerability details
aws ecr describe-image-scan-findings \
      --repository-name app/backend \
      --image-id imageTag=v1.0.0 \
      --output table

# Remediation options:
# 1. Update dependencies in Dockerfile
# 2. Use latest base image
# 3. Remove unnecessary dependencies

# Improved Dockerfile example
# FROM node:20-alpine (use latest alpine)
# RUN npm ci --only=production (don't install dev deps)
# RUN npm audit fix (fix vulnerabilities)

# Rebuild image
docker build -t app/backend:v1.0.1 .

# Re-scan
aws ecr describe-image-scan-findings \
      --repository-name app/backend \
      --image-id imageTag=v1.0.1

# Ignore vulnerabilities if necessary (with documentation)
# But always prefer to fix!
```
  
### Lifecycle policy deleting wrong images

**Symptoms:** Important images are deleted by lifecycle policy

**Causes:**
1. Wrong priority rule
2. Selection too broad (tagStatus: any)
3. Count number too low

**Solutions:**
```bash
# View current policy
aws ecr get-lifecycle-policy \
      --repository-name app/backend

# Test policy before applying (conceptual dry-run)
# Review which images would be deleted

# Example: more conservative policy
lifecyclePolicy:
      policyText: |
        {
          "rules": [
            {
              "rulePriority": 1,
              "description": "Keep all tagged",
              "selection": {
                "tagStatus": "tagged",
                "countType": "imageCountMoreThan",
                "countNumber": 100
              },
              "action": {"type": "expire"}
            },
            {
              "rulePriority": 2,
              "description": "Delete old untagged images",
              "selection": {
                "tagStatus": "untagged",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 30
              },
              "action": {"type": "expire"}
            }
          ]
        }

# Update policy
kubectl patch ecrrepository app-backend \
      --type merge \
      -p '{"spec":{"lifecyclePolicy":"..."}}'

# Check images before and after
aws ecr list-images --repository-name app/backend
```
  
### Cross-account access denied

**Symptoms:** Error trying to pull image from another AWS account

**Cause:** Repository policy does not allow access from other account

**Solutions:**
```bash
# Get ARN of role from other account
# Example: arn:aws:iam::999888777666:role/ECS-Task-Role

# Update repository policy
kubectl patch ecrrepository shared-images \
      --type merge \
      -p '{
        "spec": {
          "repositoryPolicyText": "{
            \"Version\": \"2012-10-17\",
            \"Statement\": [{
              \"Effect\": \"Allow\",
              \"Principal\": {
                \"AWS\": \"arn:aws:iam::999888777666:role/ECS-Task-Role\"
              },
              \"Action\": [
                \"ecr:GetDownloadUrlForLayer\",
                \"ecr:BatchGetImage\"
              ]
            }]
          }"
        }
      }'

# Check if policy was applied
aws ecr get-repository-policy \
      --repository-name shared/images

# In the other account, test pull
# Ensure the role has ecr permission in the other account too
aws ecr get-authorization-token --endpoint-url https://ecr.us-east-1.amazonaws.com

# If using assume role, check trust relationship
aws iam get-role --role-name ECS-Task-Role
```
  
### High ECR costs

**Symptoms:** AWS account with unexpected ECR costs

**Causes:**
1. Many images stored
2. Image size too large
3. Old versions not deleted
4. Cross-region replication active

**Solutions:**
```bash
# Check space used
aws ecr describe-repositories --repository-names app/backend

# View size of all images
aws ecr describe-images \
      --repository-name app/backend \
      --query 'imageDetails[].{Tag:imageTags[0],Size:imageSizeBytes}' \
      --output table

# Implement more aggressive lifecycle policy
lifecyclePolicy:
      policyText: |
        {
          "rules": [{
            "rulePriority": 1,
            "description": "Keep only 10 images",
            "selection": {
              "tagStatus": "any",
              "countType": "imageCountMoreThan",
              "countNumber": 10
            },
            "action": {"type": "expire"}
          }]
        }

# Delete specific large images
aws ecr batch-delete-image \
      --repository-name app/backend \
      --image-ids imageTag=old-build-1234

# Optimize image size in Dockerfile
# - Use multi-stage builds
# - Remove unnecessary layers
# - Use alpine base images
# - Combine RUN commands

# Optimized Dockerfile example
# FROM golang:1.21 AS builder
# COPY . /src
# WORKDIR /src
# RUN go build -o app
#
# FROM alpine:3.18
# RUN apk add --no-cache ca-certificates
# COPY --from=builder /src/app /usr/local/bin/
# CMD ["app"]
```
  
### Repository stuck in NotReady

**Symptoms:** Repository remains NotReady after creation

**Causes:**
1. Insufficient IAM permissions
2. KMS key not accessible (if using encryption)
3. Connectivity problem

**Solutions:**
```bash
# View detailed events
kubectl describe ecrrepository app-backend

# View operator logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100 | grep -i ecr

# Check AWSProvider is ready
kubectl get awsprovider
kubectl describe awsprovider production-aws

# If using KMS, check if it exists
aws kms describe-key --key-id alias/ecr-encryption

# If using KMS, check trust policy
aws kms get-key-policy --key-id alias/ecr-encryption --policy-name default

# Force synchronization
kubectl annotate ecrrepository app-backend \
      force-sync="$(date +%s)" --overwrite

# Last resort: delete and recreate
kubectl patch ecrrepository app-backend \
      --type merge \
      -p '{"spec":{"deletionPolicy":"Retain"}}'

kubectl delete ecrrepository app-backend

# Then recreate
kubectl apply -f ecr-repository.yaml
```
  
## Best Practices

:::note Best Practices

- **Enable scanOnPush for all repositories** — Automatically detect vulnerabilities, prevent deployment of images with critical CVEs, and monitor scanning findings regularly
- **Use IMMUTABLE tags in production** — Prevent overwrite of versioned tags, ensure v1.0.0 is always v1.0.0, and use semantic versioning (v1.0.0, v2.1.3)
- **Implement lifecycle policies** — Automatically delete old images, save storage costs, keep last N images, and delete untagged after X days
- **Use KMS encryption for compliance** — Encryption with custom key for HIPAA/PCI-DSS requirements, access control via IAM, and audit via CloudTrail
- **Configure repository policies carefully** — Controlled cross-account access, restrict pull/push by role/user, follow principle of least privilege
- **Tag all repositories consistently** — Include environment (dev/staging/prod), application, team, and CostCenter for governance and billing
- **Use consistent naming conventions** — Pattern `company/application-name`, descriptive names, use prefix for organization
- **Optimize Docker layer caching** — Structure Dockerfile to reuse layers, frequently changing commands at end, use multi-stage builds
- **Monitor ECR costs** — ~$0.10/GB/month, use lifecycle policies to save costs, use cost allocation tags
- **Integrate with Kubernetes properly** — EKS requires imagePullSecrets if cross-account, use IAM roles not static credentials, deploy with versioned tags
- **Consider cross-region replication** — Low latency in each region, DR (disaster recovery), automatic replication based on rules
- **Maintain audit trail** — Logging records all accesses, immutable tags for audit, image manifest digest for versioning
- **Keep images small** — Use alpine base (20MB vs 100MB+), multi-stage builds, remove unnecessary layers, use distroless when possible

:::
  
## Workflow CI/CD

### Build → Tag → Push → Deploy

**Typical pipeline:**

```yaml
# GitLab CI Example
stages:
  - build
  - push
  - deploy

build:
  stage: build
  script:
- docker build -t app:rev1 .
  artifacts:
reports:
      dotenv: build.env

push:
  stage: push
  script:
- aws ecr get-authorization-token | base64 -d | docker login --username AWS --password-stdin
- docker tag app:rev1 $ECR_REGISTRY/app/backend:rev1
- docker tag app:rev1 $ECR_REGISTRY/app/backend:latest
- docker push $ECR_REGISTRY/app/backend:rev1
- docker push $ECR_REGISTRY/app/backend:latest

deploy:
  stage: deploy
  script:
- kubectl set image deployment/app app=$ECR_REGISTRY/app/backend:rev1
```

**Integration with GitHub Actions:**

```yaml
name: Build and Push to ECR

on:
  push:
branches: [ main ]

jobs:
  build:
runs-on: ubuntu-latest
steps:
- uses: actions/checkout@v3

- name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        role-to-assume: arn:aws:iam::123456789012:role/github-actions-role
        aws-region: us-east-1

- name: Login to ECR
      id: login-ecr
      uses: aws-actions/amazon-ecr-login@v1

- name: Build and push
      env:
        ECR_REGISTRY: ${{ steps.login-ecr.outputs.registry }}
        ECR_REPOSITORY: app/backend
        IMAGE_TAG: ${{ github.sha }}
      run: |
        docker build -t $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG .
        docker push $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG
        docker tag $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG $ECR_REGISTRY/$ECR_REPOSITORY:latest
        docker push $ECR_REGISTRY/$ECR_REPOSITORY:latest
```

**Scan results integration:**

```bash
# Wait for scan to complete after push
aws ecr wait image-scan-complete \
  --repository-name app/backend \
  --image-id imageTag=latest

# Check for critical vulnerabilities
FINDINGS=$(aws ecr describe-image-scan-findings \
  --repository-name app/backend \
  --image-id imageTag=latest \
  --query 'imageScanFindings.findingSeverityCounts.CRITICAL')

if [ "$FINDINGS" -gt 0 ]; then
  echo "Critical vulnerabilities found!"
  exit 1
fi
```

## Related Resources

- [ECS - Elastic Container Service](/services/compute/ecs)

  - [EKS - Elastic Kubernetes Service](/services/compute/eks)

  - [Lambda Container](/services/compute/lambda)

  - [CodePipeline](/services/devops/codepipeline)

  - [KMS - Key Management Service](/services/security/kms)

  - [IAM - Roles and Policies](/services/security/iam)

---