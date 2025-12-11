---
title: 'DynamoDB - NoSQL Database'
description: 'Serverless and scalable NoSQL database on AWS'
sidebar_position: 1
---

Create fully managed, scalable, and high-performance NoSQL tables on AWS.

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

To use IRSA in production, you need to create an IAM Role with the required permissions:

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

**IAM Policy - DynamoDB (dynamodb-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "dynamodb:CreateTable",
        "dynamodb:DeleteTable",
        "dynamodb:DescribeTable",
        "dynamodb:UpdateTable",
        "dynamodb:ListTables",
        "dynamodb:TagResource",
        "dynamodb:UntagResource",
        "dynamodb:ListTagsOfResource",
        "dynamodb:UpdateTimeToLive",
        "dynamodb:DescribeTimeToLive",
        "dynamodb:UpdateContinuousBackups",
        "dynamodb:DescribeContinuousBackups"
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
  --role-name infra-operator-dynamodb-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator DynamoDB management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-dynamodb-role \
  --policy-name DynamoDBManagement \
  --policy-document file://dynamodb-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-dynamodb-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-dynamodb-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon DynamoDB is a fully managed NoSQL database service that offers:

- **Serverless**: No need to manage infrastructure
- **Automatic Scalability**: Scales automatically with your demand
- **Predictable Performance**: Millisecond latency
- **Highly Available**: Automatic replication across multiple AZs
- **Security**: Encryption at rest, VPC endpoints, and granular access control
- **Flexible Billing Models**: PAY_PER_REQUEST or PROVISIONED capacity

## Quick Start

The simplest configuration of a DynamoDB table:

**Advanced Table with GSI:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: e2e-test-table
  namespace: default
spec:
  tableName: e2e-test-users-table
  providerRef:
    name: localstack
  billingMode: PAY_PER_REQUEST
  hashKey:
    name: UserID
    type: "S"
  rangeKey:
    name: Timestamp
    type: "N"
  attributes:
  - name: Email
    type: "S"
  globalSecondaryIndexes:
  - indexName: EmailIndex
    hashKey: Email
    projectionType: ALL
  streamEnabled: true
  streamViewType: NEW_AND_OLD_IMAGES
  tags:
    Environment: test
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Simple Table:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: e2e-simple-table
  namespace: default
spec:
  tableName: e2e-simple-table
  providerRef:
    name: localstack
  billingMode: PAY_PER_REQUEST
  hashKey:
    name: ID
    type: "S"
  deletionPolicy: Delete
```

**Apply:**

```bash
kubectl apply -f dynamodb.yaml
```

**Verify Status:**

```bash
kubectl get dynamodbtable e2e-test-table
kubectl describe dynamodbtable e2e-test-table
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource

  Name of the AWSProvider resource

DynamoDB table name (1 to 255 alphanumeric characters and underscores)

  **Requirements:**
  - Must be unique within the region
  - Case-sensitive
  - Example: `users`, `orders_v2`, `product_catalog`

Billing model for the table

  **Options:**
  - `PAY_PER_REQUEST`: Pay per request (ideal for variable loads)
  - `PROVISIONED`: Configure fixed capacity (ideal for predictable workloads)

  :::tip

Use PAY_PER_REQUEST for applications with variable traffic or in development
:::


List of attributes used in keys and indexes

  Attribute name

Attribute type:
- `S`: String
- `N`: Number
- `B`: Binary

**Note:** Only define attributes used in keySchema or indexes

Defines the table's primary key

  Attribute name (must be in `attributes`)

Key type:
- `HASH`: Partition key (required)
- `RANGE`: Sort key (optional)

**Standard:**
  - Must always have exactly 1 HASH key (partition key)
  - Optionally can have 1 RANGE key (sort key)

### Optional Fields

Capacity configuration for PROVISIONED mode

  Provisioned read units

Provisioned write units

:::note

Required when `billingMode: PROVISIONED`
:::


Global secondary indexes for alternative queries

  Unique index name

Index keys (can be different from primary key)

Which attributes to include in the index:
- `type: ALL`: All attributes
- `type: KEYS_ONLY`: Keys only
- `type: INCLUDE` with `nonKeyAttributes`: Specific attributes

Separate capacity for the index (if PROVISIONED)

Local secondary indexes (same partition key, different sort key)

  Index name

Must contain identical HASH key to the table + different RANGE key

Attributes to include

:::note

LSI has a 10 GB limit per partition key value
:::


Enable DynamoDB Streams to capture changes

  Type of information in the stream:
- `NEW_IMAGE`: New item only
- `OLD_IMAGE`: Previous item only
- `NEW_AND_OLD_IMAGES`: Both
- `KEYS_ONLY`: Keys only

Enable point-in-time recovery for backup/restore

  If true, allows restore to any point in the last 35 days

Encryption configuration

  Enable encryption at rest

Key type:
- `AWS_OWNED`: AWS-managed key (no cost)
- `AWS_MANAGED`: AWS KMS managed key (no cost, more control)
- `CUSTOMER_MANAGED`: Your custom KMS key (additional cost)

KMS key ARN (required if `type: CUSTOMER_MANAGED`)

Key-value pairs for organization and billing

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: myapp
    Team: platform
    CostCenter: engineering
  ```

What happens to the table when the CR is deleted

  **Options:**
  - `Delete`: Table is deleted from AWS
  - `Retain`: Table remains in AWS
  - `Orphan`: Table remains but CR loses ownership

## Status Fields

After the table is created, the following status fields are populated:

Full table ARN (e.g., `arn:aws:dynamodb:us-east-1:123456789012:table/users`)

Current table state:
  - `CREATING`: Table is being created
  - `ACTIVE`: Ready for use
  - `DELETING`: Being deleted
  - `UPDATING`: Configuration is being updated

Number of items in the table

Total table size in bytes

`true` when the table is ACTIVE and ready for queries

Timestamp of the last sync with AWS

## Examples

### Simple Table with Partition Key

Table for storing user profiles:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: user-profiles
spec:
  providerRef:
    name: production-aws
  tableName: user_profiles
  billingMode: PAY_PER_REQUEST

  # Attributes: userId and email are strings
  attributes:
  - name: userId
    type: S
  - name: email
    type: S

  # Partition key only
  keySchema:
  - attributeName: userId
    keyType: HASH

  tags:
    Application: user-service
    Environment: production

  deletionPolicy: Retain
```

### Table with Partition Key + Sort Key

Table for storing orders with history:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: orders-table
spec:
  providerRef:
    name: production-aws
  tableName: orders
  billingMode: PROVISIONED
  billingModeConfig:
    readCapacityUnits: 10
    writeCapacityUnits: 10

  attributes:
  - name: customerId
    type: S
  - name: orderDate
    type: S  # ISO 8601 format
  - name: orderId
    type: S

  keySchema:
  - attributeName: customerId
    keyType: HASH       # Partition key
  - attributeName: orderDate
    keyType: RANGE      # Sort key

  tags:
    Application: order-service
    Environment: production

  deletionPolicy: Retain
```

### Table with Global Secondary Indexes (GSI)

Table for queries by email or by status:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: users-with-gsi
spec:
  providerRef:
    name: production-aws
  tableName: users_v2
  billingMode: PAY_PER_REQUEST

  attributes:
  - name: userId
    type: S
  - name: email
    type: S
  - name: status
    type: S
  - name: createdAt
    type: S

  keySchema:
  - attributeName: userId
    keyType: HASH

  # GSI for query by email
  globalSecondaryIndexes:
  - indexName: email-index
    keys:
    - attributeName: email
      keyType: HASH
    projection:
      type: ALL

  # GSI for query by status + date
  - indexName: status-created-index
    keys:
    - attributeName: status
      keyType: HASH
    - attributeName: createdAt
      keyType: RANGE
    projection:
      type: KEYS_ONLY  # Keys only for cost savings

  tags:
    Application: user-service

  deletionPolicy: Retain
```

### Table with Streams (for Lambda)

Table that captures changes to trigger events:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: events-table
spec:
  providerRef:
    name: production-aws
  tableName: domain_events
  billingMode: PAY_PER_REQUEST

  attributes:
  - name: aggregateId
    type: S
  - name: eventTime
    type: N

  keySchema:
  - attributeName: aggregateId
    keyType: HASH
  - attributeName: eventTime
    keyType: RANGE

  # Enable DynamoDB Streams
  streamSpecification:
    streamViewType: NEW_AND_OLD_IMAGES

  tags:
    Application: event-sourcing

  deletionPolicy: Retain
```

### Table with PITR (Point-in-Time Recovery)

Production table with automatic backup:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: critical-data
spec:
  providerRef:
    name: production-aws
  tableName: critical_data
  billingMode: PROVISIONED
  billingModeConfig:
    readCapacityUnits: 25
    writeCapacityUnits: 25

  attributes:
  - name: dataId
    type: S

  keySchema:
  - attributeName: dataId
    keyType: HASH

  # Enable Point-in-Time Recovery
  pointInTimeRecoverySpecification:
    pointInTimeRecoveryEnabled: true

  # Encryption with KMS key
  encryption:
    enabled: true
    type: AWS_MANAGED

  tags:
    Environment: production
    BackupRequired: "true"
    CriticalData: "true"

  deletionPolicy: Retain
```

### Table with Local Secondary Index (LSI)

Table with local index for alternative queries:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: leaderboard-table
spec:
  providerRef:
    name: production-aws
  tableName: game_leaderboard
  billingMode: PAY_PER_REQUEST

  attributes:
  - name: gameId
    type: S
  - name: playerId
    type: S
  - name: score
    type: N
  - name: timestamp
    type: N

  keySchema:
  - attributeName: gameId
    keyType: HASH
  - attributeName: playerId
    keyType: RANGE

  # LSI: query by score instead of playerId
  localSecondaryIndexes:
  - indexName: score-index
    keys:
    - attributeName: gameId
      keyType: HASH       # Same partition key
    - attributeName: score
      keyType: RANGE      # Different sort key
    projection:
      type: ALL

  tags:
    Application: gaming

  deletionPolicy: Delete
```

## Verification

### Verify Status via kubectl

**Command:**

```bash
# List all tables
kubectl get dynamodb

# Get detailed information
kubectl get dynamodb users-table -o yaml

# Watch creation in real time
kubectl get dynamodb users-table -w

# View status fields only
kubectl get dynamodb users-table -o jsonpath='{.status}'
```

### Verify in AWS

**AWS CLI:**

```bash
# List tables
aws dynamodb list-tables --region us-east-1

# Describe specific table
aws dynamodb describe-table \
      --table-name users \
      --region us-east-1 \
      --output json | jq '.Table | {Name, Status, ItemCount, TableArn}'

# View size and items
aws dynamodb describe-table \
      --table-name users \
      --query 'Table.{Status,Items:ItemCount,Size:TableSizeBytes}' \
      --region us-east-1

# List indexes
aws dynamodb describe-table \
      --table-name users \
      --query 'Table.GlobalSecondaryIndexes[*].{Name:IndexName,Status:IndexStatus}' \
      --region us-east-1
```

**LocalStack:**

```bash
# Point to LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

# List tables
aws dynamodb list-tables

# Describe table
aws dynamodb describe-table --table-name users

# Scan all items
aws dynamodb scan --table-name users
```

### Expected Output

**Example:**

```yaml
status:
  tableArn: arn:aws:dynamodb:us-east-1:123456789012:table/users
  tableStatus: ACTIVE
  itemCount: 1234
  tableSizeBytes: 5242880
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### Table stuck in CREATING

**Symptoms:** `tableStatus: CREATING` for more than 5 minutes

**Common causes:**
1. Invalid AWSProvider credentials
2. Table limit reached in the account
3. AWS connectivity problem

**Solutions:**
```bash
# Verify AWSProvider status
kubectl describe awsprovider production-aws

# View controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      -f | grep -i dynamodb

# View resource events
kubectl describe dynamodb users-table

# Check table limit in AWS
aws dynamodb list-tables | jq '.TableNames | length'
```

### Throughput exceeded (Provisioned)

**Error:** `ProvisionedThroughputExceededException`

**Cause:** Application exceeded configured capacity

**Solutions:**
```bash
# Increase capacity
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "replace", "path": "/spec/billingModeConfig/readCapacityUnits", "value":25}]'

# Or switch to PAY_PER_REQUEST
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "replace", "path": "/spec/billingMode", "value":"PAY_PER_REQUEST"}]'
```

### GSI not creating

**Symptoms:** GSI status remains pending, table in UPDATING indefinitely

**Causes:**
1. GSI references attribute not defined in `attributes`
2. Capacity problem in PROVISIONED mode
3. Index name conflict

**Solutions:**
```bash
# Check if attribute exists
kubectl get dynamodb users-table -o yaml | grep -A 5 "attributes:"

# Check errors in events
kubectl describe dynamodb users-table | grep -i "events" -A 10

# Remove problematic GSI
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "remove", "path": "/spec/globalSecondaryIndexes/0"}]'
```

### Deletion stuck or timeout

**Symptoms:** Deletion takes too long or doesn't complete

**Common cause:** Too much data to backup before deleting

**Solutions:**
```bash
# View deletion state
kubectl get dynamodb users-table -o yaml

# Force delete if necessary (last resort)
kubectl patch dynamodb users-table \
      -p '{"metadata":{"finalizers":[]}}' \
      --type=merge

# Or use deletionPolicy: Orphan to not delete the table
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "replace", "path": "/spec/deletionPolicy", "value":"Orphan"}]'

# Then delete CR
kubectl delete dynamodb users-table
```

### Hot partition (degraded performance)

**Symptoms:** High latency, throttling even with sufficient capacity

**Cause:** Uneven partition key distribution (e.g., many items on the same day)

**Solutions:**
```bash
# Add random numbers to partition key
# userId#2025-11-22#randomNumber

# Use GSI with better distribution
# Redistribute data through application
```

### Very high costs

**Symptoms:** AWS account with unexpected DynamoDB costs

**Possible causes:**
1. Inefficient scans without filters
2. PROVISIONED mode with high capacity
3. Unnecessary LSI/GSI

**Solutions:**
```bash
# Switch to PAY_PER_REQUEST if load is variable
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "replace", "path": "/spec/billingMode", "value":"PAY_PER_REQUEST"}]'

# Remove unused indexes
kubectl patch dynamodb users-table --type='json' \
      -p='[{"op": "remove", "path": "/spec/globalSecondaryIndexes"}]'

# View estimated cost
aws ce get-cost-and-usage \
      --time-period Start=2025-11-01,End=2025-11-22 \
      --granularity DAILY \
      --metrics BlendedCost \
      --filter file://dynamodb-filter.json \
      --group-by Type=DIMENSION,Key=SERVICE
```

## Best Practices

:::note Best Practices

- **Design partition keys carefully** — Distribute evenly, use prefixes (USER#, ORDER#) for entities, avoid hot keys
- **Choose billing mode wisely** — PAY_PER_REQUEST for development/unpredictable, PROVISIONED with autoscaling for predictable production
- **Enable backups** — PITR for production (35 days backup), use Streams for event sourcing
- **Optimize queries** — Query with key is O(1), scan is O(n), use projection for only necessary fields
- **Use GSI for alternative queries** — Global Secondary Indexes for non-primary key queries
- **Enable encryption** — No additional cost, always enable for compliance
- **Tag for cost allocation** — Environment, application, cost center tags for billing visibility

:::

## Architecture Patterns

### Single-Table Model

Use a single table for multiple entities:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDB
metadata:
  name: app-db
spec:
  providerRef:
    name: production-aws
  tableName: app_db
  billingMode: PAY_PER_REQUEST

  attributes:
  - name: pk
    type: S  # USER#user123, ORDER#order456
  - name: sk
    type: S  # PROFILE, ORDERS#2025-11-22
  - name: gsi1pk
    type: S  # EMAIL#email@example.com
  - name: gsi1sk
    type: S  # For sorting

  keySchema:
  - attributeName: pk
    keyType: HASH
  - attributeName: sk
    keyType: RANGE

  globalSecondaryIndexes:
  - indexName: gsi1
    keys:
    - attributeName: gsi1pk
      keyType: HASH
    - attributeName: gsi1sk
      keyType: RANGE
    projection:
      type: ALL

  tags:
    Architecture: single-table
```

### Event Sourcing Model

Table optimized for event sourcing:

```
PK: AggregateID#Type (e.g., USER#user123)
SK: Version#Timestamp (e.g., 0001#2025-11-22T20:00:00Z)
Attributes: eventType, eventData, causedBy, etc
```

### Prefix Query Pattern

Use prefixes in the key for efficient queries:

```
PK: USER#123
SK: PROFILE#2025-11-22 (fetch everything for that user)
SK: ORDER#2025-11-22#order1
SK: ORDER#2025-11-20#order2
```

## Use Cases

### User Profiles & Sessions

**Example:**

```yaml
# Users with active sessions
tableName: user_sessions
# PK: userId (USER#user123)
# SK: sessionId (SESSION#abc123)
# TTL for automatic cleanup
```

### Gaming Leaderboards

**Example:**

```yaml
# Player ranking by game
tableName: game_leaderboards
# PK: gameId
# SK: score (with playerId for tiebreaker)
# GSI: userId to fetch player rankings
```

### IoT Time Series

**Example:**

```yaml
# Sensor data
tableName: sensor_data
# PK: deviceId (SENSOR#device123)
# SK: timestamp (ordered)
# Streams to process data in real time
```

### Shopping Cart

**Example:**

```yaml
# Shopping carts
tableName: shopping_carts
# PK: userId
# SK: cartId (for multiple carts)
# TTL for abandoned carts
```

### Message Queue

**Example:**

```yaml
# Message queue (alternative to SQS)
tableName: message_queue
# PK: queueId
# SK: timestamp (natural order)
# Streams to process in real-time
# PITR for reprocessing
```

## Related Resources

- [RDS](/services/database/rds)

  - [Lambda](/services/compute/lambda)

  - [SQS](/services/messaging/sqs)

  - [ElastiCache](/services/caching/elasticache)

  - [Guide: Single-Table Design](/guides/dynamodb-single-table)
