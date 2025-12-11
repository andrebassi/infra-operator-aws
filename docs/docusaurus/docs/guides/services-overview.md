# AWS Services Guide - Infra Operator

Complete guide for using all AWS services provided by the Infra Operator.

## Table of Contents

- [Getting Started](#getting-started)
- [Networking Services](#networking-services)
  - [VPC](#vpc---virtual-private-cloud)
  - [Subnet](#subnet---network-segmentation)
  - [Internet Gateway](#internet-gateway---internet-connectivity)
  - [NAT Gateway](#nat-gateway---outbound-internet-for-private-subnets)
- [Storage Services](#storage-services)
  - [S3 Bucket](#s3-bucket---object-storage)
- [Database Services](#database-services)
  - [DynamoDB Table](#dynamodb-table---nosql-database)
  - [RDS Instance](#rds-instance---relational-database)
- [Compute Services](#compute-services)
  - [EC2 Instance](#ec2-instance---virtual-machines)
  - [Lambda Function](#lambda-function---serverless-compute)
- [Messaging Services](#messaging-services)
  - [SQS Queue](#sqs-queue---message-queuing)
  - [SNS Topic](#sns-topic---pubsub-messaging)
- [Security Services](#security-services)
  - [IAM Role](#iam-role---identity-and-access-management)
  - [Secrets Manager](#secrets-manager---secrets-management)
  - [KMS Key](#kms-key---encryption-key-management)
- [Container Services](#container-services)
  - [ECR Repository](#ecr-repository---container-registry)
- [Caching Services](#caching-services)
  - [ElastiCache Cluster](#elasticache-cluster---in-memory-caching)
- [Service Status Matrix](#service-status-matrix)

---

## Getting Started

Before creating any AWS resources, you need to configure an **AWSProvider** with your credentials.

### AWSProvider - Authentication

The AWSProvider manages AWS authentication and credentials for all other resources.

#### Using IRSA (Recommended for EKS)

**Example:**

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

#### Using Static Credentials (Development/Testing)

**Example:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  access-key-id: AKIAXXXXXXXXXXXXXXXX
  secret-access-key: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
  namespace: default
spec:
  region: us-east-1
  endpoint: http://localstack:4566  # For LocalStack testing
  accessKeyIDRef:
    name: aws-credentials
    key: access-key-id
  secretAccessKeyRef:
    name: aws-credentials
    key: secret-access-key
  defaultTags:
    managed-by: infra-operator
    environment: testing
```

#### Check AWSProvider Status

**Command:**

```bash
kubectl get awsproviders
kubectl describe awsprovider production-aws
```

**Status Fields:**
- `ready`: true when credentials are validated
- `accountID`: AWS account ID
- `identity`: AWS identity ARN
- `lastSyncTime`: Last validation time

---

## Networking Services

### VPC - Virtual Private Cloud

Create isolated virtual networks in AWS.

#### Basic VPC

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  instanceTenancy: default  # or "dedicated"
  tags:
    Name: production-vpc
    Environment: production
  deletionPolicy: Delete  # or "Retain"
```

#### Check VPC Status

**Command:**

```bash
kubectl get vpcs
kubectl describe vpc production-vpc
```

**Status Fields:**
- `vpcID`: AWS VPC ID (e.g., vpc-xxx)
- `cidrBlock`: Assigned CIDR block
- `state`: available, pending, etc.
- `ready`: true when VPC is available

#### AWS CLI Verification

**Command:**

```bash
export AWS_ENDPOINT_URL=http://localhost:4566  # For LocalStack
aws ec2 describe-vpcs --vpc-ids vpc-xxx
```

---

### Subnet - Network Segmentation

Create subnets within a VPC for resource placement.

#### Public Subnet (with auto-assign public IP)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet-1a
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd  # Reference to VPC
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true  # Auto-assign public IPs
  tags:
    Name: public-subnet-1a
    Type: public
  deletionPolicy: Delete
```

#### Private Subnet

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: private-subnet-1b
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: "10.0.2.0/24"
  availabilityZone: us-east-1b
  mapPublicIpOnLaunch: false  # No public IPs
  tags:
    Name: private-subnet-1b
    Type: private
  deletionPolicy: Delete
```

#### Check Subnet Status

**Command:**

```bash
kubectl get subnets
kubectl get subnet public-subnet-1a -o yaml
```

**Status Fields:**
- `subnetID`: AWS Subnet ID (e.g., subnet-xxx)
- `vpcID`: Parent VPC ID
- `state`: available, pending, etc.
- `availableIpAddressCount`: Number of available IPs

#### Multi-AZ Subnet Example

**Example:**

```yaml
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-xxx
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
  tags:
    Name: public-1a
    kubernetes.io/role/elb: "1"
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet-1b
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-xxx
  cidrBlock: "10.0.2.0/24"
  availabilityZone: us-east-1b
  mapPublicIpOnLaunch: true
  tags:
    Name: public-1b
    kubernetes.io/role/elb: "1"
```

---

### Internet Gateway - Internet Connectivity

Provides internet access to resources in public subnets.

#### Create Internet Gateway

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: production-igw
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-f3ea9b1b36fce09cd  # Will be attached to this VPC
  tags:
    Name: production-internet-gateway
    Environment: production
  deletionPolicy: Delete
```

#### Check IGW Status

**Command:**

```bash
kubectl get internetgateways
kubectl describe internetgateway production-igw
```

**Status Fields:**
- `internetGatewayID`: AWS IGW ID (e.g., igw-xxx)
- `vpcID`: Attached VPC ID
- `state`: available, attached, etc.
- `ready`: true when attached to VPC

**Important Notes:**
- Internet Gateway is automatically attached to the VPC
- One IGW per VPC
- Required for public subnets to access internet
- Automatically detached on deletion

---

### NAT Gateway - Outbound Internet for Private Subnets

Allows resources in private subnets to access the internet while remaining private.

#### Public NAT Gateway

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: production-nat-1a
  namespace: default
spec:
  providerRef:
    name: production-aws
  subnetID: subnet-12853af5337079de5  # Must be a PUBLIC subnet
  connectivityType: public  # or "private"
  # allocationID: eipalloc-xxx  # Optional: provide existing EIP
  tags:
    Name: production-nat-gateway
    Environment: production
  deletionPolicy: Delete
```

#### Check NAT Gateway Status

**Command:**

```bash
kubectl get natgateways
kubectl get natgateway production-nat-1a -o wide
```

**Status Fields:**
- `natGatewayID`: AWS NAT Gateway ID (e.g., nat-xxx)
- `subnetID`: Subnet where NAT is placed
- `publicIP`: Elastic IP address
- `privateIP`: Private IP in subnet
- `state`: pending, available, deleted, etc.
- `allocationID`: Elastic IP allocation ID

**Important Notes:**
- NAT Gateway must be placed in a **public subnet**
- Elastic IP is automatically allocated if not provided
- Can take 1-2 minutes to become available
- Charges apply when running
- Use `connectivityType: private` for private NAT gateway (no EIP)

#### Multi-AZ NAT Gateway Example

**Example:**

```yaml
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-gateway-1a
spec:
  providerRef:
    name: production-aws
  subnetID: subnet-public-1a  # Public subnet in us-east-1a
  connectivityType: public
  tags:
    Name: nat-1a
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-gateway-1b
spec:
  providerRef:
    name: production-aws
  subnetID: subnet-public-1b  # Public subnet in us-east-1b
  connectivityType: public
  tags:
    Name: nat-1b
```

---

## Storage Services

### S3 Bucket - Object Storage

Create and manage S3 buckets for object storage.

#### Basic S3 Bucket

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: my-app-data
  namespace: default
spec:
  providerRef:
    name: production-aws
  bucketName: mycompany-app-data-prod  # Must be globally unique
  tags:
    Application: my-app
    Environment: production
  deletionPolicy: Retain  # Keep bucket on CR deletion
```

#### S3 Bucket with Versioning and Encryption

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: secure-data-bucket
spec:
  providerRef:
    name: production-aws
  bucketName: mycompany-secure-data

  # Versioning
  versioning:
    enabled: true

  # Encryption
  encryption:
    algorithm: AES256  # or "aws_kms"
    # kmsKeyID: "arn:aws:kms:us-east-1:123456789012:key/xxx"

  # Block all public access
  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true

  tags:
    Compliance: required
    Encryption: enabled

  deletionPolicy: Retain
```

#### S3 Bucket with Lifecycle Rules

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: logs-bucket
spec:
  providerRef:
    name: production-aws
  bucketName: mycompany-application-logs

  lifecycleRules:
  - id: archive-old-logs
    enabled: true
    prefix: logs/
    transitions:
    - days: 30
      storageClass: STANDARD_IA
    - days: 90
      storageClass: GLACIER
    expiration:
      days: 365

  - id: cleanup-temp-files
    enabled: true
    prefix: temp/
    expiration:
      days: 7

  deletionPolicy: Delete
```

#### Check S3 Bucket Status

**Command:**

```bash
kubectl get s3buckets
kubectl describe s3bucket my-app-data
```

**Status Fields:**
- `bucketName`: S3 bucket name
- `region`: AWS region
- `arn`: S3 bucket ARN
- `ready`: true when bucket is available

**Deletion Policies:**
- `Delete`: Bucket is deleted when CR is deleted (default)
- `Retain`: Bucket is kept in AWS
- `Orphan`: Bucket is kept but not managed

---

## Database Services

### DynamoDB Table - NoSQL Database

Create DynamoDB tables for NoSQL data storage.

#### Basic DynamoDB Table

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: users-table
  namespace: default
spec:
  providerRef:
    name: production-aws
  tableName: production-users

  # Partition key
  hashKey:
    name: userID
    type: S  # S (String), N (Number), B (Binary)

  # Optional sort key
  rangeKey:
    name: timestamp
    type: N

  # Billing mode
  billingMode: PAY_PER_REQUEST  # or "PROVISIONED"

  tags:
    Application: user-service
    Environment: production

  deletionPolicy: Retain
```

#### DynamoDB with Provisioned Capacity

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: products-table
spec:
  providerRef:
    name: production-aws
  tableName: production-products

  hashKey:
    name: productID
    type: S

  billingMode: PROVISIONED

  # Provisioned throughput
  readCapacityUnits: 5
  writeCapacityUnits: 5

  # Global Secondary Index
  globalSecondaryIndexes:
  - indexName: CategoryIndex
    hashKey:
      name: category
      type: S
    rangeKey:
      name: price
      type: N
    projection:
      type: ALL  # or "KEYS_ONLY", "INCLUDE"
    readCapacityUnits: 5
    writeCapacityUnits: 5

  deletionPolicy: Delete
```

#### Check DynamoDB Status

**Command:**

```bash
kubectl get dynamodbtables
kubectl describe dynamodbtable users-table
```

**Status Fields:**
- `tableName`: DynamoDB table name
- `tableStatus`: CREATING, ACTIVE, DELETING, etc.
- `tableARN`: Table ARN
- `itemCount`: Number of items (approximate)

---

### RDS Instance - Relational Database

Create and manage RDS database instances.

**⚠️ Note:** RDS requires LocalStack Pro or AWS account.

#### PostgreSQL Database

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: production-postgres
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Instance configuration
  dbInstanceIdentifier: production-postgres-01
  dbInstanceClass: db.t3.medium
  engine: postgres
  engineVersion: "14.7"

  # Storage
  allocatedStorage: 100  # GB
  storageType: gp3
  storageEncrypted: true

  # Master credentials
  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: postgres-password
    key: password

  # Database name
  dbName: production

  # Multi-AZ for high availability
  multiAZ: true

  # Backup
  backupRetentionPeriod: 7
  preferredBackupWindow: "03:00-04:00"

  # Maintenance
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  # Network
  vpcSecurityGroupIDs:
  - sg-xxx
  dbSubnetGroupName: production-db-subnet-group
  publiclyAccessible: false

  # Deletion protection
  deletionProtection: true
  deletionPolicy: Retain

  tags:
    Application: main-app
    Environment: production
```

#### MySQL Database

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: app-mysql
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: app-mysql-01
  dbInstanceClass: db.t3.small
  engine: mysql
  engineVersion: "8.0.33"

  allocatedStorage: 50
  storageType: gp2

  masterUsername: admin
  masterUserPasswordSecretRef:
    name: mysql-password
    key: password

  dbName: application

  # Enable automated backups
  backupRetentionPeriod: 7

  # Parameter group
  dbParameterGroupName: custom-mysql-params

  deletionPolicy: Delete
```

#### Check RDS Status

**Command:**

```bash
kubectl get rdsinstances
kubectl describe rdsinstance production-postgres
```

**Status Fields:**
- `dbInstanceIdentifier`: RDS instance identifier
- `dbInstanceStatus`: available, creating, backing-up, etc.
- `endpoint`: Connection endpoint
- `port`: Connection port

---

## Compute Services

### EC2 Instance - Virtual Machines

Create and manage EC2 instances.

#### Basic EC2 Instance

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EC2Instance
metadata:
  name: web-server
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Instance configuration
  instanceName: production-web-01
  instanceType: t3.medium
  imageID: ami-0c55b159cbfafe1f0  # Amazon Linux 2

  # Network
  subnetID: subnet-xxx
  securityGroupIDs:
  - sg-xxx

  # Storage
  ebsVolumes:
  - deviceName: /dev/xvda
    volumeSize: 30
    volumeType: gp3
    deleteOnTermination: true

  # User data script
  userData: |
    #!/bin/bash
    yum update -y
    yum install -y httpd
    systemctl start httpd
    systemctl enable httpd
    echo "Hello from EC2" > /var/www/html/index.html

  # IAM role
  iamInstanceProfile: ec2-web-server-role

  # Monitoring
  monitoring: true

  tags:
    Name: production-web-01
    Role: web-server

  deletionPolicy: Delete
```

#### Check EC2 Status

**Command:**

```bash
kubectl get ec2instances
kubectl describe ec2instance web-server
```

**Status Fields:**
- `instanceID`: EC2 instance ID
- `instanceName`: Instance name tag
- `state`: running, stopped, terminated, etc.
- `publicIP`: Public IP address
- `privateIP`: Private IP address

---

### Lambda Function - Serverless Compute

Create and manage Lambda functions.

#### Basic Lambda Function

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: api-handler
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Function configuration
  functionName: production-api-handler
  runtime: python3.11
  handler: index.lambda_handler

  # Code
  codeS3Bucket: my-lambda-code
  codeS3Key: api-handler/v1.0.0.zip

  # IAM role
  roleARN: arn:aws:iam::123456789012:role/lambda-execution-role

  # Memory and timeout
  memorySize: 512  # MB
  timeout: 30  # seconds

  # Environment variables
  environment:
    variables:
      ENV: production
      LOG_LEVEL: INFO

  # VPC configuration (optional)
  vpcConfig:
    subnetIDs:
    - subnet-xxx
    securityGroupIDs:
    - sg-xxx

  tags:
    Application: api
    Environment: production

  deletionPolicy: Delete
```

#### Check Lambda Status

**Command:**

```bash
kubectl get lambdafunctions
kubectl describe lambdafunction api-handler
```

**Status Fields:**
- `functionName`: Lambda function name
- `functionARN`: Function ARN
- `state`: Active, Pending, Inactive, Failed
- `runtime`: Runtime version

---

## Messaging Services

### SQS Queue - Message Queuing

Create and manage SQS queues for asynchronous messaging.

#### Standard SQS Queue

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: task-queue
  namespace: default
spec:
  providerRef:
    name: production-aws

  queueName: production-task-queue

  # Queue type
  fifoQueue: false  # Standard queue

  # Message retention
  messageRetentionPeriod: 345600  # 4 days (seconds)

  # Visibility timeout
  visibilityTimeout: 30  # seconds

  # Dead letter queue
  deadLetterQueue:
    queueARN: arn:aws:sqs:us-east-1:123456789012:production-dlq
    maxReceiveCount: 3

  # Encryption
  encryption:
    enabled: true
    kmsKeyID: alias/aws/sqs

  tags:
    Application: task-processor
    Environment: production

  deletionPolicy: Delete
```

#### FIFO SQS Queue

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: orders-queue
spec:
  providerRef:
    name: production-aws

  queueName: production-orders.fifo  # Must end with .fifo

  # FIFO configuration
  fifoQueue: true
  contentBasedDeduplication: true
  deduplicationScope: messageGroup  # or "queue"
  fifoThroughputLimit: perMessageGroupId  # or "perQueue"

  visibilityTimeout: 60
  messageRetentionPeriod: 1209600  # 14 days

  tags:
    Application: order-processing
    Type: fifo

  deletionPolicy: Delete
```

#### Check SQS Status

**Command:**

```bash
kubectl get sqsqueues
kubectl describe sqsqueue task-queue
```

**Status Fields:**
- `queueName`: SQS queue name
- `queueURL`: Queue URL for API calls
- `queueARN`: Queue ARN
- `approximateNumberOfMessages`: Current message count

---

### SNS Topic - Pub/Sub Messaging

Create and manage SNS topics for publish/subscribe messaging.

#### Standard SNS Topic

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: notifications
  namespace: default
spec:
  providerRef:
    name: production-aws

  topicName: production-notifications
  displayName: Production Notifications

  # Subscriptions
  subscriptions:
  - protocol: email
    endpoint: alerts@company.com
  - protocol: sqs
    endpoint: arn:aws:sqs:us-east-1:123456789012:notification-queue

  # Encryption
  encryption:
    enabled: true
    kmsKeyID: alias/aws/sns

  tags:
    Application: notification-service
    Environment: production

  deletionPolicy: Delete
```

#### FIFO SNS Topic

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: order-events
spec:
  providerRef:
    name: production-aws

  topicName: production-order-events.fifo  # Must end with .fifo
  displayName: Order Events

  # FIFO configuration
  fifoTopic: true
  contentBasedDeduplication: true

  # Subscriptions (only FIFO SQS queues)
  subscriptions:
  - protocol: sqs
    endpoint: arn:aws:sqs:us-east-1:123456789012:order-processor.fifo

  deletionPolicy: Delete
```

#### Check SNS Status

**Command:**

```bash
kubectl get snstopics
kubectl describe snstopic notifications
```

**Status Fields:**
- `topicName`: SNS topic name
- `topicARN`: Topic ARN
- `subscriptionsConfirmed`: Number of confirmed subscriptions
- `subscriptionsPending`: Number of pending confirmations

---

## Security Services

### IAM Role - Identity and Access Management

Create and manage IAM roles for AWS resources.

#### EC2 Instance Role

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: ec2-web-server-role
  namespace: default
spec:
  providerRef:
    name: production-aws

  roleName: production-ec2-web-server
  description: IAM role for EC2 web servers

  # Trust policy
  assumeRolePolicyDocument: |
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "ec2.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }

  # Attach managed policies
  managedPolicyARNs:
  - arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore
  - arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy

  # Inline policy
  inlinePolicies:
  - policyName: s3-access
    policyDocument: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "s3:GetObject",
              "s3:PutObject"
            ],
            "Resource": "arn:aws:s3:::my-bucket/*"
          }
        ]
      }

  tags:
    Environment: production
    ManagedBy: infra-operator

  deletionPolicy: Delete
```

#### Lambda Execution Role

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
metadata:
  name: lambda-execution-role
spec:
  providerRef:
    name: production-aws

  roleName: production-lambda-execution
  description: Execution role for Lambda functions

  assumeRolePolicyDocument: |
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "lambda.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }

  managedPolicyARNs:
  - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
  - arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole

  deletionPolicy: Delete
```

#### Check IAM Role Status

**Command:**

```bash
kubectl get iamroles
kubectl describe iamrole ec2-web-server-role
```

**Status Fields:**
- `roleName`: IAM role name
- `roleID`: Unique role ID
- `arn`: Role ARN
- `attachedPolicies`: Number of attached policies

---

### Secrets Manager - Secrets Management

Store and manage secrets securely.

#### Database Password Secret

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: database-password
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: production/database/password
  description: Master password for production database

  # Secret value (use carefully, prefer K8s Secret reference)
  secretString: "MySecurePassword123!"

  # Or use K8s Secret
  secretStringSecretRef:
    name: db-password
    key: password

  # Automatic rotation
  rotationEnabled: true
  rotationLambdaARN: arn:aws:lambda:us-east-1:123456789012:function:rotate-db-password
  rotationRules:
    automaticallyAfterDays: 30

  # Encryption
  kmsKeyID: alias/aws/secretsmanager

  tags:
    Application: database
    Rotation: enabled

  deletionPolicy: Retain
  recoveryWindowInDays: 30  # For soft delete
```

#### API Key Secret

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: api-key
spec:
  providerRef:
    name: production-aws

  secretName: production/api/key
  description: API key for external service

  secretString: "sk-1234567890abcdef"

  deletionPolicy: Delete
```

#### Check Secrets Manager Status

**Command:**

```bash
kubectl get secretsmanagersecrets
kubectl describe secretsmanagersecret database-password
```

**Status Fields:**
- `secretName`: Secret name
- `arn`: Secret ARN
- `rotationEnabled`: Whether rotation is enabled
- `lastRotationDate`: Last rotation timestamp

---

### KMS Key - Encryption Key Management

Create and manage KMS keys for encryption.

#### Symmetric Encryption Key

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: data-encryption-key
  namespace: default
spec:
  providerRef:
    name: production-aws

  description: Key for encrypting application data

  # Key spec
  keySpec: SYMMETRIC_DEFAULT  # AES-256
  keyUsage: ENCRYPT_DECRYPT

  # Key policy
  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Sid": "Enable IAM User Permissions",
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::123456789012:root"
          },
          "Action": "kms:*",
          "Resource": "*"
        },
        {
          "Sid": "Allow S3 to use the key",
          "Effect": "Allow",
          "Principal": {
            "Service": "s3.amazonaws.com"
          },
          "Action": [
            "kms:Decrypt",
            "kms:GenerateDataKey"
          ],
          "Resource": "*"
        }
      ]
    }

  # Multi-region
  multiRegion: false

  # Automatic rotation
  enableKeyRotation: true

  tags:
    Purpose: data-encryption
    Environment: production

  deletionPolicy: Retain
  pendingWindowInDays: 30  # Waiting period before deletion
```

#### Signing Key (Asymmetric)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: signing-key
spec:
  providerRef:
    name: production-aws

  description: RSA key for signing

  keySpec: RSA_2048
  keyUsage: SIGN_VERIFY

  deletionPolicy: Delete
```

#### Check KMS Key Status

**Command:**

```bash
kubectl get kmskeys
kubectl describe kmskey data-encryption-key
```

**Status Fields:**
- `keyID`: KMS key ID
- `arn`: Key ARN
- `keyState`: Enabled, Disabled, PendingDeletion, etc.
- `keyManager`: AWS or CUSTOMER

---

## Container Services

### ECR Repository - Container Registry

Create and manage ECR repositories for Docker images.

**⚠️ Note:** ECR requires LocalStack Pro or AWS account.

#### Basic ECR Repository

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: application-images
  namespace: default
spec:
  providerRef:
    name: production-aws

  repositoryName: production/application

  # Image scanning
  imageScanningConfiguration:
    scanOnPush: true

  # Immutable tags
  imageTagMutability: IMMUTABLE  # or "MUTABLE"

  # Encryption
  encryptionConfiguration:
    encryptionType: AES256  # or "KMS"
    # kmsKey: arn:aws:kms:us-east-1:123456789012:key/xxx

  tags:
    Application: main-app
    Environment: production

  deletionPolicy: Retain
```

#### ECR with Lifecycle Policy

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
metadata:
  name: app-images
spec:
  providerRef:
    name: production-aws

  repositoryName: production/app

  imageScanningConfiguration:
    scanOnPush: true

  imageTagMutability: MUTABLE

  # Lifecycle policy (delete old images)
  lifecyclePolicy: |
    {
      "rules": [
        {
          "rulePriority": 1,
          "description": "Keep last 10 images",
          "selection": {
            "tagStatus": "any",
            "countType": "imageCountMoreThan",
            "countNumber": 10
          },
          "action": {
            "type": "expire"
          }
        }
      ]
    }

  deletionPolicy: Delete
```

#### Check ECR Status

**Command:**

```bash
kubectl get ecrrepositories
kubectl describe ecrrepository application-images
```

**Status Fields:**
- `repositoryName`: Repository name
- `repositoryURI`: Full repository URI for docker push/pull
- `arn`: Repository ARN
- `imageCount`: Number of images

---

## Caching Services

### ElastiCache Cluster - In-Memory Caching

Create and manage ElastiCache clusters (Redis/Memcached).

**⚠️ Note:** ElastiCache requires LocalStack Pro or AWS account.

#### Redis Cluster

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: session-cache
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterID: production-redis-01
  engine: redis
  engineVersion: "7.0"

  # Node configuration
  nodeType: cache.t3.medium
  numCacheNodes: 1

  # Network
  subnetGroupName: production-cache-subnet-group
  securityGroupIDs:
  - sg-xxx

  # Snapshot and backup
  snapshotRetentionLimit: 5
  snapshotWindow: "03:00-05:00"

  # Maintenance
  preferredMaintenanceWindow: "sun:05:00-sun:07:00"

  # Encryption
  transitEncryptionEnabled: true
  atRestEncryptionEnabled: true

  tags:
    Application: session-store
    Environment: production

  deletionPolicy: Retain
```

#### Memcached Cluster

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: object-cache
spec:
  providerRef:
    name: production-aws

  clusterID: production-memcached-01
  engine: memcached
  engineVersion: "1.6.17"

  nodeType: cache.t3.small
  numCacheNodes: 3

  azMode: cross-az  # Distribute nodes across AZs

  deletionPolicy: Delete
```

#### Check ElastiCache Status

**Command:**

```bash
kubectl get elasticacheclusters
kubectl describe elasticachecluster session-cache
```

**Status Fields:**
- `clusterID`: Cluster identifier
- `clusterStatus`: available, creating, modifying, etc.
- `endpoint`: Connection endpoint
- `port`: Connection port

---

## Service Status Matrix

| Service | Domain | Adapter | UseCase | Controller | Tested | LocalStack | Status |
|---------|--------|---------|---------|------------|--------|------------|--------|
| **Networking** |
| VPC | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| Subnet | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| InternetGateway | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| NATGateway | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| RouteTable | ✅ | ✅ | ❌ | ❌ | ❌ | Community | CRD Only |
| SecurityGroup | ✅ | ✅ | ❌ | ❌ | ❌ | Community | CRD Only |
| **Storage** |
| S3Bucket | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| **Database** |
| DynamoDB | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| RDS | ✅ | ✅ | ✅ | ✅ | ⚠️ | Pro Only | AWS/Pro Only |
| **Compute** |
| EC2Instance | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| Lambda | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| **Messaging** |
| SQS | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| SNS | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| **Security** |
| IAMRole | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| SecretsManager | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| KMSKey | ✅ | ✅ | ✅ | ✅ | ✅ | Community | Production Ready |
| **Container** |
| ECR | ✅ | ✅ | ✅ | ✅ | ⚠️ | Pro Only | AWS/Pro Only |
| **Caching** |
| ElastiCache | ✅ | ✅ | ✅ | ✅ | ⚠️ | Pro Only | AWS/Pro Only |

**Legend:**
- ✅ Implemented and tested
- ⚠️ Implemented but requires LocalStack Pro or AWS
- ❌ Not implemented yet

---

## Common Patterns

### Multi-Tier Application Network

**Example:**

```yaml
---
# VPC
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: app-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: application-vpc

---
# Public Subnet 1
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-xxx  # Update after VPC creation
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
  tags:
    Name: public-1a
    kubernetes.io/role/elb: "1"

---
# Private Subnet 1
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: private-1a
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-xxx
  cidrBlock: "10.0.10.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: false
  tags:
    Name: private-1a
    kubernetes.io/role/internal-elb: "1"

---
# Internet Gateway
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: app-igw
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-xxx
  tags:
    Name: application-igw

---
# NAT Gateway
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-1a
spec:
  providerRef:
    name: production-aws
  subnetID: subnet-xxx  # Public subnet ID
  connectivityType: public
  tags:
    Name: nat-gateway-1a
```

### Serverless Application Stack

**Example:**

```yaml
---
# Lambda Function
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: api-handler
spec:
  providerRef:
    name: production-aws
  functionName: api-handler
  runtime: python3.11
  handler: index.handler
  codeS3Bucket: lambda-code
  codeS3Key: api-handler.zip
  roleARN: arn:aws:iam::xxx:role/lambda-role
  memorySize: 512
  timeout: 30

---
# DynamoDB Table
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: data-table
spec:
  providerRef:
    name: production-aws
  tableName: application-data
  hashKey:
    name: id
    type: S
  billingMode: PAY_PER_REQUEST

---
# SQS Queue
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: processing-queue
spec:
  providerRef:
    name: production-aws
  queueName: processing-queue
  visibilityTimeout: 300

---
# S3 Bucket
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: data-bucket
spec:
  providerRef:
    name: production-aws
  bucketName: application-data-bucket
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
```

---

## Troubleshooting

### Common Issues

#### 1. AWSProvider Not Ready

**Command:**

```bash
kubectl describe awsprovider production-aws
```

**Check:**
- Credentials are valid
- IAM permissions are sufficient
- Network connectivity to AWS (or LocalStack)

#### 2. Resource Stuck in Pending

**Command:**

```bash
kubectl get <resource-type> <name> -o yaml
kubectl describe <resource-type> <name>
```

**Check:**
- Controller logs: `kubectl logs -n infra-operator-system deploy/infra-operator-controller-manager`
- AWS API errors in events
- Resource dependencies (e.g., VPC must exist before Subnet)

#### 3. Deletion Hanging

Resources with `deletionPolicy: Delete` may hang if:
- AWS resource has dependencies
- Finalizers are blocking deletion

**Fix:**
```bash
# Check finalizers
kubectl get <resource> <name> -o yaml | grep finalizers

# Force remove finalizer (use with caution)
kubectl patch <resource> <name> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

#### 4. LocalStack Connection Issues

**Check LocalStack:**
```bash
curl http://localhost:4566/_localstack/health
```

**Verify endpoint in AWSProvider:**
```yaml
spec:
  endpoint: http://localstack:4566  # or http://localhost:4566
```

---

## Best Practices

:::note Best Practices

- **Use Deletion Policies wisely** — Use Retain for production databases and S3 buckets with important data, Delete for dev/test environments
- **Tag all resources** — Include Environment, Application, Team, CostCenter tags for governance and cost allocation
- **Use K8s Secrets for sensitive data** — Never hardcode credentials in CRs, always reference Secrets for passwords and keys
- **Enable encryption everywhere** — S3, RDS, EBS, ElastiCache - all should have encryption enabled for compliance
- **Multi-AZ for production** — Enable multiAZ for RDS, ElastiCache, and spread subnets across AZs for high availability
- **Monitor resource status** — Use kubectl get/describe commands to check resource health, set up alerts on status.ready

:::

## Next Steps

- **RouteTable Controller**: Implement route table management for custom routing
- **SecurityGroup Controller**: Implement security group rules management
- **EKS Support**: Add EKS cluster provisioning
- **Cross-Resource References**: Use status fields to reference resources automatically

---

**Generated by Infra Operator**
Version: v1.0.0
Last Updated: 2025-11-22
