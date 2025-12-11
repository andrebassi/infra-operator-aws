---
title: 'ElastiCache - Managed In-Memory Cache'
description: 'Managed in-memory cache with Redis and Memcached for high-performance applications'
sidebar_position: 1
---

Create and manage fully managed Redis or Memcached clusters on AWS for high-performance and scalable applications.

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

**IAM Policy - ElastiCache (elasticache-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticache:CreateCacheCluster",
        "elasticache:DeleteCacheCluster",
        "elasticache:DescribeCacheClusters",
        "elasticache:ModifyCacheCluster",
        "elasticache:CreateReplicationGroup",
        "elasticache:DeleteReplicationGroup",
        "elasticache:DescribeReplicationGroups",
        "elasticache:ModifyReplicationGroup",
        "elasticache:AddTagsToResource",
        "elasticache:RemoveTagsFromResource",
        "elasticache:ListTagsForResource",
        "elasticache:CreateCacheSubnetGroup",
        "elasticache:ModifyCacheSubnetGroup"
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
  --role-name infra-operator-elasticache-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator ElastiCache management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-elasticache-role \
  --policy-name ElastiCacheManagement \
  --policy-document file://elasticache-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-elasticache-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-elasticache-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon ElastiCache is a fully managed in-memory caching service that provides performance and scalability for modern applications. It supports Redis and Memcached as cache engines with built-in high availability, security, and monitoring.

**Features:**

- **In-Memory Cache**: Sub-millisecond latency for reads/writes
- **Two Supported Engines**: Redis (more features) and Memcached (simpler)
- **Cluster Mode**: Redis Cluster for horizontal scalability
- **Multi-AZ**: Automatic high availability with failover
- **Replication**: Automatic failover with synchronized replicas
- **Encryption**: Encryption in transit (TLS) and at rest
- **Authentication**: Redis AUTH with secure tokens
- **Automatic Backups**: Snapshots for data recovery (Redis)
- **Auto Scaling**: Increase/decrease nodes based on demand
- **VPC Support**: Network isolation in private subnets
- **Parameter Groups**: Custom cache configuration
- **Security Groups**: Granular network access control
- **ElastiCache Cluster Mode**: Data distribution across multiple nodes
- **Pub/Sub (Redis)**: Messaging between applications
- **Transactions (Redis)**: ACID transactions with MULTI/EXEC

**Status**: ⚠️ Requires LocalStack Pro or Real AWS

## Quick Start

**Redis Single Node:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: e2e-redis-simple
  namespace: default
spec:
  providerRef:
    name: localstack

  # Cluster identifier (2-63 characters)
  clusterID: e2e-redis-simple

  # Engine and version
  engine: redis
  engineVersion: "7.0"

  # Node size
  nodeType: cache.t3.micro

  # Number of nodes (single mode)
  numCacheNodes: 1

  # Tagging
  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing

  deletionPolicy: Delete
```

**Memcached Cluster:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: e2e-memcached
  namespace: default
spec:
  providerRef:
    name: localstack

  # Identifier
  clusterID: e2e-memcached-test

  # Engine
  engine: memcached
  engineVersion: "1.6.17"

  # Node size and quantity
  nodeType: cache.t3.micro
  numCacheNodes: 2

  # Tagging
  tags:
    environment: test
    managed-by: infra-operator
    engine: memcached
    purpose: e2e-testing

  deletionPolicy: Delete
```

**Redis Cluster Mode with Replication:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: e2e-redis-cluster
  namespace: default
spec:
  providerRef:
    name: localstack

  # Identifier
  clusterID: e2e-redis-cluster

  # Engine
  engine: redis
  engineVersion: "7.0"

  # Node size
  nodeType: cache.t3.small

  # Replication group description
  replicationGroupDescription: "E2E Test Redis Cluster"

  # Cluster Mode
  numNodeGroups: 2
  replicasPerNodeGroup: 1
  automaticFailoverEnabled: true
  multiAZEnabled: false

  # Tagging
  tags:
    environment: test
    managed-by: infra-operator
    cluster-mode: enabled
    purpose: e2e-testing

  deletionPolicy: Delete
```

**Production with Full HA:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: app-cache-prod
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Identifier
  clusterID: app-redis-prod

  # Engine
  engine: redis
  engineVersion: "7.0"

  # Cluster Mode
  replicationGroupDescription: "Production Redis Cluster"
  automaticFailoverEnabled: true
  multiAZEnabled: true

  # Shard configuration
  numNodeGroups: 2
  replicasPerNodeGroup: 2
  nodeType: cache.r6g.xlarge

  # Network
  subnetGroupName: private-cache-subnet
  securityGroupIds:
  - sg-0123456789abcdef0

  # Backup
  snapshotRetentionLimit: 7
  snapshotWindow: "03:00-05:00"

  # Security
  transitEncryptionEnabled: true
  atRestEncryptionEnabled: true
  authTokenRef:
    name: redis-auth
    key: token

  tags:
    Environment: production
    Critical: "true"

  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f elasticache.yaml
```

**Verify Status:**

```bash
kubectl get elasticachecluster
kubectl describe elasticachecluster e2e-redis-simple
kubectl get elasticachecluster e2e-redis-simple -o yaml
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource for authentication

  AWSProvider resource name

Unique cluster identifier (2 to 63 characters)

  **Rules:**
  - Alphanumerics and hyphens only
  - Cannot start or end with hyphen
  - Must be unique per region
  - Cannot be modified after creation

  **Example:**

  ```yaml
  clusterID: app-redis-cache-prod
  ```

Cache engine

  **Options:**
  - `redis` - Redis (recommended for production)
  - `memcached` - Memcached (simple, no persistence)

  **Example:**

  ```yaml
  engine: redis
  ```

Cache engine version

  **Examples:**
  - Redis: `7.0`, `6.2`, `5.0`
  - Memcached: `1.6.22`, `1.5.16`

  **Example:**

  ```yaml
  engineVersion: "7.0"
  ```

  **Note:** Always specify version precisely

Node type (CPU, RAM, performance)

  **cache.t3 Family (Burstable - Dev/Test):**
  - `cache.t3.micro` - 1 vCPU, 1 GB RAM (~$0.017/hour)
  - `cache.t3.small` - 1 vCPU, 2 GB RAM (~$0.034/hour)
  - `cache.t3.medium` - 1 vCPU, 4 GB RAM (~$0.068/hour)

  **cache.r6g Family (Memory Optimized):**
  - `cache.r6g.large` - 2 vCPU, 16 GB RAM (~$0.280/hour)
  - `cache.r6g.xlarge` - 4 vCPU, 32 GB RAM (~$0.560/hour)
  - `cache.r6g.2xlarge` - 8 vCPU, 64 GB RAM (~$1.120/hour)

  **cache.m6g Family (General Purpose):**
  - `cache.m6g.large` - 2 vCPU, 8 GB RAM (~$0.140/hour)
  - `cache.m6g.xlarge` - 4 vCPU, 16 GB RAM (~$0.280/hour)

  **Example:**

  ```yaml
  nodeType: cache.t3.micro
  ```

Number of nodes in cluster (single mode)

  **Range:** 1 - 300 nodes

  **Note:** Use `numCacheNodes` for single mode. For cluster mode, use `numNodeGroups`:

  ```yaml
  numCacheNodes: 1
  ```

### Optional Fields - Cluster Mode and HA

Replication group description (for cluster mode)

  **Used with:**
  - `numNodeGroups`: Number of shards
  - `replicasPerNodeGroup`: Replicas per shard
  - `automaticFailoverEnabled`: Automatic failover
  - `multiAZEnabled`: Multi-AZ

  **Example:**

  ```yaml
  replicationGroupDescription: "Production Redis Cluster"
  ```

Enable automatic failover (requires Multi-AZ and replication)

  **Benefits:**
  - Automatic failover in case of failure
  - No manual intervention
  - Application continues working
  - Requires more replicas

  **Example:**

  ```yaml
  automaticFailoverEnabled: true
  ```

  **Recommended:** true in production

Distribute nodes across multiple availability zones

  **Benefits:**
  - Tolerance to AZ failures
  - Automatic failover
  - ~50% cost increase

  **Example:**

  ```yaml
  multiAZEnabled: true
  ```

  **Recommended:** true in production

Number of shards in cluster mode

  **Range:** 1 - 500 shards

  **Note:** Use with `replicationGroupId` for cluster mode:

  ```yaml
  numNodeGroups: 3
  ```

Number of replicas per shard

  **Range:** 0 - 5 replicas per shard:

  ```yaml
  replicasPerNodeGroup: 2
  ```

  **Note:** Recommended 1-2 replicas in production

### Optional Fields - Network and Security

Cache subnet group name (private subnets)

  **Important:** MUST exist previously in AWS:

  ```yaml
  subnetGroupName: private-cache-subnet
  ```

  **Create subnet group via AWS CLI:**
  ```bash
  aws elasticache create-cache-subnet-group \
--cache-subnet-group-name private-cache-subnet \
--cache-subnet-group-description "Private subnets for ElastiCache" \
--subnet-ids subnet-xxx subnet-yyy
  ```

AWS Security Group IDs for access control

  **Example:**

  ```yaml
  securityGroupIds:
  - sg-0123456789abcdef0
  - sg-0987654321fedcba0
  ```

Enable encryption in transit (TLS)

  **Details:**
  - Encrypts data in motion on the network
  - Requires authToken if enabled
  - Performance: minimal overhead
  - Recommended: always enabled

  **Example:**

  ```yaml
  transitEncryptionEnabled: true
  ```

Enable encryption at rest

  **Details:**
  - Encrypts stored data
  - Valid only for Redis (not Memcached)
  - Performance: negligible impact
  - Regulatory compliance

  **Example:**

  ```yaml
  atRestEncryptionEnabled: true
  ```

Reference to Secret containing Redis authentication token (password)

  **Structure:**
  - `name`: Secret name
  - `namespace`: Secret namespace (optional)
  - `key`: Key within Secret

  **Example:**

  ```yaml
  authTokenRef:
    name: redis-auth
    key: token
  ```

  **Corresponding Secret:**
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: redis-auth
  stringData:
    token: MySecureRedisToken123!
  ```

  **Note:** Always use in production

### Optional Fields - Backup and Recovery

Days to retain snapshots (Redis only)

  **Range:** 0 - 35 days

  **Recommendations:**
  - Development: 0-1 days
  - Staging: 1-7 days
  - Production: 7-35 days

  **Example:**

  ```yaml
  snapshotRetentionLimit: 7
  ```

  **Note:** 0 = no automatic snapshots

Daily snapshot window (UTC, format HH:MM-HH:MM)

  **Default:** Automatically selected:

  ```yaml
  snapshotWindow: "03:00-05:00"  # 3AM-5AM UTC
  ```

  **Tip:** Choose low-usage time

Automatically update minor versions

  **Example:**

  ```yaml
  enableAutoMinorVersionUpgrade: true
  ```

### Optional Fields - Other

Key-value pairs for organization and billing

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: web-app
    Team: backend
    CostCenter: engineering
    ManagedBy: infra-operator
  ```

What happens to the cluster when the CR is deleted

  **Options:**
  - `Delete`: Cluster is deleted from AWS
  - `Retain`: Cluster remains in AWS but not managed
  - `Orphan`: Remove only management

  **Example:**

  ```yaml
  deletionPolicy: Retain  # For production
  ```

## Status Fields

After the cluster is created, the following status fields are populated:

`true` when cluster is available and ready to use

Current cluster state

  **Possible values:**
  - `creating` - Creating
  - `available` - Available and ready
  - `modifying` - Configuration being changed
  - `snapshotting` - Snapshot in progress
  - `deleting` - Being deleted
  - `failed` - Creation error

Full ElastiCache cluster ARN

  ```
  arn:aws:elasticache:us-east-1:123456789012:cluster:app-redis-cache
  ```

Cluster configuration endpoint (cluster mode)

  Cache hostname for connection

Connection port


Cluster primary endpoint (replication group)

  Primary node hostname

Connection port (default: 6379 Redis, 11211 Memcached)


Read endpoint (read replicas)

  Hostname for reads

Connection port


List of individual endpoints for each node

  Individual node hostname

Node port


Cluster node type (e.g., cache.t3.micro)

Running engine version

List of member clusters (for replication groups)

Timestamp when cluster was created

Timestamp of last sync with AWS

Cluster status conditions

  Condition type (e.g., Ready, Available)

Condition status (True, False, Unknown)

Reason for current status

Descriptive message


## Examples

### Redis Single Node for Development

Simple cluster for development and testing:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: dev-redis
  namespace: default
spec:
  providerRef:
    name: dev-aws

  clusterID: dev-redis-cache
  engine: redis
  engineVersion: "7.0"

  # Small instance for dev
  nodeType: cache.t3.micro
  numCacheNodes: 1

  # No Multi-AZ to save costs
  multiAZEnabled: false

  # No backup
  snapshotRetentionLimit: 0

  subnetGroupName: dev-cache-subnet

  # No encryption for better performance in dev
  transitEncryptionEnabled: false

  tags:
    Environment: development
    Application: my-app

  deletionPolicy: Delete
```

### Redis Cluster Mode with Full HA

Cluster for production with high availability:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: production-redis
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Replication group for cluster mode
  clusterID: app-redis-prod
  replicationGroupDescription: "Production Redis Cluster"

  # Engine
  engine: redis
  engineVersion: "7.0"

  # Cluster configuration
  nodeType: cache.r6g.xlarge
  numNodeGroups: 3          # 3 shards
  replicasPerNodeGroup: 2   # 2 replicas per shard

  # High Availability
  multiAZEnabled: true
  automaticFailoverEnabled: true

  # Network
  subnetGroupName: private-cache-subnet
  securityGroupIds:
  - sg-0123456789abcdef0

  # Backup
  snapshotRetentionLimit: 7
  snapshotWindow: "03:00-05:00"

  # Security
  transitEncryptionEnabled: true
  atRestEncryptionEnabled: true
  authTokenRef:
    name: redis-auth
    key: token

  tags:
    Environment: production
    Critical: "true"
    Team: backend
    CostCenter: infrastructure

  deletionPolicy: Retain
```

### Redis with Full Encryption and Auth

Critical cluster with maximum security:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: secure-redis
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterID: secure-redis-prod
  replicationGroupDescription: "Secure Redis Cluster"

  engine: redis
  engineVersion: "7.0"

  # Performance nodes
  nodeType: cache.r6g.2xlarge
  numNodeGroups: 4
  replicasPerNodeGroup: 2

  # HA
  multiAZEnabled: true
  automaticFailoverEnabled: true

  # Private network
  subnetGroupName: critical-cache-subnet
  securityGroupIds:
  - sg-critical-redis-001

  # Maximum security
  transitEncryptionEnabled: true
  atRestEncryptionEnabled: true
  authTokenRef:
    name: redis-auth-token
    key: password

  # Full backups
  snapshotRetentionLimit: 35
  snapshotWindow: "02:00-03:00"
  preferredMaintenanceWindow: "sun:03:00-sun:04:00"

  tags:
    Environment: production
    CriticalData: "true"
    BackupRequired: "true"
    Compliance: "required"

  deletionPolicy: Retain
```

### Memcached for Session Storage

Memcached cluster for storing sessions:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: session-cache
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterID: session-cache-prod

  engine: memcached
  engineVersion: "1.6.22"

  nodeType: cache.t3.small
  numCacheNodes: 3

  subnetGroupName: web-cache-subnet
  securityGroupIds:
  - sg-memcached-001

  # Memcached does not support Multi-AZ
  multiAZEnabled: false

  # No backup (data does not persist)
  snapshotRetentionLimit: 0

  tags:
    Environment: production
    Type: session-cache
    Application: web-app

  deletionPolicy: Delete
```

### Redis for Rate Limiting and Leaderboards

Cluster optimized for fast operations:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: realtime-redis
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterID: realtime-cache
  replicationGroupDescription: "Realtime Redis Cluster"

  engine: redis
  engineVersion: "7.0"

  # Optimized for low latency
  nodeType: cache.r6g.large
  numNodeGroups: 2
  replicasPerNodeGroup: 1

  # Fast failover
  automaticFailoverEnabled: true
  multiAZEnabled: true

  subnetGroupName: realtime-cache-subnet
  securityGroupIds:
  - sg-realtime-redis-001

  # Security
  transitEncryptionEnabled: true
  authTokenRef:
    name: realtime-redis-auth
    key: token

  # Normal backups
  snapshotRetentionLimit: 3
  snapshotWindow: "02:00-03:00"

  tags:
    Environment: production
    Type: realtime-cache
    UseCase: rate-limiting-leaderboards

  deletionPolicy: Retain
```

## Verification

### Verify Status via kubectl

**Command:**

```bash
# List all ElastiCache clusters
kubectl get elasticachecluster

# Get detailed information
kubectl get elasticachecluster app-cache -o yaml

# Watch creation in real-time
kubectl get elasticachecluster app-cache -w

# View events and status
kubectl describe elasticachecluster app-cache
```

### Verify on AWS

**AWS CLI:**

```bash
# List clusters
aws elasticache describe-cache-clusters \
      --query 'CacheClusters[].{Id:CacheClusterId,Status:CacheClusterStatus,Engine:Engine,NodeType:CacheNodeType}' \
      --output table

# Get full details
aws elasticache describe-cache-clusters \
      --cache-cluster-id app-redis-cache \
      --output json | jq '.CacheClusters[0]'

# View connection endpoint
aws elasticache describe-cache-clusters \
      --cache-cluster-id app-redis-cache \
      --query 'CacheClusters[0].CacheNodes[0].Endpoint'

# Test connection (Redis)
redis-cli -h app-redis-cache.xxxxx.cache.amazonaws.com \
              -p 6379 \
              -a MyToken123! \
              PING

# View snapshots
aws elasticache describe-snapshots \
      --cache-cluster-id app-redis-cache

```

**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws elasticache describe-cache-clusters

aws elasticache describe-cache-clusters \
      --cache-cluster-id app-redis-cache
```

**redis-cli (Redis):**

```bash
# Connect to cache
redis-cli -h <endpoint> -p 6379 -a <password>

# Test connection
PING
# Response: PONG

# View server information
INFO server

# View memory usage
INFO memory

# View all keys
KEYS *

# Disconnect
EXIT
```

**memcached-tool (Memcached):**

```bash
# Connect and get stats
echo "stats" | nc <endpoint> 11211

# View largest memory consumer
memcached-tool <endpoint>:11211 display

# Flush all (clear data)
echo "flush_all" | nc <endpoint> 11211
```

### Expected Output

**Example:**

```yaml
status:
  ready: true
  clusterStatus: available
  cacheClusterARN: arn:aws:elasticache:us-east-1:123456789012:cluster:app-redis-cache
  primaryEndpoint:
    address: app-redis-cache.xxxxx.cache.amazonaws.com
    port: 6379
  cacheNodeType: cache.t3.micro
  engineVersion: "7.0"
  clusterCreateTime: "2025-11-22T20:30:15Z"
  lastSyncTime: "2025-11-22T20:45:22Z"
  conditions:
  - type: Ready
    status: "True"
    reason: ClusterAvailable
    message: "ElastiCache cluster is available"
    lastTransitionTime: "2025-11-22T20:35:10Z"
```

## Troubleshooting

### Cluster stuck in creating for more than 30 minutes

**Symptoms:** `cacheClusterStatus: creating` indefinitely

**Common causes:**
1. Subnet group does not exist or is invalid
2. Security group not found
3. Cluster quota reached
4. Connectivity issue with AWS

**Solutions:**
```bash
# Check detailed status
kubectl describe elasticachecluster app-cache

# View operator logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100 | grep -i elasticache

# Verify subnet group exists
aws elasticache describe-cache-subnet-groups \
      --cache-subnet-group-name cache-subnet-group

# Verify AWSProvider is ready
kubectl get awsprovider
kubectl describe awsprovider production-aws

# Force sync
kubectl annotate elasticachecluster app-cache \
      force-sync="$(date +%s)" --overwrite
```

### Connection timeout when connecting to cache

**Symptoms:** `Connection refused` or `Cannot connect to host`

**Common causes:**
1. Security group does not allow connection
2. Cache is not accessible (private)
3. Incorrect endpoint/port
4. Network not routed correctly

**Solutions:**
```bash
# Get correct endpoint
aws elasticache describe-cache-clusters \
      --cache-cluster-id app-redis-cache \
      --query 'CacheClusters[0].CacheNodes[0].Endpoint'

# Verify security group allows inbound
aws ec2 describe-security-groups \
      --group-ids sg-0123456789abcdef0 \
      --query 'SecurityGroups[0].IpPermissions'

# MUST have rule like:
# IpProtocol: tcp, FromPort: 6379, ToPort: 6379
# CidrIp: 10.0.0.0/16 (your VPC)

# If connecting from outside VPC:
# 1. Cache MUST be publicly accessible (not recommended)
# 2. Or use bastion/jump host
# 3. Or use VPN

# Test with telnet/nc
nc -zv app-redis-cache.xxxxx.cache.amazonaws.com 6379

# Test with redis-cli
redis-cli -h app-redis-cache.xxxxx.cache.amazonaws.com \
             -p 6379 \
             -a password \
             PING
```

### Out of Memory (cache full)

**Symptoms:** Error `OOM command not allowed` when writing data

**Causes:**
1. Data grew beyond expected
2. TTL not configured (data never expires)
3. Insufficient node size

**Solutions:**
```bash
# Connect to Redis and view usage
redis-cli -h <endpoint> -a password
INFO memory

# Increase node size (resize)
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"cacheNodeType":"cache.t3.small"}}'

# Increase number of nodes (scale up)
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"numCacheNodes":2}}'

# Clear data manually (Redis)
redis-cli -h <endpoint> -a password
FLUSHDB  # Clear only current database
FLUSHALL # Clear all databases

# Implement TTL in application
SET key value EX 3600  # Expires in 1 hour
```

### Slow Performance/High Latency

**Symptoms:** Slow read/write even in cache

**Causes:**
1. CPU or memory saturated
2. Congested network
3. Aggressive eviction policy
4. Insufficient node size

**Solutions:**
```bash
# View slow queries (Redis)
redis-cli -h <endpoint> -a password
SLOWLOG GET 10

# Increase node size
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"cacheNodeType":"cache.r6g.xlarge"}}'

# Increase replicas to distribute load
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"replicasPerNodeGroup":2}}'

# Enable cluster mode for distribution
# Requires recreating cluster
```

### Slow failover or not working

**Symptoms:** After failure, cache stays offline for long time

**Cause:** Automatic failover may be disabled or insufficient replicas

**Solutions:**
```bash
# Verify automatic failover
aws elasticache describe-replication-groups \
      --replication-group-id app-redis-rg \
      --query 'ReplicationGroups[0].AutomaticFailoverEnabled'

# Enable if not
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"automaticFailoverEnabled":true}}'

# Verify Multi-AZ
aws elasticache describe-replication-groups \
      --replication-group-id app-redis-rg \
      --query 'ReplicationGroups[0].MultiAZ'

# Enable Multi-AZ
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"multiAZEnabled":true}}'

# Test failover manually
# WARNING: Causes downtime!
aws elasticache test-failover \
      --replication-group-id app-redis-rg

# View failover progress
aws elasticache describe-replication-groups \
      --replication-group-id app-redis-rg
```

### High ElastiCache costs

**Symptoms:** AWS account with unexpected cache expenses

**Common causes:**
1. Nodes too large
2. Multi-AZ in dev/test
3. Snapshots retained too long
4. Cross-region replication

**Solutions:**
```bash
# Calculate current cost
# Use AWS Pricing Calculator
# cache.t3.micro: ~$0.017/hour * 730 = ~$12/month
# cache.r6g.xlarge: ~$0.560/hour * 730 = ~$409/month

# Reduce class (if possible)
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"cacheNodeType":"cache.t3.micro"}}'

# Reduce replicas in dev
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"replicasPerNodeGroup":0}}'

# Reduce snapshot retention
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"snapshotRetentionLimit":3}}'

# For dev, delete after use
kubectl delete elasticachecluster dev-cache
```

### Snapshot fails or takes too long

**Symptoms:** Status stays in `snapshotting` for hours

**Causes:**
1. Cache too large
2. IO saturated during snapshot
3. Many transactions during snapshot

**Solutions:**
```bash
# View snapshot status
aws elasticache describe-snapshots \
      --cache-cluster-id app-redis-cache

# View completed snapshots
aws elasticache describe-snapshots \
      --query 'Snapshots[].{SnapshotName:SnapshotName,Status:SnapshotStatus}'

# Increase snapshot window
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"snapshotWindow":"02:00-04:00"}}'

# Create manual snapshot (Redis)
aws elasticache create-snapshot \
      --cache-cluster-id app-redis-cache \
      --snapshot-name manual-backup-$(date +%Y%m%d-%H%M%S)

# Delete old snapshots
aws elasticache delete-snapshot \
      --snapshot-name snapshot-name
```

### Error deleting cluster (finalizer stuck)

**Symptoms:** `kubectl delete elasticachecluster` pending indefinitely

**Cause:** Finalizer cannot delete cluster

**Solutions:**
```bash
# View details
kubectl describe elasticachecluster app-cache

# View finalizers
kubectl get elasticachecluster app-cache -o yaml | grep finalizers

# Option 1: Change deletionPolicy before deleting
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"deletionPolicy":"Retain"}}'

# Then delete
kubectl delete elasticachecluster app-cache

# Option 2: Force delete CR
kubectl patch elasticachecluster app-cache \
      -p '{"metadata":{"finalizers":[]}}' \
      --type=merge

# Then delete manually on AWS if needed
aws elasticache delete-cache-cluster \
      --cache-cluster-id app-redis-cache
```

### Invalid or expired authentication token

**Symptoms:** Error `WRONGPASS invalid username-password pair` when connecting

**Cause:** Incorrect, expired, or unconfigured auth token

**Solutions:**
```bash
# Verify auth token is configured
kubectl get elasticachecluster app-cache -o yaml | grep authToken

# Test connecting without auth
redis-cli -h <endpoint> -p 6379 PING

# If fails, check on AWS
aws elasticache describe-cache-clusters \
      --cache-cluster-id app-redis-cache

# Connect with correct token
redis-cli -h <endpoint> -p 6379 -a MyToken123! PING

# Update token (requires restart)
kubectl patch elasticachecluster app-cache \
      --type merge \
      -p '{"spec":{"authToken":"NewToken456!"}}'

# Verify transitEncryption is enabled
kubectl get elasticachecluster app-cache -o yaml | grep transitEncryption
```

## Best Practices

:::note Best Practices

- **Enable Multi-AZ for production** — Automatic failover in case of failure, ~50% higher cost but essential for critical data
- **Enable encryption everywhere** — Transit encryption (TLS) always enabled, at-rest encryption for sensitive data, auth tokens in production for regulatory compliance
- **Use strong auth tokens** — Complex tokens (32+ characters), store in Kubernetes Secret, rotate regularly, never hardcode, use different tokens per environment
- **Use Cluster Mode for scale** — Enable for >100 GB of data, automatic sharding, unlimited horizontal scalability, better performance than single mode
- **Configure replicas properly** — Minimum 1 replica in production, 2 replicas for high concurrency, read-heavy workloads benefit from more replicas
- **Right-size node types** — Start with t3.micro, monitor memory usage, use r6g for memory-intensive or m6g for general purpose workloads
- **Implement TTL and eviction** — Always use TTL on keys, set eviction policy (allkeys-lru), avoid data that never expires, monitor evictions
- **Configure backups** — Automatic snapshots (7-35 days), manual backups before upgrades, test restore regularly, copy snapshots to another region
- **Use connection pooling** — Don't create new connections per request, reuse existing connections, configure appropriate timeout
- **Tag all resources** — Environment (dev/staging/prod), application, cost center, owner/team, critical flag
- **Secure network access** — Cache in private subnet, restrictive security group, only necessary IPs/SGs, never 0.0.0.0/0

:::

## Usage Patterns

### 1. Session Storage (Memcached)

Store user sessions:

```yaml
engine: memcached
nodeType: cache.t3.small
numCacheNodes: 3
```

**Benefits:**
- Fast and simple
- Share sessions between instances
- No persistence (OK for sessions)

### 2. Database Query Cache (Redis)

Cache query results:

```yaml
engine: redis
nodeType: cache.r6g.large
replicationGroupDescription: "Query Cache Redis"
snapshotRetentionLimit: 7
```

**Benefits:**
- Reduce database load
- Sub-millisecond latency
- Automatic backup

### 3. Rate Limiting (Redis)

Limit requests per user/IP:

```yaml
engine: redis
nodeType: cache.r6g.large
numNodeGroups: 2
replicasPerNodeGroup: 1
automaticFailoverEnabled: true
```

**Benefits:**
- Very fast for increments
- Distributed across multiple nodes
- Highly reliable

### 4. Real-time Leaderboards (Redis)

Real-time ranking with sorted sets:

```yaml
engine: redis
nodeType: cache.r6g.xlarge
numNodeGroups: 3
replicasPerNodeGroup: 2
```

**Benefits:**
- O(log N) operations
- Fast score updates
- Efficient queries

### 5. Pub/Sub Messaging (Redis)

Messaging system between services:

```yaml
engine: redis
nodeType: cache.r6g.large
automaticFailoverEnabled: true
multiAZEnabled: true
```

**Benefits:**
- Low latency
- Broadcast to multiple subscribers
- Simple pattern matching

### 6. Analytics Data (Redis)

Real-time data aggregation:

```yaml
engine: redis
nodeType: cache.r6g.2xlarge
numNodeGroups: 4
replicasPerNodeGroup: 2
snapshotRetentionLimit: 7
```

**Benefits:**
- HyperLogLog for approximate counting
- Bitmaps for fast operations
- Streams for temporal data

## Related Resources

- [VPC and Subnets](/services/networking/vpc)

  - [Security Groups](/services/networking/security-group)

  - [RDS Instance](/services/database/rds)

  - [Lambda](/services/compute/lambda)

  - [DynamoDB](/services/database/dynamodb)
