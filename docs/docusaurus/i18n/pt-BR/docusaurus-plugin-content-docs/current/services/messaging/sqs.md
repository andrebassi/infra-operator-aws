---
title: 'SQS Queue - Message Queues'
description: 'Managed, scalable, and durable message queues for asynchronous communication'
sidebar_position: 2
---

Managed, scalable, and durable message queues for asynchronous communication between your application components.

## Prerequisite: AWSProvider Configuration

Before creating any AWS resource, you need to configure an **AWSProvider** that manages credentials and authentication with AWS.

**IRSA (Recommended for Production):**

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

**Static Credentials (Development/LocalStack):**

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

**Trust Policy (`trust-policy.json`):**

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

**IAM Policy - SQS (`sqs-policy.json`):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "sqs:CreateQueue",
        "sqs:DeleteQueue",
        "sqs:GetQueueAttributes",
        "sqs:SetQueueAttributes",
        "sqs:TagQueue",
        "sqs:UntagQueue",
        "sqs:ListQueueTags"
      ],
      "Resource": "*"
}
  ]
}
```

**Create Role with AWS CLI:**

```bash
# 1. Get OIDC Provider from EKS cluster
export CLUSTER_NAME=my-cluster
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

# 2. Update trust-policy.json with correct values
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
}
  ]
}
EOF

# 3. Create IAM Role
aws iam create-role \
  --role-name infra-operator-sqs-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator SQS management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-sqs-role \
  --policy-name SQSManagement \
  --policy-document file://sqs-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-sqs-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-sqs-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon SQS (Simple Queue Service) is a fully managed message queuing service that decouples components of distributed applications. With SQS, you can:

- Decouple distributed application components
- Process messages asynchronously
- Scale automatically based on load
- Ensure message durability with replication
- Implement patterns like producer-consumer and worker pools
- Integrate with Lambda and SNS

## Quick Start

**Basic Standard Queue:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: e2e-test-queue
  namespace: default
spec:
  queueName: e2e-test-messages-queue
  providerRef:
    name: localstack
  delaySeconds: 0
  maximumMessageSize: 262144
  messageRetentionPeriod: 345600
  visibilityTimeout: 30
  receiveMessageWaitTimeSeconds: 10
  tags:
    Environment: test
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Queue with DLQ:**

```yaml
# First create the DLQ
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: e2e-dlq-queue
  namespace: default
spec:
  queueName: e2e-test-dlq
  providerRef:
    name: localstack
  messageRetentionPeriod: 1209600
  deletionPolicy: Delete
---
# Then create main queue with DLQ reference
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: e2e-queue-with-dlq
  namespace: default
spec:
  queueName: e2e-test-main-queue
  providerRef:
    name: localstack
  visibilityTimeout: 60
  deadLetterQueue:
targetArn: arn:aws:sqs:us-east-1:000000000000:e2e-test-dlq
maxReceiveCount: 3
  tags:
    Application: test-app
    Team: platform
  deletionPolicy: Delete
```

**FIFO Queue:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: e2e-fifo-queue
  namespace: default
spec:
  queueName: e2e-test-fifo.fifo
  providerRef:
    name: localstack
  fifoQueue: true
  contentBasedDeduplication: true
  tags:
    Type: FIFO
    Environment: test
  deletionPolicy: Delete
```

**Apply:**

```bash
kubectl apply -f sqs-queue.yaml
```

**Verify Status:**

```bash
kubectl get sqsqueues
kubectl describe sqsqueue e2e-test-queue
kubectl get sqsqueue e2e-test-queue -o yaml
```

**Watch Creation:**

```bash
kubectl get sqsqueue e2e-test-queue -w
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource that manages AWS authentication

  Name of the AWSProvider resource to use

SQS queue name

  **Requirements:**
  - Standard queue: Name between 1-80 characters
  - FIFO queue: Name must end with `.fifo`, between 1-80 characters
  - Only alphanumeric characters, hyphens, and underscores
  - Example: `orders-processing` or `payments.fifo`

  :::note

The name must be unique within the AWS region
:::


### Optional Fields

If `true`, creates a FIFO (First-In-First-Out) queue with guaranteed processing order

  **Implications:**
  - FIFO: Guaranteed order, but lower throughput (~300 msg/s)
  - Standard: Better throughput (unlimited), but no order guarantee

  **Use FIFO when:** Processing order is critical (payments, transactions)

For FIFO queues, enables content-based deduplication using message body hash

  :::note

Only applicable when `fifoQueue: true`
:::


Message retention period in seconds (default: 4 days / 345,600 seconds)

  **Valid range:** 60 seconds to 1,209,600 seconds (14 days)

  **Examples:**
  - 60: 1 minute
  - 300: 5 minutes
  - 3600: 1 hour
  - 86400: 1 day
  - 345600: 4 days (default)
  - 1209600: 14 days (maximum)

Time in seconds a message is invisible after being consumed

  **Valid range:** 0 to 43,200 seconds (12 hours)

  **Configuration guide:**
  - Set to 6x the maximum expected processing time
  - If processing takes 10s, use 60s timeout
  - Too short timeout: messages reprocessed multiple times
  - Too long timeout: slow recovery on failure

Long polling time in seconds when no messages available

  **Valid range:** 0 to 20 seconds

  **Long polling benefits:**
  - Reduces API costs (fewer empty requests)
  - Reduces latency (immediate response when message arrives)
  - 20s is the maximum allowed

  **Recommendation:** Use 20 in production

Delay time before message becomes visible in the queue

  **Valid range:** 0 to 900 seconds (15 minutes)

  **Use:** Schedule future message processing

Maximum message size in bytes

  **Valid range:** 1,024 to 262,144 bytes (256 KB)

  **Default:** 262,144 bytes (256 KB)

Dead letter queue configuration for messages that could not be processed

  ARN of the target DLQ queue

      Example: `arn:aws:sqs:us-east-1:123456789012:orders-dlq`

:::note

DLQ queue must exist before referencing
:::


Maximum number of consumption attempts before sending to DLQ

**Valid range:** 1 to 1,000

**Recommendation:** 3 for most cases

**Importance:** DLQ is essential for tracking problematic messages

ID or ARN of the AWS KMS key for message encryption at rest

  Example: `arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012`

  **When to use:**
  - Sensitive data (financial information, PII)
  - Regulatory compliance (HIPAA, PCI-DSS)
  - Production with critical data

Key-value pairs to tag and categorize the queue

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: order-service
    Team: backend
    CostCenter: engineering
  ```

Policy for when the CustomResource is deleted

  **Options:**
  - `Delete`: Queue is deleted from AWS
  - `Retain`: Queue remains in AWS but unmanaged
  - `Orphan`: Queue remains and CR ownership is removed

  **Recommendation:** Use `Retain` in production

## Status Fields

After the SQS queue is created, the following status fields are populated:

URL of the SQS queue for sending/receiving messages

  Example: `https://sqs.us-east-1.amazonaws.com/123456789012:orders-processing`

ARN (Amazon Resource Name) of the queue

  Example: `arn:aws:sqs:us-east-1:123456789012:orders-processing`

Approximate number of messages available in the queue

Approximate number of messages in processing (invisible)

Approximate number of scheduled/delayed messages

`true` when the queue is created and ready for use

Timestamp of last AWS synchronization

## Examples

### Standard Queue for High Throughput

Simple queue optimized for fast high-volume processing:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: event-stream-queue
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Standard queue for better throughput
  queueName: event-stream-processing

  # Short retention for temporary events
  messageRetentionPeriod: 86400  # 1 day

  # Timeout adjusted for fast processing
  visibilityTimeout: 10

  # Long polling for efficiency
  receiveMessageWaitTimeSeconds: 20

  tags:
    Environment: production
    Type: event-stream
    Throughput: high

  # Delete queue on cleanup (ephemeral)
  deletionPolicy: Delete
```

### FIFO Queue with Order Guarantee

FIFO queue for sequential processing (e.g., transactions):

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: payment-transactions-queue
  namespace: default
spec:
  providerRef:
    name: production-aws

  # FIFO name
  queueName: payment-transactions.fifo

  # Enable FIFO
  fifoQueue: true

  # Automatic deduplication
  contentBasedDeduplication: true

  # Long retention for audit
  messageRetentionPeriod: 1209600  # 14 days

  # Longer timeout for transaction processing
  visibilityTimeout: 120

  # Long polling
  receiveMessageWaitTimeSeconds: 20

  # Maximum 1000 tps for FIFO
  maximumMessageSize: 262144

  tags:
    Environment: production
    Type: financial-transactions
    Compliance: required
    CriticalData: "true"

  # Keep queue in production
  deletionPolicy: Retain
```

### Queue with Dead Letter Queue (DLQ)

Robust configuration with failure handling:

```yaml
# First, create the DLQ
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: order-processing-dlq
  namespace: default
spec:
  providerRef:
    name: production-aws
  queueName: order-processing-dlq
  messageRetentionPeriod: 1209600  # 14 days for investigation
  visibilityTimeout: 300
  tags:
    Environment: production
    Type: dlq
    Parent: order-processing
  deletionPolicy: Retain

---
# Then, create main queue with configured DLQ
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: order-processing-queue
  namespace: default
spec:
  providerRef:
    name: production-aws
  queueName: order-processing

  messageRetentionPeriod: 345600  # 4 days
  visibilityTimeout: 60
  receiveMessageWaitTimeSeconds: 20

  # Configure DLQ
  deadLetterQueue:
# Use ARN of DLQ created above
targetArn: arn:aws:sqs:us-east-1:123456789012:order-processing-dlq
# Send to DLQ after 3 attempts
maxReceiveCount: 3

  tags:
    Environment: production
    Application: order-service
    HasDLQ: "true"

  deletionPolicy: Retain
```

### Queue with KMS Encryption

Queue with sensitive data encrypted at rest:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: sensitive-data-queue
  namespace: default
spec:
  providerRef:
    name: production-aws

  queueName: sensitive-data-processing

  # KMS encryption for sensitive data
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012

  messageRetentionPeriod: 86400
  visibilityTimeout: 45
  receiveMessageWaitTimeSeconds: 20

  deadLetterQueue:
targetArn: arn:aws:sqs:us-east-1:123456789012:sensitive-data-dlq
maxReceiveCount: 2

  tags:
    Environment: production
    DataClassification: confidential
    Encryption: kms-required
    Compliance: hipaa-pci-dss

  deletionPolicy: Retain
```

## Verification

### Verify Queue Status

**Command:**

```bash
# List all SQS queues
kubectl get sqsqueues

# Get detailed queue information
kubectl get sqsqueue order-queue -o yaml

# Watch queue creation
kubectl get sqsqueue order-queue -w

# Verify creation events
kubectl describe sqsqueue order-queue
```

### Verify in AWS

**AWS CLI:**

```bash
# List queues
aws sqs list-queues --region us-east-1

# Get queue attributes
aws sqs get-queue-attributes \
      --queue-url https://sqs.us-east-1.amazonaws.com/123456789012/orders-processing \
      --attribute-names All \
      --region us-east-1

# Get approximate message count
aws sqs get-queue-attributes \
      --queue-url https://sqs.us-east-1.amazonaws.com/123456789012/orders-processing \
      --attribute-names ApproximateNumberOfMessages \
      --region us-east-1

# Send test message
aws sqs send-message \
      --queue-url https://sqs.us-east-1.amazonaws.com/123456789012/orders-processing \
      --message-body '{"order_id": "12345", "status": "test"}' \
      --region us-east-1
```

**LocalStack:**

```bash
# For LocalStack testing
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_REGION=us-east-1

# List queues
aws sqs list-queues

# Get attributes
aws sqs get-queue-attributes \
      --queue-url http://localhost:4566/000000000000/orders-processing \
      --attribute-names All
```

**AWS Console:**

1. Access AWS Management Console
2. Go to SQS
3. Search for queue by name
4. Open queue to see details
5. Monitor: Messages available, In flight, Delayed

### Expected Output

**Example:**

```yaml
status:
  queueUrl: https://sqs.us-east-1.amazonaws.com/123456789012/orders-processing
  queueArn: arn:aws:sqs:us-east-1:123456789012:orders-processing
  approximateNumberOfMessages: 42
  approximateNumberOfMessagesNotVisible: 5
  approximateNumberOfMessagesDelayed: 0
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### Queue stuck in pending state

**Symptoms:** Queue `ready: false` for more than 2 minutes

**Common causes:**
1. Invalid AWSProvider credentials
2. Network connectivity issues
3. Queue name already exists in AWS account
4. AWS API rate limiting

**Solutions:**
```bash
# Check AWSProvider status
kubectl describe awsprovider production-aws

# Check controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100

# Check queue events
kubectl describe sqsqueue order-queue

# Check if queue exists in AWS
aws sqs list-queues --query 'QueueUrls' | grep orders-processing
```

### Messages not being processed

**Symptoms:** `approximateNumberOfMessages` grows indefinitely

**Common causes:**
1. Queue consumer is offline
2. Visibility timeout too short (message returns quickly)
3. Consumer error preventing delete
4. DLQ policy sending to DLQ prematurely

**Solutions:**
```bash
# Check message count
aws sqs get-queue-attributes \
      --queue-url <QUEUE_URL> \
      --attribute-names ApproximateNumberOfMessages

# Increase visibility timeout if needed
kubectl patch sqsqueue order-queue --type merge -p \
      '{"spec":{"visibilityTimeout": 120}}'

# Check consumer is running
kubectl get pods -l app=order-consumer
kubectl logs -l app=order-consumer --tail=50
```

### DLQ filling with messages

**Symptoms:** DLQ grows but no action taken

**Common causes:**
1. Consumer didn't implement retry logic correctly
2. Malformed or invalid messages
3. External dependency (database, API) offline
4. maxReceiveCount too low

**Solutions:**
```bash
# Receive message from DLQ for investigation
aws sqs receive-message \
      --queue-url <DLQ_URL> \
      --attribute-names All \
      --message-attribute-names All

# Increase maxReceiveCount if appropriate
kubectl patch sqsqueue order-queue --type merge -p \
      '{"spec":{"deadLetterQueue":{"maxReceiveCount": 5}}}'

# Purge DLQ after fixing
aws sqs purge-queue --queue-url <DLQ_URL>
```

### Visibility Timeout too short/long

**Symptoms:**
- Too short: Duplicate messages in processing
- Too long: Slow failure recovery

**Solution:**
```bash
# Measure processing time
# Formula: visibilityTimeout = 6 × max_processing_time

# If processing takes up to 30 seconds:
kubectl patch sqsqueue order-queue --type merge -p \
      '{"spec":{"visibilityTimeout": 180}}'  # 30s × 6
```

### Duplicate messages in Standard Queue

**Symptoms:** Messages processed multiple times

**Cause:** Standard Queue doesn't guarantee deduplication

**Solutions:**
1. Implement deduplication in consumer (use message ID or idempotency key)
2. Use FIFO Queue if order is important
3. Implement database idempotency (unique constraint)

**Example:**

```yaml
# Migrate to FIFO if order is critical
spec:
      queueName: orders.fifo
      fifoQueue: true
      contentBasedDeduplication: true
```

### FIFO Queue throughput problems

**Symptoms:** Messages queued, not processed quickly

**Cause:** FIFO queues have limit of ~300 messages/second

**Solutions:**
1. Use `MessageGroupId` to parallelize across multiple groups
2. Increase number of consumers per group
3. Consider using Standard Queue if order isn't critical

**Command:**

```bash
# Send messages in different MessageGroupIds
aws sqs send-message \
      --queue-url <QUEUE_URL> \
      --message-body 'message' \
      --message-group-id 'group-1'  # Different values for parallelism
```

### Error referencing DLQ

**Error:** `DLQ target ARN does not exist`

**Cause:** DLQ queue wasn't created first

**Solution:**
```bash
# Create DLQ first
kubectl apply -f dlq.yaml

# Wait for DLQ to be ready
kubectl get sqsqueue order-processing-dlq -w

# Then create main queue with DLQ reference
kubectl apply -f order-queue.yaml
```

## Best Practices

:::note Best Practices

- **Always configure Dead Letter Queue** — Essential for production, enables tracking failed messages
- **Set appropriate visibility timeout** — Should exceed message processing time
- **Use long polling** — Reduces empty responses and API costs (waitTimeSeconds: 20)
- **Configure maxReceiveCount wisely** — Balance retry attempts vs DLQ overflow
- **Enable encryption** — KMS encryption for sensitive message data

:::

## Integration Patterns

### Simple Producer-Consumer

Publishing application sends messages, workers consume:

![SQS Producer Consumer](/img/diagrams/sqs-producer-consumer.svg)

**Implementation:**
- Producer: `send_message()` to enqueue
- Consumer: `receive_message()` in loop with long polling
- Delete: `delete_message()` after successful processing

### Fan-Out with SNS + SQS

Message published to multiple queues via SNS:

![SQS Fan-out Pattern](/img/diagrams/sqs-fanout-pattern.svg)

**Use case:** One action triggers multiple workflows

### Backpressure Handling

Controlling processing rate with visibilityTimeout:

![SQS Throttling](/img/diagrams/sqs-throttling.svg)

**Control:**
- Increase `visibilityTimeout` slows reprocessing
- Implement exponential backoff in consumer
- Scale number of consumers to increase throughput

## Related Resources

- [SNS Topic](/services/messaging/sns)

  - [Lambda](/services/compute/lambda)
