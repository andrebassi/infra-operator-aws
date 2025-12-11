---
title: 'SNS Topic - Pub/Sub Messaging'
description: 'Scalable, fully managed pub/sub messaging service for notifications'
sidebar_position: 1
---

Fully managed, scalable, and highly available pub/sub messaging service for notifications, alerts, and distributed communication.

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

**IAM Policy - SNS (sns-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "sns:CreateTopic",
        "sns:DeleteTopic",
        "sns:GetTopicAttributes",
        "sns:SetTopicAttributes",
        "sns:Subscribe",
        "sns:Unsubscribe",
        "sns:TagResource",
        "sns:UntagResource",
        "sns:ListTagsForResource"
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
  --role-name infra-operator-sns-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator SNS management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-sns-role \
  --policy-name SNSManagement \
  --policy-document file://sns-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-sns-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-sns-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon SNS (Simple Notification Service) is a fully managed pub/sub service that enables publishing messages to multiple subscribers (subscribers) simultaneously. With SNS, you can:

- Publish messages to multiple subscribers (fan-out pattern)
- Send notifications via email, SMS, push notifications
- Integrate with SQS, Lambda, HTTP endpoints and applications
- Ensure delivery with automatic retry
- Use FIFO topics for guaranteed message ordering
- Apply filter policies for intelligent routing
- Implement patterns like fan-out and broadcast

## Quick Start

The simplest SNS configuration with fan-out:

**Standard SNS Topic:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: e2e-test-topic
  namespace: default
spec:
  topicName: e2e-notifications
  providerRef:
    name: localstack
  displayName: E2E Test Notifications
  subscriptions:
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:000000000000:e2e-test-messages-queue
  tags:
    Environment: test
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**FIFO SNS Topic:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: e2e-fifo-topic
  namespace: default
spec:
  topicName: e2e-notifications.fifo
  providerRef:
    name: localstack
  fifoTopic: true
  contentBasedDeduplication: true
  subscriptions:
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:000000000000:e2e-test-fifo.fifo
  tags:
    Type: FIFO
    Environment: test
  deletionPolicy: Delete
```

**Production Topic:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: order-notifications
  namespace: default
spec:
  providerRef:
    name: production-aws
  topicName: order-notifications
  displayName: Order Notifications
  subscriptions:
  - protocol: email
endpoint: team@example.com
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:order-queue
  tags:
    Environment: production
    Team: backend
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f sns-topic.yaml
```

**Verify Status:**

```bash
kubectl get snstopics
kubectl describe snstopic order-notifications
kubectl get snstopic order-notifications -o yaml
```

**Watch Creation:**

```bash
kubectl get snstopic order-notifications -w
```

**Publish Test Message:**

```bash
aws sns publish \
  --topic-arn arn:aws:sns:us-east-1:123456789012:order-notifications \
  --message "Test message" \
  --subject "Test Subject" \
  --region us-east-1
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource that manages AWS authentication

  Name of the AWSProvider resource to use

SNS topic name

  **Requirements:**
  - Standard topic: Name between 1-256 characters
  - FIFO topic: Name must end with `.fifo`, between 1-256 characters
  - Only alphanumeric characters, hyphens, and underscores
  - Example: `order-notifications` or `payments.fifo`

  :::note

The name must be unique within the AWS region
:::


### Optional Fields

Readable name of topic displayed in email/SMS notifications

  **Example**: `Order Notifications`

If `true`, creates a FIFO (First-In-First-Out) topic with guaranteed processing order

  **Implications:**
  - FIFO: Guaranteed order, throughput ~300 msg/s
  - Standard: Better throughput (unlimited), no order guarantee

  **Use FIFO when:** Order of processing is critical (payments, transactions, critical events)

For FIFO topics, enables content-based deduplication using message body hash

  :::note

Only applicable when `fifoTopic: true`
:::


List of subscriptions (subscribers) that will receive published messages

  Message delivery protocol

**Supported values:**
- `email`: Send to email address
- `email-json`: Email with structured JSON
- `sms`: Send SMS to phone number
- `sqs`: Send to SQS queue
- `lambda`: Invoke Lambda function
- `http`: POST to HTTP endpoint
- `https`: POST to HTTPS endpoint
- `application`: Mobile push notification

**Cost:** Email free, SMS has cost, others vary

Message destination

**Examples by protocol:**
- email: `team@example.com`
- sms: `+5511999999999`
- sqs: `arn:aws:sqs:us-east-1:123456789012:order-queue`
- lambda: `arn:aws:lambda:us-east-1:123456789012:function:process-event`
- http: `https://api.example.com/webhook`
- application: `arn:aws:sns:us-east-1:123456789012:app/GCM/MyApp/abcdef`

If `true`, SNS sends only the message body (without SNS wrapper)

**Use when:** Integrating with SQS or Lambda (less overhead)

Policy to filter which messages are delivered to this subscription

**Example:**
      ```json
      {
        "store": ["example_corp"],
        "order_type": ["order-placed", "order-canceled"],
        "price": [{"numeric": [">", 100]}]
      }
      ```

**Benefit:** Reduces costs (doesn't send unnecessary messages)

Dead Letter Queue for messages that failed delivery

**Properties:**
- `deadLetterTargetArn`: ARN of SQS queue for failures
- `maxReceiveCount`: How many attempts before sending to DLQ

**Importance:** Subscriptions define who and how messages are delivered

ID or ARN of AWS KMS key for message encryption at rest

  Example: `arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012`

  **When to use:**
  - Sensitive data (financial information, PII)
  - Regulatory compliance (HIPAA, PCI-DSS)
  - Production with critical data

JSON policy that defines retry behavior and timeout for delivery

  **Example with exponential retry:**
  ```json
  {
"http": {
      "defaultHealthyRetryPolicy": {
        "minDelayTarget": 20,
        "maxDelayTarget": 20,
        "numRetries": 3,
        "numMaxDelayThresholds": 0,
        "numNoDelayTransitions": 0,
        "numWithDelayTransitions": 0,
        "maxReceiveCount": 100000
      }
}
  }
  ```

Key-value pairs to tag and categorize the topic

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
  - `Delete`: Topic is deleted from AWS
  - `Retain`: Topic remains in AWS but unmanaged
  - `Orphan`: Topic remains and CR ownership is removed

  **Recommendation:** Use `Retain` in production

## Status Fields

After the SNS topic is created, the following status fields are populated:

ARN (Amazon Resource Name) of the SNS topic

  Example: `arn:aws:sns:us-east-1:123456789012:order-notifications`

List of created subscriptions with their details

  **Fields per subscription:**
  - `subscriptionArn`: Unique ARN of the subscription
  - `protocol`: Protocol used
  - `endpoint`: Message destination
  - `status`: Subscription status (Subscribed, PendingConfirmation, etc)

`true` when the topic is created and ready to publish messages

Timestamp of last AWS synchronization

## Examples

### Standard SNS Topic for Fan-Out

Simple topic that publishes to multiple subscribers simultaneously:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: order-events
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Unique topic name
  topicName: order-events-production
  displayName: Order Events

  # Subscriptions for different systems
  subscriptions:
  # Notify via email
  - protocol: email
endpoint: orders@company.com
rawMessageDelivery: false

  # Send to processing queue
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:order-processing
rawMessageDelivery: true

  # Invoke Lambda function for analysis
  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:analyze-order

  # Webhook to external system
  - protocol: https
endpoint: https://analytics.company.com/orders

  tags:
    Environment: production
    Type: event-stream
    Pattern: fan-out

  deletionPolicy: Retain
```

### FIFO SNS Topic with Order Guarantee

FIFO topic to ensure message ordering (e.g., transactions):

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: payment-transactions
  namespace: default
spec:
  providerRef:
    name: production-aws

  # FIFO name
  topicName: payment-transactions.fifo
  displayName: Payment Transactions

  # Enable FIFO
  fifoTopic: true
  contentBasedDeduplication: true

  # Subscriptions for sequential processing
  subscriptions:
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:payments.fifo
rawMessageDelivery: true
# Filter for approved payments
filterPolicy:
      status: ["approved"]
      amount: [{"numeric": [">", 0]}]

  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:process-payment
filterPolicy:
      status: ["approved"]

  # Notify fraud
  - protocol: email
endpoint: fraud-team@company.com
filterPolicy:
      risk_level: ["high", "critical"]

  tags:
    Environment: production
    Type: financial
    Compliance: required
    CriticalData: "true"

  deletionPolicy: Retain
```

### SNS with Multiple Protocols (Email + SMS + SQS + Lambda)

Alert topic with notifications through multiple channels:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: critical-alerts
  namespace: default
spec:
  providerRef:
    name: production-aws

  topicName: critical-alerts
  displayName: Critical System Alerts

  subscriptions:
  # Email for support
  - protocol: email
endpoint: support@company.com
filterPolicy:
      severity: ["critical", "emergency"]

  # SMS for oncall
  - protocol: sms
endpoint: +5511987654321
filterPolicy:
      severity: ["critical", "emergency"]

  # Queue for automatic processing
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:alerts-queue
rawMessageDelivery: true

  # Lambda for automatic escalation
  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:escalate-alert
filterPolicy:
      severity: ["critical"]

  # Store in S3 via Lambda
  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:store-alert

  tags:
    Environment: production
    Application: monitoring
    AlertLevel: critical

  deletionPolicy: Retain
```

### SNS with Filter Policy for Smart Routing

Topic that routes messages based on attributes:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: store-events
  namespace: default
spec:
  providerRef:
    name: production-aws

  topicName: store-events-production
  displayName: Store Events

  subscriptions:
  # Sales notifications for sales team
  - protocol: email
endpoint: sales@company.com
filterPolicy:
      event_type: ["sale", "refund"]
      store: ["nyc", "los-angeles", "chicago"]

  # Inventory analysis
  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:update-inventory
filterPolicy:
      event_type: ["stock-low", "restock"]

  # Reports from all stores
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:reports-queue
rawMessageDelivery: true
filterPolicy:
      event_type: ["sale", "refund"]
      amount: [{"numeric": [">", 100]}]

  # Notify store manager
  - protocol: https
endpoint: https://store-manager.company.com/notify
filterPolicy:
      event_type: ["incident", "issue"]

  tags:
    Environment: production
    Type: retail
    Pattern: content-routing

  deletionPolicy: Retain
```

### SNS with Dead Letter Queue

Topic with robust failure handling:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: orders-dlq
  namespace: default
spec:
  providerRef:
    name: production-aws

  topicName: orders-dlq
  displayName: Orders DLQ

  subscriptions:
  # Main queue for processing
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:orders-processing
rawMessageDelivery: true
# Dead Letter Queue for failures
redrivePolicy:
      deadLetterTargetArn: arn:aws:sqs:us-east-1:123456789012:orders-dlq
      maxReceiveCount: 3

  # Lambda with custom retry policy
  - protocol: lambda
endpoint: arn:aws:lambda:us-east-1:123456789012:function:process-order
redrivePolicy:
      deadLetterTargetArn: arn:aws:sqs:us-east-1:123456789012:order-failures

  tags:
    Environment: production
    Application: orders
    HasDLQ: "true"

  deletionPolicy: Retain
```

### SNS with KMS Encryption

Topic with encrypted sensitive data:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: secure-notifications
  namespace: default
spec:
  providerRef:
    name: production-aws

  topicName: secure-notifications
  displayName: Secure Notifications

  # KMS encryption for sensitive data
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012

  subscriptions:
  - protocol: email
endpoint: security@company.com
rawMessageDelivery: false

  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:secure-logs
rawMessageDelivery: true

  # Delivery policy with robust retry
  deliveryPolicy: |
{
      "http": {
        "defaultHealthyRetryPolicy": {
          "minDelayTarget": 20,
          "maxDelayTarget": 20,
          "numRetries": 3,
          "numMaxDelayThresholds": 0,
          "numNoDelayTransitions": 0,
          "numWithDelayTransitions": 0,
          "maxReceiveCount": 100000
        },
        "disableSubscriptionOverrides": false
      }
}

  tags:
    Environment: production
    DataClassification: confidential
    Encryption: kms-required
    Compliance: hipaa-pci-dss

  deletionPolicy: Retain
```

## Verification

### Verify Topic Status

**Command:**

```bash
# List all SNS topics
kubectl get snstopics

# Get detailed topic information
kubectl get snstopic order-notifications -o yaml

# Watch topic creation
kubectl get snstopic order-notifications -w

# Verify creation events
kubectl describe snstopic order-notifications
```

### Verify in AWS

**AWS CLI:**

```bash
# List topics
aws sns list-topics --region us-east-1

# Get topic attributes
aws sns get-topic-attributes \
      --topic-arn arn:aws:sns:us-east-1:123456789012:order-notifications \
      --region us-east-1

# List topic subscriptions
aws sns list-subscriptions-by-topic \
      --topic-arn arn:aws:sns:us-east-1:123456789012:order-notifications \
      --region us-east-1

# Publish test message
aws sns publish \
      --topic-arn arn:aws:sns:us-east-1:123456789012:order-notifications \
      --message '{"order_id": "12345", "status": "test"}' \
      --message-attributes '{"order_id":{"DataType":"String","StringValue":"12345"}}' \
      --region us-east-1

# Publish with attributes to test filter policy
aws sns publish \
      --topic-arn arn:aws:sns:us-east-1:123456789012:store-events \
      --message "Store event message" \
      --message-attributes \
        event_type="{DataType=String,StringValue=sale}" \
        store="{DataType=String,StringValue=nyc}" \
        amount="{DataType=Number,StringValue=150}" \
      --region us-east-1
```

**LocalStack:**

```bash
# For LocalStack testing
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_REGION=us-east-1

# List topics
aws sns list-topics

# Get attributes
aws sns get-topic-attributes \
      --topic-arn arn:aws:sns:us-east-1:000000000000:order-notifications

# Publish message
aws sns publish \
      --topic-arn arn:aws:sns:us-east-1:000000000000:order-notifications \
      --message "Test message"
```

**AWS Console:**

1. Access AWS Management Console
2. Go to SNS
3. Search for topic by name
4. Open topic to see details
5. See "Subscriptions" to confirm subscribers
6. Use "Publish message" to test

### Expected Output

**Example:**

```yaml
status:
  topicArn: arn:aws:sns:us-east-1:123456789012:order-notifications
  subscriptions:
  - subscriptionArn: arn:aws:sns:us-east-1:123456789012:order-notifications:12345678-1234-1234-1234-123456789012
protocol: email
endpoint: team@example.com
status: PendingConfirmation
  - subscriptionArn: arn:aws:sns:us-east-1:123456789012:order-notifications:87654321-4321-4321-4321-210987654321
protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:order-queue
status: Subscribed
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### Subscription pending confirmation (Email)

**Symptoms:** Status `PendingConfirmation` for email subscriptions

**Cause:** Confirmation email needs to be accepted

**Solutions:**
```bash
# 1. Check subscription status
aws sns list-subscriptions-by-topic \
      --topic-arn <TOPIC_ARN> \
      --query 'Subscriptions[*].[SubscriptionArn,Endpoint,SubscriptionArn]'

# 2. User accesses email and clicks confirmation link
# 3. Confirm it changed to Subscribed
aws sns list-subscriptions-by-topic \
      --topic-arn <TOPIC_ARN>

# 4. If email doesn't arrive, check spam/junk
# 5. Recreate subscription if needed
kubectl delete snstopic order-notifications
kubectl apply -f sns-topic.yaml
```

**Note**: Confirmation is only needed for email/SMS

### Messages not being delivered

**Symptoms:** Publishing message doesn't result in delivery

**Common causes:**
1. Filter policy blocking message
2. Subscription still in PendingConfirmation
3. Permission issues (IAM, SQS access)
4. Invalid or unreachable endpoint

**Solutions:**
```bash
# Verify filter policy is correct
aws sns get-subscription-attributes \
      --subscription-arn <SUBSCRIPTION_ARN> \
      --attribute-name FilterPolicy

# Publish with attributes that pass filter
aws sns publish \
      --topic-arn <TOPIC_ARN> \
      --message "test" \
      --message-attributes \
        store="{DataType=String,StringValue=nyc}"

# Verify SQS can receive from SNS (policy)
aws sqs get-queue-attributes \
      --queue-url <QUEUE_URL> \
      --attribute-names Policy

# Check Lambda logs
aws logs tail /aws/lambda/process-order --follow
```

### Filter Policy not working

**Symptoms:** Messages arriving even without passing filter

**Cause:** Incorrect filter policy syntax

**Solution:**
```bash
# Validate filter syntax
# Must be valid JSON and attributes need to match

# Correct example:
{
      "event_type": ["sale", "refund"],
      "amount": [{"numeric": [">", 100]}],
      "store": ["nyc", "los-angeles"]
}

# Update filter policy
aws sns set-subscription-attributes \
      --subscription-arn <SUBSCRIPTION_ARN> \
      --attribute-name FilterPolicy \
      --attribute-value '{"event_type":["sale"]}'

# Confirm it was applied
aws sns get-subscription-attributes \
      --subscription-arn <SUBSCRIPTION_ARN> \
      --attribute-name FilterPolicy
```

**Note**: Use `FilterPolicy` in Kubernetes (capitalized)

### High SMS costs

**Symptoms:** AWS bill with very high SMS charges

**Cause:** SMS sent to many users

**Solutions:**
1. Use filter policy to reduce recipients
2. Send only for critical alerts
3. Use email instead of SMS when possible
4. Implement rate limiting

**Example:**

```yaml
# Example: SMS only for critical alerts
subscriptions:
- protocol: sms
      endpoint: +5511987654321
      filterPolicy:
        severity: ["critical"]  # Only for critical
        alert_type: ["security"]  # Only certain types
```

### Lambda invocation throttling

**Symptoms:** Lambda not invoked for all messages

**Cause:** Lambda concurrency limit reached

**Solutions:**
```bash
# Increase Lambda reserved concurrency
aws lambda put-function-concurrency \
      --function-name process-order \
      --reserved-concurrent-executions 100

# Use DLQ to capture failures
kubectl patch snstopic order-notifications --type merge -p \
      '{"spec":{"subscriptions":[{"protocol":"lambda","redrivePolicy":{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:lambda-failures"}}]}}'

# Check Lambda logs
aws logs tail /aws/lambda/process-order --follow
```

### FIFO Topic throughput problems

**Symptoms:** Messages queued, processed slowly

**Cause:** FIFO topics limited to ~300 messages/second

**Solutions:**
1. Use `MessageGroupId` to parallelize across multiple groups
2. Increase consumers per group
3. Consider Standard topic if order isn't critical

**Command:**

```bash
# Publish with different group IDs for parallelism
aws sns publish \
      --topic-arn <FIFO_TOPIC_ARN> \
      --message "msg" \
      --message-group-id "group-1"

aws sns publish \
      --topic-arn <FIFO_TOPIC_ARN> \
      --message "msg" \
      --message-group-id "group-2"
```

### Topic stuck in pending state

**Symptoms:** Topic `ready: false` for more than 2 minutes

**Common causes:**
1. Invalid AWSProvider credentials
2. Connectivity issues
3. Topic name already exists
4. AWS API rate limiting

**Solutions:**
```bash
# Check AWSProvider status
kubectl describe awsprovider production-aws

# Check controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100

# Check topic events
kubectl describe snstopic order-notifications

# Check if topic exists in AWS
aws sns list-topics | grep order-notifications
```

### Error creating subscriptions

**Error:** `InvalidParameter: Invalid parameter: TopicArn Reason: Invalid ARN`

**Cause:** Topic hasn't been created yet or ARN is invalid

**Solution:**
```bash
# Create topic first
kubectl apply -f sns-topic.yaml

# Wait for topic to be ready
kubectl wait --for=condition=ready snstopic/order-notifications

# Then add subscriptions
kubectl patch snstopic order-notifications --type merge -p \
      '{"spec":{"subscriptions":[{"protocol":"email","endpoint":"team@example.com"}]}}'
```

### Delivery Policy not working

**Symptoms:** Retry not retrying as expected

**Cause:** Incorrect delivery policy syntax

**Solution:**
```bash
# Delivery policy must be valid JSON
aws sns set-topic-attributes \
      --topic-arn <TOPIC_ARN> \
      --attribute-name DeliveryPolicy \
      --attribute-value '{
        "http": {
          "defaultHealthyRetryPolicy": {
            "minDelayTarget": 20,
            "maxDelayTarget": 20,
            "numRetries": 3
          }
        }
      }'

# Confirm it was applied
aws sns get-topic-attributes \
      --topic-arn <TOPIC_ARN> \
      --attribute-name DeliveryPolicy
```

## Best Practices

:::note Best Practices

- **Choose FIFO vs Standard wisely** — FIFO for transactions/payments, Standard for high-volume events
- **Enable message deduplication** — Prevent duplicate processing in FIFO topics
- **Use message filtering** — Reduce unnecessary subscriber notifications
- **Configure DLQ for failed deliveries** — Capture messages that fail to deliver
- **Enable encryption** — KMS encryption for sensitive message content

:::

## Integration Patterns

### Fan-Out Pattern (SNS → Multiple SQS)

Publish one message to multiple queues:

![SNS Fan-out](/img/diagrams/sns-fanout.svg)

**Use case:** One action triggers multiple independent workflows

**Implementation:**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: order-created
spec:
  providerRef:
    name: production-aws
  topicName: order-created
  subscriptions:
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:order-processing
rawMessageDelivery: true
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:inventory-update
rawMessageDelivery: true
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:notification-queue
rawMessageDelivery: true
```

### Distributed Alerts and Notifications

Distribute alerts via multiple channels:

![SNS Alerts](/img/diagrams/sns-alerts.svg)

**Implementation:**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: critical-alerts
spec:
  providerRef:
    name: production-aws
  topicName: critical-alerts
  subscriptions:
  - protocol: email
endpoint: oncall@company.com
filterPolicy:
      severity: ["critical", "emergency"]
  - protocol: sms
endpoint: +5511987654321
filterPolicy:
      severity: ["critical"]
  - protocol: https
endpoint: https://hooks.slack.com/services/XXXX
```

### Event Broadcasting with Filter Policy

Central publisher with multiple subscribers filtering by interest:

![SNS Pub/Sub](/img/diagrams/sns-pubsub.svg)

**Implementation:**
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: store-events
spec:
  providerRef:
    name: production-aws
  topicName: store-events
  subscriptions:
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:order-events
filterPolicy:
      event_type: ["order-placed", "order-fulfilled"]
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:payment-events
filterPolicy:
      event_type: ["payment-received", "payment-failed"]
  - protocol: sqs
endpoint: arn:aws:sqs:us-east-1:123456789012:inventory-events
filterPolicy:
      event_type: ["stock-updated", "reorder-needed"]
```

### Cross-Account Messaging

SNS topic in one account sending to SQS in another account:

![SNS Cross-Account](/img/diagrams/sns-cross-account.svg)

**Requires:** SQS queue policy allowing SNS from other account

## Related Resources

- [SQS Queue](/services/messaging/sqs)

  - [Lambda](/services/compute/lambda)
