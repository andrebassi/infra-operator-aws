---
title: 'IAM Role - Access Management'
description: 'Create and manage IAM roles with access policies'
sidebar_position: 2
---

Manage identities and access permissions for AWS resources directly from Kubernetes.

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

**Verify Status:**

```bash
kubectl get awsprovider
kubectl describe awsprovider production-aws
```
:::warning

For production, always use **IRSA** (IAM Roles for Service Accounts) instead of static credentials.

:::

### Create IAM Role for IRSA

To use IRSA in production, you need to create an IAM Role with the necessary permissions:

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

**IAM Policy - IAM Role Management (iam-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:DeleteRole",
        "iam:GetRole",
        "iam:UpdateRole",
        "iam:AttachRolePolicy",
        "iam:DetachRolePolicy",
        "iam:PutRolePolicy",
        "iam:DeleteRolePolicy",
        "iam:GetRolePolicy",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies",
        "iam:TagRole",
        "iam:UntagRole",
        "iam:ListRoleTags"
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
  --role-name infra-operator-iam-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator IAM management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-iam-role \
  --policy-name IAMManagement \
  --policy-document file://iam-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-iam-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-iam-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

IAM Roles in Infra Operator allow you to create and manage access roles and their associated policies. Roles define who can access which AWS resources, with what permissions and under what conditions. They are essential for implementing the Least Privilege principle and ensuring security in your infrastructure.

### Key Concepts

- **IAM Role**: Entity that defines access permissions
- **Trust Relationship (AssumeRolePolicyDocument)**: Defines who can use the role
- **Managed Policies**: Pre-defined policies by AWS or custom
- **Inline Policies**: Specific policies embedded in the role
- **Permissions Boundary**: Maximum limit of permissions the role can have
- **Service-Linked Roles**: Automatic roles for specific services

## Quick Start

**EC2 Service Role:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: e2e-ec2-role
  namespace: default
spec:
  providerRef:
    name: localstack
  roleName: e2e-ec2-service-role
  description: "IAM role for EC2 instances - E2E testing"
  path: /e2e/
  maxSessionDuration: 3600

  # Who can assume this role (Trust Relationship)
  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "ec2.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
}

  # AWS managed policies
  managedPolicyArns:
  - arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
  - arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy

  tags:
environment: test
managed-by: infra-operator
purpose: e2e-testing

  deletionPolicy: Delete
```

**Lambda with Inline Policy:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: e2e-lambda-role
  namespace: default
spec:
  providerRef:
    name: localstack
  roleName: e2e-lambda-execution-role
  description: "Lambda execution role with custom inline policy - E2E testing"
  path: /lambda/
  maxSessionDuration: 7200

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "lambda.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  # Custom inline policy
  inlinePolicy:
policyName: CustomDynamoDBAccess
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "dynamodb:GetItem",
              "dynamodb:PutItem",
              "dynamodb:UpdateItem",
              "dynamodb:Query"
            ],
            "Resource": "arn:aws:dynamodb:*:*:table/e2e-*"
          }
        ]
      }

  tags:
environment: test
managed-by: infra-operator
service: lambda
purpose: e2e-testing

  deletionPolicy: Delete
```

**Production Role:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: lambda-execution-role
  namespace: default
spec:
  providerRef:
    name: production-aws
  roleName: lambda-execution-role
  description: "Role for Lambda function execution"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  tags:
    Environment: production
    Application: my-app

  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f iam-role.yaml
```

**Verify:**

```bash
kubectl get iamroles
kubectl describe iamrole lambda-execution-role
kubectl get iamrole lambda-execution-role -o jsonpath='{.status.roleArn}'
```
## Configuration

### Required Fields

| Field | Description | Example |
|-------|-----------|---------|
| `providerRef.name` | Name of configured AWSProvider | `production-aws` |
| `roleName` | Name of role in AWS IAM | `lambda-execution-role` |
| `assumeRolePolicyDocument` | JSON document that defines who can assume the role | JSON policy document |

### Optional Fields

| Field | Description | Default |
|-------|-----------|--------|
| `description` | Human-readable description of the role | "" |
| `path` | Role path in IAM (/service-role/, /custom/, etc) | "/" |
| `managedPolicyArns` | List of ARNs of managed policies | [] |
| `inlinePolicies` | Inline policies embedded in the role | [] |
| `maxSessionDuration` | Maximum session duration in seconds | 3600 |
| `permissionsBoundary` | ARN of policy that limits permissions | "" |
| `tags` | Tags for categorization and governance | {} |

### Complete Example

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: application-service-role
  namespace: default
spec:
  providerRef:
    name: production-aws

  roleName: application-service-role
  description: "Role for access to storage and database resources"
  path: /service-roles/

  # Define the trust relationship
  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {"Service": "ec2.amazonaws.com"},
          "Action": "sts:AssumeRole"
        },
        {
          "Effect": "Allow",
          "Principal": {"AWS": "arn:aws:iam::123456789012:role/other-role"},
          "Action": "sts:AssumeRole",
          "Condition": {"StringEquals": {"sts:ExternalId": "unique-external-id"}}
        }
      ]
}

  # Managed policies
  managedPolicyArns:
  - arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
  - arn:aws:iam::aws:policy/CloudWatchLogsFullAccess

  # Custom inline policies
  inlinePolicies:
  - policyName: dynamodb-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "dynamodb:GetItem",
              "dynamodb:PutItem",
              "dynamodb:UpdateItem",
              "dynamodb:Query"
            ],
            "Resource": "arn:aws:dynamodb:us-east-1:123456789012:table/users"
          }
        ]
      }

  - policyName: secretsmanager-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["secretsmanager:GetSecretValue"],
            "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/*"
          }
        ]
      }

  # Maximum permissions limit
  permissionsBoundary: arn:aws:iam::aws:policy/PowerUserAccess

  # Maximum session duration: 12 hours
  maxSessionDuration: 43200

  # Governance tags
  tags:
    Environment: production
    Team: platform
    CostCenter: engineering
    ManagedBy: infra-operator
```

## Status

After applying the role, you can verify its status:

```bash
kubectl describe iamrole lambda-execution-role
```

### Status Fields

| Field | Description |
|-------|-----------|
| `roleArn` | Complete ARN of created role |
| `roleId` | Unique ID of role in AWS IAM |
| `createDate` | Role creation date |
| `ready` | Boolean indicating if role is ready for use |
| `conditions` | Details of errors or warnings |

Example status:

```yaml
status:
  roleArn: arn:aws:iam::123456789012:role/lambda-execution-role
  roleId: AIDAQ2EXAMPLE4DCDFG7
  createDate: "2025-11-22T10:30:00Z"
  ready: true
```

## Practical Examples

### 1. Lambda Execution Role

Role to allow Lambda functions to write logs to CloudWatch:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: lambda-exec-role
spec:
  providerRef:
    name: production-aws
  roleName: lambda-execution-role
  description: "Role for Lambda function execution with CloudWatch logs"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  tags:
    Service: lambda
    Environment: production
```

### 2. EC2 Instance Profile Role

Role for EC2 instances with access to S3 and SSM:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: ec2-app-server-role
spec:
  providerRef:
    name: production-aws
  roleName: EC2-AppServer-Role
  description: "Role for EC2 application servers"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "ec2.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore
  - arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy

  inlinePolicies:
  - policyName: s3-app-bucket-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["s3:GetObject", "s3:ListBucket"],
            "Resource": [
              "arn:aws:s3:::my-app-bucket",
              "arn:aws:s3:::my-app-bucket/*"
            ]
          }
        ]
      }

  tags:
    Environment: production
    Type: ec2-instance
```

### 3. Cross-Account Access Role

Role to assume access in another AWS account:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: cross-account-role
spec:
  providerRef:
    name: production-aws
  roleName: CrossAccount-Production-Access
  description: "Role for cross-account access from development account"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {
          "AWS": "arn:aws:iam::111111111111:root"
        },
        "Action": "sts:AssumeRole",
        "Condition": {
          "StringEquals": {"sts:ExternalId": "unique-external-id-123"}
        }
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/ReadOnlyAccess

  maxSessionDuration: 28800  # 8 hours

  tags:
    Purpose: cross-account-access
    SourceAccount: "111111111111"
```

### 4. Service Role for ECS

Role for ECS tasks with access to ECR and SecretsManager:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: ecs-task-role
spec:
  providerRef:
    name: production-aws
  roleName: ecsTaskExecutionRole
  description: "Role for ECS task execution"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "ecs-tasks.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

  inlinePolicies:
  - policyName: ecr-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "ecr:GetAuthorizationToken",
              "ecr:BatchGetImage",
              "ecr:GetDownloadUrlForLayer"
            ],
            "Resource": "*"
          }
        ]
      }

  - policyName: secrets-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["secretsmanager:GetSecretValue"],
            "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:ecs/*"
          }
        ]
      }

  tags:
    Service: ecs
    Environment: production
```

### 5. Role with Permissions Boundary

Role with permissions limit to ensure compliance:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: developer-role
spec:
  providerRef:
    name: production-aws
  roleName: Developer-Boundary-Role
  description: "Role for developers with permissions boundary"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {
          "AWS": "arn:aws:iam::123456789012:root"
        },
        "Action": "sts:AssumeRole"
      }]
}

  # Role can have these permissions
  managedPolicyArns:
  - arn:aws:iam::aws:policy/PowerUserAccess

  # But is limited by this boundary
  permissionsBoundary: arn:aws:iam::123456789012:policy/DeveloperBoundaryPolicy

  tags:
    Type: developer
    Environment: development
```

### 6. Role for IRSA (EKS ServiceAccount)

Role for integration with EKS Service Accounts (IRSA):

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: eks-app-role
spec:
  providerRef:
    name: production-aws
  roleName: EKS-App-ServiceAccount-Role
  description: "Role for pods in EKS via IRSA"

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {
          "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/ABC123"
        },
        "Action": "sts:AssumeRoleWithWebIdentity",
        "Condition": {
          "StringEquals": {
            "oidc.eks.us-east-1.amazonaws.com/id/ABC123:sub": "system:serviceaccount:default:app-sa"
          }
        }
      }]
}

  inlinePolicies:
  - policyName: s3-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["s3:*"],
            "Resource": "arn:aws:s3:::eks-app-data/*"
          }
        ]
      }

  tags:
    Cluster: production-eks
    Purpose: irsa
```

## Verification and Tests

### Verify Created Role

**Command:**

```bash
# List all roles
kubectl get iamroles

# Role details
kubectl describe iamrole lambda-execution-role

# Get role ARN
kubectl get iamrole lambda-execution-role -o jsonpath='{.status.roleArn}'

# View trust policy
kubectl get iamrole lambda-execution-role -o yaml | grep -A 50 assumeRolePolicy
```

### Test Assume Role with AWS CLI

**Command:**

```bash
# Assume the role from your AWS user
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/lambda-execution-role \
  --role-session-name test-session

# Use returned credentials for requests
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...

# Verify assumed identity
aws sts get-caller-identity
```

### Validate Permissions

**Command:**

```bash
# List role permissions
aws iam get-role --role-name lambda-execution-role

# List attached policies
aws iam list-attached-role-policies --role-name lambda-execution-role

# List inline policies
aws iam list-role-policies --role-name lambda-execution-role

# View specific inline policy
aws iam get-role-policy \
  --role-name lambda-execution-role \
  --policy-name dynamodb-access
```

## Troubleshooting

### Error: "Access Denied" when Creating Role

**Problem**: Operator doesn't have permission to create roles.

**Solution**:
1. Verify IAM permissions of operator user/role
2. Operator needs `iam:CreateRole`, `iam:AttachRolePolicy`, `iam:PutRolePolicy`
3. Verify trust policy is correct in AWSProvider

**Command:**

```bash
# Verify credentials
aws sts get-caller-identity

# Verify test permissions
aws iam list-roles --max-items 1
```

### Error: "Invalid Trust Relationship"

**Problem**: `assumeRolePolicyDocument` has invalid syntax.

**Solution**:
1. Validate JSON using `python3 -m json.tool`
2. Ensure principal syntax is correct
3. Verify ARNs actually exist

**Example:**

```yaml
# Correct ✅
assumeRolePolicyDocument: |
  {
"Version": "2012-10-17",
"Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "lambda.amazonaws.com"},
      "Action": "sts:AssumeRole"
}]
  }

# Incorrect ❌
assumeRolePolicyDocument: '{
  "Version": "2012-10-17",
  # Comments are not allowed in JSON
}'
```

### Error: "Policy Too Permissive"

**Problem**: Policy allows more than expected.

**Solution**:
1. Review each statement in the policy
2. Use specific Resources instead of "*"
3. Use Conditions to restrict access
4. Implement Permissions Boundary

**Example:**

```yaml
# Improved ✅
inlinePolicies:
- policyName: s3-limited-access
  policyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Action": ["s3:GetObject", "s3:ListBucket"],
        "Resource": "arn:aws:s3:::my-bucket/*",
        "Condition": {
          "IpAddress": {
            "aws:SourceIp": ["10.0.0.0/8"]
          }
        }
      }]
}
```

### Error: "Circular Dependency"

**Problem**: Role A assumes Role B which assumes Role A.

**Solution**:
1. Review role hierarchy
2. Use External IDs to avoid confusion
3. Consider a different assumeRole design

### Error: "Session Duration Exceeded"

**Problem**: `maxSessionDuration` is lower than necessary.

**Solution**:
1. Increase value (maximum: 43200 seconds = 12 hours)
2. Verify if application really needs long sessions

**Example:**

```yaml
# Minimum: 900 (15 min)
# Maximum: 43200 (12 hours)
maxSessionDuration: 28800  # 8 hours
```

## Best Practices

:::note Best Practices

- **Principle of Least Privilege** — Always use minimum necessary permissions:
- **Use Managed Policies for Common Patterns** — **Example:**
- **Implement Permissions Boundary** — Always use permissions boundary for assumable roles:
- **Use Tags for Governance** — **Example:**
- **Review and Audit Regularly** — **Command:**
- **MFA for Sensitive Roles** — **Example:**
- **Rotation Policies** — Document when roles were created and their next review date:
- **Service Control Policies (SCP)** — Combine with SCPs for maximum security:

:::

## Common Patterns

### Lambda with S3 and DynamoDB Access

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: lambda-data-processor
spec:
  providerRef:
    name: production-aws
  roleName: lambda-data-processor

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  inlinePolicies:
  - policyName: data-access
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["s3:GetObject", "s3:PutObject"],
            "Resource": "arn:aws:s3:::data-bucket/*"
          },
          {
            "Effect": "Allow",
            "Action": [
              "dynamodb:GetItem",
              "dynamodb:PutItem",
              "dynamodb:Query"
            ],
            "Resource": "arn:aws:dynamodb:us-east-1:*:table/data"
          }
        ]
      }
```

### EC2 with Access to Multiple Services

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: ec2-application-role
spec:
  providerRef:
    name: production-aws
  roleName: EC2-Application-Role

  assumeRolePolicyDocument: |
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "ec2.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}

  managedPolicyArns:
  - arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore
  - arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy
  - arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly

  inlinePolicies:
  - policyName: s3-and-dynamodb
policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": ["s3:*"],
            "Resource": "arn:aws:s3:::app-bucket/*"
          },
          {
            "Effect": "Allow",
            "Action": ["dynamodb:*"],
            "Resource": "arn:aws:dynamodb:*:*:table/app-*"
          },
          {
            "Effect": "Allow",
            "Action": ["secretsmanager:GetSecretValue"],
            "Resource": "arn:aws:secretsmanager:*:*:secret:app/*"
          }
        ]
      }
```

## Related Resources

### AWS Documentation
- [IAM Roles Documentation](https://docs.aws.amazon.com/iam/latest/userguide/id_roles.html)
- [Trust Relationships](https://docs.aws.amazon.com/iam/latest/userguide/id_roles_create_for-user.html)
- [Permissions Boundary](https://docs.aws.amazon.com/iam/latest/userguide/access_policies_boundaries.html)
- [Policy Simulator](https://policysim.aws.amazon.com/)

### Useful Tools
- [IAM Policy Generator](https://awspolicygen.s3.amazonaws.com/policygen.html)
- [IAM Access Analyzer](https://aws.amazon.com/iam/access-analyzer/)
- [Policy Validator](https://docs.aws.amazon.com/IAM/latest/UserGuide/access-analyzer-policy-validation.html)

### Integration with Infra Operator
- [EC2 Instance - Role Profile](/services/compute/ec2)
- [Lambda Function - Execution Role](/services/compute/lambda)
- [ECS Task - Execution Role](/services/compute/ecs)
- [Secrets Manager - Role Access](/services/security/secretsmanager)

---
