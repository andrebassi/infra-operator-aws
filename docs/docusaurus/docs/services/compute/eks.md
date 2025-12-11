---
title: 'EKS - Elastic Kubernetes Service'
description: 'Manage managed Kubernetes clusters on AWS'
sidebar_position: 2
---

Create and manage fully managed Kubernetes clusters on AWS with Amazon EKS.

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

### Required IAM Permissions

**IAM Policy - EKS (eks-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "eks:CreateCluster",
        "eks:DeleteCluster",
        "eks:DescribeCluster",
        "eks:UpdateClusterConfig",
        "eks:UpdateClusterVersion",
        "eks:TagResource",
        "eks:UntagResource",
        "eks:ListClusters"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:PassRole"
      ],
      "Resource": "arn:aws:iam::*:role/eks-cluster-role"
    }
  ]
}
```
## Overview

Amazon EKS (Elastic Kubernetes Service) is a managed service that makes it easy to run Kubernetes on AWS without needing to install, operate, and maintain your own Kubernetes control plane.

**Key Benefits:**
- 🚀 Fully managed control plane
- 🔒 Integrated security with IAM
- 📊 Native integration with AWS services
- 🔄 Automatic version updates
- 💰 Pay only for the resources you use

## Quick Start

**Basic Cluster:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: my-eks-cluster
  namespace: default
spec:
  providerRef:
    name: production-aws
  clusterName: my-eks-cluster
  version: "1.28"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-abc123
      - subnet-def456
      - subnet-ghi789
    endpointPublicAccess: true
    endpointPrivateAccess: true
  tags:
    Environment: production
    Team: platform
  deletionPolicy: Delete
```

**Cluster with Logging:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: production-eks
  namespace: default
spec:
  providerRef:
    name: production-aws
  clusterName: production-eks
  version: "1.29"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-abc123
      - subnet-def456
      - subnet-ghi789
    securityGroupIDs:
      - sg-12345678
    endpointPublicAccess: false
    endpointPrivateAccess: true
  logging:
    enabledTypes:
      - api
      - audit
      - authenticator
      - controllerManager
      - scheduler
  tags:
    Environment: production
    Team: platform
  deletionPolicy: Retain
```

**Cluster with Encryption:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: secure-eks
  namespace: default
spec:
  providerRef:
    name: production-aws
  clusterName: secure-eks
  version: "1.29"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-abc123
      - subnet-def456
      - subnet-ghi789
    endpointPublicAccess: false
    endpointPrivateAccess: true
  encryption:
    resources:
      - secrets
    providerKeyArn: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012
  logging:
    enabledTypes:
      - api
      - audit
  tags:
    Environment: production
    Compliance: pci-dss
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f eks-cluster.yaml
```

**Check Status:**

```bash
kubectl get ekscluster
kubectl describe ekscluster my-eks-cluster
```
## Configuration Reference

### Required Fields

AWSProvider resource reference

  AWSProvider resource name

Unique EKS cluster name

  :::warning

The name must be unique in the AWS region

:::


Kubernetes version (e.g., "1.28", "1.29")

  **Supported versions:** 1.24 to 1.30+

IAM Role ARN for the EKS cluster

  This role needs the trust policy `eks.amazonaws.com` and permissions to manage resources

VPC network configuration

  List of subnet IDs (minimum 2)

**Important:** Must be in different Availability Zones for high availability

Additional security groups for the cluster

Enable public access to the API endpoint

Enable private access to the API endpoint

Allowed CIDRs for public access

      Example: `["203.0.113.0/24", "198.51.100.0/24"]`

### Optional Fields

Cluster logging configuration

  Log types to enable

      Possible values:
- `api` - API server logs
- `audit` - Audit logs
- `authenticator` - Authentication logs
- `controllerManager` - Controller manager logs
- `scheduler` - Scheduler logs

Secret encryption configuration

  Resources to encrypt (typically `["secrets"]`)

KMS key ARN for encryption

Custom tags for the cluster

  Example:
  ```yaml
  tags:
    Environment: production
    Team: platform
    CostCenter: engineering
  ```

Resource deletion policy

  **Possible values:**
  - `Delete` - Delete cluster when removing CR
  - `Retain` - Keep cluster when removing CR

## Status

The cluster status is automatically updated by the operator:

```yaml
status:
  ready: true
  arn: arn:aws:eks:us-east-1:123456789012:cluster/my-eks-cluster
  endpoint: https://ABC123.gr7.us-east-1.eks.amazonaws.com
  status: ACTIVE
  version: "1.28"
  platformVersion: eks.3
  certificateAuthority: LS0tLS1CRUdJTi...
  lastSyncTime: "2025-01-23T10:30:00Z"
```

### Status Fields

Indicates if the cluster is ready (ACTIVE)

EKS cluster ARN

Kubernetes API endpoint URL

Cluster status: CREATING, ACTIVE, UPDATING, DELETING, FAILED

Certificate authority data (base64) for kubeconfig

## Use Cases

### 1. Development Cluster

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: dev-cluster
  namespace: development
spec:
  providerRef:
    name: dev-aws
  clusterName: dev-cluster
  version: "1.29"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-dev-1
      - subnet-dev-2
    endpointPublicAccess: true
    endpointPrivateAccess: false
  tags:
    Environment: development
    AutoShutdown: "true"
  deletionPolicy: Delete
```

### 2. Production Cluster with High Security

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: prod-cluster
  namespace: production
spec:
  providerRef:
    name: prod-aws
  clusterName: prod-cluster
  version: "1.29"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-prod-private-1a
      - subnet-prod-private-1b
      - subnet-prod-private-1c
    securityGroupIDs:
      - sg-cluster-control-plane
    endpointPublicAccess: false
    endpointPrivateAccess: true
  encryption:
    resources:
      - secrets
    providerKeyArn: arn:aws:kms:us-east-1:123456789012:key/prod-key
  logging:
    enabledTypes:
      - api
      - audit
      - authenticator
  tags:
    Environment: production
    Compliance: hipaa
    BackupPolicy: daily
  deletionPolicy: Retain
```

### 3. Multi-AZ Cluster for High Availability

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: ha-cluster
  namespace: production
spec:
  providerRef:
    name: prod-aws
  clusterName: ha-cluster
  version: "1.29"
  roleARN: arn:aws:iam::123456789012:role/eks-cluster-role
  vpcConfig:
    subnetIDs:
      - subnet-us-east-1a
      - subnet-us-east-1b
      - subnet-us-east-1c
      - subnet-us-east-1d
    endpointPublicAccess: true
    endpointPrivateAccess: true
    publicAccessCidrs:
      - "203.0.113.0/24"  # Office IP
  logging:
    enabledTypes:
      - api
      - audit
  tags:
    Environment: production
    HighAvailability: "true"
  deletionPolicy: Retain
```

## Common Operations

### Check Cluster Status

**Command:**

```bash
# List all clusters
kubectl get ekscluster

# View cluster details
kubectl describe ekscluster my-eks-cluster

# View status only
kubectl get ekscluster my-eks-cluster -o jsonpath='{.status.status}'
```

### Update Kubernetes Version

**Command:**

```bash
# Edit the CR and change the spec.version field
kubectl edit ekscluster my-eks-cluster

# Or apply a patch
kubectl patch ekscluster my-eks-cluster \
  --type merge \
  -p '{"spec":{"version":"1.29"}}'
```

:::warning

Version updates can only be done one minor version at a time (e.g., 1.27 → 1.28)

:::

### Configure Local Kubeconfig

**Command:**

```bash
# Get cluster data
CLUSTER_ENDPOINT=$(kubectl get ekscluster my-eks-cluster -o jsonpath='{.status.endpoint}')
CA_DATA=$(kubectl get ekscluster my-eks-cluster -o jsonpath='{.status.certificateAuthority}')

# Configure AWS CLI
aws eks update-kubeconfig \
  --name my-eks-cluster \
  --region us-east-1
```

### Enable Audit Logs

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
metadata:
  name: my-eks-cluster
spec:
  # ... other fields
  logging:
    enabledTypes:
      - audit
```

## Troubleshooting

### Cluster stuck in CREATING

### Check IAM Role

**Command:**

```bash
# Check if role exists and has correct permissions
aws iam get-role --role-name eks-cluster-role

# Check trust policy
aws iam get-role --role-name eks-cluster-role \
      --query 'Role.AssumeRolePolicyDocument'
```

### Check Subnets

**Command:**

```bash
# Check if subnets exist
aws ec2 describe-subnets --subnet-ids subnet-abc123 subnet-def456

# Check if they're in different AZs
aws ec2 describe-subnets \
      --subnet-ids subnet-abc123 subnet-def456 \
      --query 'Subnets[*].[SubnetId, AvailabilityZone]'
```

### View Operator Logs

**Command:**

```bash
kubectl logs -n infra-operator-system \
      -l control-plane=controller-manager \
      --tail=100
```

### Cluster FAILED

**Command:**

```bash
# View Kubernetes events
kubectl describe ekscluster my-eks-cluster

# View detailed logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager \
  | grep my-eks-cluster
```

### Can't delete cluster

If the cluster won't delete, it may be due to dependent resources:

```bash
# Force removal of finalizer (use with caution!)
kubectl patch ekscluster my-eks-cluster \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge
```

## Best Practices

:::note Best Practices

- **Use managed node groups** — Simplified updates and scaling vs self-managed
- **Enable cluster autoscaler** — Automatically adjust node count based on demand
- **Use private endpoint** — Restrict API server access to VPC
- **Enable audit logging** — Send to CloudWatch for compliance and troubleshooting
- **Version upgrades regularly** — Stay within supported Kubernetes versions

:::

## Related Resources

- [VPC](/services/networking/vpc)

  - [Subnet](/services/networking/subnet)

  - [Security Group](/services/networking/security-group)

  - [IAM Role](/services/security/iam)
---

## SetupEKS - Complete Infrastructure in a Single Resource

**SetupEKS** is a high-level CRD that creates all the AWS infrastructure needed for a functional EKS cluster with a single YAML manifest. It automates the creation of:

- VPC with configurable CIDR
- Public and private subnets in multiple AZs
- Internet Gateway and Route Tables
- NAT Gateway (Single or HighAvailability)
- Security Groups for cluster and nodes
- IAM Roles for cluster and nodes
- EKS Cluster with add-ons
- Configurable Node Groups

### Why use SetupEKS?

Create a complete EKS cluster with less than 20 lines of YAML

Automatically follows AWS best practices for EKS

Automatically deletes LoadBalancers created by Kubernetes before removing subnets

Use existing VPC or let the operator create everything automatically

### Quick Start - SetupEKS

**Minimal Cluster:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SetupEKS
metadata:
  name: my-cluster
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  vpcCIDR: "10.100.0.0/16"
  kubernetesVersion: "1.29"
  nodePools:
    - name: general
      instanceTypes:
        - t3.medium
      scalingConfig:
        minSize: 1
        maxSize: 3
        desiredSize: 2
```

**Cluster with NAT Gateway:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SetupEKS
metadata:
  name: eks-production
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  clusterName: my-production-cluster
  kubernetesVersion: "1.30"
  vpcCIDR: "10.200.0.0/16"
  natGatewayMode: Single  # or HighAvailability
  nodePools:
    - name: apps
      instanceTypes:
        - m5.large
        - m5a.large
      capacityType: ON_DEMAND
      scalingConfig:
        minSize: 2
        maxSize: 10
        desiredSize: 3
      labels:
        workload-type: apps
      subnetSelector: private
  tags:
    Environment: production
```

**Cluster with SPOT and ON_DEMAND:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SetupEKS
metadata:
  name: eks-mixed
  namespace: infra-operator
spec:
  providerRef:
    name: aws-develop
  vpcCIDR: "10.150.0.0/16"
  kubernetesVersion: "1.29"
  natGatewayMode: Single
  nodePools:
    # ON_DEMAND pool for critical workloads
    - name: on-demand
      instanceTypes:
        - t3.medium
      capacityType: ON_DEMAND
      scalingConfig:
        minSize: 1
        maxSize: 3
        desiredSize: 2
      labels:
        capacity-type: on-demand
    # SPOT pool for fault-tolerant workloads
    - name: spot
      instanceTypes:
        - t3.large
        - t3.xlarge
        - m5.large
      capacityType: SPOT
      scalingConfig:
        minSize: 0
        maxSize: 10
        desiredSize: 3
      labels:
        capacity-type: spot
      taints:
        - key: spot
          value: "true"
          effect: PREFER_NO_SCHEDULE
```

**Apply and Check:**

```bash
kubectl apply -f setupeks.yaml
kubectl get setupeks -n infra-operator -w
```

**Example 4: Existing VPC:**

```yaml
# Use your existing VPC/Subnets
# SetupEKS does NOT create network infrastructure
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SetupEKS
metadata:
  name: eks-existing-vpc
  namespace: infra-operator
spec:
  providerRef:
    name: aws-develop
  clusterName: my-cluster
  kubernetesVersion: "1.31"

  # Existing VPC - does NOT create VPC, Subnets, IGW, NAT
  existingVpcID: vpc-0123456789abcdef0
  existingSubnetIDs:
    - subnet-aaaa1111  # AZ us-east-1a (private or public)
    - subnet-bbbb2222  # AZ us-east-1b (private or public)

  # vpcCIDR and natGatewayMode are IGNORED
  # when existingVpcID is present

  nodePools:
    - name: workers
      instanceTypes:
        - t3.medium
      capacityType: ON_DEMAND
      scalingConfig:
        minSize: 1
        maxSize: 5
        desiredSize: 2
```
:::info

**Using Existing VPC**: When `existingVpcID` and `existingSubnetIDs` are provided:
- The operator does **NOT create** VPC, Subnets, Internet Gateway, NAT Gateway or Route Tables
- Creates **only** the EKS cluster and Node Groups using your existing infrastructure
- On deletion, does **NOT remove** existing VPC/Subnets (removes only EKS + Node Groups)
- LoadBalancer cleanup still works (deletes ALB/NLB created by Kubernetes)
- Requires **minimum 2 subnets** in different AZs (EKS requirement)

:::

### Configuration Reference - SetupEKS

#### Required Fields

AWSProvider reference for authentication

VPC CIDR block (e.g., "10.0.0.0/16")

List of node pools (minimum 1)

#### Optional Fields

EKS cluster name (uses metadata.name if not specified)

Kubernetes version (1.28, 1.29, 1.30)

NAT Gateway mode:
  - `Single` - One NAT Gateway (cost savings)
  - `HighAvailability` - NAT Gateway per AZ (production)
  - `None` - No NAT Gateway (nodes in public subnets)

List of AZs (minimum 2). Auto-detects if not specified.

Existing VPC ID to use instead of creating new one

Existing subnet IDs (requires existingVpcID)

Endpoint access configuration:
  - `publicAccess` (default: true)
  - `privateAccess` (default: true)
  - `publicAccessCIDRs` (default: ["0.0.0.0/0"])

Enable CloudWatch logs:
  - `apiServer`, `audit`, `authenticator`, `controllerManager`, `scheduler`

Secret encryption with KMS:
  - `enabled` (default: false)
  - `kmsKeyARN` (creates new key if not specified)

Enable IAM Roles for Service Accounts

Install essential add-ons (vpc-cni, coredns, kube-proxy)

### Node Pool Configuration

Unique node pool name

EC2 instance types (e.g., ["t3.medium", "t3.large"])

Auto-scaling configuration:
  - `minSize` (default: 1)
  - `maxSize` (default: 3)
  - `desiredSize` (default: 2)

Capacity type: `ON_DEMAND` or `SPOT`

AMI type:
  - `AL2_x86_64` - Amazon Linux 2 (x86)
  - `AL2_ARM_64` - Amazon Linux 2 (Graviton)
  - `AL2_x86_64_GPU` - Amazon Linux 2 with GPU
  - `BOTTLEROCKET_x86_64` - Bottlerocket
  - `BOTTLEROCKET_ARM_64` - Bottlerocket (Graviton)

Disk size in GB (20-16384)

Subnet selector: `private`, `public`, or `all`

Kubernetes labels applied to nodes

Taints applied to nodes:
  - `key`, `value`, `effect` (NO_SCHEDULE, NO_EXECUTE, PREFER_NO_SCHEDULE)

### Automatic LoadBalancer Cleanup

:::warning

SetupEKS automatically deletes all LoadBalancers (ALB, NLB) and Target Groups created within the VPC before deleting subnets.

:::

When you install services like NGINX Ingress Controller or AWS Load Balancer Controller in the cluster, they create LoadBalancers in AWS that are not managed by the operator. During SetupEKS deletion, these LoadBalancers block subnet removal due to ENIs (Elastic Network Interfaces) in use.

The operator resolves this automatically:

1. Lists all LoadBalancers in the VPC
2. Deletes listeners from each LoadBalancer
3. Deletes the LoadBalancers
4. Waits for complete deletion
5. Deletes orphaned Target Groups
6. Proceeds with subnet deletion

**Command:**

```bash
# Example log output during deletion:
# Found LoadBalancer in VPC, deleting... {"name": "k8s-...", "type": "network"}
# Deleting listener {"arn": "arn:aws:elasticloadbalancing:..."}
# LoadBalancer deletion initiated
```

### SetupEKS Status

**Example:**

```yaml
status:
  ready: true
  phase: Ready
  message: "All resources created successfully"
  vpc:
    id: vpc-0123456789abcdef0
    cidr: "10.100.0.0/16"
    state: available
  cluster:
    name: my-cluster
    arn: arn:aws:eks:us-east-1:123456789012:cluster/my-cluster
    endpoint: https://ABC123.gr7.us-east-1.eks.amazonaws.com
    status: ACTIVE
    version: "1.29"
  nodePools:
    - name: general
      status: ACTIVE
      desiredSize: 2
      minSize: 1
      maxSize: 3
  kubeconfigCommand: "aws eks update-kubeconfig --name my-cluster --region us-east-1"
```

### Check Status

**Command:**

```bash
# List SetupEKS
kubectl get setupeks -n infra-operator

# View details
kubectl describe setupeks my-cluster -n infra-operator

# View status in YAML
kubectl get setupeks my-cluster -n infra-operator -o yaml

# Monitor creation
kubectl get setupeks -n infra-operator -w
```

### Get Kubeconfig

**Command:**

```bash
# Get command from status
kubectl get setupeks my-cluster -n infra-operator \
  -o jsonpath='{.status.kubeconfigCommand}'

# Execute to configure local kubectl
aws eks update-kubeconfig --name my-cluster --region us-east-1
```

### Delete SetupEKS

**Command:**

```bash
# Delete (automatic LoadBalancer cleanup)
kubectl delete setupeks my-cluster -n infra-operator

# Monitor deletion
kubectl get setupeks my-cluster -n infra-operator -w
```

:::note

Deletion can take 15-20 minutes as it needs to delete Node Groups, EKS Cluster, NAT Gateways and VPC in the correct order.

:::

---

## References

- [Official Amazon EKS Documentation](https://docs.aws.amazon.com/eks/)
- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [AWS EKS API Reference](https://docs.aws.amazon.com/eks/latest/APIReference/)
