---
title: 'Subnet - Network Segmentation'
description: 'Create subnets within a VPC for resource segmentation and isolation'
sidebar_position: 8
---

Create subnets within a VPC to organize and isolate your AWS resources.

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

**IAM Policy - Subnet (subnet-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ec2:CreateSubnet",
        "ec2:DeleteSubnet",
        "ec2:DescribeSubnets",
        "ec2:ModifySubnetAttribute",
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
  --role-name infra-operator-subnet-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator Subnet management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-subnet-role \
  --policy-name SubnetManagement \
  --policy-document file://subnet-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-subnet-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-subnet-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

A Subnet is a division of a VPC that allows you to segment your network into smaller IP address ranges. With Subnets, you can:

- Organize resources into tiers (public, private, data)
- Distribute resources across Availability Zones for high availability
- Implement security isolation
- Control network traffic routing

## Quick Start

**Public Subnet:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: e2e-public-subnet
  namespace: default
spec:
  providerRef:
    name: localstack
  vpcID: REPLACE_WITH_VPC_ID
  cidrBlock: "10.0.1.0/24"
  availabilityZone: "us-east-1a"
  mapPublicIpOnLaunch: true
  tags:
    Name: e2e-public-subnet
    Type: public
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Private Subnet:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: e2e-private-subnet
  namespace: default
spec:
  providerRef:
    name: localstack
  vpcID: REPLACE_WITH_VPC_ID
  cidrBlock: "10.0.2.0/24"
  availabilityZone: "us-east-1b"
  mapPublicIpOnLaunch: false
  tags:
    Name: e2e-private-subnet
    Type: private
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Apply and Verify:**

```bash
# Apply the subnet
kubectl apply -f subnet.yaml

# Verify status
kubectl get subnets
kubectl describe subnet e2e-public-subnet
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource

  Name of AWSProvider resource

ID of the VPC where the subnet will be created (example: `vpc-xxx`)

  :::note

The VPC must exist before creating the subnet
:::


IPv4 CIDR block for the subnet (example: "10.0.1.0/24")

  **Requirements:**
  - Must be within the VPC CIDR
  - Cannot overlap with other subnets in the VPC
  - Allowed netmask: /16 to /28

### Optional Fields

Availability Zone where the subnet will be created (example: `us-east-1a`)

  :::tip

Distribute subnets across multiple AZs for high availability
:::


If `true`, instances launched in this subnet receive public IP automatically

  **Use cases:**
  - `true`: Public subnets (web servers, load balancers)
  - `false`: Private subnets (databases, app servers)

Tags for organization and billing

  **Example:**

  ```yaml
  tags:
    Name: public-subnet-1a
    Type: public
kubernetes.io/role/elb: "1"  # For EKS load balancers
  ```

Deletion policy when the CR is removed

  **Options:**
  - `Delete`: Subnet is deleted from AWS
  - `Retain`: Subnet remains in AWS
  - `Orphan`: Removes management but keeps in AWS

## Status Fields

AWS identifier for the subnet (example: `subnet-12853af5337079de5`)

Parent VPC ID

CIDR block assigned to the subnet

AZ where the subnet is located

Current state of the subnet
  - `pending`: Being created
  - `available`: Ready for use

Number of available IP addresses in the subnet

`true` when the subnet is available

## Examples

### Multi-AZ Architecture with Public and Private Subnets

**Example:**

```yaml
---
# Public Subnet - AZ 1a
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
  tags:
    Name: public-subnet-1a
    Type: public
kubernetes.io/role/elb: "1"
  deletionPolicy: Delete

---
# Public Subnet - AZ 1b
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-1b
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.2.0/24"
  availabilityZone: us-east-1b
  mapPublicIpOnLaunch: true
  tags:
    Name: public-subnet-1b
    Type: public
kubernetes.io/role/elb: "1"
  deletionPolicy: Delete

---
# Private Subnet - AZ 1a
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: private-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.10.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: false
  tags:
    Name: private-subnet-1a
    Type: private
kubernetes.io/role/internal-elb: "1"
  deletionPolicy: Delete

---
# Private Subnet - AZ 1b
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: private-1b
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.11.0/24"
  availabilityZone: us-east-1b
  mapPublicIpOnLaunch: false
  tags:
    Name: private-subnet-1b
    Type: private
kubernetes.io/role/internal-elb: "1"
  deletionPolicy: Delete
```

### Database Subnet (Isolated)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: database-subnet-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.20.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: false
  tags:
    Name: database-subnet-1a
    Type: database
    Tier: data
    Encryption: required
  deletionPolicy: Retain  # Keep data subnet
```

## Verification

### Verify Subnet Status

**Command:**

```bash
# List all subnets
kubectl get subnets

# View details with available IPs
kubectl get subnet public-1a -o wide

# Full YAML
kubectl get subnet public-1a -o yaml
```

### Verify in AWS

**AWS CLI:**

```bash
# List subnets in a VPC
aws ec2 describe-subnets \
      --filters "Name=vpc-id,Values=vpc-xxx"

# Details of a specific subnet
aws ec2 describe-subnets \
      --subnet-ids subnet-xxx \
      --query 'Subnets[0]' \
      --output table
```

**LocalStack:**

```bash
export AWS_ENDPOINT_URL=http://localhost:4566

# List subnets
aws ec2 describe-subnets \
      --filters "Name=vpc-id,Values=vpc-xxx" \
      --query 'Subnets[*].[SubnetId,CidrBlock,AvailabilityZone,State]' \
      --output table
```

## Troubleshooting

### Error: VPC ID does not exist

**Error:** `VpcID vpc-xxx does not exist`

**Causes:**
1. VPC has not been created yet
2. Incorrect VPC ID in spec
3. VPC was deleted

**Solutions:**
```bash
# Check if VPC exists
kubectl get vpcs
kubectl get vpc <vpc-name> -o jsonpath='{.status.vpcID}'

# Update subnet with correct VPC ID
kubectl patch subnet <name> --type='json' \
      -p='[{"op": "replace", "path": "/spec/vpcID", "value":"vpc-xxx"}]'
```

### CIDR Conflict

**Error:** `CIDR block 10.0.1.0/24 conflicts with existing subnet`

**Cause:** A subnet with overlapping CIDR already exists

**Solutions:**
1. Use a different CIDR
2. Check existing subnets:
       ```bash
       aws ec2 describe-subnets \
         --filters "Name=vpc-id,Values=vpc-xxx" \
         --query 'Subnets[*].CidrBlock'
       ```

### Subnet not becoming ready

**Symptoms:** `ready: false` for more than 2 minutes

**Debug:**
```bash
# View events
kubectl describe subnet <name>

# Controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      | grep subnet

# Detailed status
kubectl get subnet <name> -o yaml
```

### No available IPs

**Symptoms:** `availableIpAddressCount: 0`

**Causes:**
- Subnet too small (/28 = only 11 usable IPs)
- Too many ENIs created

**Solutions:**
1. Create larger subnet (/24 = 251 IPs)
2. Clean up unused resources
3. Delete orphaned ENIs

## Best Practices

:::note Best Practices

- **Size subnets appropriately** — /24 for public (251 IPs), /22 for private (1019 IPs)
- **Use multiple AZs** — Spread subnets across 2-3 AZs for high availability
- **Separate public and private** — Public for load balancers, private for applications
- **Plan CIDR allocation** — Leave room for additional subnets in each AZ
- **Tag with AZ and tier** — Include availability zone and subnet type (public/private/database)

:::

## Architecture Patterns

### 3-Tier Architecture

Classic three-tier architecture with subnets organized by function and distributed across multiple Availability Zones for high availability:

![3-Tier Architecture](/img/diagrams/subnet-3tier-architecture.svg)

### EKS-Ready Subnets

**Example:**

```yaml
# Subnets for EKS need specific tags
tags:
  kubernetes.io/cluster/my-cluster: shared
  kubernetes.io/role/elb: "1"              # Public
  kubernetes.io/role/internal-elb: "1"     # Private
```

## Related Resources

- [VPC](/services/networking/vpc)

  - [NAT Gateway](/services/networking/nat-gateway)

  - [Security Groups](/services/networking/security-group)

  - [Multi-Tier Guide](/guides/multi-tier-network)