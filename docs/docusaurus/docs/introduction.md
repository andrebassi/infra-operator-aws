---
title: 'Introduction'
description: 'Manage AWS infrastructure directly from Kubernetes with Infra Operator'
slug: /
sidebar_position: 1
---

Manage AWS resources directly from Kubernetes using Custom Resources and GitOps.

## What is Infra Operator?

**Infra Operator** is a Kubernetes operator that allows you to provision and manage AWS resources using Custom Resource Definitions (CRDs). Instead of using separate tools like Terraform or CloudFormation, you can manage your AWS infrastructure using `kubectl` and GitOps tools like ArgoCD.

### Key Benefits

Manage infrastructure alongside applications using Git as the source of truth

Use kubectl, Helm, and familiar Kubernetes ecosystem tools

Well-organized, testable (100% coverage), and maintainable code following architecture patterns

Full support for networking, compute, storage, database, messaging, CDN, security, and more

## Supported Services (26 Total)

### Networking (9 services)

| Service | Description |
|---------|-------------|
| **VPC** | Virtual Private Cloud |
| **Subnet** | VPC Subnets |
| **Internet Gateway** | Internet access for VPC |
| **NAT Gateway** | Outbound internet for private subnets |
| **Security Group** | Firewall rules |
| **Route Table** | Network routing |
| **ALB** | Application Load Balancer (Layer 7) |
| **NLB** | Network Load Balancer (Layer 4) |
| **Elastic IP** | Static public IP addresses |

### Compute (3 services)

| Service | Description |
|---------|-------------|
| **EC2 Instance** | Virtual machines |
| **Lambda** | Serverless functions |
| **EKS** | Kubernetes clusters |

### Storage & Database (3 services)

| Service | Description |
|---------|-------------|
| **S3 Bucket** | Object storage |
| **RDS Instance** | Relational databases (PostgreSQL, MySQL, etc.) |
| **DynamoDB Table** | NoSQL database |

### Messaging (2 services)

| Service | Description |
|---------|-------------|
| **SQS Queue** | Message queues |
| **SNS Topic** | Pub/Sub notifications |

### API & CDN (2 services)

| Service | Description |
|---------|-------------|
| **API Gateway** | REST, HTTP, WebSocket APIs |
| **CloudFront** | Content Delivery Network (CDN) |

### Security (4 services)

| Service | Description |
|---------|-------------|
| **IAM Role** | Identity and access management |
| **Secrets Manager** | Secrets storage |
| **KMS Key** | Encryption keys |
| **ACM Certificate** | SSL/TLS certificates |

### Containers (2 services)

| Service | Description |
|---------|-------------|
| **ECR Repository** | Container registry |
| **ECS Cluster** | Container orchestration |

### Caching (1 service)

| Service | Description |
|---------|-------------|
| **ElastiCache** | In-memory cache (Redis, Memcached) |

## Quick Example

**Example:**

```yaml
---
# Create VPC
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  providerRef:
    name: aws-provider
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true

---
# Create Public Subnet
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet
spec:
  providerRef:
    name: aws-provider
  vpcID: vpc-xxx  # Will be filled automatically
  cidrBlock: "10.0.1.0/24"
  mapPublicIpOnLaunch: true

---
# Create S3 Bucket
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: app-data
spec:
  providerRef:
    name: aws-provider
  bucketName: myapp-production-data
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
```

**Command:**

```bash
kubectl apply -f infrastructure.yaml
```

## Architecture

### System Overview

Infra Operator follows the Kubernetes controller pattern architecture:

**How It Works:**

1. **GitOps / kubectl** - Creates Custom Resources (CRs) in Kubernetes
2. **Controllers** - Detect changes in CRs and reconcile state
3. **AWS SDK** - Controllers use AWS SDK to provision resources
4. **Status Update** - AWS resource state is reflected in the CR

**Main Components:**

- **26 Controllers**: One for each AWS service (VPC, S3, EC2, Lambda, etc.)
- **26 CRDs**: Custom Resource Definitions for each resource type
- **AWS SDK v2**: Communication with AWS APIs
- **Reconciliation Loop**: Ensures desired state = actual state

### Clean Architecture

Implementation following **Clean Architecture** principles for testable (100% coverage), maintainable, and decoupled code:

**Architecture Layers:**

**1. Domain Layer (Core)**
- Pure business models (VPC, S3, EC2, etc.)
- Validation rules
- No external dependencies
- 100% test coverage

**2. Use Cases Layer**
- Application logic
- Create, Update, Delete, GetStatus
- Domain orchestration
- Interface with Ports

**3. Ports Layer (Interfaces)**
- Repository interfaces
- Cloud Provider interfaces
- Dependency inversion principle
- Abstract contracts

**4. Adapters Layer**
- AWS SDK implementations
- Kubernetes client
- Concrete implementations of Ports
- Communication with external systems

**5. Controllers Layer**
- Reconciliation loops
- Kubernetes API integration
- Event handling
- CR to Domain mapping

**Benefits:**
- Testability: 100% coverage in domain
- Maintainability: Clear separation of concerns
- Flexibility: Easy to add new services
- Independence: Core decoupled from frameworks

## Next Steps

- [Installation](/installation) - How to install Infra Operator
- [Quick Start](/quickstart) - Getting started
- [AWS Services](/services/networking/vpc) - Services documentation
- [API Reference](/api-reference/overview) - Complete API reference
