---
title: 'ComputeStack - All-in-One Infrastructure'
description: 'Create complete VPC + EC2 infrastructure with a single resource'
sidebar_position: 4
---

# ComputeStack

Create a complete infrastructure stack (VPC, Subnet, Internet Gateway, Security Group, EC2 Bastion) with a single Kubernetes resource.

## Overview

ComputeStack is a high-level CRD that orchestrates the creation of multiple AWS resources:

- **VPC** with configurable CIDR
- **Public Subnet** (auto-generated if not specified)
- **Internet Gateway** + Route Table
- **Security Group** for SSH access
- **EC2 Key Pair** (auto-generated if not specified)
- **Kubernetes Secret** with SSH private key
- **EC2 Bastion Instance** with public IP

## Quick Start

### Minimal Configuration

**title="computestack-minimal.yaml":**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: my-stack
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  vpcCIDR: "10.200.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.micro
```

This creates:
- VPC: `10.200.0.0/16`
- Subnet: `10.200.1.0/24` (auto-generated)
- Key Pair: `my-stack-key` (auto-generated)
- Secret: `my-stack-ssh-key` (contains private key)
- EC2 Instance: t3.micro with public IP

### With Cloud-Init

**title="computestack-cloudinit.yaml":**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: docker-stack
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  vpcCIDR: "10.200.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.small
    userData: |
      #!/bin/bash
      yum update -y
      yum install -y docker
      systemctl start docker
      systemctl enable docker
      usermod -aG docker ec2-user
```

### With Existing Key Pair

**title="computestack-existing-key.yaml":**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: existing-key-stack
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  vpcCIDR: "10.200.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.micro
    keyName: my-existing-aws-keypair  # Use existing AWS key
```

## Specification

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `spec.providerRef.name` | string | AWSProvider reference |
| `spec.vpcCIDR` | string | VPC CIDR block (e.g., "10.200.0.0/16") |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `spec.publicSubnets` | array | auto | List of public subnet CIDRs |
| `spec.privateSubnets` | array | - | List of private subnet CIDRs |
| `spec.availabilityZones` | array | auto | AZs to use |
| `spec.tags` | object | - | Tags for all resources |
| `spec.deletionPolicy` | string | Delete | Delete, Retain, or Orphan |

### Bastion Instance Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `spec.bastionInstance.enabled` | boolean | false | Enable bastion instance |
| `spec.bastionInstance.instanceType` | string | t3.micro | EC2 instance type |
| `spec.bastionInstance.keyName` | string | auto | SSH key pair name |
| `spec.bastionInstance.userData` | string | - | Cloud-init script |
| `spec.bastionInstance.userDataSecretRef` | object | - | Secret reference for userData |
| `spec.bastionInstance.ami` | string | auto | AMI ID (defaults to latest Amazon Linux 2) |

### UserData from Secret

For sensitive scripts, use a Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bastion-init-script
  namespace: infra-operator
type: Opaque
stringData:
  script: |
    #!/bin/bash
    # Sensitive initialization script
    echo "secret-token" > /etc/myapp/token
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ComputeStack
metadata:
  name: secure-stack
spec:
  providerRef:
    name: aws-production
  vpcCIDR: "10.200.0.0/16"
  bastionInstance:
    enabled: true
    instanceType: t3.micro
    userDataSecretRef:
      name: bastion-init-script
      key: script
```

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `status.vpcID` | string | Created VPC ID |
| `status.subnetIDs` | array | Created subnet IDs |
| `status.internetGatewayID` | string | Created IGW ID |
| `status.securityGroupID` | string | Created SG ID |
| `status.bastionInstance.instanceID` | string | EC2 instance ID |
| `status.bastionInstance.publicIP` | string | Public IP address |
| `status.bastionInstance.privateIP` | string | Private IP address |
| `status.bastionInstance.keyPairName` | string | Key pair name used |
| `status.bastionInstance.sshSecretName` | string | Secret with SSH key |
| `status.ready` | boolean | All resources ready |

## Auto-Generation Features

### Auto-Subnet

If `publicSubnets` is not specified, ComputeStack automatically creates a subnet:

- VPC CIDR: `10.200.0.0/16` → Subnet: `10.200.1.0/24`
- VPC CIDR: `10.100.0.0/16` → Subnet: `10.100.1.0/24`

### Auto-KeyPair

If `keyName` is not specified:

1. Creates EC2 Key Pair: `{stack-name}-key`
2. Stores private key in Secret: `{stack-name}-ssh-key`
3. Secret contains:
   - `private-key`: PEM-encoded private key
   - `key-pair-name`: Name of the key pair

## Usage Examples

### Deploy and Connect

**Command:**

```bash
# 1. Deploy ComputeStack
kubectl apply -f computestack.yaml

# 2. Wait for ready
kubectl get computestack my-stack -n infra-operator -w

# 3. Get SSH key
kubectl get secret my-stack-ssh-key -n infra-operator \
  -o jsonpath='{.data.private-key}' | base64 -d > my-stack.pem
chmod 600 my-stack.pem

# 4. Get public IP
BASTION_IP=$(kubectl get computestack my-stack -n infra-operator \
  -o jsonpath='{.status.bastionInstance.publicIP}')

# 5. SSH to bastion
ssh -i my-stack.pem ec2-user@$BASTION_IP
```

### Verify Resources

**Command:**

```bash
# Check all resources
kubectl describe computestack my-stack -n infra-operator

# Verify VPC
aws ec2 describe-vpcs --vpc-ids $(kubectl get computestack my-stack \
  -o jsonpath='{.status.vpcID}')

# Verify instance
aws ec2 describe-instances --instance-ids $(kubectl get computestack my-stack \
  -o jsonpath='{.status.bastionInstance.instanceID}')
```

## Cloud-Init Examples

### Install Docker + Docker Compose

**Example:**

```yaml
userData: |
  #!/bin/bash
  yum update -y
  yum install -y docker git
  systemctl start docker
  systemctl enable docker
  usermod -aG docker ec2-user

  # Install Docker Compose
  curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
```

### Install Kubernetes Tools

**Example:**

```yaml
userData: |
  #!/bin/bash
  yum update -y

  # Install kubectl
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

  # Install Helm
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

  # Install AWS CLI v2
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
  unzip awscliv2.zip
  ./aws/install
```

### Install Monitoring Tools

**Example:**

```yaml
userData: |
  #!/bin/bash
  yum update -y

  # Install neofetch for system info
  yum install -y epel-release
  yum install -y neofetch htop

  # Display on login
  echo 'neofetch' >> /home/ec2-user/.bashrc
```

## Cleanup

**Command:**

```bash
# Delete ComputeStack (deletes all resources)
kubectl delete computestack my-stack -n infra-operator

# Verify cleanup
aws ec2 describe-vpcs --filters "Name=tag:Name,Values=my-stack*"
```

## Troubleshooting

### Instance Not Starting

**Command:**

```bash
# Check ComputeStack status
kubectl describe computestack my-stack

# Check operator logs
kubectl logs -n infra-operator deploy/infra-operator --tail=100

# Check EC2 console for instance errors
aws ec2 describe-instances --instance-ids i-xxx --query 'Reservations[].Instances[].StateReason'
```

### SSH Key Not Found

**Command:**

```bash
# Verify Secret exists
kubectl get secret my-stack-ssh-key -n infra-operator

# Check Secret contents
kubectl get secret my-stack-ssh-key -n infra-operator -o yaml
```

### Cloud-Init Not Running

**Command:**

```bash
# SSH to instance and check logs
ssh -i my-stack.pem ec2-user@$BASTION_IP

# Check cloud-init logs
sudo cat /var/log/cloud-init.log
sudo cat /var/log/cloud-init-output.log
```

## Best Practices

:::note Best Practices

- **Use private subnets for production workloads** — Keep production instances isolated from internet
- **Restrict Security Group to specific IPs** — Never allow 0.0.0.0/0 for SSH access
- **Use userDataSecretRef for sensitive scripts** — Keep credentials and tokens out of plain YAML
- **Rotate SSH keys regularly** — Generate new key pairs periodically for security
- **Use t3.micro for bastion** — Free tier eligible, sufficient for SSH jump host
- **Enable auto-shutdown for dev environments** — Save costs by stopping instances when not in use
- **Delete stacks when not in use** — Clean up development stacks to avoid unnecessary charges
- **Plan subnet CIDR allocation** — Use consistent patterns like 10.0.1.0/24 (us-east-1a), 10.0.2.0/24 (us-east-1b)

:::
