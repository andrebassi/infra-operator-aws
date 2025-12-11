---
title: 'Security Group - Virtual Firewall'
description: 'Stateful firewall for network traffic control on EC2 instances'
sidebar_position: 7
---

Create and manage AWS Security Groups with granular control over inbound (ingress) and outbound (egress) traffic to protect your EC2 instances, RDS, Load Balancers, and other resources.

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

**IAM Policy - Security Groups (sg-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "ec2:CreateSecurityGroup",
        "ec2:DeleteSecurityGroup",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeSecurityGroupRules",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:AuthorizeSecurityGroupEgress",
        "ec2:RevokeSecurityGroupIngress",
        "ec2:RevokeSecurityGroupEgress",
        "ec2:UpdateSecurityGroupRuleDescriptionsIngress",
        "ec2:UpdateSecurityGroupRuleDescriptionsEgress",
        "ec2:ModifySecurityGroupRules",
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
# 1. Get EKS cluster OIDC Provider
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
  --role-name infra-operator-sg-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator Security Group management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-sg-role \
  --policy-name SecurityGroupManagement \
  --policy-document file://sg-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-sg-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-sg-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

AWS Security Groups act as a stateful virtual firewall that controls inbound (ingress) and outbound (egress) traffic to AWS resources such as EC2 instances, RDS databases, Load Balancers, and others. They are fundamental to network security in AWS.

**Features:**
- **Stateful Firewall**: Responses to allowed traffic are automatically permitted
- **Granular Control**: Rules per protocol (TCP/UDP/ICMP), port, and source/destination
- **VPC-Scoped**: Security Groups are created within a specific VPC
- **Multiple Rules**: Up to 60 ingress and 60 egress rules per Security Group
- **Source/Destination Flexibility**: CIDR blocks, another Security Group, or Prefix Lists
- **Default Deny**: All traffic is blocked unless explicitly allowed
- **No Outbound Restrictions**: By default, all outbound traffic is allowed
- **Real-time Changes**: Rule changes are applied immediately
- **Multiple Attachments**: A Security Group can be used by multiple resources
- **Chaining**: Reference other Security Groups in rules (no IP needed)
- **Auditing**: Flow logs via VPC Flow Logs
- **Tagging**: Organize and manage via tags
- **No Additional Cost**: Security Groups are free

**Status**: ✅ Works on Real AWS and LocalStack

## Quick Start

**Basic Security Group:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: e2e-web-server-sg
  namespace: default
spec:
  providerRef:
    name: localstack

  vpcId: vpc-0123456789abcdef0
  groupName: e2e-web-server-sg
  description: Security group for web servers

  ingressRules:
  - ipProtocol: tcp
    fromPort: 80
    toPort: 80
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTP from internet

  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTPS from internet

  egressRules:
  - ipProtocol: -1
    cidrIpv4: 0.0.0.0/0
    description: Allow all outbound traffic

  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing

  deletionPolicy: Delete
```

**Security Group with Source SG:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: e2e-database-sg
  namespace: default
spec:
  providerRef:
    name: localstack

  vpcId: vpc-0123456789abcdef0
  groupName: e2e-database-sg
  description: Security group for RDS database

  ingressRules:
  # PostgreSQL access only from web server SG
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-0987654321fedcba0
    description: Allow PostgreSQL from web servers

  egressRules:
  # No explicit egress = default deny all

  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing

  deletionPolicy: Delete
```

**Complete Security Group:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: app-backend-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  # VPC where the SG will be created
  vpcId: vpc-0123456789abcdef0

  # Security Group name (must be unique in VPC)
  groupName: app-backend-production

  # Description (cannot be changed later)
  description: Security group for backend application servers

  # Ingress Rules (incoming traffic)
  ingressRules:
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8080
    referencedGroupId: sg-alb123456
    description: Allow HTTP from ALB

  - ipProtocol: tcp
    fromPort: 9090
    toPort: 9090
    cidrIpv4: 10.0.0.0/8
    description: Allow metrics from internal network

  - ipProtocol: tcp
    fromPort: 22
    toPort: 22
    referencedGroupId: sg-bastion123
    description: SSH from bastion host

  # Egress Rules (outgoing traffic)
  egressRules:
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database456
    description: PostgreSQL to database

  - ipProtocol: tcp
    fromPort: 6379
    toPort: 6379
    referencedGroupId: sg-redis789
    description: Redis to cache cluster

  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS to internet for APIs

  # Tags for organization
  tags:
    Environment: production
    Application: backend
    Team: platform
    ManagedBy: infra-operator

  # Keep SG if CR is deleted
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f security-group.yaml
```

**Verify Status:**

```bash
kubectl get securitygroups
# or shortname
kubectl get sg

kubectl describe securitygroup e2e-web-server-sg
kubectl get securitygroup e2e-web-server-sg -o yaml
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource for authentication

  Name of AWSProvider resource

ID of the VPC where the Security Group will be created

  **Example:**

  ```yaml
  vpcId: vpc-0123456789abcdef0
  ```

  **Notes:**
  - VPC must exist before creating the Security Group
  - Security Group cannot be moved between VPCs
  - Use VPC CR created by infra-operator or existing VPC
  - Format: `vpc-` followed by 17 hexadecimal characters

Security Group name (must be unique within VPC)

  **Rules:**
  - 1 to 255 characters
  - Letters, numbers, spaces, `._-:/()#,@[]+=&;{}!$*`
  - Cannot start with `sg-`
  - Case sensitive

  **Example:**

  ```yaml
  groupName: my-app-backend-sg
  ```

Security Group description (cannot be changed later)

  **Example:**

  ```yaml
  description: Security group for application backend servers
  ```

  **Important:**
  - Description CANNOT be modified after creation
  - If you need to change it, must delete and recreate the Security Group
  - Use clear and detailed descriptions from the start
  - Maximum 255 characters

### Optional Fields - Ingress Rules

List of inbound traffic rules (ingress)

  **Example:**

  ```yaml
  ingressRules:
  - ipProtocol: tcp
    fromPort: 80
    toPort: 80
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTP from anywhere

  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv6: ::/0
    description: Allow HTTPS from anywhere IPv6

  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-app123456
    description: PostgreSQL from app servers
  ```

  IP protocol of the rule

**Options:**
- `tcp`: Transmission Control Protocol
- `udp`: User Datagram Protocol
- `icmp`: Internet Control Message Protocol (ping)
- `icmpv6`: ICMP for IPv6
- `58`: ICMPv6 (protocol number)
- `-1`: All protocols (any)
- Protocol number (0-255): [Complete list](https://www.iana.org/assignments/protocol-numbers/)

**Example:**

```yaml
ipProtocol: tcp
# or to allow everything
ipProtocol: -1
```

Start port of range (required for TCP/UDP)

**Example:**

```yaml
fromPort: 80
```

**Details:**
- Range: 0-65535
- Required for `tcp` and `udp`
- Not used for `icmp` or `-1`
- For single port, `fromPort` = `toPort`

End port of range (required for TCP/UDP)

**Example:**

```yaml
toPort: 80
# or range
fromPort: 8000
toPort: 8999
```

Source IPv4 CIDR block (mutually exclusive with referencedGroupId)

**Example:**

```yaml
cidrIpv4: 0.0.0.0/0          # Entire internet
cidrIpv4: 10.0.0.0/8          # Class A private network
cidrIpv4: 192.168.1.0/24      # Specific subnet
cidrIpv4: 203.0.113.25/32     # Single IP
```

**Common formats:**
- `0.0.0.0/0`: All IPv4 traffic (internet)
- `10.0.0.0/8`: RFC1918 private (10.x.x.x)
- `172.16.0.0/12`: RFC1918 private (172.16-31.x.x)
- `192.168.0.0/16`: RFC1918 private (192.168.x.x)
- `x.x.x.x/32`: Single IP

Source IPv6 CIDR block (mutually exclusive with referencedGroupId)

**Example:**

```yaml
cidrIpv6: ::/0                          # Entire IPv6 internet
cidrIpv6: 2001:db8::/32                 # IPv6 subnet
cidrIpv6: 2001:db8::1/128               # Single IPv6 IP
```

ID of another Security Group as source (instead of CIDR)

**Example:**

```yaml
referencedGroupId: sg-0123456789abcdef0
```

**Advantages:**
- No need to know specific IPs
- Rule automatically adjusts when instances change
- Recommended pattern for communication between AWS resources
- Can reference SG in another account (with peering)

**Important:**
- Mutually exclusive with `cidrIpv4` and `cidrIpv6`
- Referenced Security Group can be in same VPC or peered VPC

ID of a managed Prefix List (for AWS services)

**Example:**

```yaml
prefixListId: pl-12345678
```

**Usage:**
- AWS Managed Prefix Lists (S3, DynamoDB, CloudFront)
- Customer Managed Prefix Lists
- Example: allow access to S3 endpoints in region

Rule description (highly recommended)

**Example:**

```yaml
description: Allow HTTPS from CloudFront distribution
```

**Best practices:**
- Always add description
- Explain WHO and WHY has access
- Maximum 255 characters
- Facilitates auditing and troubleshooting

### Optional Fields - Egress Rules

List of outbound traffic rules (egress)

  **Example:**

  ```yaml
  egressRules:
  # Allow all outbound (AWS default)
  - ipProtocol: -1
    cidrIpv4: 0.0.0.0/0
    description: Allow all outbound traffic

  # Or restricted
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS to internet

  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database123
    description: PostgreSQL to database
  ```

  **Default behavior:**
  - AWS automatically creates `0.0.0.0/0` egress rule
  - If you specify `egressRules`, the default rule is REMOVED
  - To allow everything, explicitly add the `ipProtocol: -1` rule

  **Fields:** Same fields as `ingressRules` (ipProtocol, fromPort, toPort, cidrIpv4, cidrIpv6, referencedGroupId, prefixListId, description)

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
    Compliance: pci-dss
  ```

What happens to the Security Group when the CR is deleted

  **Options:**
  - `Delete`: Security Group is deleted from AWS (⚠️ may fail if in use)
  - `Retain`: Security Group remains in AWS but not managed
  - `Orphan`: Remove only management

  **Example:**

  ```yaml
  deletionPolicy: Retain
  ```

  **Important:**
  - Cannot delete SG that is in use (attached to ENI)
  - AWS returns error: "resource has a dependent object"
  - Use `Retain` if SG may be in use by unmanaged resources

## Status Fields

After the Security Group is created, the following status fields are populated:

`true` when the Security Group is created and ready for use

Security Group ID created in AWS

  ```
  sg-0123456789abcdef0
  ```

Security Group name (confirmation)

VPC where the Security Group was created (confirmation)

AWS account ID owning the Security Group

Number of configured ingress rules

Number of configured egress rules

Timestamp of last sync with AWS (ISO 8601 format)

Additional status message (errors, warnings, etc)

## Examples

### Security Group for Web Server (HTTP/HTTPS)

Allows HTTP and HTTPS traffic from internet:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: web-server-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcId: vpc-0123456789abcdef0
  groupName: web-server-public
  description: Security group for public web servers

  ingressRules:
  # HTTP from internet
  - ipProtocol: tcp
    fromPort: 80
    toPort: 80
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTP from internet

  # HTTPS from internet
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTPS from internet

  # HTTPS IPv6
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv6: ::/0
    description: Allow HTTPS from internet IPv6

  # SSH only from corporate VPN
  - ipProtocol: tcp
    fromPort: 22
    toPort: 22
    cidrIpv4: 203.0.113.0/24
    description: SSH from corporate VPN

  egressRules:
  # Allow all outbound traffic
  - ipProtocol: -1
    cidrIpv4: 0.0.0.0/0
    description: Allow all outbound traffic

  tags:
    Environment: production
    Type: web-server
    Tier: frontend

  deletionPolicy: Delete
```

### Security Group for Database (PostgreSQL/MySQL)

Allows database access only from specific servers:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: database-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcId: vpc-0123456789abcdef0
  groupName: rds-database-private
  description: Security group for RDS PostgreSQL database

  ingressRules:
  # PostgreSQL from app backend
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-backend123456
    description: PostgreSQL from backend servers

  # PostgreSQL from app worker
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-worker123456
    description: PostgreSQL from worker servers

  # PostgreSQL from bastion (for maintenance)
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-bastion123456
    description: PostgreSQL from bastion host for maintenance

  egressRules:
  # Database doesn't need egress
  # (not specifying egressRules = deny all)

  tags:
    Environment: production
    Type: database
    Engine: postgresql
    Tier: data

  deletionPolicy: Retain
```

### Security Group for Load Balancer (ALB/NLB)

Public load balancer forwarding to private backend:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: alb-public-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcId: vpc-0123456789abcdef0
  groupName: alb-public-frontend
  description: Security group for public Application Load Balancer

  ingressRules:
  # HTTP from internet
  - ipProtocol: tcp
    fromPort: 80
    toPort: 80
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTP from internet

  # HTTPS from internet
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: Allow HTTPS from internet

  # HTTPS IPv6
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv6: ::/0
    description: Allow HTTPS from internet IPv6

  egressRules:
  # Egress to backend servers on port 8080
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8080
    referencedGroupId: sg-backend123456
    description: Forward traffic to backend servers

  # Health checks to backend
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8080
    cidrIpv4: 10.0.0.0/8
    description: Health checks to backend in VPC

  tags:
    Environment: production
    Type: load-balancer
    Tier: frontend

  deletionPolicy: Retain
```

### Security Group for Internal Services (Microservices)

Internal communication between microservices:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: internal-services-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcId: vpc-0123456789abcdef0
  groupName: internal-microservices
  description: Security group for internal microservices communication

  ingressRules:
  # gRPC between services
  - ipProtocol: tcp
    fromPort: 50051
    toPort: 50051
    referencedGroupId: sg-self
    description: gRPC from other microservices

  # Internal HTTP API
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8099
    cidrIpv4: 10.0.0.0/8
    description: HTTP APIs from internal VPC

  # Metrics (Prometheus)
  - ipProtocol: tcp
    fromPort: 9090
    toPort: 9090
    referencedGroupId: sg-monitoring123
    description: Prometheus metrics scraping

  egressRules:
  # Redis cluster
  - ipProtocol: tcp
    fromPort: 6379
    toPort: 6379
    referencedGroupId: sg-redis123456
    description: Redis for caching

  # RabbitMQ
  - ipProtocol: tcp
    fromPort: 5672
    toPort: 5672
    referencedGroupId: sg-rabbitmq123
    description: RabbitMQ for messaging

  # PostgreSQL
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database123
    description: PostgreSQL database

  # HTTPS to external APIs
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS to external APIs

  # DNS
  - ipProtocol: udp
    fromPort: 53
    toPort: 53
    cidrIpv4: 0.0.0.0/0
    description: DNS resolution

  tags:
    Environment: production
    Type: microservices
    Tier: application

  deletionPolicy: Retain
```

### Security Group for Bastion Host (Jump Box)

SSH access server for administration:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: bastion-sg
  namespace: default
spec:
  providerRef:
    name: production-aws

  vpcId: vpc-0123456789abcdef0
  groupName: bastion-jump-host
  description: Security group for SSH bastion/jump host

  ingressRules:
  # SSH only from corporate IPs
  - ipProtocol: tcp
    fromPort: 22
    toPort: 22
    cidrIpv4: 203.0.113.0/24
    description: SSH from corporate office

  - ipProtocol: tcp
    fromPort: 22
    toPort: 22
    cidrIpv4: 198.51.100.0/24
    description: SSH from VPN gateway

  egressRules:
  # SSH to any server in VPC
  - ipProtocol: tcp
    fromPort: 22
    toPort: 22
    cidrIpv4: 10.0.0.0/8
    description: SSH to servers in VPC

  # PostgreSQL to databases (troubleshooting)
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database123
    description: PostgreSQL for database maintenance

  # HTTPS for package downloads
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS for package updates

  # DNS
  - ipProtocol: udp
    fromPort: 53
    toPort: 53
    cidrIpv4: 0.0.0.0/0
    description: DNS resolution

  tags:
    Environment: production
    Type: bastion
    Purpose: administration

  deletionPolicy: Retain
```

## Verification

### Verify Status via kubectl

**Command:**

```bash
# List all Security Groups
kubectl get securitygroups
# or shortname
kubectl get sg

# Get detailed information
kubectl get securitygroup web-server-sg -o yaml

# Watch creation in real-time
kubectl get securitygroup web-server-sg -w

# View events and status
kubectl describe securitygroup web-server-sg

# View only the created SG ID
kubectl get securitygroup web-server-sg -o jsonpath='{.status.securityGroupId}'

# View rule count
kubectl get securitygroup web-server-sg -o jsonpath='{.status.ingressRuleCount}'
```

### Verify in AWS

**AWS CLI:**

```bash
# List all Security Groups
aws ec2 describe-security-groups

# Get specific SG by ID
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0

# Get SG by name
aws ec2 describe-security-groups \
      --filters "Name=group-name,Values=web-server-public"

# Get SGs from a VPC
aws ec2 describe-security-groups \
      --filters "Name=vpc-id,Values=vpc-0123456789abcdef0"

# View ingress rules
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissions'

# View egress rules
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissionsEgress'

# List resources using the SG
aws ec2 describe-network-interfaces \
      --filters "Name=group-id,Values=sg-0123456789abcdef0"

# View rules in friendly format (table)
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --output table

# Verify tags
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].Tags'
```

**kubectl:**

```bash
# View all ingress rules in readable format
kubectl get securitygroup web-server-sg -o json | \
      jq '.spec.ingressRules[] | "Port: \(.fromPort)-\(.toPort) Protocol: \(.ipProtocol) From: \(.cidrIpv4 // .referencedGroupId)"'

# Export as YAML for backup
kubectl get securitygroup web-server-sg -o yaml > backup-sg.yaml

# View all SGs with their IDs
kubectl get sg -o custom-columns=NAME:.metadata.name,SG-ID:.status.securityGroupId,VPC:.spec.vpcId

# Check if SG is ready
kubectl get sg -o custom-columns=NAME:.metadata.name,READY:.status.ready
```

**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws ec2 describe-security-groups

# Create test rule
aws ec2 authorize-security-group-ingress \
      --group-id sg-test123 \
      --protocol tcp \
      --port 8080 \
      --cidr 0.0.0.0/0

# View created rules
aws ec2 describe-security-groups \
      --group-ids sg-test123
```

### Expected Output

**Example:**

```yaml
status:
  securityGroupId: sg-0123456789abcdef0
  groupName: web-server-public
  vpcId: vpc-0123456789abcdef0
  ownerId: "123456789012"
  ingressRuleCount: 4
  egressRuleCount: 1
  ready: true
  lastSyncTime: "2025-11-22T20:30:45Z"
```

## Troubleshooting

### Security Group is not created - 'vpc not found'

**Symptoms:** Error creating SG, message "VPC vpc-xxx not found"

**Common causes:**
1. Incorrect VPC ID or does not exist
2. VPC in different region
3. AWS credentials without DescribeVpcs permission

**Solutions:**
```bash
# Verify VPC exists
aws ec2 describe-vpcs --vpc-ids vpc-0123456789abcdef0

# List all VPCs in region
aws ec2 describe-vpcs --query 'Vpcs[*].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output table

# Check region in AWSProvider
kubectl get awsprovider production-aws -o yaml | grep region

# Check if VPC is in correct region
aws ec2 describe-vpcs --region us-east-1

# If VPC was created via CR, check if it's ready
kubectl get vpc my-vpc -o yaml

# Fix spec.vpcId
kubectl patch securitygroup web-server-sg \
      --type merge \
      -p '{"spec":{"vpcId":"vpc-CORRECT-ID"}}'

# Verify creation again
kubectl describe securitygroup web-server-sg
```

### Cannot delete Security Group - 'has dependent object'

**Symptoms:** Deleting SG fails with "resource has a dependent object"

**Cause:** Security Group is in use by EC2, RDS, ENI, Load Balancer, etc

**Solutions:**
```bash
# View which resources are using the SG
aws ec2 describe-network-interfaces \
      --filters "Name=group-id,Values=sg-0123456789abcdef0"

# View EC2 instances using the SG
aws ec2 describe-instances \
      --filters "Name=instance.group-id,Values=sg-0123456789abcdef0" \
      --query 'Reservations[*].Instances[*].[InstanceId,State.Name,Tags[?Key==`Name`].Value|[0]]' \
      --output table

# View Load Balancers using the SG
aws elbv2 describe-load-balancers \
      --query 'LoadBalancers[?SecurityGroups[?contains(@, `sg-0123456789abcdef0`)]].[LoadBalancerName,LoadBalancerArn]' \
      --output table

# View RDS instances using the SG
aws rds describe-db-instances \
      --query 'DBInstances[*].[DBInstanceIdentifier,VpcSecurityGroups[?VpcSecurityGroupId==`sg-0123456789abcdef0`]]'

# Resolution options:
# 1. Change SG of dependent resources first
aws ec2 modify-instance-attribute \
      --instance-id i-1234567890abcdef0 \
      --groups sg-other-id

# 2. Use deletionPolicy: Retain
kubectl patch securitygroup web-server-sg \
      --type merge \
      -p '{"spec":{"deletionPolicy":"Retain"}}'

# 3. Delete dependent resources before SG
kubectl delete ec2instance my-instance
# wait to complete
kubectl delete securitygroup web-server-sg

# 4. If everything fails, orphan the CR
kubectl patch securitygroup web-server-sg \
      -p '{"metadata":{"finalizers":[]}}' --type=merge
```

### Ingress/egress rules are not applied

**Symptoms:** Traffic doesn't work despite configured rules

**Causes:**
1. Rules were not synced
2. Wrong port range
3. Incorrect CIDR
4. Referenced SG does not exist
5. Network ACLs blocking

**Solutions:**
```bash
# Check if rules were applied in AWS
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissions'

# View operator logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100 | grep -i security

# Force reconciliation
kubectl annotate securitygroup web-server-sg \
      force-sync="$(date +%s)" --overwrite

# Check if referenced SG exists
aws ec2 describe-security-groups \
      --group-ids sg-referenced123

# Test connectivity (from within instance)
# SSH to instance
aws ssm start-session --target i-instance123

# Test port
nc -zv target-host 8080
curl -v http://target-host:8080

# Check NACLs (can block independently of SG)
aws ec2 describe-network-acls \
      --filters "Name=vpc-id,Values=vpc-0123456789abcdef0"

# View Flow Logs for debug
aws ec2 describe-flow-logs

# Recreate specific rules
kubectl patch securitygroup web-server-sg \
      --type json \
      -p '[{
        "op": "replace",
        "path": "/spec/ingressRules/0/fromPort",
        "value": 80
      }]'
```

### Referenced Security Group doesn't work

**Symptoms:** Traffic blocked even using referencedGroupId

**Causes:**
1. Referenced SG does not exist
2. SG in different VPC (without peering)
3. Typo in SG ID
4. Source instances don't have correct SG

**Solutions:**
```bash
# Check if referenced SG exists
aws ec2 describe-security-groups \
      --group-ids sg-backend123456

# Check if in same VPC (or peered)
aws ec2 describe-security-groups \
      --group-ids sg-backend123456 \
      --query 'SecurityGroups[0].VpcId'

# View which instances HAVE the source SG
aws ec2 describe-instances \
      --filters "Name=instance.group-id,Values=sg-backend123456" \
      --query 'Reservations[*].Instances[*].[InstanceId,PrivateIpAddress]' \
      --output table

# If no instances, traffic will never come!
# Check if SG was applied to instance
aws ec2 describe-instances \
      --instance-ids i-instance123 \
      --query 'Reservations[0].Instances[0].SecurityGroups'

# Fix referenced SG
kubectl patch securitygroup database-sg \
      --type json \
      -p '[{
        "op": "replace",
        "path": "/spec/ingressRules/0/referencedGroupId",
        "value": "sg-CORRECT-ID"
      }]'

# Add SG to source instance
aws ec2 modify-instance-attribute \
      --instance-id i-backend123 \
      --groups sg-backend123456 sg-other-needed

# Verify rule was created
aws ec2 describe-security-group-rules \
      --filters "Name=group-id,Values=sg-database123"
```

### Hit rule limit (60 per direction)

**Symptoms:** Error adding rule: "Rules limit exceeded"

**Cause:** Security Groups have limit of 60 ingress and 60 egress rules

**Solutions:**
```bash
# View current rule count
kubectl get securitygroup web-server-sg -o jsonpath='{.status.ingressRuleCount}'
kubectl get securitygroup web-server-sg -o jsonpath='{.status.egressRuleCount}'

# List all rules
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissions[*].[IpProtocol,FromPort,ToPort,IpRanges[*].CidrIp]' \
      --output table

# Resolution options:
# 1. Consolidate rules with port ranges
# Instead of:
#   fromPort: 8080, toPort: 8080
#   fromPort: 8081, toPort: 8081
#   fromPort: 8082, toPort: 8082
# Use:
#   fromPort: 8080, toPort: 8082

# 2. Use Prefix Lists to group CIDRs
aws ec2 create-managed-prefix-list \
      --prefix-list-name corporate-offices \
      --entries Cidr=203.0.113.0/24 Cidr=198.51.100.0/24 \
      --address-family IPv4 \
      --max-entries 50

# Use prefix list in rule
ingressRules:
- ipProtocol: tcp
  fromPort: 443
  toPort: 443
  prefixListId: pl-12345678
  description: HTTPS from corporate offices

# 3. Split into multiple Security Groups
# Create additional SG and associate both to resource
aws ec2 modify-instance-attribute \
      --instance-id i-123 \
      --groups sg-main sg-additional

# 4. Re-evaluate if all rules are necessary
# Remove unused rules
```

### Security Group stuck in NotReady

**Symptoms:** SG remains in NotReady after creation

**Causes:**
1. Insufficient IAM permissions
2. VPC does not exist
3. Connectivity issue with AWS
4. Duplicate GroupName in VPC

**Solutions:**
```bash
# View detailed events
kubectl describe securitygroup web-server-sg

# View operator logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100 | grep -i security

# Check AWSProvider is ready
kubectl get awsprovider
kubectl describe awsprovider production-aws

# Test IAM permissions manually
aws ec2 describe-security-groups --max-results 1

# Check if SG with same name already exists in VPC
aws ec2 describe-security-groups \
      --filters \
        "Name=vpc-id,Values=vpc-0123456789abcdef0" \
        "Name=group-name,Values=web-server-public"

# If SG already exists, import or change name
# Option 1: Change name
kubectl patch securitygroup web-server-sg \
      --type merge \
      -p '{"spec":{"groupName":"web-server-public-v2"}}'

# Option 2: Delete duplicate SG in AWS
aws ec2 delete-security-group \
      --group-id sg-duplicate123

# Force synchronization
kubectl annotate securitygroup web-server-sg \
      force-sync="$(date +%s)" --overwrite

# Last resort: delete and recreate
kubectl patch securitygroup web-server-sg \
      --type merge \
      -p '{"spec":{"deletionPolicy":"Orphan"}}'

kubectl delete securitygroup web-server-sg
kubectl apply -f security-group.yaml
```

### Changes to rules are not reflected

**Symptoms:** Changing spec.ingressRules/egressRules doesn't update AWS

**Cause:** Operator didn't reconcile or didn't detect change

**Solutions:**
```bash
# Check if spec was updated
kubectl get securitygroup web-server-sg -o yaml | grep -A 20 ingressRules

# View generation and observedGeneration
kubectl get securitygroup web-server-sg -o yaml | grep -E 'generation|observedGeneration'

# If generation != observedGeneration, operator didn't reconcile
# Force reconciliation
kubectl annotate securitygroup web-server-sg \
      force-sync="$(date +%s)" --overwrite

# View operator logs during update
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      -f | grep web-server-sg

# Check if rules changed in AWS
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissions'

# If operator is not running
kubectl get pods -n infra-operator-system

# Restart operator if necessary
kubectl rollout restart deployment/infra-operator-controller-manager \
      -n infra-operator-system

# Last resort: update via AWS CLI and force reverse sync
aws ec2 authorize-security-group-ingress \
      --group-id sg-0123456789abcdef0 \
      --protocol tcp \
      --port 8080 \
      --cidr 0.0.0.0/0

# Force operator to detect drift
kubectl annotate securitygroup web-server-sg \
      force-sync="$(date +%s)" --overwrite
```

## Best Practices

:::note Best Practices

- **Open only necessary ports** — Minimize attack surface by limiting exposed ports
- **Restrict CIDR to minimum** — Use specific IPs or security group references, never 0.0.0.0/0 for SSH
- **Use security group references** — Reference other SGs instead of IP ranges when possible
- **Separate groups by function** — Web, app, database tiers should have distinct security groups
- **Document all rules** — Include description for every ingress/egress rule

:::

## Common Architecture Patterns

### Three-Tier Architecture (Web/App/Database)

**Example:**

```yaml
# Tier 1: Public Load Balancer
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: alb-public-sg
spec:
  providerRef:
    name: production-aws
  vpcId: vpc-prod123
  groupName: tier1-alb-public
  description: Public ALB for internet traffic

  ingressRules:
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS from internet

  egressRules:
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8080
    referencedGroupId: sg-backend
    description: Forward to backend tier

---
# Tier 2: Application Servers
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: backend-app-sg
spec:
  providerRef:
    name: production-aws
  vpcId: vpc-prod123
  groupName: tier2-backend-app
  description: Backend application servers

  ingressRules:
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8080
    referencedGroupId: sg-alb
    description: HTTP from ALB

  egressRules:
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database
    description: PostgreSQL to database tier
  - ipProtocol: tcp
    fromPort: 443
    toPort: 443
    cidrIpv4: 0.0.0.0/0
    description: HTTPS for external APIs

---
# Tier 3: Database
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: database-sg
spec:
  providerRef:
    name: production-aws
  vpcId: vpc-prod123
  groupName: tier3-database
  description: PostgreSQL database

  ingressRules:
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-backend
    description: PostgreSQL from backend tier

  # No egress = deny all
```

### Microservices with Service Mesh

**Example:**

```yaml
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
metadata:
  name: service-mesh-sg
spec:
  providerRef:
    name: production-aws
  vpcId: vpc-prod123
  groupName: microservices-mesh
  description: Security group for service mesh communication

  ingressRules:
  # Service mesh data plane (Envoy)
  - ipProtocol: tcp
    fromPort: 15001
    toPort: 15001
    referencedGroupId: sg-self
    description: Envoy proxy mesh traffic

  # Application ports
  - ipProtocol: tcp
    fromPort: 8080
    toPort: 8099
    referencedGroupId: sg-self
    description: Inter-service HTTP APIs

  # gRPC
  - ipProtocol: tcp
    fromPort: 50051
    toPort: 50051
    referencedGroupId: sg-self
    description: gRPC between services

  egressRules:
  # Self-reference for mesh
  - ipProtocol: tcp
    fromPort: 15001
    toPort: 15001
    referencedGroupId: sg-self
    description: Envoy mesh egress

  # Database
  - ipProtocol: tcp
    fromPort: 5432
    toPort: 5432
    referencedGroupId: sg-database
    description: PostgreSQL

  # Redis
  - ipProtocol: tcp
    fromPort: 6379
    toPort: 6379
    referencedGroupId: sg-redis
    description: Redis cache
```

## Related Resources

- [VPC - Virtual Private Cloud](/services/networking/vpc)

  - [Subnet - Subnets](/services/networking/subnet)

  - [EC2 Instance](/services/compute/ec2)

  - [RDS Instance](/services/database/rds)

  - [Load Balancer (ALB/NLB)](/services/networking/load-balancer)

  - [NAT Gateway](/services/networking/nat-gateway)

---
