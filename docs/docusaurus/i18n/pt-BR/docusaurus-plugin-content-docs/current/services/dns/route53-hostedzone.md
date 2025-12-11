---
title: "Route53 Hosted Zone"
description: "Manage public and private DNS zones in AWS Route53"
sidebar_position: 1
---

# Route53 Hosted Zone

The `Route53HostedZone` resource allows you to create and manage DNS zones (hosted zones) in AWS Route53 using native Kubernetes resources.

## Overview

Route53 Hosted Zones are containers for DNS records that define how to route traffic to a domain and its subdomains. It supports both public zones (for internet) and private zones (for VPCs).

**Use Cases:**
- Manage public DNS for web applications
- Configure private DNS for internal communication in VPCs
- Migrate DNS management to GitOps
- Automate DNS zone creation for ephemeral environments

## Basic Example

### Public Zone

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53HostedZone
metadata:
  name: example-public-zone
  namespace: default
spec:
  providerRef:
    name: aws-provider
  name: example.com
  comment: "Production public DNS zone"
  privateZone: false
  tags:
    Environment: production
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

### Private Zone (VPC)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53HostedZone
metadata:
  name: internal-zone
  namespace: default
spec:
  providerRef:
    name: aws-provider
  name: internal.example.com
  comment: "Private DNS for VPC"
  privateZone: true
  vpcId: vpc-0123456789abcdef0
  vpcRegion: us-east-1
  tags:
    Environment: production
    Type: private
  deletionPolicy: Retain
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `providerRef` | `ProviderReference` | Yes | Reference to AWSProvider |
| `name` | `string` | Yes | Domain name (e.g., example.com) |
| `comment` | `string` | No | Comment about the zone |
| `privateZone` | `bool` | No | If true, creates private zone (default: false) |
| `vpcId` | `string` | Conditional | VPC ID (required if privateZone=true) |
| `vpcRegion` | `string` | Conditional | VPC region (required if privateZone=true) |
| `tags` | `map[string]string` | No | AWS tags for the zone |
| `deletionPolicy` | `string` | No | Delete or Retain (default: Delete) |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | `bool` | Indicates if the zone was created |
| `hostedZoneID` | `string` | Hosted zone ID in Route53 |
| `nameServers` | `[]string` | Authoritative name servers |
| `resourceRecordSetCount` | `int64` | Number of record sets in the zone |
| `lastSyncTime` | `*metav1.Time` | Last synchronization time |

## Advanced Examples

### Multi-VPC Private Zone

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53HostedZone
metadata:
  name: shared-internal-zone
spec:
  providerRef:
    name: aws-provider
  name: shared.internal
  comment: "Shared private zone across VPCs"
  privateZone: true
  vpcId: vpc-primary123
  vpcRegion: us-east-1
  tags:
    Shared: "true"
    CostCenter: engineering
```

### Development Zone with Retain Policy

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53HostedZone
metadata:
  name: dev-zone
spec:
  providerRef:
    name: aws-provider
  name: dev.example.com
  comment: "Development environment DNS"
  deletionPolicy: Retain  # Preserva zona ao deletar CR
  tags:
    Environment: development
    AutoDelete: "false"
```

## Validations

### Private Zone
- If `privateZone: true`, the fields `vpcId` and `vpcRegion` are **required**
- VPC must exist in the specified region

### Public Zone
- If `privateZone: false`, cannot specify `vpcId` or `vpcRegion`

### Domain Name
- Must be a valid FQDN (Fully Qualified Domain Name)
- Cannot be empty

## Operations

### Create Zone

**Command:**

```bash
kubectl apply -f hostedzone.yaml
```

### Check Status

**Command:**

```bash
# List zones
kubectl get route53hostedzone
kubectl get r53hz  # shortname

# Zone details
kubectl describe route53hostedzone example-public-zone

# View name servers
kubectl get route53hostedzone example-public-zone -o jsonpath='{.status.nameServers}'
```

### Update Tags

**Example:**

```yaml
spec:
  tags:
    Environment: production
    CostCenter: platform
    Team: sre
```

**Command:**

```bash
kubectl apply -f hostedzone.yaml
```

### Delete Zone

**Command:**

```bash
# With deletionPolicy: Delete (default)
kubectl delete route53hostedzone example-public-zone

# With deletionPolicy: Retain
# Zone is preserved in Route53
kubectl delete route53hostedzone example-public-zone
```

## Deletion Policies

### Delete (Default)

**Example:**

```yaml
spec:
  deletionPolicy: Delete
```
- Deletes the hosted zone in Route53 when the CR is removed
- **Warning:** Zone must be empty (no record sets except NS and SOA)

### Retain

**Example:**

```yaml
spec:
  deletionPolicy: Retain
```
- Preserves the hosted zone in Route53
- Only removes the CR from Kubernetes
- Useful for production environments

## Integration with Record Sets

After creating a hosted zone, use the `hostedZoneID` from the status to create record sets:

```yaml
# 1. Create hosted zone
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53HostedZone
metadata:
  name: my-zone
spec:
  providerRef:
    name: aws-provider
  name: example.com

---
# 2. Get hostedZoneID from status
# kubectl get route53hostedzone my-zone -o jsonpath='{.status.hostedZoneID}'

# 3. Create record set
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: www-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC  # From status above
  name: www.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.1
```

## Troubleshooting

### Zone Not Ready

**Problem:** `status.ready: false`

**Solutions:**
1. Check IAM permissions:
   ```
   route53:CreateHostedZone
   route53:GetHostedZone
   route53:ListHostedZones
   route53:ChangeTagsForResource
   ```

2. Check controller logs:
   ```bash
   kubectl logs -n infra-operator-system deployment/infra-operator-controller-manager
   ```

3. Check AWSProvider:
   ```bash
   kubectl get awsprovider -n default
   ```

### Error Deleting Zone

**Problem:** Error deleting hosted zone

**Cause:** Zone contains record sets beyond NS and SOA

**Solution:**
```bash
# Delete all record sets first
kubectl delete route53recordset -l hostedZone=my-zone

# Then delete the zone
kubectl delete route53hostedzone my-zone
```

### VPC Not Found

**Problem:** "VPC not found" error for private zone

**Solutions:**
1. Check if VPC exists:
   ```bash
   aws ec2 describe-vpcs --vpc-ids vpc-xxxxx --region us-east-1
   ```

2. Verify correct region in spec

3. Check EC2 permissions:
   ```
   ec2:DescribeVpcs
   ec2:DescribeVpcAttribute
   ```

## Required IAM Permissions

**Example:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "route53:CreateHostedZone",
        "route53:GetHostedZone",
        "route53:DeleteHostedZone",
        "route53:ListHostedZones",
        "route53:UpdateHostedZoneComment",
        "route53:ChangeTagsForResource",
        "route53:ListTagsForResource"
      ],
      "Resource": "*"
},
{
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVpcs",
        "ec2:DescribeVpcAttribute"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": ["us-east-1"]
        }
      }
}
  ]
}
```

## Best Practices

:::note Best Practices

- **One private zone per VPC or group of related VPCs**
- **Avoid sharing private zones between environments**

:::

## References

- [AWS Route53 Hosted Zones](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html)
- [Private Hosted Zones](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-private.html)
- [Route53 Pricing](https://aws.amazon.com/route53/pricing/)