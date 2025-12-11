---
title: 'API Reference Overview'
description: 'Complete reference for all Infra Operator CRDs and APIs'
sidebar_position: 1
---

# API Reference

Complete reference documentation for Infra Operator Custom Resource Definitions (CRDs).

## API Group

All Infra Operator resources use the following API group:

**API Group:**

```
aws-infra-operator.runner.codes/v1alpha1
```

## Available Resources

### Core Resources

| Kind | Description | Status |
|------|-------------|--------|
| `AWSProvider` | AWS credentials and configuration | Stable |

### Networking Resources

| Kind | Description | Status |
|------|-------------|--------|
| `VPC` | Virtual Private Cloud | Stable |
| `Subnet` | VPC Subnet | Stable |
| `InternetGateway` | Internet Gateway | Stable |
| `NATGateway` | NAT Gateway | Stable |
| `RouteTable` | Route Table | Stable |
| `SecurityGroup` | Security Group | Stable |
| `ElasticIP` | Elastic IP Address | Stable |
| `ALB` | Application Load Balancer | Stable |
| `NLB` | Network Load Balancer | Stable |

### Compute Resources

| Kind | Description | Status |
|------|-------------|--------|
| `EC2Instance` | EC2 Instance | Stable |
| `EKSCluster` | EKS Kubernetes Cluster | Stable |
| `ECSCluster` | ECS Container Cluster | Stable |
| `LambdaFunction` | Lambda Function | Stable |
| `ComputeStack` | All-in-one Infrastructure | Stable |

### Storage Resources

| Kind | Description | Status |
|------|-------------|--------|
| `S3Bucket` | S3 Bucket | Stable |

### Database Resources

| Kind | Description | Status |
|------|-------------|--------|
| `RDSInstance` | RDS Database Instance | Stable |
| `DynamoDBTable` | DynamoDB Table | Stable |
| `ElastiCacheCluster` | ElastiCache Cluster | Stable |

### Container Resources

| Kind | Description | Status |
|------|-------------|--------|
| `ECRRepository` | ECR Container Registry | Stable |

### Messaging Resources

| Kind | Description | Status |
|------|-------------|--------|
| `SQSQueue` | SQS Queue | Stable |
| `SNSTopic` | SNS Topic | Stable |

### Security Resources

| Kind | Description | Status |
|------|-------------|--------|
| `IAMRole` | IAM Role | Stable |
| `KMSKey` | KMS Encryption Key | Stable |
| `SecretsManagerSecret` | Secrets Manager Secret | Stable |
| `Certificate` | ACM Certificate | Stable |
| `EC2KeyPair` | EC2 SSH Key Pair | Stable |

### CDN & DNS Resources

| Kind | Description | Status |
|------|-------------|--------|
| `CloudFront` | CloudFront Distribution | Stable |
| `Route53HostedZone` | Route53 Hosted Zone | Stable |
| `Route53RecordSet` | Route53 DNS Record | Stable |

### API Management

| Kind | Description | Status |
|------|-------------|--------|
| `APIGateway` | API Gateway | Stable |

## Common Fields

### ProviderRef

All AWS resources require a reference to an AWSProvider:

**Example:**

```yaml
spec:
  providerRef:
    name: aws-production  # Name of AWSProvider resource
```

### Tags

Most resources support AWS tags:

**Example:**

```yaml
spec:
  tags:
    Environment: production
    Team: platform
    ManagedBy: infra-operator
```

### DeletionPolicy

Controls what happens when the Kubernetes resource is deleted:

**Example:**

```yaml
spec:
  deletionPolicy: Delete  # Delete | Retain | Orphan
```

- `Delete`: Delete the AWS resource when CR is deleted (default)
- `Retain`: Keep the AWS resource but remove from operator management
- `Orphan`: Keep the AWS resource and remove ownership metadata

## Status Fields

All resources expose common status fields:

| Field | Type | Description |
|-------|------|-------------|
| `ready` | boolean | Whether resource is ready for use |
| `lastSyncTime` | string | Last successful sync with AWS |
| `conditions` | array | Detailed status conditions |

## Example

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Environment: production
  deletionPolicy: Retain
status:
  vpcID: vpc-0123456789abcdef0
  cidrBlock: "10.0.0.0/16"
  state: available
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Next Steps

- [AWSProvider Configuration](/api-reference/awsprovider)
- [CRD Specifications](/api-reference/crds)
- [Networking Resources](/services/networking/vpc)
