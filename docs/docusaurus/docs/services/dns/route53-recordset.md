---
title: "Route53 Record Set"
description: "Manage DNS records in AWS Route53"
sidebar_position: 2
---

# Route53 Record Set

The `Route53RecordSet` resource allows you to create and manage DNS records (resource record sets) in AWS Route53 using native Kubernetes resources.

## Overview

Route53 Record Sets define how traffic is routed to resources (servers, load balancers, etc.). It supports all standard DNS record types and advanced routing policies.

**Use Cases:**
- Simple DNS records (A, AAAA, CNAME, TXT, MX)
- Alias records for AWS resources (ELB, CloudFront, S3)
- Weight-based, latency-based, or geolocation-based routing
- Automatic failover with health checks
- Declarative DNS management via GitOps

## Supported Record Types

- **A** - IPv4 address
- **AAAA** - IPv6 address
- **CNAME** - Canonical name
- **MX** - Mail exchange
- **TXT** - Text record
- **PTR** - Pointer record
- **SRV** - Service locator
- **SPF** - Sender Policy Framework
- **NS** - Name server
- **SOA** - Start of authority
- **CAA** - Certification Authority Authorization
- **NAPTR** - Name Authority Pointer

## Basic Examples

### Simple A Record

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: www-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: www.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.1
- 192.0.2.2
```

### CNAME Record

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: blog-cname
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: blog.example.com
  type: CNAME
  ttl: 300
  resourceRecords:
- www.example.com
```

### Alias Record (Load Balancer)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: api-alias
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: api.example.com
  type: A
  aliasTarget:
hostedZoneID: Z215JYRZR1TBD5  # ELB hosted zone
dnsName: my-lb-123.us-east-1.elb.amazonaws.com
evaluateTargetHealth: true
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `providerRef` | `ProviderReference` | Yes | Reference to AWSProvider |
| `hostedZoneID` | `string` | Yes | Hosted zone ID |
| `name` | `string` | Yes | Full DNS name (FQDN) |
| `type` | `string` | Yes | Record type (A, AAAA, CNAME, etc.) |
| `ttl` | `*int64` | Conditional | TTL in seconds (required for non-alias) |
| `resourceRecords` | `[]string` | Conditional | Record values (required for non-alias) |
| `aliasTarget` | `*AliasTarget` | Conditional | Target for alias records |
| `setIdentifier` | `string` | Conditional | Identifier for routing policies |
| `weight` | `*int64` | No | Weight (0-255) for weighted routing |
| `region` | `string` | No | Region for latency-based routing |
| `geoLocation` | `*GeoLocation` | No | Location for geolocation routing |
| `failover` | `string` | No | PRIMARY or SECONDARY |
| `multiValueAnswer` | `bool` | No | Enable multivalue answer |
| `healthCheckID` | `string` | No | Health check ID |
| `deletionPolicy` | `string` | No | Delete or Retain (default: Delete) |

## Routing Policies

### 1. Simple Routing (Default)

A single record with one or more values:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: simple-a-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.1
- 192.0.2.2
- 192.0.2.3
```

### 2. Weighted Routing

Distributes traffic based on weights:

```yaml
---
# 70% of traffic
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: app-server-1
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.10
  setIdentifier: "server-1"
  weight: 70

---
# 30% of traffic
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: app-server-2
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.20
  setIdentifier: "server-2"
  weight: 30
```

### 3. Latency-Based Routing

Routes to region with lowest latency:

```yaml
---
# US East
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: global-us-east
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: global.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.10
  setIdentifier: "us-east-1"
  region: us-east-1

---
# EU West
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: global-eu-west
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: global.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.20
  setIdentifier: "eu-west-1"
  region: eu-west-1
```

### 4. Geolocation Routing

Routes based on user location:

```yaml
---
# US users
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: geo-us
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: geo.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.10
  setIdentifier: "us-users"
  geoLocation:
countryCode: US

---
# European users
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: geo-eu
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: geo.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.20
  setIdentifier: "eu-users"
  geoLocation:
continentCode: EU

---
# Default (rest of the world)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: geo-default
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: geo.example.com
  type: A
  ttl: 300
  resourceRecords:
- 192.0.2.30
  setIdentifier: "default-location"
  geoLocation:
continentCode: "*"
```

### 5. Failover Routing

Active/passive configuration:

```yaml
---
# Primary
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: failover-primary
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.10
  setIdentifier: "primary"
  failover: PRIMARY
  healthCheckID: abc123-health-check

---
# Secondary (backup)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: failover-secondary
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.20
  setIdentifier: "secondary"
  failover: SECONDARY
```

### 6. Multivalue Answer Routing

Returns multiple values with health checks:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: multivalue-1
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: multi.example.com
  type: A
  ttl: 60
  resourceRecords:
- 192.0.2.10
  setIdentifier: "server-1"
  multiValueAnswer: true
  healthCheckID: health-check-1
```

## Special Records

### MX (Mail Exchange)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: mx-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: example.com
  type: MX
  ttl: 3600
  resourceRecords:
- 10 mail1.example.com
- 20 mail2.example.com
- 30 mail3.example.com
```

### TXT (SPF, DKIM, DMARC)

**Example:**

```yaml
---
# SPF Record
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: spf-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: example.com
  type: TXT
  ttl: 300
  resourceRecords:
- '"v=spf1 include:_spf.google.com ~all"'

---
# DMARC Record
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: dmarc-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: _dmarc.example.com
  type: TXT
  ttl: 300
  resourceRecords:
- '"v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"'
```

### SRV (Service Records)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: srv-record
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: _service._tcp.example.com
  type: SRV
  ttl: 300
  resourceRecords:
- 10 60 5060 sipserver.example.com
```

## Alias Records for AWS Resources

### Application Load Balancer (ALB)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: alb-alias
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: app.example.com
  type: A
  aliasTarget:
hostedZoneID: Z35SXDOTRQ7X7K  # ALB hosted zone (us-east-1)
dnsName: my-alb-123.us-east-1.elb.amazonaws.com
evaluateTargetHealth: true
```

### CloudFront Distribution

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: cloudfront-alias
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: cdn.example.com
  type: A
  aliasTarget:
hostedZoneID: Z2FDTNDATAQYW2  # CloudFront hosted zone
dnsName: d123abc.cloudfront.net
evaluateTargetHealth: false
```

### S3 Website Endpoint

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Route53RecordSet
metadata:
  name: s3-website-alias
spec:
  providerRef:
    name: aws-provider
  hostedZoneID: Z1234567890ABC
  name: static.example.com
  type: A
  aliasTarget:
hostedZoneID: Z3AQBSTGFYJSTF  # S3 website hosted zone (us-east-1)
dnsName: mybucket.s3-website-us-east-1.amazonaws.com
evaluateTargetHealth: false
```

## Operations

### Create Record

**Command:**

```bash
kubectl apply -f recordset.yaml
```

### Check Status

**Command:**

```bash
# List records
kubectl get route53recordset
kubectl get r53rs  # shortname

# Details
kubectl describe route53recordset www-record

# View change status
kubectl get route53recordset www-record -o jsonpath='{.status.changeStatus}'
```

### Update Record

**Example:**

```yaml
spec:
  ttl: 600  # Update TTL
  resourceRecords:
- 192.0.2.100  # New IP
```

**Command:**

```bash
kubectl apply -f recordset.yaml
```

### Delete Record

**Command:**

```bash
kubectl delete route53recordset www-record
```

## Troubleshooting

### Record Not Ready

**Solutions:**
1. Check hosted zone exists
2. Check IAM permissions
3. Validate record type and values
4. Check controller logs

### Validation Error

**Problem:** Alias target and resource records cannot coexist

**Solution:** Use only one:
```yaml
# Either aliasTarget
spec:
  aliasTarget:
hostedZoneID: Z123
dnsName: lb.example.com

# OR resourceRecords + ttl
spec:
  ttl: 300
  resourceRecords:
- 192.0.2.1
```

### Change Status PENDING

**Normal:** DNS changes can take up to 60 seconds to propagate

**Check:**
```bash
kubectl get route53recordset my-record -w
```

## IAM Permissions

**Example:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "route53:ChangeResourceRecordSets",
        "route53:GetChange",
        "route53:ListResourceRecordSets"
      ],
      "Resource": [
        "arn:aws:route53:::hostedzone/*",
        "arn:aws:route53:::change/*"
      ]
},
{
      "Effect": "Allow",
      "Action": [
        "route53:GetHostedZone",
        "route53:ListHostedZones"
      ],
      "Resource": "*"
}
  ]
}
```

## Best Practices

:::note Best Practices

- **Set appropriate TTL** — Development: 60-300s, Production: 300-3600s, reduce before changes
- **Use alias records for AWS resources** — Free, faster resolution for ALB/CloudFront/S3
- **Implement health checks** — Enable failover routing for high availability
- **Document record purposes** — Clear naming conventions and descriptions
- **Use routing policies wisely** — Simple for basic, weighted for A/B, failover for DR

:::

## AWS Resource Hosted Zone IDs

### ELB/ALB by Region
- us-east-1: Z35SXDOTRQ7X7K
- us-west-2: Z1H1FL5HABSF5
- eu-west-1: Z32O12XQLNTSW2

### CloudFront
- Global: Z2FDTNDATAQYW2

### S3 Website by Region
- us-east-1: Z3AQBSTGFYJSTF
- us-west-2: Z3BJ6K6RIION7M

[Complete list in AWS documentation](https://docs.aws.amazon.com/general/latest/gr/elb.html)

## References

- [Route53 Record Types](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html)
- [Routing Policies](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html)
- [Alias Records](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-choosing-alias-non-alias.html)