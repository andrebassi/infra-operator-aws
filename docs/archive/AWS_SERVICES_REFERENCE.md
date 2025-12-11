# AWS Services Reference - Infra Operator

Este documento lista todos os serviços AWS suportados pelo infra-operator, com exemplos de uso e configuração.

## Índice

1. [Credentials & Identity](#credentials--identity)
2. [Compute](#compute)
3. [Storage](#storage)
4. [Database](#database)
5. [Networking](#networking)
6. [Messaging & Events](#messaging--events)
7. [Security](#security)

---

## Credentials & Identity

### AWSProvider

**Status**: ✅ Controller implementado
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/AWSProvider`
**Shortnames**: `awsp`

Gerencia credenciais AWS para outros recursos.

**Exemplos**:

```yaml
# IRSA (IAM Roles for Service Accounts) - Recomendado
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: production
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role
  defaultTags:
    environment: production
    managed-by: infra-operator
```

```yaml
# Static Credentials
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: dev
spec:
  region: us-west-2
  accessKeyIDRef:
    name: aws-credentials
    key: access-key-id
  secretAccessKeyRef:
    name: aws-credentials
    key: secret-access-key
```

---

## Compute

### Lambda Function

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/LambdaFunction`
**Shortnames**: `lambda`

Gerencia funções AWS Lambda serverless.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: api-processor
spec:
  providerRef:
    name: production
  functionName: api-data-processor
  runtime: python3.11
  handler: lambda_function.lambda_handler
  role: arn:aws:iam::123456789012:role/lambda-execution-role

  # Código do S3
  code:
    s3Bucket: my-lambda-code
    s3Key: functions/processor-v1.0.0.zip

  # Configuração
  memorySize: 512
  timeout: 30

  # Variáveis de ambiente
  environment:
    LOG_LEVEL: INFO
    DYNAMODB_TABLE: data-table

  # VPC (opcional)
  vpcConfig:
    subnetIds:
      - subnet-abc123
      - subnet-def456
    securityGroupIds:
      - sg-123456

  # Dead Letter Queue
  deadLetterConfig:
    targetArn: arn:aws:sqs:us-east-1:123456789012:dlq

  # X-Ray tracing
  tracingMode: Active

  # Concorrência reservada
  reservedConcurrentExecutions: 10

  tags:
    team: backend
    application: api
```

```yaml
# Lambda com código inline (até 4KB)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: simple-function
spec:
  providerRef:
    name: dev
  functionName: hello-world
  runtime: python3.11
  handler: index.handler
  role: arn:aws:iam::123456789012:role/lambda-role

  code:
    zipFile: |
      import json
      def handler(event, context):
          return {
              'statusCode': 200,
              'body': json.dumps('Hello from Lambda!')
          }

  memorySize: 128
  timeout: 3
```

```yaml
# Lambda com container image
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: container-function
spec:
  providerRef:
    name: production
  functionName: ml-inference
  runtime: "" # Não necessário para container
  handler: "" # Não necessário para container
  role: arn:aws:iam::123456789012:role/lambda-container-role

  code:
    imageUri: 123456789012.dkr.ecr.us-east-1.amazonaws.com/ml-model:latest

  memorySize: 2048
  timeout: 60
```

### EC2 Instance

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/EC2Instance`
**Shortnames**: `ec2`

Gerencia instâncias EC2.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EC2Instance
metadata:
  name: web-server
spec:
  providerRef:
    name: production
  instanceName: web-server-01
  instanceType: t3.medium
  imageID: ami-0c55b159cbfafe1f0  # Amazon Linux 2

  keyName: my-keypair
  subnetID: subnet-abc123
  securityGroupIDs:
    - sg-web-servers

  # IAM instance profile
  iamInstanceProfile: arn:aws:iam::123456789012:instance-profile/ec2-role

  # User data script
  userData: |
    #!/bin/bash
    yum update -y
    yum install -y nginx
    systemctl start nginx
    systemctl enable nginx

  # EBS volumes
  blockDeviceMappings:
    - deviceName: /dev/xvda
      ebs:
        volumeSize: 30
        volumeType: gp3
        deleteOnTermination: true
        encrypted: true
    - deviceName: /dev/xvdf
      ebs:
        volumeSize: 100
        volumeType: gp3
        deleteOnTermination: false

  monitoring: true
  ebsOptimized: true

  tags:
    Name: web-server-01
    Role: webserver
```

---

## Storage

### S3 Bucket

**Status**: ✅ Controller implementado
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/S3Bucket`
**Shortnames**: `s3`

Gerencia buckets S3 com configuração completa.

**Exemplo** (ver config/samples/s3bucket_sample.yaml para mais exemplos):

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: data-lake
spec:
  providerRef:
    name: production
  bucketName: company-data-lake-prod

  versioning:
    enabled: true

  encryption:
    algorithm: aws:kms
    kmsKeyID: arn:aws:kms:us-east-1:123456789012:key/abc-123

  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true

  lifecycleRules:
    - id: archive-old-data
      enabled: true
      prefix: raw-data/
      transitions:
        - days: 30
          storageClass: STANDARD_IA
        - days: 90
          storageClass: GLACIER
      expiration:
        days: 2555  # 7 years

  tags:
    data-classification: confidential
    compliance: gdpr

  deletionPolicy: Retain
```

---

## Database

### RDS Instance

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/RDSInstance`
**Shortnames**: `rds`

Gerencia instâncias RDS (PostgreSQL, MySQL, MariaDB, etc).

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: app-database
spec:
  providerRef:
    name: production
  dbInstanceIdentifier: app-db-prod
  engine: postgres
  engineVersion: "15.4"
  dbInstanceClass: db.r6g.xlarge
  allocatedStorage: 100
  storageType: gp3

  masterUsername: dbadmin
  masterPasswordRef:
    name: db-credentials
    key: password

  dbName: application

  vpcSecurityGroupIDs:
    - sg-database

  dbSubnetGroupName: private-subnets

  multiAZ: true
  publiclyAccessible: false

  backupRetentionPeriod: 30
  preferredBackupWindow: "03:00-04:00"
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  storageEncrypted: true
  kmsKeyID: arn:aws:kms:us-east-1:123456789012:key/abc-123

  enableCloudwatchLogsExports:
    - postgresql

  deletionProtection: true
  deletionPolicy: Snapshot
  finalSnapshotIdentifier: app-db-final-snapshot

  tags:
    application: main-app
    tier: database
```

### DynamoDB Table

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/DynamoDBTable`
**Shortnames**: `dynamodb`, `ddb`

Gerencia tabelas DynamoDB NoSQL.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: user-sessions
spec:
  providerRef:
    name: production
  tableName: user-sessions

  # Atributos
  attributeDefinitions:
    - attributeName: userId
      attributeType: S
    - attributeName: sessionId
      attributeType: S
    - attributeName: createdAt
      attributeType: N

  # Chave primária
  keySchema:
    - attributeName: userId
      keyType: HASH
    - attributeName: sessionId
      keyType: RANGE

  # On-demand billing (recomendado)
  billingMode: PAY_PER_REQUEST

  # Global Secondary Index
  globalSecondaryIndexes:
    - indexName: SessionsByDate
      keySchema:
        - attributeName: userId
          keyType: HASH
        - attributeName: createdAt
          keyType: RANGE
      projection:
        projectionType: ALL

  # DynamoDB Streams
  streamSpecification:
    streamEnabled: true
    streamViewType: NEW_AND_OLD_IMAGES

  # Encryption at rest
  sseSpecification:
    enabled: true
    sseType: KMS
    kmsMasterKeyId: arn:aws:kms:us-east-1:123456789012:key/abc-123

  # TTL
  timeToLiveSpecification:
    enabled: true
    attributeName: expiresAt

  # Point-in-time recovery
  pointInTimeRecoveryEnabled: true

  # Deletion protection
  deletionProtectionEnabled: true

  tags:
    application: auth-service
```

```yaml
# Provisioned billing com auto-scaling
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: products
spec:
  providerRef:
    name: production
  tableName: products

  attributeDefinitions:
    - attributeName: productId
      attributeType: S

  keySchema:
    - attributeName: productId
      keyType: HASH

  billingMode: PROVISIONED
  provisionedThroughput:
    readCapacityUnits: 5
    writeCapacityUnits: 5
```

### ElastiCache Cluster

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/ElastiCacheCluster`
**Shortnames**: `elasticache`, `cache`

Gerencia clusters Redis e Memcached.

**Exemplo Redis**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: session-cache
spec:
  providerRef:
    name: production
  clusterID: session-cache-prod
  engine: redis
  engineVersion: "7.0"
  nodeType: cache.r6g.large

  # Redis replication group
  replicationGroupDescription: "Session cache with HA"
  numNodeGroups: 2  # Cluster mode
  replicasPerNodeGroup: 2  # 2 replicas per shard
  automaticFailoverEnabled: true
  multiAZEnabled: true

  # Network
  subnetGroupName: cache-subnets
  securityGroupIds:
    - sg-cache

  # Backups
  snapshotRetentionLimit: 7
  snapshotWindow: "03:00-05:00"
  preferredMaintenanceWindow: "sun:05:00-sun:07:00"

  # Security
  atRestEncryptionEnabled: true
  transitEncryptionEnabled: true
  authTokenRef:
    name: redis-auth
    key: token

  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/abc-123

  tags:
    purpose: session-storage
```

```yaml
# Memcached cluster
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
metadata:
  name: app-cache
spec:
  providerRef:
    name: dev
  clusterID: app-cache-dev
  engine: memcached
  engineVersion: "1.6.17"
  nodeType: cache.t3.micro
  numCacheNodes: 2

  subnetGroupName: cache-subnets
  securityGroupIds:
    - sg-cache
```

---

## Messaging & Events

### SQS Queue

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/SQSQueue`
**Shortnames**: `sqs`

Gerencia filas SQS.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: order-processing
spec:
  providerRef:
    name: production
  queueName: order-processing

  # FIFO queue
  fifoQueue: true
  contentBasedDeduplication: true

  # Message configuration
  visibilityTimeout: 300  # 5 minutes
  messageRetentionPeriod: 1209600  # 14 days
  maxMessageSize: 262144  # 256 KB
  receiveMessageWaitTimeSeconds: 20  # Long polling

  # Dead Letter Queue
  deadLetterQueue:
    targetARN: arn:aws:sqs:us-east-1:123456789012:order-dlq.fifo
    maxReceiveCount: 3

  # Encryption
  kmsMasterKeyID: arn:aws:kms:us-east-1:123456789012:key/abc-123
  kmsDataKeyReusePeriodSeconds: 300

  tags:
    application: order-system
```

```yaml
# Standard queue simples
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
metadata:
  name: notifications
spec:
  providerRef:
    name: production
  queueName: app-notifications

  visibilityTimeout: 30
  messageRetentionPeriod: 345600  # 4 days
```

### SNS Topic

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/SNSTopic`
**Shortnames**: `sns`

Gerencia tópicos SNS para pub/sub.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
metadata:
  name: order-events
spec:
  providerRef:
    name: production
  topicName: order-events

  # FIFO topic
  fifoTopic: true
  contentBasedDeduplication: true

  # Encryption
  kmsMasterKeyId: arn:aws:kms:us-east-1:123456789012:key/abc-123

  # Subscriptions
  subscriptions:
    # SQS subscription
    - protocol: sqs
      endpoint: arn:aws:sqs:us-east-1:123456789012:order-fulfillment
      rawMessageDelivery: true

    # Lambda subscription
    - protocol: lambda
      endpoint: arn:aws:lambda:us-east-1:123456789012:function:process-order

    # Email subscription
    - protocol: email
      endpoint: alerts@company.com

    # HTTP endpoint com filter
    - protocol: https
      endpoint: https://api.company.com/webhooks/orders
      filterPolicy: |
        {
          "eventType": ["order.created", "order.completed"]
        }

  tags:
    application: order-system
```

---

## Security

### KMS Key

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/KMSKey`
**Shortnames**: `kms`

Gerencia chaves de criptografia KMS.

**Exemplo**:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: app-data-key
spec:
  providerRef:
    name: production
  description: "Encryption key for application data"

  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT

  # Auto-rotation anual
  enableKeyRotation: true

  # Multi-region key
  multiRegion: false

  # Key policy (IAM JSON)
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
          "Sid": "Allow use by application",
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::123456789012:role/app-role"
          },
          "Action": [
            "kms:Decrypt",
            "kms:Encrypt",
            "kms:GenerateDataKey"
          ],
          "Resource": "*"
        }
      ]
    }

  tags:
    application: main-app
    compliance: pci-dss

  deletionPolicy: Retain
  pendingWindowInDays: 30
```

### Secrets Manager Secret

**Status**: 🟡 CRD definido (controller pendente)
**CRD**: `aws-infra-operator.runner.codes/v1alpha1/SecretsManagerSecret`
**Shortnames**: `awssecret`, `sm`

Gerencia secrets no AWS Secrets Manager.

**Exemplo**:

```yaml
# Kubernetes Secret com o valor
apiVersion: v1
kind: Secret
metadata:
  name: db-password-source
type: Opaque
stringData:
  password: MyS3cr3tP@ssw0rd!
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: database-credentials
spec:
  providerRef:
    name: production
  secretName: prod/database/credentials

  description: "Database master password"

  # Referência ao Secret do Kubernetes
  secretStringRef:
    name: db-password-source
    key: password

  # Encryption
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/abc-123

  # Automatic rotation
  rotationEnabled: true
  rotationLambdaARN: arn:aws:lambda:us-east-1:123456789012:function:rotate-db-password
  automaticallyAfterDays: 30

  tags:
    application: main-app
    environment: production

  deletionPolicy: Retain
  recoveryWindowInDays: 30
```

```yaml
# Secrets complexos (JSON)
apiVersion: v1
kind: Secret
metadata:
  name: api-keys-source
type: Opaque
stringData:
  credentials: |
    {
      "api_key": "AKIAXXXXXXXXXXXXXXXX",
      "api_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "webhook_url": "https://api.example.com/webhook"
    }
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: external-api-credentials
spec:
  providerRef:
    name: production
  secretName: prod/external-api/credentials

  secretStringRef:
    name: api-keys-source
    key: credentials

  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/abc-123
```

---

## Resumo de Status

| Serviço | CRD | Controller | Prioridade |
|---------|-----|------------|------------|
| **AWSProvider** | ✅ | ✅ | Alta |
| **S3Bucket** | ✅ | ✅ | Alta |
| **Lambda** | ✅ | 🟡 | **Alta** |
| **DynamoDB** | ✅ | 🟡 | **Alta** |
| **RDS** | ✅ | 🟡 | Alta |
| **ElastiCache** | ✅ | 🟡 | Média |
| **SQS** | ✅ | 🟡 | Média |
| **SNS** | ✅ | 🟡 | Média |
| **EC2** | ✅ | 🟡 | Baixa |
| **KMS** | ✅ | 🟡 | Média |
| **Secrets Manager** | ✅ | 🟡 | Média |

**Legenda**:
- ✅ = Implementado e funcional
- 🟡 = CRD definido, controller pendente
- ❌ = Não implementado

## Próximos Controllers a Implementar

**Prioridade Alta** (Cloud Native essenciais):
1. **Lambda** - Serverless compute
2. **DynamoDB** - NoSQL database

**Prioridade Média** (Infraestrutura):
3. **SQS** - Message queuing
4. **SNS** - Pub/sub messaging
5. **ElastiCache** - In-memory cache
6. **KMS** - Encryption keys
7. **Secrets Manager** - Secret storage

**Prioridade Baixa**:
8. **RDS** - Já tem alternativas (CloudNativePG, etc)
9. **EC2** - Workloads devem estar em containers

## Padrões Comuns

### 1. Referências entre Recursos

```yaml
# KMS Key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: app-key
spec:
  providerRef:
    name: production
  description: "Application encryption key"
---
# S3 Bucket usando a KMS Key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: encrypted-bucket
spec:
  providerRef:
    name: production
  bucketName: encrypted-data
  encryption:
    algorithm: aws:kms
    kmsKeyID: "{{ .status.keyId }}"  # Referência ao KMS Key
```

### 2. Multi-Environment Pattern

```yaml
# Production
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: users
  namespace: production
spec:
  providerRef:
    name: aws-prod
  tableName: users-prod
  billingMode: PROVISIONED
  provisionedThroughput:
    readCapacityUnits: 100
    writeCapacityUnits: 50
  deletionProtectionEnabled: true
---
# Development
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
metadata:
  name: users
  namespace: development
spec:
  providerRef:
    name: aws-dev
  tableName: users-dev
  billingMode: PAY_PER_REQUEST
  deletionProtectionEnabled: false
```

### 3. GitOps Ready

Todos os CRDs são projetados para serem gerenciados via GitOps (ArgoCD, Flux):

```yaml
# argocd-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: infrastructure
spec:
  project: default
  source:
    repoURL: https://github.com/company/infra
    path: aws-resources
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: infrastructure
  syncPolicy:
    automated:
      prune: false  # Cuidado com deleção automática!
      selfHeal: true
```

---

Para mais detalhes sobre cada CRD, consulte os arquivos em `api/v1alpha1/*_types.go`.
