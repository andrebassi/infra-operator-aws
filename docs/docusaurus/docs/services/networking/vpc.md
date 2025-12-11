---
title: 'VPC - Virtual Private Cloud'
description: 'Create isolated virtual networks on AWS'
sidebar_position: 9
---

Create isolated virtual networks on AWS to securely host your resources.

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

**IAM Policy - VPC (vpc-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ec2:CreateVpc",
        "ec2:DeleteVpc",
        "ec2:DescribeVpcs",
        "ec2:ModifyVpcAttribute",
        "ec2:CreateTags",
        "ec2:DeleteTags",
        "ec2:DescribeTags"
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
  --role-name infra-operator-vpc-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator VPC management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-vpc-role \
  --policy-name VPCManagement \
  --policy-document file://vpc-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-vpc-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-vpc-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

A Virtual Private Cloud (VPC) is a logically isolated section of the AWS cloud where you can launch AWS resources in a virtual network that you define. With VPC, you have complete control over your virtual network environment, including:

- Selection of IP address ranges
- Creation of subnets
- Configuration of route tables
- Configuration of network gateways

## Quick Start

**Basic VPC:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: e2e-test-vpc
  namespace: default
spec:
  providerRef:
    name: localstack
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: e2e-test-vpc
    Environment: testing
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Production VPC:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc
    Environment: production
    Team: platform
    CostCenter: engineering
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f vpc.yaml
```

**Verify Status:**

```bash
kubectl get vpc
kubectl describe vpc e2e-test-vpc
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource

  Name of AWSProvider resource

IPv4 CIDR block for the VPC (e.g., "10.0.0.0/16")

  **Valid ranges:**
  - 10.0.0.0 - 10.255.255.255 (10/8 prefix)
  - 172.16.0.0 - 172.31.255.255 (172.16/12 prefix)
  - 192.168.0.0 - 192.168.255.255 (192.168/16 prefix)

  **Allowed netmask:** /16 to /28

### Optional Fields

Enables DNS resolution in the VPC. When enabled, instances in the VPC can resolve DNS hostnames.

Enables DNS hostnames in the VPC. Instances receive public DNS hostnames that correspond to their public IP addresses.

  :::note

Requires `enableDnsSupport: true`
:::


Tenancy option for instances launched in the VPC

  **Options:**
  - `default`: Instances run on shared hardware
  - `dedicated`: Instances run on single-tenant hardware (additional cost)

Key-value pairs to tag the VPC

  **Example:**

  ```yaml
  tags:
    Name: production-vpc
    Environment: production
    Team: platform
  ```

What happens to the VPC when the CR is deleted

  **Options:**
  - `Delete`: VPC is deleted from AWS
  - `Retain`: VPC remains in AWS but not managed
  - `Orphan`: VPC remains but CR ownership is removed

## Status Fields

After the VPC is created, the following status fields are populated:

AWS identifier of the VPC (e.g., `vpc-f3ea9b1b36fce09cd`)

CIDR block assigned to the VPC

Current state of the VPC
  - `pending`: VPC is being created
  - `available`: VPC is ready for use

`true` when the VPC is available and ready for use

Timestamp of last sync with AWS

## Examples

### Production VPC

High-availability VPC for production workloads:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Large CIDR for many subnets
  cidrBlock: "10.0.0.0/16"

  # Enable DNS for service discovery
  enableDnsSupport: true
  enableDnsHostnames: true

  # Shared hardware (cost-effective)
  instanceTenancy: default

  tags:
    Name: production-vpc
    Environment: production
    ManagedBy: infra-operator
    CostCenter: engineering

  # Retain VPC if CR is deleted
  deletionPolicy: Retain
```

### Development VPC

Smaller VPC for development environment:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: dev-vpc
  namespace: default
spec:
  providerRef:
    name: localstack

  # Smaller CIDR for dev
  cidrBlock: "172.16.0.0/16"

  enableDnsSupport: true
  enableDnsHostnames: true

  tags:
    Name: development-vpc
    Environment: development
    AutoShutdown: "true"

  # Delete VPC on cleanup
  deletionPolicy: Delete
```

### Multi-Environment VPCs

Separate VPCs for each environment:

**Production:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: prod-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc
    Environment: production
  deletionPolicy: Retain
```

**Staging:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: staging-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.1.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: staging-vpc
    Environment: staging
  deletionPolicy: Delete
```

**Development:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: dev-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.2.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: development-vpc
    Environment: development
  deletionPolicy: Delete
```
## Verification

### Verify VPC Status

**Command:**

```bash
# List all VPCs
kubectl get vpcs

# Get detailed VPC information
kubectl get vpc production-vpc -o yaml

# Watch VPC creation
kubectl get vpc production-vpc -w
```

### Verify in AWS

**AWS CLI:**

```bash
# List VPCs
aws ec2 describe-vpcs --vpc-ids vpc-xxx

# Get VPC details
aws ec2 describe-vpcs \
      --vpc-ids vpc-xxx \
      --query 'Vpcs[0]' \
      --output json
```

**LocalStack:**

```bash
# For LocalStack testing
export AWS_ENDPOINT_URL=http://localhost:4566

aws ec2 describe-vpcs \
      --vpc-ids vpc-xxx
```

### Expected Output

**Example:**

```yaml
status:
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: 10.0.0.0/16
  state: available
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### VPC stuck in pending state

**Symptoms:** VPC `state` is `pending` for more than 2 minutes

**Common causes:**
1. Invalid AWSProvider credentials
2. Network connectivity issues
3. AWS API rate limiting

**Solutions:**
```bash
# Check AWSProvider status
kubectl describe awsprovider production-aws

# Check controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100

# Check VPC events
kubectl describe vpc production-vpc
```

### CIDR block already exists

**Error:** `"CIDR block 10.0.0.0/16 conflicts with existing VPC"`

**Cause:** Another VPC with the same CIDR exists in the account

**Solutions:**
- Use a different CIDR block
- Delete the conflicting VPC
- Use VPC peering if connectivity is needed

### Deletion stuck

**Symptoms:** VPC deletion takes too long or gets stuck

**Cause:** Resources still attached to the VPC (subnets, IGW, etc.)

**Solutions:**
```bash
# Check attached resources
aws ec2 describe-subnets --filters "Name=vpc-id,Values=vpc-xxx"
aws ec2 describe-internet-gateways --filters "Name=attachment.vpc-id,Values=vpc-xxx"

# Delete dependent resources first
kubectl delete subnet <subnet-name>
kubectl delete internetgateway <igw-name>

# Then delete the VPC
kubectl delete vpc production-vpc
```

### DNS not working

**Symptoms:** Instances cannot resolve DNS hostnames

**Solutions:**
1. Ensure `enableDnsSupport: true`
2. Ensure `enableDnsHostnames: true` for public DNS
3. Check VPC DHCP options

**Example:**

```yaml
spec:
      enableDnsSupport: true
      enableDnsHostnames: true
```

## Best Practices

:::note Best Practices

- **Use /16 CIDR for production** — 65,536 IPs provides room for growth
- **Reserve space for future growth** — Plan CIDR blocks to avoid overlaps with peered VPCs
- **Enable DNS hostnames** — Required for private hosted zones and service discovery
- **Enable DNS support** — Required for VPC DNS resolution
- **Use consistent CIDR patterns** — Example: 10.{env}.0.0/16 where env=0 for prod, 1 for staging

:::

## Architecture Patterns

### Single VPC Architecture

Typical configuration with public and private subnets distributed across multiple availability zones for high availability:

![Single VPC Architecture](/img/diagrams/vpc-single-architecture.svg)

### Multi-VPC Architecture

Architecture with multiple VPCs isolated by environment, connected via VPC Peering for secure communication:

![Multi-VPC Architecture](/img/diagrams/vpc-multi-architecture.svg)

## Related Resources

- [Subnet](/services/networking/subnet)

  - [Internet Gateway](/services/networking/internet-gateway)

  - [NAT Gateway](/services/networking/nat-gateway)

  - [Multi-Tier Network Guide](/guides/multi-tier-network)