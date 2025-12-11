---
title: 'CRD Specifications'
description: 'Complete CRD specifications for all Infra Operator resources'
sidebar_position: 3
---

# CRD Specifications

This page provides detailed specifications for all Custom Resource Definitions (CRDs) in Infra Operator.

## Installing CRDs

CRDs are automatically installed when you deploy Infra Operator via Helm:

**Command:**

```bash
helm install infra-operator ./chart -n infra-operator --create-namespace
```

To verify CRDs are installed:

**Command:**

```bash
kubectl get crds | grep aws-infra-operator.runner.codes
```

**Expected output (30 CRDs):**

```
albs.aws-infra-operator.runner.codes
apigateways.aws-infra-operator.runner.codes
awsproviders.aws-infra-operator.runner.codes
certificates.aws-infra-operator.runner.codes
cloudfronts.aws-infra-operator.runner.codes
computestacks.aws-infra-operator.runner.codes
dynamodbtables.aws-infra-operator.runner.codes
ec2instances.aws-infra-operator.runner.codes
ec2keypairs.aws-infra-operator.runner.codes
ecrrepositories.aws-infra-operator.runner.codes
ecsclusters.aws-infra-operator.runner.codes
eksclusters.aws-infra-operator.runner.codes
elasticacheclusters.aws-infra-operator.runner.codes
elasticips.aws-infra-operator.runner.codes
iamroles.aws-infra-operator.runner.codes
internetgateways.aws-infra-operator.runner.codes
kmskeys.aws-infra-operator.runner.codes
lambdafunctions.aws-infra-operator.runner.codes
natgateways.aws-infra-operator.runner.codes
nlbs.aws-infra-operator.runner.codes
rdsinstances.aws-infra-operator.runner.codes
route53hostedzones.aws-infra-operator.runner.codes
route53recordsets.aws-infra-operator.runner.codes
routetables.aws-infra-operator.runner.codes
s3buckets.aws-infra-operator.runner.codes
secretsmanagersecrets.aws-infra-operator.runner.codes
securitygroups.aws-infra-operator.runner.codes
snstopics.aws-infra-operator.runner.codes
sqsqueues.aws-infra-operator.runner.codes
subnets.aws-infra-operator.runner.codes
vpcs.aws-infra-operator.runner.codes
```

## CRD Naming Convention

| Kubernetes Kind | Plural | Short Name |
|----------------|--------|------------|
| VPC | vpcs | vpc |
| Subnet | subnets | subnet |
| InternetGateway | internetgateways | igw |
| NATGateway | natgateways | nat |
| RouteTable | routetables | rt |
| SecurityGroup | securitygroups | sg |
| ElasticIP | elasticips | eip |
| ALB | albs | alb |
| NLB | nlbs | nlb |
| EC2Instance | ec2instances | ec2 |
| EKSCluster | eksclusters | eks |
| ECSCluster | ecsclusters | ecs |
| LambdaFunction | lambdafunctions | lambda |
| S3Bucket | s3buckets | s3 |
| RDSInstance | rdsinstances | rds |
| DynamoDBTable | dynamodbtables | ddb |
| ElastiCacheCluster | elasticacheclusters | ec |
| ECRRepository | ecrrepositories | ecr |
| SQSQueue | sqsqueues | sqs |
| SNSTopic | snstopics | sns |
| IAMRole | iamroles | iam |
| KMSKey | kmskeys | kms |
| SecretsManagerSecret | secretsmanagersecrets | secret |
| Certificate | certificates | cert |
| CloudFront | cloudfronts | cf |
| APIGateway | apigateways | apigw |
| Route53HostedZone | route53hostedzones | r53hz |
| Route53RecordSet | route53recordsets | r53rs |
| ComputeStack | computestacks | cs |
| AWSProvider | awsproviders | provider |

## Using Short Names

**Command:**

```bash
# These are equivalent:
kubectl get vpcs
kubectl get vpc

kubectl get securitygroups
kubectl get sg

kubectl get s3buckets
kubectl get s3
```

## Viewing CRD Schema

**Command:**

```bash
# View full CRD definition
kubectl get crd vpcs.aws-infra-operator.runner.codes -o yaml

# Explain spec fields
kubectl explain vpc.spec

# Explain nested fields
kubectl explain vpc.spec.tags
```

## Printer Columns

Each CRD defines custom columns for `kubectl get`:

### VPC

**Command:**

```bash
kubectl get vpc
```

```
NAME     VPC-ID                 CIDR          STATE      READY   AGE
my-vpc   vpc-0123456789abcdef0  10.0.0.0/16   available  true    5m
```

### S3Bucket

**Command:**

```bash
kubectl get s3
```

```
NAME        BUCKET-NAME              REGION      VERSIONING   READY   AGE
my-bucket   my-bucket-production     us-east-1   Enabled      true    10m
```

### EC2Instance

**Command:**

```bash
kubectl get ec2
```

```
NAME         INSTANCE-ID          TYPE       STATE     PUBLIC-IP      READY   AGE
my-instance  i-0123456789abcdef0  t3.micro   running   54.123.45.67   true    15m
```

## Validation

All CRDs include validation rules enforced by the Kubernetes API server:

### Required Fields

**Example:**

```yaml
# This will fail - missing cidrBlock
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  # cidrBlock is required!
```

### Pattern Validation

**Example:**

```yaml
# This will fail - invalid CIDR format
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  cidrBlock: "invalid"  # Must be valid CIDR notation
```

### Enum Validation

**Example:**

```yaml
# This will fail - invalid deletionPolicy
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  cidrBlock: "10.0.0.0/16"
  deletionPolicy: Invalid  # Must be Delete, Retain, or Orphan
```

## Webhooks (Optional)

Infra Operator includes validation webhooks for additional validation:

- Cross-field validation
- AWS-specific constraints
- Resource dependency checks

Enable webhooks in Helm values:

**Example:**

```yaml
webhooks:
  enabled: true
```

:::note
Webhooks require cert-manager for TLS certificates.
:::

## Upgrading CRDs

When upgrading Infra Operator, CRDs are automatically updated:

**Command:**

```bash
# Upgrade Helm release
helm upgrade infra-operator ./chart -n infra-operator

# Verify CRDs are updated
kubectl get crd vpcs.aws-infra-operator.runner.codes -o jsonpath='{.metadata.annotations.controller-gen\.kubebuilder\.io/version}'
```

## Backup and Restore

### Backup CRDs and Resources

**Command:**

```bash
# Backup CRDs
kubectl get crds -o yaml | grep -A 1000 "aws-infra-operator.runner.codes" > crds-backup.yaml

# Backup all resources
for resource in vpc subnet sg s3 ec2 rds; do
  kubectl get $resource -A -o yaml > ${resource}-backup.yaml
done
```

### Restore

**Command:**

```bash
# Restore CRDs (if needed)
kubectl apply -f crds-backup.yaml

# Restore resources
kubectl apply -f vpc-backup.yaml
kubectl apply -f subnet-backup.yaml
# ... etc
```
