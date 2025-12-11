---
title: 'Elastic IP'
description: 'Manage AWS Elastic IP addresses with Kubernetes'
sidebar_position: 2
---

## Overview

The **ElasticIP** resource manages AWS Elastic IP (EIP) addresses, providing static IPv4 addresses for your VPC resources.

## Use Cases

- **Static Public IPs**: Assign persistent public IP addresses to instances
- **NAT Gateway IPs**: Allocate IPs for NAT Gateways
- **Load Balancer IPs**: Use for Network Load Balancers
- **Failover**: Quickly remap IPs between instances

## Basic Example

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
metadata:
  name: my-elastic-ip
  namespace: default
spec:
  providerRef:
    name: aws-provider

  # Domain type (vpc or standard)
  domain: vpc

  # Tags
  tags:
    Environment: production
    ManagedBy: infra-operator

  # Deletion policy
  deletionPolicy: Delete
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `providerRef` | Object | Yes | Reference to AWSProvider |
| `domain` | String | No | Domain type: `vpc` (default) or `standard` |
| `networkBorderGroup` | String | No | Network border group for the IP |
| `publicIpv4Pool` | String | No | Public IPv4 pool to allocate from |
| `tags` | Map | No | Key-value tags |
| `deletionPolicy` | String | No | `Delete` (default), `Retain`, or `Orphan` |

## Status Fields

| Field | Description |
|-------|-------------|
| `ready` | Boolean indicating if EIP is allocated |
| `allocationId` | AWS allocation ID |
| `publicIp` | The allocated public IP address |
| `associationId` | Association ID if attached to instance |
| `privateIpAddress` | Associated private IP |
| `networkInterfaceId` | Associated network interface |
| `instanceId` | Associated EC2 instance ID |

## Advanced Examples

### With Network Border Group

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
metadata:
  name: regional-eip
spec:
  providerRef:
    name: aws-provider
  domain: vpc
  networkBorderGroup: us-east-1-wl1-bos-wlz-1
  tags:
    Region: us-east-1
    Zone: wavelength
```

### With Custom IPv4 Pool

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
metadata:
  name: byoip-eip
spec:
  providerRef:
    name: aws-provider
  domain: vpc
  publicIpv4Pool: ipv4pool-ec2-012345abcde67890f
  tags:
    Pool: custom-byoip
```

### Retain on Deletion

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
metadata:
  name: persistent-eip
spec:
  providerRef:
    name: aws-provider
  domain: vpc
  deletionPolicy: Retain  # Don't delete EIP when CR is deleted
  tags:
    Lifecycle: retain
```

## Domain Types

### VPC Domain (Default)
- For use with VPC instances
- Modern AWS accounts
- Supports all EC2-VPC features

**Example:**

```yaml
domain: vpc
```

### Standard Domain (Classic)
- For EC2-Classic (legacy)
- Rare use case
- Not recommended for new deployments

**Example:**

```yaml
domain: standard
```

## Association

Elastic IPs are automatically associated with resources when specified in their configuration:

### With NAT Gateway

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: my-nat
spec:
  providerRef:
    name: aws-provider
  subnetId: subnet-12345
  allocationId: eipalloc-67890  # From ElasticIP status
```

### With Network Load Balancer

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NLB
metadata:
  name: my-nlb
spec:
  providerRef:
    name: aws-provider
  loadBalancerName: my-nlb
  subnets:
- subnet-12345
  subnetMappings:
- subnetId: subnet-12345
      allocationId: eipalloc-67890  # From ElasticIP status
```

## Deletion Policies

### Delete (Default)
Release the Elastic IP when CR is deleted:
```yaml
deletionPolicy: Delete
```

### Retain
Keep the EIP in AWS but remove CR:
```yaml
deletionPolicy: Retain
```

### Orphan
Remove CR but leave EIP running:
```yaml
deletionPolicy: Orphan
```

## Monitoring

Check Elastic IP status:

```bash
kubectl get elasticip my-elastic-ip -o yaml
```

Example output:
```yaml
status:
  ready: true
  allocationId: eipalloc-0123456789abcdef0
  publicIp: 54.123.45.67
  domain: vpc
  lastSyncTime: "2025-01-23T10:30:00Z"
```

## Best Practices

:::note Best Practices

- **Release unused EIPs** — AWS charges for unattached Elastic IPs (~$3.60/month)
- **Associate EIP before NAT Gateway** — NAT requires EIP for internet connectivity
- **Use consistent naming** — Include purpose (nat, bastion, api) in Name tag
- **Plan IP allocation** — Account for HA (multiple NATs) and future growth
- **Document IP associations** — Track which EIPs are used for what purpose

:::

## Troubleshooting

### EIP Not Allocating

Check provider credentials:
```bash
kubectl describe awsprovider aws-provider
```

Check operator logs:
```bash
kubectl logs -n infra-operator-system -l control-plane=controller-manager
```

### Address Limit Exceeded

Request limit increase:
- AWS Service Quotas Console
- Default limit: 5 EIPs per region
- Can be increased to 100+

### EIP Stuck in Pending

Check CloudFormation limits or VPC quotas:
```bash
kubectl get elasticip my-elastic-ip -o jsonpath='{.status.conditions}'
```

## Related Resources

- [NAT Gateway](/services/networking/nat-gateway)
- [NLB](/services/networking/nlb)
- [VPC](/services/networking/vpc)

## AWS Documentation

- [Elastic IP Addresses](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html)
- [EIP Pricing](https://aws.amazon.com/ec2/pricing/on-demand/#Elastic_IP_Addresses)