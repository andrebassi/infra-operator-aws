---
title: 'Internet Gateway - Internet Connectivity'
description: 'Provides internet access for resources in public subnets'
sidebar_position: 3
---

Provide internet access for resources in VPC public subnets.

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

**IAM Policy - Internet Gateway (igw-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ec2:CreateInternetGateway",
        "ec2:DeleteInternetGateway",
        "ec2:DescribeInternetGateways",
        "ec2:AttachInternetGateway",
        "ec2:DetachInternetGateway",
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
  --role-name infra-operator-igw-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator Internet Gateway management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-igw-role \
  --policy-name InternetGatewayManagement \
  --policy-document file://igw-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-igw-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-igw-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

An Internet Gateway (IGW) is a VPC component that enables communication between resources in the VPC and the internet. It is horizontally scalable, redundant, and highly available by design.

**Features:**
- Allows egress traffic to the internet
- Allows ingress traffic from the internet
- Supports IPv4 and IPv6
- No additional cost (only data traffic)
- One IGW per VPC

## Quick Start

**Basic Internet Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: e2e-igw
  namespace: default
spec:
  providerRef:
    name: localstack
  vpcID: REPLACE_WITH_VPC_ID
  tags:
    Name: e2e-internet-gateway
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Production Internet Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: production-igw
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-0a1b2c3d4e5f6g7h8
  tags:
    Name: production-internet-gateway
    Environment: production
    ManagedBy: infra-operator
    CostCenter: networking
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f internet-gateway.yaml
```

**Check Status:**

```bash
kubectl get internetgateway
kubectl describe internetgateway e2e-igw
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource

  Name of the AWSProvider resource

AWS VPC ID where the Internet Gateway will be attached (e.g., `vpc-0a1b2c3d4e5f6g7h8`)

  :::note

Use `REPLACE_WITH_VPC_ID` during deployment and replace with the actual VPC ID after its creation, or use the `status.vpcID` field from an existing VPC resource.

:::


### Optional Fields

Key-value pairs to tag the Internet Gateway

  **Example:**

  ```yaml
  tags:
    Name: production-igw
    Environment: production
    Team: platform
  ```

What happens to the IGW when the CR is deleted

  **Options:**
  - `Delete`: IGW is deleted from AWS
  - `Retain`: IGW remains in AWS but unmanaged

## Status Fields

After the Internet Gateway is created, the following status fields are populated:

AWS Internet Gateway identifier (e.g., `igw-0a1b2c3d4e5f6g7h8`)

ID of the VPC to which the IGW is attached

IGW attachment state
  - `attaching`: IGW is being attached to the VPC
  - `attached`: IGW is attached and ready for use
  - `detaching`: IGW is being detached
  - `detached`: IGW has been detached

`true` when the IGW is attached and ready for use

Timestamp of the last synchronization with AWS

## Examples

### Internet Gateway for Production

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: production-igw
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcID: vpc-0a1b2c3d4e5f6g7h8

  tags:
    Name: production-internet-gateway
    Environment: production
    ManagedBy: infra-operator
    CostCenter: networking

  # Keep IGW if CR is deleted
  deletionPolicy: Retain
```

### Internet Gateway for Development

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: dev-igw
  namespace: default
spec:
  providerRef:
    name: localstack

  vpcID: vpc-f3ea9b1b36fce09cd

  tags:
    Name: development-internet-gateway
    Environment: development
    AutoShutdown: "true"

  # Delete IGW on cleanup
  deletionPolicy: Delete
```

### Complete Configuration with VPC

**VPC:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: app-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: application-vpc
  deletionPolicy: Delete
```

**Internet Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: app-igw
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: REPLACE_WITH_VPC_ID
  tags:
    Name: application-igw
  deletionPolicy: Delete
```

**Public Subnet:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: app-public-subnet
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: REPLACE_WITH_VPC_ID
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
  tags:
    Name: public-subnet-1a
    Type: public
  deletionPolicy: Delete
```
## Verification

### Check IGW Status

**Command:**

```bash
# List all Internet Gateways
kubectl get internetgateways

# Get detailed information
kubectl get internetgateway production-igw -o yaml

# Watch creation progress
kubectl get internetgateway production-igw -w
```

### Check on AWS

**AWS CLI:**

```bash
# List Internet Gateways
aws ec2 describe-internet-gateways --internet-gateway-ids igw-xxx

# Get IGW details
aws ec2 describe-internet-gateways \
      --internet-gateway-ids igw-xxx \
      --query 'InternetGateways[0]' \
      --output json
```

**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws ec2 describe-internet-gateways \
      --internet-gateway-ids igw-xxx
```

### Expected Output

**Example:**

```yaml
status:
  internetGatewayID: igw-0a1b2c3d4e5f6g7h8
  vpcID: vpc-f3ea9b1b36fce09cd
  state: attached
  ready: true
  lastSyncTime: "2025-11-22T20:30:15Z"
```

## Troubleshooting

### IGW does not attach to VPC

**Symptoms:** IGW remains in `attaching` state

**Common causes:**
1. VPC does not exist or is not ready
2. Another IGW is already attached to the VPC
3. Insufficient IAM permissions

**Solutions:**
```bash
# Check if VPC exists and is ready
kubectl get vpc my-vpc

# Check if an IGW already exists in the VPC
aws ec2 describe-internet-gateways \
      --filters "Name=attachment.vpc-id,Values=vpc-xxx"

# Check controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100
```

### VPC already has IGW attached

**Error:** `"Resource has a dependency violation"`

**Cause:** A VPC can only have one Internet Gateway attached

**Solutions:**
1. Use the existing IGW
2. Or detach the old IGW before attaching a new one

**Command:**

```bash
# List IGWs in the VPC
aws ec2 describe-internet-gateways \
      --filters "Name=attachment.vpc-id,Values=vpc-xxx"
```

### Deletion stuck

**Symptoms:** IGW deletion takes too long or gets stuck

**Cause:** IGW still has dependencies (route tables pointing to it)

**Solutions:**
```bash
# Check route tables using the IGW
aws ec2 describe-route-tables \
      --filters "Name=route.gateway-id,Values=igw-xxx"

# Remove routes before deleting IGW
# (normally managed automatically by the operator)

# Force deletion if necessary
kubectl delete internetgateway my-igw --force --grace-period=0
```

### Internet does not work after creating IGW

**Symptoms:** Resources in public subnet cannot access the internet

**Cause:** Missing route table configuration

**Solutions:**
1. Check if the route table has a route to 0.0.0.0/0 pointing to the IGW
2. Check if the subnet is associated with the correct route table
3. Check security groups and NACLs

**Command:**

```bash
# Check route tables
aws ec2 describe-route-tables \
      --filters "Name=vpc-id,Values=vpc-xxx"
```

## Best Practices

:::note Best Practices

- **One IGW per VPC** — AWS allows only one IGW attached to a VPC
- **Reuse IGW for all public subnets** — No need for multiple IGWs in same VPC
- **Attach before routing** — IGW must be attached to VPC before adding routes
- **Tag consistently** — Include VPC name and environment in IGW tags
- **Never delete IGW with active routes** — Remove route table entries first

:::

## Public Network Architecture

![Internet Gateway Architecture](/img/diagrams/internet-gateway-architecture.svg)

## Related Resources

- [VPC](/services/networking/vpc)

  - [Subnet](/services/networking/subnet)

  - [NAT Gateway](/services/networking/nat-gateway)

  - [Route Table](/services/networking/route-table)
---
