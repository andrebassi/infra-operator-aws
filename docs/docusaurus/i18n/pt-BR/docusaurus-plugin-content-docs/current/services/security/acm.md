---
title: 'ACM (Certificate Manager)'
description: 'Manage AWS ACM SSL/TLS certificates with Kubernetes'
sidebar_position: 1
---

## Overview

The **Certificate** resource manages AWS Certificate Manager (ACM) certificates for SSL/TLS encryption.

## Use Cases

- **HTTPS/TLS**: Secure web applications and APIs
- **Load Balancers**: Use with ALB, NLB, CloudFront
- **Domain Validation**: DNS or Email validation
- **Wildcard Certs**: Protect multiple subdomains

## Basic Example

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: my-certificate
  namespace: default
spec:
  providerRef:
    name: aws-provider

  # Primary domain
  domainName: example.com

  # Subject Alternative Names (SANs)
  subjectAlternativeNames:
- www.example.com
- api.example.com

  # Validation method: DNS or EMAIL
  validationMethod: DNS

  # Tags
  tags:
    Environment: production
    Domain: example.com

  # Deletion policy
  deletionPolicy: Delete
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `providerRef` | Object | Yes | Reference to AWSProvider |
| `domainName` | String | Yes | Primary domain name |
| `subjectAlternativeNames` | Array | No | Additional domain names (SANs) |
| `validationMethod` | String | No | `DNS` (default) or `EMAIL` |
| `tags` | Map | No | Key-value tags |
| `deletionPolicy` | String | No | `Delete`, `Retain`, or `Orphan` |

## Status Fields

| Field | Description |
|-------|-------------|
| `ready` | Boolean indicating if cert is issued |
| `certificateARN` | AWS ARN of the certificate |
| `status` | Certificate status (PENDING_VALIDATION, ISSUED, FAILED) |
| `validationRecords` | DNS validation records for Route53 |

## Validation Methods

### DNS Validation (Recommended)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: dns-validated-cert
spec:
  providerRef:
    name: aws-provider
  domainName: example.com
  validationMethod: DNS
```

**Validation Records:**
After creation, check status for DNS records:

```bash
kubectl get certificate dns-validated-cert -o yaml
```

Output:
```yaml
status:
  validationRecords:
- domainName: example.com
      resourceRecordName: _abc123.example.com
      resourceRecordType: CNAME
      resourceRecordValue: _xyz789.acm-validations.aws
```

Add this CNAME to your DNS (Route53):
```yaml
apiVersion: route53.aws.upbound.io/v1beta1
kind: Record
metadata:
  name: acm-validation
spec:
  name: _abc123.example.com
  type: CNAME
  ttl: 300
  records:
- _xyz789.acm-validations.aws
```

### Email Validation

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: email-validated-cert
spec:
  providerRef:
    name: aws-provider
  domainName: example.com
  validationMethod: EMAIL
```

AWS will send validation emails to:
- admin@example.com
- administrator@example.com
- hostmaster@example.com
- postmaster@example.com
- webmaster@example.com

## Advanced Examples

### Wildcard Certificate

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: wildcard-cert
spec:
  providerRef:
    name: aws-provider
  domainName: "*.example.com"
  subjectAlternativeNames:
- example.com  # Include apex domain
  validationMethod: DNS
  tags:
    Type: wildcard
```

### Multi-Domain Certificate

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: multi-domain-cert
spec:
  providerRef:
    name: aws-provider
  domainName: example.com
  subjectAlternativeNames:
- www.example.com
- api.example.com
- app.example.com
- "*.example.com"
  validationMethod: DNS
```

### Certificate for CloudFront

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: cloudfront-cert
spec:
  providerRef:
    name: aws-provider-us-east-1  # Must be us-east-1 for CloudFront!
  domainName: cdn.example.com
  validationMethod: DNS
  tags:
    Service: cloudfront
```

**Important**: CloudFront certificates MUST be in us-east-1 region!

## Certificate States

| State | Description |
|-------|-------------|
| `PENDING_VALIDATION` | Awaiting domain validation |
| `ISSUED` | Certificate successfully issued |
| `INACTIVE` | Certificate not in use |
| `EXPIRED` | Certificate has expired |
| `VALIDATION_TIMED_OUT` | Validation took too long |
| `REVOKED` | Certificate was revoked |
| `FAILED` | Issuance failed |

## Using Certificates

### With ALB

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: https-alb
spec:
  providerRef:
    name: aws-provider
  loadBalancerName: my-alb
  subnets:
- subnet-abc123
- subnet-def456
  listeners:
- protocol: HTTPS
      port: 443
      certificateArn: arn:aws:acm:us-east-1:123456789012:certificate/abc-123  # From Certificate status
      defaultActions:
        - type: forward
          targetGroupArn: arn:aws:elasticloadbalancing:...
```

### With CloudFront

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: CloudFront
metadata:
  name: my-distribution
spec:
  providerRef:
    name: aws-provider
  aliases:
- cdn.example.com
  viewerCertificate:
acmCertificateArn: arn:aws:acm:us-east-1:123456789012:certificate/abc-123
minimumProtocolVersion: TLSv1.2_2021
sslSupportMethod: sni-only
```

## Automatic Renewal

ACM certificates auto-renew if:
- Domain validation still works
- Certificate is in use
- Domain is still owned

**No action required** for DNS-validated certs with Route53!

## Monitoring

Check certificate status:

```bash
kubectl get certificate my-certificate -o yaml
```

Example output:
```yaml
status:
  ready: true
  certificateARN: arn:aws:acm:us-east-1:123456789012:certificate/abc-123
  status: ISSUED
  validationRecords:
- domainName: example.com
      resourceRecordName: _abc.example.com
      resourceRecordType: CNAME
      resourceRecordValue: _xyz.acm-validations.aws
```

## Best Practices

:::note Best Practices

- **Use DNS validation over email** — Faster, automated, no manual approval needed
- **Request certificates before deployment** — Allow time for validation and propagation
- **Enable certificate transparency logging** — Required for public certificates
- **Monitor certificate expiration** — Set alerts 30-60 days before expiry
- **Use wildcard certificates wisely** — Reduces management overhead but increases blast radius

:::

## Troubleshooting

### Certificate Stuck in PENDING_VALIDATION

For DNS validation:
1. Check validation records in status
2. Verify CNAME records in DNS
3. Wait up to 72 hours for validation

**Command:**

```bash
kubectl get certificate my-cert -o jsonpath='{.status.validationRecords}'
```

### Validation Failed

Check:
- Domain ownership (WHOIS)
- DNS propagation (dig/nslookup)
- Email deliverability (for EMAIL validation)

### Certificate Not Auto-Renewing

Ensure:
- Certificate is in use (attached to resource)
- DNS validation records still exist
- Domain still owned

## Complete Example

**Example:**

```yaml
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
metadata:
  name: production-cert
  namespace: production
spec:
  providerRef:
    name: aws-provider

  # Primary domain
  domainName: example.com

  # Additional domains
  subjectAlternativeNames:
- www.example.com
- api.example.com
- "*.example.com"

  # DNS validation for automation
  validationMethod: DNS

  # Tags for organization
  tags:
    Environment: production
    ManagedBy: infra-operator
    CostCenter: platform

  # Retain cert if CR deleted
  deletionPolicy: Retain
```

## Related Resources

- [ALB](/services/networking/alb)
- [CloudFront](/services/networking/cloudfront)
- [Route53 (External)](/guides/external-dns)

## AWS Documentation

- [AWS Certificate Manager](https://docs.aws.amazon.com/acm/)
- [DNS Validation](https://docs.aws.amazon.com/acm/latest/userguide/dns-validation.html)
- [Certificate Renewal](https://docs.aws.amazon.com/acm/latest/userguide/managed-renewal.html)