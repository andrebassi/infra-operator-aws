---
title: 'Troubleshooting'
description: 'Common issues and solutions for Infra Operator'
sidebar_position: 3
---

# Troubleshooting

This guide covers common issues you may encounter when using Infra Operator and how to resolve them.

## Diagnostic Commands

### Check Operator Status

**Command:**

```bash
# Verify operator is running
kubectl get pods -n infra-operator

# Check operator logs
kubectl logs -n infra-operator deploy/infra-operator --tail=100

# Follow logs in real-time
kubectl logs -n infra-operator deploy/infra-operator -f
```

### Check Resource Status

**Command:**

```bash
# List all resources
kubectl get vpc,subnet,sg,s3,ec2 -A

# Get detailed status
kubectl describe vpc my-vpc

# Check events
kubectl get events -n infra-operator --sort-by='.lastTimestamp'
```

### Check AWSProvider

**Command:**

```bash
# Verify provider is ready
kubectl get awsprovider

# Check provider details
kubectl describe awsprovider aws-production
```

## Common Issues

### Operator Not Starting

**Symptoms:** Operator pod is in CrashLoopBackOff or Error state

**Check logs:**

```bash
kubectl logs -n infra-operator deploy/infra-operator --previous
```

**Common causes:**

1. **Missing CRDs**

   **Command:**

   ```bash
   # Verify CRDs are installed
   kubectl get crds | grep aws-infra-operator.runner.codes

   # Reinstall if missing
   kubectl apply -f chart/crds/
   ```

2. **Invalid RBAC**

   **Command:**

   ```bash
   # Check ServiceAccount
   kubectl get sa -n infra-operator

   # Check ClusterRole
   kubectl get clusterrole | grep infra-operator
   ```

3. **Resource limits too low**

   **Example:**

   ```yaml
   # Increase in values.yaml
   operator:
     resources:
       limits:
         memory: 512Mi
       requests:
         memory: 256Mi
   ```

### AWSProvider Not Ready

**Symptoms:** AWSProvider `ready: false`

**Check:**

```bash
kubectl describe awsprovider aws-production
```

**Common causes:**

1. **Invalid credentials**

   **Command:**

   ```bash
   # Verify Secret exists
   kubectl get secret aws-credentials -n infra-operator

   # Test credentials
   AWS_ACCESS_KEY_ID=$(kubectl get secret aws-credentials -n infra-operator \
     -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
   AWS_SECRET_ACCESS_KEY=$(kubectl get secret aws-credentials -n infra-operator \
     -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)
   aws sts get-caller-identity
   ```

2. **Wrong region**

   **Example:**

   ```yaml
   # Verify region in provider
   spec:
     region: us-east-1  # Must match your AWS resources
   ```

3. **IRSA not configured (EKS)**

   **Command:**

   ```bash
   # Check ServiceAccount annotation
   kubectl get sa infra-operator -n infra-operator -o yaml | grep eks.amazonaws.com

   # Verify IAM role trust policy
   aws iam get-role --role-name infra-operator-role
   ```

### Resource Stuck in Pending

**Symptoms:** Resource stays in `pending` state indefinitely

**Check:**

```bash
kubectl describe vpc my-vpc
kubectl logs -n infra-operator deploy/infra-operator | grep "my-vpc"
```

**Common causes:**

1. **AWSProvider not ready**

   **Command:**

   ```bash
   kubectl get awsprovider
   # Ensure provider referenced by resource is ready
   ```

2. **AWS API errors**

   **Command:**

   ```bash
   # Check operator logs for AWS errors
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "error\|failed"
   ```

3. **Rate limiting**

   **Command:**

   ```bash
   # Look for throttling errors
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "throttl"
   ```

### Resource Won't Delete

**Symptoms:** Resource stuck in `Terminating` state

**Check:**

```bash
kubectl get vpc my-vpc -o yaml | grep -A 10 finalizers
```

**Solutions:**

1. **Check for dependent resources**

   **Command:**

   ```bash
   # VPC can't be deleted if it has subnets, IGWs, etc.
   aws ec2 describe-subnets --filters "Name=vpc-id,Values=vpc-xxx"
   aws ec2 describe-internet-gateways --filters "Name=attachment.vpc-id,Values=vpc-xxx"
   ```

2. **Remove finalizer (last resort)**

   **Command:**

   ```bash
   # WARNING: This may leave AWS resources orphaned
   kubectl patch vpc my-vpc -p '{"metadata":{"finalizers":[]}}' --type=merge
   ```

3. **Force delete with timeout**

   **Command:**

   ```bash
   kubectl delete vpc my-vpc --timeout=30s
   ```

### Drift Detected

**Symptoms:** Resource shows drift between Kubernetes spec and AWS

**Check:**

```bash
kubectl describe vpc my-vpc | grep -A 5 "Drift"
```

**Solutions:**

1. **Update spec to match AWS**

   **Command:**

   ```bash
   # Get current AWS state
   aws ec2 describe-vpcs --vpc-ids vpc-xxx

   # Update Kubernetes resource to match
   kubectl edit vpc my-vpc
   ```

2. **Force reconciliation**

   **Command:**

   ```bash
   # Add annotation to trigger reconcile
   kubectl annotate vpc my-vpc force-reconcile="$(date +%s)" --overwrite
   ```

3. **Enable auto-remediation**

   **Example:**

   ```yaml
   spec:
     driftDetection:
       enabled: true
       autoRemediate: true
   ```

### EC2 Instance Won't Start

**Symptoms:** EC2Instance stuck in `pending` or `stopped`

**Check:**

```bash
kubectl describe ec2instance my-instance
aws ec2 describe-instances --instance-ids i-xxx
```

**Common causes:**

1. **Invalid AMI**

   **Command:**

   ```bash
   # Check if AMI exists in region
   aws ec2 describe-images --image-ids ami-xxx
   ```

2. **Invalid instance type**

   **Command:**

   ```bash
   # Check available instance types
   aws ec2 describe-instance-types --instance-types t3.micro
   ```

3. **Subnet/Security Group issues**

   **Command:**

   ```bash
   # Verify subnet exists
   aws ec2 describe-subnets --subnet-ids subnet-xxx

   # Verify security group
   aws ec2 describe-security-groups --group-ids sg-xxx
   ```

4. **Insufficient capacity**

   **Command:**

   ```bash
   # Try different AZ or instance type
   aws ec2 describe-instance-type-offerings \
     --location-type availability-zone \
     --filters Name=instance-type,Values=t3.micro
   ```

### S3 Bucket Permission Denied

**Symptoms:** S3Bucket creation fails with access denied

**Check:**

```bash
kubectl logs -n infra-operator deploy/infra-operator | grep "s3\|bucket"
```

**Solutions:**

1. **Check IAM permissions**

   **JSON:**

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "s3:CreateBucket",
           "s3:DeleteBucket",
           "s3:GetBucketLocation",
           "s3:GetBucketTagging",
           "s3:PutBucketTagging",
           "s3:GetBucketVersioning",
           "s3:PutBucketVersioning",
           "s3:GetEncryptionConfiguration",
           "s3:PutEncryptionConfiguration"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

2. **Bucket name already exists**

   **Command:**

   ```bash
   # S3 bucket names are globally unique
   aws s3api head-bucket --bucket my-bucket-name
   ```

### LocalStack Connection Issues

**Symptoms:** Resources fail when using LocalStack

**Check:**

```bash
# Verify LocalStack is running
kubectl get pods | grep localstack

# Test connectivity
kubectl run test --rm -it --image=curlimages/curl -- \
  curl http://localstack.default.svc.cluster.local:4566/_localstack/health
```

**Solutions:**

1. **Check endpoint URL**

   **Example:**

   ```yaml
   # In AWSProvider
   spec:
     endpoint: http://localstack.default.svc.cluster.local:4566
   ```

2. **Check LocalStack services**

   **Command:**

   ```bash
   # List running services
   curl http://localhost:4566/_localstack/health | jq
   ```

## Performance Issues

### Slow Reconciliation

**Symptoms:** Resources take long time to sync

**Solutions:**

1. **Increase concurrency**

   **Example:**

   ```yaml
   # In operator deployment
   args:
     - --max-concurrent-reconciles=10
   ```

2. **Check rate limits**

   **Command:**

   ```bash
   # Monitor AWS API calls
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "rate\|limit"
   ```

### High Memory Usage

**Symptoms:** Operator using excessive memory

**Solutions:**

1. **Increase memory limits**

   **Example:**

   ```yaml
   operator:
     resources:
       limits:
         memory: 1Gi
   ```

2. **Reduce cache size**

   **Example:**

   ```yaml
   args:
     - --cache-size=100
   ```

## Getting Help

### Collect Debug Information

**Command:**

```bash
# Create debug bundle
mkdir debug-bundle
kubectl get pods -n infra-operator -o yaml > debug-bundle/pods.yaml
kubectl logs -n infra-operator deploy/infra-operator > debug-bundle/logs.txt
kubectl get crds | grep aws-infra-operator.runner.codes > debug-bundle/crds.txt
kubectl get awsprovider,vpc,subnet,sg -A -o yaml > debug-bundle/resources.yaml
kubectl get events -n infra-operator > debug-bundle/events.txt
```

### Report Issues

When reporting issues, include:

1. Operator version
2. Kubernetes version
3. Cloud provider (AWS/LocalStack)
4. Resource YAML (redacted credentials)
5. Operator logs
6. Error messages

**GitHub Issues:** https://github.com/andrebassi/infra-operator/issues
