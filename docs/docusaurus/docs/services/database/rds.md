---
title: 'RDS Instance - Relational Database'
description: 'Managed relational databases (PostgreSQL, MySQL, MariaDB, SQL Server, Oracle)'
sidebar_position: 2
---

Create and manage fully managed and scalable relational databases on AWS with PostgreSQL, MySQL, MariaDB, SQL Server, or Oracle.

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

**Check Status:**

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

**IAM Policy - RDS (rds-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:CreateDBInstance",
        "rds:DeleteDBInstance",
        "rds:DescribeDBInstances",
        "rds:ModifyDBInstance",
        "rds:StartDBInstance",
        "rds:StopDBInstance",
        "rds:CreateDBSnapshot",
        "rds:DeleteDBSnapshot",
        "rds:AddTagsToResource",
        "rds:RemoveTagsFromResource",
        "rds:ListTagsForResource",
        "rds:CreateDBParameterGroup",
        "rds:ModifyDBParameterGroup"
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
  --role-name infra-operator-rds-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator RDS management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-rds-role \
  --policy-name RDSManagement \
  --policy-document file://rds-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-rds-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator's ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-rds-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

Amazon RDS (Relational Database Service) is a fully managed relational database service that offers:

**Features:**
- **Fully Managed**: AWS manages backups, patches, replication, and failover
- **High Availability**: Automatic Multi-AZ with failover in minutes
- **Multiple Engines**: PostgreSQL, MySQL, MariaDB, SQL Server, Oracle
- **Scalability**: Increase or decrease capacity as needed
- **Automated Backups**: Configurable retention (up to 35 days)
- **Point-in-Time Recovery (PITR)**: Restore to any point in the last 35 days
- **Encryption at Rest**: AES-256 with AWS KMS
- **Encryption in Transit**: Automatic SSL/TLS
- **Performance Insights**: Monitor and optimize performance
- **Enhanced Monitoring**: Detailed OS, IO, CPU metrics
- **Read Replicas**: Read replication across regions (synchronous or asynchronous)
- **Automated Patching**: Configurable maintenance windows
- **Parameter Groups**: Custom database configuration
- **Security Groups**: Network-level access control

**Status**: ⚠️ Requires LocalStack Pro or Real AWS

## Quick Start

**RDS PostgreSQL:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: e2e-postgres-db
  namespace: default
spec:
  providerRef:
    name: localstack

  # Unique database identifier
  dbInstanceIdentifier: e2e-test-postgres

  # Database engine
  engine: postgres
  engineVersion: "14.7"

  # Instance class
  dbInstanceClass: db.t3.micro

  # Storage
  allocatedStorage: 20

  # Administrator credentials
  masterUsername: dbadmin
  masterUserPassword: Test123456!

  # Initial database name
  dbName: testdb
  port: 5432

  # Configuration
  multiAZ: false
  publiclyAccessible: true
  storageEncrypted: true

  # Backups
  backupRetentionPeriod: 7
  preferredBackupWindow: "03:00-04:00"

  # Tags
  tags:
    Environment: test
    ManagedBy: infra-operator
    Database: postgres

  deletionPolicy: Delete
```

**RDS MySQL:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: e2e-mysql-db
  namespace: default
spec:
  providerRef:
    name: localstack

  # Unique identifier
  dbInstanceIdentifier: e2e-test-mysql

  # MySQL engine
  engine: mysql
  engineVersion: "8.0.33"

  # Instance class
  dbInstanceClass: db.t3.small

  # Storage
  allocatedStorage: 30

  # Credentials
  masterUsername: admin
  masterUserPassword: MyPassword123!

  # Initial database
  dbName: mydb
  port: 3306

  # Multi-AZ enabled
  multiAZ: true
  publiclyAccessible: false
  storageEncrypted: true

  # Backups
  backupRetentionPeriod: 14

  # Tags
  tags:
    Environment: test
    ManagedBy: infra-operator
    Database: mysql

  deletionPolicy: Delete
```

**RDS PostgreSQL Production:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: app-database
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Unique database identifier (2-63 characters)
  dbInstanceIdentifier: app-db-prod

  # Database engine
  engine: postgres
  engineVersion: "15.4"

  # Instance class (capacity)
  dbInstanceClass: db.t3.micro

  # Storage
  allocatedStorage: 20
  storageType: gp3
  iops: 3000

  # Administrator credentials
  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  # Multi-AZ for high availability
  multiAZ: true

  # VPC and security
  dbSubnetGroupName: private-subnet-group
  vpcSecurityGroupRefs:
    - name: db-sg

  # Backups
  backupRetentionPeriod: 7
  preferredBackupWindow: "03:00-04:00"
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  # Deletion protection
  deletionProtection: true

  # Tagging
  tags:
    Environment: production
    Application: web-app

  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f rds.yaml
```

**Check Status:**

```bash
kubectl get rdsinstances
kubectl describe rdsinstance e2e-postgres-db
kubectl get rdsinstance e2e-postgres-db -o yaml
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource for authentication

  AWSProvider resource name

Unique identifier for the RDS instance (2 to 63 characters)

  **Rules:**
  - Alphanumeric and hyphens only
  - Cannot begin or end with hyphen
  - Must be unique per region
  - Cannot be modified after creation

  **Example:**

  ```yaml
  dbInstanceIdentifier: myapp-db-prod
  ```

Database engine

  **Options:**
  - `postgres` - PostgreSQL
  - `mysql` - MySQL
  - `mariadb` - MariaDB
  - `sqlserver-ex` - SQL Server Express
  - `sqlserver-se` - SQL Server Standard
  - `sqlserver-ee` - SQL Server Enterprise
  - `oracle-se2` - Oracle Standard Edition 2
  - `oracle-ee` - Oracle Enterprise Edition

  **Example:**

  ```yaml
  engine: postgres
  ```

Database engine version

  **Examples:**
  - PostgreSQL: `15.4`, `14.9`, `13.13`
  - MySQL: `8.0.35`, `8.0.34`, `5.7.44`
  - MariaDB: `10.6.14`, `10.5.21`, `10.4.32`

  **Example:**

  ```yaml
  engineVersion: "15.4"
  ```

  **Note:** Always specify versions with patch (e.g., 15.4, not 15)

Instance class (CPU, RAM, performance)

  **db.t3 Family (Burstable - Dev/Test):**
  - `db.t3.micro` - 1 vCPU, 1 GB RAM (~$0.017/hour)
  - `db.t3.small` - 1 vCPU, 2 GB RAM (~$0.034/hour)
  - `db.t3.medium` - 1 vCPU, 4 GB RAM (~$0.068/hour)

  **db.t4g Family (Graviton - cheaper):**
  - `db.t4g.micro` - 1 vCPU, 1 GB RAM
  - `db.t4g.small` - 1 vCPU, 2 GB RAM

  **db.m5 Family (General Purpose):**
  - `db.m5.large` - 2 vCPU, 8 GB RAM (~$0.175/hour)
  - `db.m5.xlarge` - 4 vCPU, 16 GB RAM (~$0.350/hour)
  - `db.m5.2xlarge` - 8 vCPU, 32 GB RAM (~$0.700/hour)

  **db.r5 Family (Memory Optimized - Cache/Analytics):**
  - `db.r5.large` - 2 vCPU, 16 GB RAM (~$0.280/hour)
  - `db.r5.xlarge` - 4 vCPU, 32 GB RAM (~$0.560/hour)

  **Example:**

  ```yaml
  dbInstanceClass: db.t3.micro
  ```

Allocated storage space in GB

  **Range:** 20 - 65,536 GB (depends on engine)

  **Storage types:**
  - **gp2** (General Purpose): 20-65,536 GB (default)
  - **gp3** (General Purpose v3): 20-65,536 GB (recommended, better performance)
  - **io1** (Provisioned IOPS): 100-65,536 GB (high performance)
  - **io2** (Provisioned IOPS v2): 100-65,536 GB (maximum performance)

  **Example:**

  ```yaml
  allocatedStorage: 20
  storageType: gp3
  ```

  **Note:** You can increase size later, but cannot decrease

Database administrator username

  **Rules:**
  - 1 to 16 characters (depends on engine)
  - Alphanumeric only
  - Cannot be `admin`, `root`, `postgres` (some engines)
  - Cannot begin with number

  **Example:**

  ```yaml
  masterUsername: dbadmin
  ```

Reference to Kubernetes Secret containing the password

  Kubernetes Secret name

Key within the Secret

**Example Secret:**
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: db-password
    namespace: default
  type: Opaque
  stringData:
    password: SuperSecurePassword123!
  ```

  **Password Rules:**
  - Minimum 8 characters
  - Contains: letters, numbers, special symbols
  - Cannot contain `@`, `/`, `"`, or `\`

### Optional Fields - Storage

EBS storage type

  **Options:**
  - `gp2`: General Purpose SSD (default, good for most)
  - `gp3`: General Purpose SSD v3 (recommended, better performance)
  - `io1`: Provisioned IOPS SSD (predictable performance)
  - `io2`: Provisioned IOPS SSD v2 (maximum performance)

  **Example:**

  ```yaml
  storageType: gp3
  ```

Provisioned IOPS (io1/io2 only)

  **Range:** 1,000 - 64,000 IOPS

  **Note:** For gp3, IOPS is configured separately from size:

  ```yaml
  storageType: io1
  allocatedStorage: 100
  iops: 5000
  ```

Storage throughput for gp3 (MB/s)

  **Range:** 125 - 1,000 MB/s:

  ```yaml
  storageType: gp3
  storageThroughput: 500
  ```

### Optional Fields - Network and Security

DB Subnet Group name (private subnets)

  **Important:** MUST exist previously in AWS:

  ```yaml
  dbSubnetGroupName: private-subnet-group
  ```

  **Create subnet group via AWS CLI:**
  ```bash
  aws rds create-db-subnet-group \
    --db-subnet-group-name private-subnet-group \
    --db-subnet-group-description "Private subnets for RDS" \
    --subnet-ids subnet-xxx subnet-yyy
  ```

References to Kubernetes Security Groups for access control

  SecurityGroup resource name in Kubernetes

**Example:**

```yaml
  vpcSecurityGroupRefs:
    - name: rds-security-group
  ```

  **Alternative - use direct IDs:**
  ```yaml
  vpcSecurityGroupIds:
    - sg-0123456789abcdef0
    - sg-0123456789abcdef1
  ```

Whether the database is accessible via public internet

  **⚠️ Security:** Never enable in production!:

  ```yaml
  publiclyAccessible: false  # Keep private
  ```

Protects against accidental deletion

  **Recommended:** true in production:

  ```yaml
  deletionProtection: true
  ```

### Optional Fields - Backups and Recovery

Days of automated backup retention

  **Range:** 1 - 35 days

  **Recommendations:**
  - Development: 1-7 days
  - Staging: 7-14 days
  - Production: 14-35 days

  **Example:**

  ```yaml
  backupRetentionPeriod: 30
  ```

  **Cost:** Increases with retention

Daily backup window (UTC, format HH:MM-HH:MM)

  **Default:** Automatically selected:

  ```yaml
  preferredBackupWindow: "03:00-04:00"  # 3AM-4AM UTC
  ```

  **Tip:** Choose low-usage time

Weekly maintenance window for patches (format: ddd:HH:MM-ddd:HH:MM)

  **Default:** Automatically selected:

  ```yaml
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"
  # Sunday, 4AM-5AM UTC
  ```

  **Valid days:** mon, tue, wed, thu, fri, sat, sun

Skip creating final snapshot when deleting database

  **Caution:** true may lose data!:

  ```yaml
  skipFinalSnapshot: false  # Always create final snapshot
  ```

Final snapshot identifier when deleting

  **Example:**

  ```yaml
  finalDBSnapshotIdentifier: app-db-final-snapshot-2025-11-22
  ```

### Optional Fields - High Availability and Replication

Enable Multi-AZ (high availability)

  **Benefits:**
  - Automatic failover on failure (minutes)
  - Synchronous replication for data integrity
  - ~50% cost increase

  **Recommended:** true in production:

  ```yaml
  multiAZ: true
  ```

Use IAM for authentication (instead of password)

  **Example:**

  ```yaml
  enableIAMDatabaseAuthentication: true
  ```

  **Benefits:**
  - No hardcoded password
  - Temporary tokens
  - Automatic auditing

### Optional Fields - Encryption

Enable encryption at rest

  **Default:** true (recommended):

  ```yaml
  storageEncrypted: true
  ```

AWS KMS key ARN for encryption

  **Default:** AWS-managed key (no cost):

  ```yaml
  storageEncrypted: true
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012
  ```

### Optional Fields - Monitoring and Optimization

Enable Performance Insights (detailed analysis)

  **Example:**

  ```yaml
  enablePerformanceInsights: true
  ```

  **Cost:** ~$0.02/hour additional

Enable Enhanced Monitoring (OS metrics)

  **Example:**

  ```yaml
  enableEnhancedMonitoring: true
  monitoringInterval: 60  # Every 60 seconds
  ```

  **Cost:** Based on interval

Enhanced Monitoring interval (seconds)

  **Options:** 0 (disabled), 1, 5, 10, 15, 30, 60:

  ```yaml
  monitoringInterval: 60
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

What happens to the instance when the CR is deleted

  **Options:**
  - `Delete`: Instance is deleted from AWS
  - `Retain`: Instance remains in AWS but unmanaged
  - `Orphan`: Remove only management

  **Example:**

  ```yaml
  deletionPolicy: Retain  # For production
  ```

## Status Fields

After the instance is created, the following status fields are populated:

Complete ARN of the RDS instance

  ```
  arn:aws:rds:us-east-1:123456789012:db:app-db-prod
  ```

Database connection information

  Database hostname for connection

      ```
      app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com
      ```

Connection port (default: 5432 PostgreSQL, 3306 MySQL)


Current instance state

  **Possible values:**
  - `creating` - Creating
  - `available` - Available and ready
  - `deleting` - Being deleted
  - `modifying` - Configuration being changed
  - `backing-up` - Backup in progress
  - `maintenance` - Scheduled maintenance
  - `failed` - Creation error

Allocated space in GB

Running engine version

Whether Multi-AZ is enabled

`true` when the instance is `available` and ready

Timestamp of last AWS synchronization

## Examples

### Production PostgreSQL RDS with Multi-AZ

Database for web application with high availability:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: production-postgres
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Identification
  dbInstanceIdentifier: myapp-db-prod
  engine: postgres
  engineVersion: "15.4"

  # Class and storage
  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  storageType: gp3
  iops: 3000
  storageThroughput: 250

  # Credentials
  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  # Network
  dbSubnetGroupName: private-subnet-group
  vpcSecurityGroupRefs:
    - name: rds-security-group
  publiclyAccessible: false

  # High Availability
  multiAZ: true
  deletionProtection: true

  # Backups
  backupRetentionPeriod: 30
  preferredBackupWindow: "03:00-04:00"
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  # Monitoring
  enablePerformanceInsights: true

  # Security
  storageEncrypted: true

  # Tags
  tags:
    Environment: production
    Application: myapp
    Team: backend
    CostCenter: engineering

  deletionPolicy: Retain
```

### RDS MySQL for Development

Simple database for local development:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: dev-mysql
  namespace: default
spec:
  providerRef:
    name: dev-aws

  dbInstanceIdentifier: myapp-db-dev

  engine: mysql
  engineVersion: "8.0.35"

  # Small instance for dev
  dbInstanceClass: db.t3.micro
  allocatedStorage: 20
  storageType: gp3

  masterUsername: admin
  masterUserPasswordSecretRef:
    name: dev-db-password
    key: password

  # Dev doesn't need Multi-AZ
  multiAZ: false
  deletionProtection: false

  # Less frequent backups
  backupRetentionPeriod: 7

  tags:
    Environment: development
    Application: myapp

  deletionPolicy: Delete
```

### RDS with Read Replica

Primary database with read replication for analytics:

```yaml
# Primary database
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: primary-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: myapp-db-primary
  engine: postgres
  engineVersion: "15.4"

  dbInstanceClass: db.m5.large
  allocatedStorage: 100
  storageType: gp3

  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  multiAZ: true
  backupRetentionPeriod: 35

  # Enable backups (required for replicas)
  backupRetentionPeriod: 7

  tags:
    Environment: production
    Role: primary

  deletionPolicy: Retain

---
# Read Replica (for analytics/reporting)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: read-replica-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: myapp-db-read-replica

  # Create from primary database
  replicateSourceDb: myapp-db-primary

  # Same engine, can be smaller class
  dbInstanceClass: db.t3.small

  # No need to configure password (inherits from primary)
  # Replica in different AZ for HA
  availabilityZone: us-east-1b

  tags:
    Environment: production
    Role: read-replica

  deletionPolicy: Delete
```

### RDS with Encryption and Complete PITR

Critical database with maximum security and recovery:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: critical-database
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: critical-db-prod

  engine: postgres
  engineVersion: "15.4"

  dbInstanceClass: db.r5.xlarge  # Memory optimized
  allocatedStorage: 500
  storageType: io2
  iops: 20000

  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: critical-db-password
    key: password

  # Private VPC with security
  dbSubnetGroupName: critical-subnet-group
  vpcSecurityGroupRefs:
    - name: critical-rds-sg
  publiclyAccessible: false

  # HA and protection
  multiAZ: true
  deletionProtection: true

  # Backups with complete PITR
  backupRetentionPeriod: 35
  preferredBackupWindow: "02:00-03:00"
  skipFinalSnapshot: false
  finalDBSnapshotIdentifier: critical-db-final-backup

  # Encryption with KMS
  storageEncrypted: true
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee

  # Complete monitoring
  enablePerformanceInsights: true
  enableEnhancedMonitoring: true
  monitoringInterval: 1

  # IAM authentication
  enableIAMDatabaseAuthentication: true

  # Maintenance
  preferredMaintenanceWindow: "sun:03:00-sun:04:00"

  tags:
    Environment: production
    CriticalData: "true"
    BackupRequired: "true"
    Compliance: "required"

  deletionPolicy: Retain
```

### RDS MariaDB for WordPress/CMS

Database for traditional web application:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: wordpress-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: wordpress-db

  engine: mariadb
  engineVersion: "10.6.14"

  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  storageType: gp3

  masterUsername: wordpress
  masterUserPasswordSecretRef:
    name: wordpress-db-password
    key: password

  dbSubnetGroupName: web-subnet-group
  vpcSecurityGroupRefs:
    - name: wordpress-rds-sg

  multiAZ: true
  backupRetentionPeriod: 14

  tags:
    Environment: production
    Application: wordpress

  deletionPolicy: Retain
```

## Verification

### Check Status via kubectl

**Command:**

```bash
# List all RDS instances
kubectl get rdsinstances

# Get detailed information
kubectl get rdsinstance production-postgres -o yaml

# Monitor creation in real-time
kubectl get rdsinstance production-postgres -w

# View events and status
kubectl describe rdsinstance production-postgres
```

### Check in AWS

**AWS CLI:**

```bash
# List RDS instances
aws rds describe-db-instances \
  --query 'DBInstances[].{Identifier:DBInstanceIdentifier,Status:DBInstanceStatus,Engine:Engine,Class:DBInstanceClass}' \
  --output table

# Get complete details
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --output json | jq '.DBInstances[0]'

# View connection endpoint
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].Endpoint'

# Test connection (PostgreSQL)
psql -h app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com \
  -U postgres \
  -d postgres

# View backups
aws rds describe-db-snapshots \
  --db-instance-identifier app-db-prod

# View multi-AZ status
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].MultiAZ'

```

**LocalStack:**

```bash
# For LocalStack testing
export AWS_ENDPOINT_URL=http://localhost:4566

aws rds describe-db-instances

aws rds describe-db-instances \
  --db-instance-identifier app-db-prod
```

**psql (PostgreSQL):**

```bash
# Connect to database
psql -h <endpoint> -U postgres -d postgres

# View users
\du

# View databases
\l

# View space used
SELECT pg_database.datname,
       pg_size_pretty(pg_database_size(pg_database.datname))
FROM pg_database;

# Disconnect
\q
```

**mysql (MySQL/MariaDB):**

```bash
# Connect to database
mysql -h <endpoint> -u admin -p

# View databases
SHOW DATABASES;

# View users
SELECT user, host FROM mysql.user;

# View space used
SELECT table_schema,
       ROUND(SUM(data_length+index_length)/1024/1024,2) AS size_mb
FROM information_schema.tables
GROUP BY table_schema;

# Exit
EXIT;
```

### Expected Output

**Example:**

```yaml
status:
  dbInstanceArn: arn:aws:rds:us-east-1:123456789012:db:app-db-prod
  dbInstanceStatus: available
  endpoint:
    address: app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com
    port: 5432
  engine: postgres
  engineVersion: "15.4"
  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  multiAZ: true
  storageEncrypted: true
  ready: true
  lastSyncTime: "2025-11-22T20:45:22Z"
```

## Troubleshooting

### RDS stuck in creating for more than 30 minutes

**Symptoms:** `dbInstanceStatus: creating` indefinitely

**Common causes:**
1. Subnet group doesn't exist or is invalid
2. Security group not found
3. RDS instance quota reached
4. AWS connectivity issue

**Solutions:**
```bash
# Check detailed status
kubectl describe rdsinstance app-database

# View operator logs
kubectl logs -n infra-operator-system \
  deploy/infra-operator-controller-manager \
  --tail=100 | grep -i rds

# Check if subnet group exists
aws rds describe-db-subnet-groups \
  --db-subnet-group-name private-subnet-group

# Check AWSProvider is ready
kubectl get awsprovider
kubectl describe awsprovider production-aws

# Force synchronization
kubectl annotate rdsinstance app-database \
  force-sync="$(date +%s)" --overwrite

# Last resort: delete and recreate (with Retain)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"deletionPolicy":"Retain"}}'
```

### Connection timeout when connecting to database

**Symptoms:** `psql: could not translate host name` or `Connection refused`

**Common causes:**
1. Security group doesn't allow connection (port blocked)
2. Database is not publicly accessible and connecting from outside VPC
3. Incorrect hostname/endpoint
4. Network not properly routed

**Solutions:**
```bash
# Get correct endpoint
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].Endpoint'

# Check security group allows ingress
aws ec2 describe-security-groups \
  --group-ids sg-0123456789abcdef0 \
  --query 'SecurityGroups[0].IpPermissions'

# MUST have rule like:
# IpProtocol: tcp, FromPort: 5432, ToPort: 5432
# CidrIp: 10.0.0.0/16 (your VPC)

# If connecting from outside VPC:
# 1. Enable publiclyAccessible (not recommended)
# 2. Or use bastion/jump host
# 3. Or use VPN

# Test with telnet/nc
nc -zv app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com 5432

# Test with psql verbose
psql -h app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com \
  -U postgres \
  -d postgres \
  -v

# If using Multi-AZ, failover may be in progress
# Try again in 5 minutes
```

### Out of disk space (storage full)

**Symptoms:** `Disk full` error when executing queries, application slow

**Causes:**
1. Data grew beyond expected
2. Accumulated logs or backups
3. Lack of old data cleanup

**Solutions:**
```bash
# View available space
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].AllocatedStorage'

# Increase storage (may take minutes)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"allocatedStorage":100}}'

# View progress
kubectl get rdsinstance app-database -w

# If PostgreSQL, clean up space
# Connect to database:
psql -h <endpoint> -U postgres -d postgres

# Vacuum full (reclaim space)
VACUUM FULL;
VACUUM ANALYZE;

# If MySQL, optimize tables
mysql -h <endpoint> -u admin -p

OPTIMIZE TABLE table_name;

# View database size
SELECT pg_database.datname,
       pg_size_pretty(pg_database_size(pg_database.datname))
FROM pg_database;
```

### High RDS costs

**Symptoms:** AWS account with unexpected RDS charges

**Common causes:**
1. Instance class too large
2. High provisioned IOPS
3. Many backups retained
4. Cross-region replication

**Solutions:**
```bash
# Calculate current cost
# Use AWS Pricing Calculator
# t3.small: ~$0.034/hour * 730 = ~$25/month
# gp3 storage: ~$0.12/GB/month

# Reduce class (if possible)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.t3.micro"}}'

# Reduce backup retention
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"backupRetentionPeriod":7}}'

# Delete old unnecessary snapshots
aws rds delete-db-snapshot \
  --db-snapshot-identifier app-db-snapshot-old

# For dev, delete after use
kubectl delete rdsinstance dev-database
```

### Backup fails or takes too long

**Symptoms:** Backup status stuck in `backing-up` for hours

**Causes:**
1. Database too large
2. IO saturated during backup
3. Many transactions during backup

**Solutions:**
```bash
# View backup status
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].LatestRestorableTime'

# View completed snapshots
aws rds describe-db-snapshots \
  --db-instance-identifier app-db-prod

# Increase backup window
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"preferredBackupWindow":"02:00-04:00"}}'

# Increase IOPS for backup performance
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"iops":5000}}'

# Create manual snapshot
aws rds create-db-snapshot \
  --db-instance-identifier app-db-prod \
  --db-snapshot-identifier manual-backup-$(date +%Y%m%d-%H%M%S)
```

### Multi-AZ failover slow or fails

**Symptoms:** After failure, application offline for 5+ minutes

**Cause:** Automatic failover may take time, especially if Multi-AZ not properly configured

**Solutions:**
```bash
# Check Multi-AZ is enabled
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].MultiAZ'

# Enable if not
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"multiAZ":true}}'

# Test failover manually
# CAUTION: Causes downtime!
aws rds reboot-db-instance \
  --db-instance-identifier app-db-prod \
  --force-failover

# View reboot progress
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].DBInstanceStatus'

# Implement retry in application
# Use connection pooling
# Configure application-level failover
```

### Read Replica not syncing or lagging

**Symptoms:** Replica shows outdated data, cannot connect

**Causes:**
1. Replication lag (network delay)
2. Primary database under IO pressure
3. Network issues

**Solutions:**
```bash
# View replica lag
aws rds describe-db-instances \
  --db-instance-identifier myapp-db-read-replica \
  --query 'DBInstances[0].StatusInfos'

# Increase replica class if under-resourced
kubectl patch rdsinstance read-replica-db \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.t3.small"}}'

# Increase IOPS on primary
kubectl patch rdsinstance primary-db \
  --type merge \
  -p '{"spec":{"iops":5000}}'

```

### Degraded performance (slow queries)

**Symptoms:** Normal queries become slow, high CPU/IO

**Causes:**
1. Missing indexes
2. Suboptimal query plan
3. Lack of memory/CPU
4. Transaction locks

**Solutions:**
```bash
# Enable Performance Insights
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"enablePerformanceInsights":true}}'

# Use AWS Performance Insights console for analysis

# Increase memory/CPU if needed
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.m5.large"}}'

# Connect and analyze queries
psql -h <endpoint> -U postgres -d postgres

# View slow queries (PostgreSQL)
SELECT query, calls, mean_exec_time, max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

# View missing indexes
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY schemaname, tablename;

# Create appropriate indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_date ON orders(created_at);
```

### Error deleting database (finalizer stuck)

**Symptoms:** `kubectl delete rdsinstance` pending indefinitely

**Cause:** Finalizer cannot delete final snapshot or database

**Solutions:**
```bash
# View details
kubectl describe rdsinstance app-database

# View finalizers
kubectl get rdsinstance app-database -o yaml | grep finalizers

# Option 1: Change deletionPolicy before deleting
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"deletionPolicy":"Retain"}}'

# Then delete
kubectl delete rdsinstance app-database

# Option 2: Delete final snapshot manually
aws rds delete-db-snapshot \
  --db-snapshot-identifier app-db-final-backup

# Then force delete CR
kubectl patch rdsinstance app-database \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

# Option 3: Keep final snapshot, only remove CR
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"skipFinalSnapshot":true}}'

kubectl delete rdsinstance app-database
```

## Best Practices

:::note Best Practices

- **Enable Multi-AZ for production** — Automatic failover in minutes, ~50% higher cost but essential for critical data, test failover regularly
- **Enable encryption everywhere** — Storage encryption (AES-256), AWS KMS for custom keys, SSL/TLS for connections, backup encryption for compliance
- **Configure appropriate backup retention** — 7-35 days depending on criticality, minimum 14 days for production, PITR up to 35 days, test restore regularly
- **Restrict security group access** — Only necessary IPs/SGs, never 0.0.0.0/0, segregate dev/staging/prod, regularly audit rules
- **Use private subnets only** — Never publicly accessible in production, use NAT gateway for outbound, bastion/jump host for admin access
- **Enable monitoring and insights** — Performance Insights for tuning, CloudWatch alerts for anomalies, monitor CPU/Storage/IOPS metrics
- **Tune parameter groups** — Customize settings per engine (shared_buffers, work_mem for PostgreSQL), document reason for changes, test in dev first
- **Tag all resources** — Environment (dev/staging/prod), application, cost center, owner/team, backup required flag
- **Right-size instances** — Start small and monitor growth, use t3/t4g for variable workloads, m5 for predictable production loads
- **Optimize costs** — Delete dev/staging after use, appropriate backup retention, reserved instances for production
- **Plan for disaster recovery** — Read replicas for regional DR, snapshots in another region, document RTO/RPO, automate failover
- **Enable audit and compliance** — CloudTrail for API calls, CloudWatch Logs for queries, IAM database authentication
- **Optimize queries** — Performance Insights for bottlenecks, slow query logs, appropriate index strategy, connection pooling in app

:::

## Use Cases

### 1. Transactional Web Application

E-commerce or SaaS with many small transactions:

```yaml
dbInstanceClass: db.t3.small
allocatedStorage: 100
multiAZ: true
backupRetentionPeriod: 30
# Indexes on user_id, order_id, timestamps
```

### 2. E-commerce with Carts and Orders

High-volume data with critical transactions:

```yaml
dbInstanceClass: db.m5.large
allocatedStorage: 500
storageType: io1
iops: 10000
multiAZ: true
backupRetentionPeriod: 35
# Read replica for analytics
```

### 3. CMS (WordPress, Drupal)

Web applications with dynamic content:

```yaml
engine: mysql  # or mariadb
dbInstanceClass: db.t3.small
allocatedStorage: 50
multiAZ: true
enableCloudwatchLogsExports: [slowquery, error]
```

### 4. ERP/CRM with High Concurrency

Multiple users accessing simultaneously:

```yaml
dbInstanceClass: db.m5.2xlarge
allocatedStorage: 1000
storageType: io2
iops: 64000
multiAZ: true
backupRetentionPeriod: 35
enableEnhancedMonitoring: true
```

## Related Resources

- [VPC and Subnets](/services/networking/vpc)

  - [Security Groups](/services/networking/security-group)

  - [Secrets Manager](/services/security/secrets-manager)

  - [Lambda](/services/compute/lambda)

  - [DynamoDB](/services/database/dynamodb)
---
