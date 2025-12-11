---
title: 'NAT Gateway - Internet for Private Subnets'
description: 'Allows resources in private subnets to access the internet while keeping their private IPs'
sidebar_position: 4
---

Allows resources in private subnets to access the internet securely while keeping their private IP addresses protected.

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

**IAM Policy - NAT Gateway (nat-gateway-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNatGateway",
        "ec2:DeleteNatGateway",
        "ec2:DescribeNatGateways",
        "ec2:AllocateAddress",
        "ec2:ReleaseAddress",
        "ec2:DescribeAddresses",
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
  --role-name infra-operator-nat-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator NAT Gateway management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-nat-role \
  --policy-name NATGatewayManagement \
  --policy-document file://nat-gateway-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-nat-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-nat-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

A NAT Gateway (Network Address Translation) allows instances in private subnets to initiate outbound connections to the internet or other AWS services, without receiving inbound connections. The NAT Gateway masks the instance's private IP address by replacing it with its own Elastic IP address (EIP).

**Features:**
- Allows egress traffic to the internet from private subnets
- Blocks unsolicited ingress traffic
- Uses Elastic IP (EIP) for consistency
- Automatically scales up to 45 Gbps
- Native IPv4 support
- Must be in a PUBLIC subnet
- Provides NAT flow logs
- No creation cost, pay for data usage

## Quick Start

**Basic NAT Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: e2e-nat
  namespace: default
spec:
  providerRef:
    name: localstack
  subnetID: REPLACE_WITH_PUBLIC_SUBNET_ID
  connectivityType: public
  tags:
    Name: e2e-nat-gateway
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Production NAT Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: my-nat
  namespace: default
spec:
  providerRef:
    name: production-aws

  # PUBLIC subnet where NAT will be allocated
  subnetRef:
    name: public-subnet-1a

  tags:
    Name: my-nat-gateway
    Environment: production

  deletionPolicy: Delete
```

**Apply:**

```bash
kubectl apply -f nat-gateway.yaml
```

**Check Status:**

```bash
kubectl get natgateway my-nat
kubectl describe natgateway my-nat
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource

  Name of AWSProvider resource

Reference to PUBLIC subnet where NAT Gateway will be allocated. NAT Gateway MUST be in a public subnet.

  Name of Subnet resource (must be public subnet)

### Optional Fields

Elastic IP (EIP) ID to be used by NAT Gateway. If not provided, an EIP will be created automatically.

  **Example:**

  ```yaml
  allocationID: eipalloc-0a1b2c3d4e5f6g7h8
  ```

Key-value pairs to tag the NAT Gateway

  **Example:**

  ```yaml
  tags:
    Name: production-nat-gateway
    Environment: production
    Team: platform
    CostCenter: networking
  ```

What happens to the NAT Gateway when the CR is deleted

  **Options:**
  - `Delete`: NAT Gateway is deleted from AWS
  - `Retain`: NAT Gateway remains in AWS but unmanaged

## Status Fields

After the NAT Gateway is created, the following status fields are populated:

AWS identifier of the NAT Gateway (e.g., `natgw-0a1b2c3d4e5f6g7h8`)

Elastic IP (EIP) address allocated to the NAT Gateway (e.g., `203.0.113.42`)

Private IP address of the NAT Gateway's network interface within the subnet

State of the NAT Gateway
  - `pending`: NAT Gateway is being created
  - `available`: NAT Gateway is ready for use
  - `deleting`: NAT Gateway is being deleted
  - `deleted`: NAT Gateway was deleted
  - `failed`: Creation or operation failed

`true` when NAT Gateway is available and ready for use

Timestamp of last synchronization with AWS

## Examples

### Production NAT Gateway with Multi-AZ HA

For high availability, create a NAT Gateway in each availability zone:

```yaml
# NAT Gateway in AZ 1a
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-gateway-1a
  namespace: default
spec:
  providerRef:
    name: production-aws

  subnetRef:
    name: public-subnet-1a

  tags:
    Name: nat-gateway-1a
    Environment: production
    AvailabilityZone: us-east-1a
    ManagedBy: infra-operator

  deletionPolicy: Retain

---
# NAT Gateway in AZ 1b
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-gateway-1b
  namespace: default
spec:
  providerRef:
    name: production-aws

  subnetRef:
    name: public-subnet-1b

  tags:
    Name: nat-gateway-1b
    Environment: production
    AvailabilityZone: us-east-1b
    ManagedBy: infra-operator

  deletionPolicy: Retain
```

### Development NAT Gateway

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: dev-nat
  namespace: default
spec:
  providerRef:
    name: localstack

  subnetRef:
    name: dev-public-subnet

  tags:
    Name: development-nat-gateway
    Environment: development
    AutoShutdown: "true"

  # Delete NAT during development cleanup
  deletionPolicy: Delete
```

### Complete Configuration with VPC, Subnets and Routing

**VPC:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: app-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: application-vpc
```

**Internet Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: app-igw
spec:
  providerRef:
    name: production-aws
  vpcRef:
    name: app-vpc
  tags:
    Name: application-igw
```

**Public Subnet for NAT:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet-1a
spec:
  providerRef:
    name: production-aws
  vpcRef:
    name: app-vpc
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
  tags:
    Name: public-subnet-1a
    Type: public
```

**Private Subnet for Applications:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: private-subnet-1a
spec:
  providerRef:
    name: production-aws
  vpcRef:
    name: app-vpc
  cidrBlock: "10.0.10.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: false
  tags:
    Name: private-subnet-1a
    Type: private
```

**NAT Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: app-nat
spec:
  providerRef:
    name: production-aws
  subnetRef:
    name: public-subnet-1a
  tags:
    Name: application-nat-gateway
```

**Public Route Table:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: public-rt
spec:
  providerRef:
    name: production-aws
  vpcRef:
    name: app-vpc
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      gatewayRef:
        name: app-igw
  associations:
- subnetRef:
        name: public-subnet-1a
  tags:
    Name: public-route-table
```

**Private Route Table:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-rt
spec:
  providerRef:
    name: production-aws
  vpcRef:
    name: app-vpc
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayRef:
        name: app-nat
  associations:
- subnetRef:
        name: private-subnet-1a
  tags:
    Name: private-route-table
```
## Verification

### Check NAT Gateway Status

**Command:**

```bash
# List all NAT Gateways
kubectl get natgateways

# Get detailed information
kubectl get natgateway my-nat -o yaml

# Watch creation in real-time
kubectl get natgateway my-nat -w
```

### Check in AWS

**AWS CLI:**

```bash
# List NAT Gateways
aws ec2 describe-nat-gateways

# Get specific NAT details
aws ec2 describe-nat-gateways \
      --nat-gateway-ids natgw-xxx \
      --query 'NatGateways[0]' \
      --output json

# Check Elastic IP
aws ec2 describe-addresses \
      --allocation-ids eipalloc-xxx

# List private subnets using the NAT
aws ec2 describe-route-tables \
      --filters "Name=route.nat-gateway-id,Values=natgw-xxx"
```

**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws ec2 describe-nat-gateways
aws ec2 describe-addresses
```

### Expected Output

**Example:**

```yaml
status:
  natGatewayID: natgw-0a1b2c3d4e5f6g7h8
  publicIP: 203.0.113.42
  privateIP: 10.0.1.150
  state: available
  ready: true
  lastSyncTime: "2025-11-22T20:30:15Z"
```

## Troubleshooting

### NAT stuck in pending

**Symptoms:** NAT remains in `pending` state for more than 5 minutes

**Common causes:**
1. Subnet is not in a valid availability zone
2. Subnet is private (NAT MUST be in public subnet)
3. No space to create ENI in subnet
4. Insufficient IAM permissions

**Solutions:**
```bash
# Check if subnet exists and is ready
kubectl get subnet public-subnet-1a
kubectl describe subnet public-subnet-1a

# Check controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100

# Check in AWS if ENI can be created
aws ec2 describe-subnets --subnet-ids subnet-xxx

# Verify it's actually a public subnet
aws ec2 describe-route-tables \
      --filters "Name=association.subnet-id,Values=subnet-xxx"
```

### Private subnet without internet

**Symptoms:** Pods in private subnet cannot access internet

**Cause:** Private route table doesn't have a route pointing to NAT Gateway

**Solutions:**
```bash
# Check private subnet route tables
aws ec2 describe-route-tables \
      --filters "Name=association.subnet-id,Values=subnet-privada"

# Route MUST be: 0.0.0.0/0 → natgw-xxx

# If route is missing, create via RouteTable CR:
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
      name: private-rt
spec:
      providerRef:
        name: production-aws
      vpcRef:
        name: my-vpc
      routes:
        - destinationCidrBlock: "0.0.0.0/0"
          natGatewayRef:
            name: my-nat
      associations:
        - subnetRef:
            name: private-subnet-1a
```
  
### High NAT Gateway costs

**Symptoms:** AWS bills with unexpected NAT costs

**Cause:** NAT Gateway charges per hour + data transfer

**Solutions:**
1. **Consolidate NAT to a single AZ** if possible
2. **Remove unused NATs**:
```bash
# Delete NAT
kubectl delete natgateway unused-nat

# Release associated Elastic IP
aws ec2 release-address --allocation-id eipalloc-xxx
```

3. **Use VPC Endpoints to reduce costs**:
```bash
# For S3, DynamoDB access, use Gateway Endpoints
# No data transfer charges
```

4. **Monitor CloudWatch**:
```bash
# View bytes processed
aws cloudwatch get-metric-statistics \
      --namespace AWS/NatGateway \
      --metric-name BytesOutToDestination \
      --start-time 2025-01-01T00:00:00Z \
      --end-time 2025-01-31T23:59:59Z \
      --period 3600 \
      --statistics Sum
```

### Deletion stuck

**Symptoms:** NAT deletion takes too long or gets stuck

**Cause:** Resources still use the NAT Gateway or EIP is in use

**Solutions:**
```bash
# Check if route tables still point to NAT
aws ec2 describe-route-tables \
      --filters "Name=route.nat-gateway-id,Values=natgw-xxx"

# Remove or update private routes:
kubectl patch routetable private-rt --type merge \
      -p '{"spec":{"routes":[]}}'

# Wait for finalizer to be removed
kubectl describe natgateway my-nat

# Force deletion if necessary
kubectl delete natgateway my-nat --force --grace-period=0

# If still stuck, remove finalizer
kubectl patch natgateway my-nat -p '{"metadata":{"finalizers":[]}}' --type=merge
```
  
### NAT Gateway not receiving traffic

**Symptoms:** NAT created successfully but doesn't process traffic

**Cause:** Security groups or NACLs blocking traffic

**Solutions:**
```bash
# Check private subnet security group
aws ec2 describe-security-groups \
      --group-ids sg-xxx

# MUST allow egress traffic to 0.0.0.0/0
# Example:
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
      name: private-sg
spec:
      providerRef:
        name: production-aws
      vpcRef:
        name: my-vpc
      egressRules:
        - description: Allow all outbound traffic
          protocol: -1
          fromPort: 0
          toPort: 65535
          cidrIp: "0.0.0.0/0"
```

**Command:**

```bash
# Test connectivity
# Enter pod in private subnet
kubectl exec -it pod-privado -- bash

# Try to access internet
curl -v https://www.google.com

# Check NAT flow logs
aws ec2 describe-flow-logs \
      --filter "Name=resource-type,Values=NatGateway"
```

## Best Practices

:::note Best Practices

- **Place NAT Gateway in public subnet** — Must have route to IGW for internet connectivity
- **Use one NAT per AZ for HA** — Avoids cross-AZ data transfer charges and single point of failure
- **Associate Elastic IP** — Required for consistent outbound IP address
- **Monitor data processing costs** — NAT Gateway charges per GB processed, optimize traffic patterns
- **Consider NAT Instance for dev** — Lower cost alternative for non-production environments

:::

## Network Architecture with NAT

### Network Topology

Typical architecture showing how NAT Gateway allows resources in private subnets to access the internet securely:

![NAT Gateway Architecture](/img/diagrams/nat-gateway-architecture.svg)

### Traffic Flow

The diagram below shows the path a packet takes when a private instance accesses the internet:

![NAT Gateway Packet Flow](/img/diagrams/nat-gateway-packet-flow.svg)

## Related Resources

- [VPC](/services/networking/vpc)

  - [Subnet](/services/networking/subnet)

  - [Internet Gateway](/services/networking/internet-gateway)

  - [Route Table](/services/networking/route-table)

  - [Security Group](/services/networking/security-group)

  - [Elastic IP](/services/networking/elastic-ip)
---