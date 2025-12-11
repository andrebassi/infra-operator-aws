---
title: 'ALB - Application Load Balancer'
description: 'Distribute HTTP/HTTPS traffic across multiple targets'
sidebar_position: 1
---

Intelligently distribute HTTP/HTTPS traffic across multiple targets with content-based routing.

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

**IAM Policy - ALB (alb-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:CreateLoadBalancer",
        "elasticloadbalancing:DeleteLoadBalancer",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeLoadBalancerAttributes",
        "elasticloadbalancing:ModifyLoadBalancerAttributes",
        "elasticloadbalancing:DescribeTags",
        "elasticloadbalancing:AddTags",
        "elasticloadbalancing:RemoveTags"
      ],
      "Resource": "*"
}
  ]
}
```

**Create Role:**

```bash
# Create IAM Role
aws iam create-role \
  --role-name infra-operator-alb-role \
  --assume-role-policy-document file://trust-policy.json

# Attach policy
aws iam put-role-policy \
  --role-name infra-operator-alb-role \
  --policy-name alb-policy \
  --policy-document file://alb-policy.json

# Annotate Service Account in K8s
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-alb-role
```
## Creating Application Load Balancer

**Internet-Facing ALB:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: public-alb
  namespace: default
spec:
  providerRef:
    name: production-aws

  loadBalancerName: my-public-alb
  scheme: internet-facing  # Accessible from internet

  subnets:
- subnet-abc123  # Public subnet AZ 1
- subnet-def456  # Public subnet AZ 2

  securityGroups:
- sg-web123  # Security group allowing 80/443

  ipAddressType: ipv4
  enableHttp2: true
  idleTimeout: 60

  enableDeletionProtection: false

  tags:
    Name: public-alb
    Environment: production
    Team: platform
```

**Internal ALB:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: internal-alb
  namespace: default
spec:
  providerRef:
    name: production-aws

  loadBalancerName: my-internal-alb
  scheme: internal  # Accessible only within VPC

  subnets:
- subnet-private1  # Private subnet AZ 1
- subnet-private2  # Private subnet AZ 2

  securityGroups:
- sg-internal123

  ipAddressType: ipv4
  enableHttp2: true
  idleTimeout: 120

  enableDeletionProtection: false

  tags:
    Name: internal-alb
    Environment: production
    Purpose: microservices
```

**ALB with WAF Protection:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: waf-protected-alb
  namespace: default
spec:
  providerRef:
    name: production-aws

  loadBalancerName: waf-alb
  scheme: internet-facing

  subnets:
- subnet-abc123
- subnet-def456

  securityGroups:
- sg-waf123

  enableHttp2: true
  enableWafFailOpen: true  # Allow traffic if WAF is unavailable

  enableDeletionProtection: true  # Protection against accidental deletion

  deletionPolicy: Retain  # Do not delete ALB when removing CR

  tags:
    Name: waf-protected-alb
    Environment: production
    Protected: "true"
```

**Verify Status:**

```bash
# List ALBs
kubectl get alb

# View details
kubectl describe alb public-alb

# View ALB DNS (use this DNS to access)
kubectl get alb public-alb -o jsonpath='{.status.dnsName}'

# View ALB ARN
kubectl get alb public-alb -o jsonpath='{.status.loadBalancerARN}'
```
## Specification Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `loadBalancerName` | string | ✅ | Application Load Balancer name (1-32 alphanumeric characters) |
| `scheme` | string | ❌ | ALB scheme: `internet-facing` (default) or `internal` |
| `subnets` | []string | ✅ | List of subnet IDs (minimum 2 in different AZs) |
| `securityGroups` | []string | ❌ | List of security group IDs |
| `ipAddressType` | string | ❌ | IP address type: `ipv4` (default) or `dualstack` (IPv4 + IPv6) |
| `enableDeletionProtection` | bool | ❌ | Protection against accidental deletion (default: false) |
| `enableHttp2` | bool | ❌ | Enable HTTP/2 (default: true) |
| `enableWafFailOpen` | bool | ❌ | Allow traffic if WAF is unavailable (default: false) |
| `idleTimeout` | int32 | ❌ | Idle connection timeout in seconds (1-4000, default: 60) |
| `tags` | map[string]string | ❌ | Custom tags for ALB |
| `deletionPolicy` | string | ❌ | Deletion policy: `Delete` (default) or `Retain` |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | bool | Whether the ALB is active (`state` = "active") |
| `loadBalancerARN` | string | ARN of the created ALB |
| `dnsName` | string | Public DNS of the ALB (use to access) |
| `state` | string | State: `provisioning`, `active`, `failed` |
| `vpcID` | string | VPC where the ALB was created |
| `canonicalHostedZoneID` | string | Hosted Zone ID for Route53 |
| `lastSyncTime` | time | Last synchronization with AWS |

## Use Cases

### Public ALB for Web Application

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: web-app-alb
  namespace: production
spec:
  providerRef:
    name: production-aws

  loadBalancerName: webapp-alb
  scheme: internet-facing

  subnets:
- subnet-public-1a
- subnet-public-1b
- subnet-public-1c

  securityGroups:
- sg-web-https

  enableHttp2: true
  idleTimeout: 60

  tags:
    Application: web-app
    Environment: production
    CostCenter: engineering
```

### Internal ALB for Microservices

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: microservices-alb
  namespace: backend
spec:
  providerRef:
    name: production-aws

  loadBalancerName: internal-services
  scheme: internal

  subnets:
- subnet-private-1a
- subnet-private-1b

  securityGroups:
- sg-internal-services

  enableHttp2: true
  idleTimeout: 300  # 5 minutes for long-polling

  tags:
    Purpose: microservices-mesh
    Environment: production
```

## Troubleshooting

### ALB Not Getting Ready

### Check ALB state

**Command:**

```bash
kubectl describe alb my-alb
```

Look for:
- `State: provisioning` - ALB is still being provisioned (can take 2-3 minutes)
- `State: failed` - Provisioning failed (check operator logs)

### Check subnets

**Command:**

```bash
# Check if subnets exist and are in different AZs
aws ec2 describe-subnets --subnet-ids subnet-abc123 subnet-def456

# ALB requires at least 2 subnets in different AZs
```

**Common error**: Subnets in the same AZ
**Solution**: Use subnets in different AZs

### Check security groups

**Command:**

```bash
# Check if security groups exist
aws ec2 describe-security-groups --group-ids sg-web123

# Check ingress rules
aws ec2 describe-security-groups --group-ids sg-web123 \
  --query 'SecurityGroups[0].IpPermissions'
```

**Common error**: Security group without ingress rules
**Solution**: Add rules allowing ports 80/443

### Check IAM permissions

**Command:**

```bash
# View operator logs
kubectl logs -n infra-operator-system -l control-plane=controller-manager

# Look for "AccessDenied" errors
```

**Common error**: IAM role without ELBv2 permissions
**Solution**: Add `elasticloadbalancing:*` policy to IRSA role

### ALB Stuck in "provisioning"

### Normal provisioning time

ALB typically takes 2-5 minutes to become active.

**Command:**

```bash
# Check state in AWS
aws elbv2 describe-load-balancers \
  --names my-public-alb \
  --query 'LoadBalancers[0].State'
```

If stuck for more than 10 minutes, check:
- Subnets have connectivity to AWS API
- No ALB limit reached on AWS account

### Check account limits

**Command:**

```bash
# Check ALB quota
aws service-quotas get-service-quota \
  --service-code elasticloadbalancing \
  --quota-code L-53DA6B97

# List existing ALBs
aws elbv2 describe-load-balancers --query 'length(LoadBalancers)'
```

If limit reached (default: 50 ALBs per region):
- Request quota increase via AWS Console
- Delete unused ALBs

### Error Deleting ALB

### Deletion protection enabled

**Command:**

```bash
kubectl get alb my-alb -o yaml | grep deletionProtection
```

If `enableDeletionProtection: true`:
1. Disable protection:
```bash
kubectl patch alb my-alb --type=merge -p '{"spec":{"enableDeletionProtection":false}}'
```

2. Wait for reconciliation (1-2 minutes)

3. Delete again:
```bash
kubectl delete alb my-alb
```

### ALB with listeners/target groups

ALB cannot be deleted if it has associated listeners or target groups.

**Command:**

```bash
# List listeners
aws elbv2 describe-listeners --load-balancer-arn <ALB-ARN>

# List target groups
aws elbv2 describe-target-groups --load-balancer-arn <ALB-ARN>
```

**Solution**: Delete listeners and target groups manually first:
```bash
# Delete listeners
aws elbv2 delete-listener --listener-arn <LISTENER-ARN>

# Delete target groups
aws elbv2 delete-target-group --target-group-arn <TG-ARN>

# Now delete the ALB
kubectl delete alb my-alb
```

## Deletion Policies

### Delete (Default)

When the CR is deleted, the ALB in AWS is automatically deleted:

```yaml
spec:
  deletionPolicy: Delete  # Default
```

### Retain

The ALB in AWS is retained even after deleting the CR:

```yaml
spec:
  deletionPolicy: Retain
```

**Use case**: ALBs with complex configurations (listeners, rules, certificates) that you want to keep.

:::warning

With `deletionPolicy: Retain`, the ALB continues to be charged even after deleting the CR.

:::

## Advanced Examples

### ALB with IPv6 (Dual Stack)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: ipv6-alb
spec:
  providerRef:
    name: production-aws

  loadBalancerName: dual-stack-alb
  scheme: internet-facing

  subnets:
- subnet-abc123
- subnet-def456

  ipAddressType: dualstack  # IPv4 + IPv6

  enableHttp2: true
```

### ALB with Custom Timeout

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
metadata:
  name: long-polling-alb
spec:
  providerRef:
    name: production-aws

  loadBalancerName: websocket-alb

  subnets:
- subnet-abc123
- subnet-def456

  idleTimeout: 3600  # 1 hour for WebSockets

  tags:
    Purpose: websocket-connections
```

## Next Steps

After creating the ALB:

1. **Configure Listeners** (HTTP/HTTPS) via AWS Console or Terraform
2. **Create Target Groups** pointing to your Kubernetes pods
3. **Configure Rules** for path/host-based routing
4. **Add SSL Certificates** via ACM
5. **Configure WAF** for protection against attacks

:::info

The operator only manages the ALB. Listeners, target groups, and rules must be configured separately.

:::

## References

- [AWS ALB Documentation](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/)
- [ALB Best Practices](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/best-practices.html)
- [Pricing Calculator](https://calculator.aws/)
