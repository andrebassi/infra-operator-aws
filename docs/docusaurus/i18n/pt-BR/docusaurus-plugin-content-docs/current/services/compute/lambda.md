---
title: 'Lambda Function - Serverless Computing'
description: 'Run code without managing servers with serverless functions'
sidebar_position: 3
---

Run code without managing servers. AWS Lambda is a serverless compute service that executes your code in response to events and automatically manages the required compute resources.

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

**IAM Policy - Lambda (lambda-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "lambda:CreateFunction",
        "lambda:DeleteFunction",
        "lambda:GetFunction",
        "lambda:GetFunctionConfiguration",
        "lambda:UpdateFunctionCode",
        "lambda:UpdateFunctionConfiguration",
        "lambda:TagResource",
        "lambda:UntagResource",
        "lambda:ListTags",
        "lambda:PublishVersion",
        "lambda:CreateAlias",
        "lambda:UpdateAlias"
      ],
      "Resource": "*"
},
{
      "Effect": "Allow",
      "Action": [
        "iam:PassRole"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "iam:PassedToService": "lambda.amazonaws.com"
        }
      }
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
  --role-name infra-operator-lambda-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator Lambda management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-lambda-role \
  --policy-name LambdaManagement \
  --policy-document file://lambda-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-lambda-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-lambda-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

AWS Lambda allows you to run code without provisioning or managing servers. You simply upload your code and Lambda runs it with high availability. You only pay for the compute time you use - there's no charge when your code isn't running.

**Features:**
- Run code without managing servers (serverless)
- Automatic and instant scalability
- Pay only for execution time (100ms granularity)
- Multiple languages: Python, Node.js, Java, Go, C#, Ruby
- Native integration with AWS services (S3, DynamoDB, SQS, API Gateway)
- VPC support for accessing private databases
- Layers for sharing code and libraries
- Dead Letter Queues (DLQ) for failures
- Reserved Concurrency for control
- X-Ray tracing

## Quick Start

**Lambda Python:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: e2e-hello-function
  namespace: default
spec:
  providerRef:
    name: localstack
  functionName: e2e-hello-world
  runtime: python3.12
  handler: index.handler
  role: arn:aws:iam::000000000000:role/lambda-role
  timeout: 30
  memorySize: 256
  code:
zipFile: UEsDBBQAAAAIAHRLdluO3E/MUgAAAFMAAAAIABwAaW5kZXgucHlVVAkAAzusIWk7rCFpdXgLAAEE9QEAAAQAAAAAS0lNU8hIzEvJSS3SSC1LzSvRUUjOzytJrSjRtOJSAIKi1JLSojyFavXiksSS0mLn/JRUdSsFIwMDHQX1pPyUSiBH3SM1JydfITy/KCdFvZYLAFBLAQIeAxQAAAAIAHRLdluO3E/MUgAAAFMAAAAIABgAAAAAAAEAAACkgQAAAABpbmRleC5weVVUBQADO6whaXV4CwABBPUBAAAEAAAAAFBLBQYAAAAAAQABAE4AAACUAAAAAAA=
  environment:
variables:
      ENV: test
      LOG_LEVEL: debug
  tags:
    Environment: test
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**Lambda Node.js:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: e2e-nodejs-function
  namespace: default
spec:
  providerRef:
    name: localstack
  functionName: e2e-nodejs-handler
  runtime: nodejs20.x
  handler: index.handler
  role: arn:aws:iam::000000000000:role/lambda-role
  timeout: 10
  memorySize: 128
  code:
zipFile: UEsDBBQAAAAIANxLdlvWeVkhYQAAAGQAAAAIABwAaW5kZXguanNVVAkAAwCtIWkArSFpdXgLAAEE9QEAAAQAAAAADckxCoNAEEbh3lP8nRGCiKVBmzSpvMPqTgiy7sjMrCiSu7uPr3t0bCym9c9FH0jQw+kZZzxop2gV+gFXgZyQJYm4oOYs6Zs9dWib5omJ/dmh/FAIjK/wijHPetES/1eR3VBLAQIeAxQAAAAIANxLdlvWeVkhYQAAAGQAAAAIABgAAAAAAAEAAACkgQAAAABpbmRleC5qc1VUBQADAK0haXV4CwABBPUBAAAEAAAAAFBLBQYAAAAAAQABAE4AAACjAAAAAAA=
  tags:
    Environment: test
    Runtime: nodejs
  deletionPolicy: Delete
```

**Production Lambda:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: process-orders
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Function name in AWS
  functionName: process-orders

  # Runtime and handler
  runtime: python3.11
  handler: index.handler

  # Inline code (simple)
  code:
zipFile: |
      import json
      def handler(event, context):
          return {
              'statusCode': 200,
              'body': json.dumps('Hello World!')
          }

  # IAM Role for execution
  role: arn:aws:iam::123456789012:role/lambda-execution-role

  # Configuration
  memorySize: 256
  timeout: 30

  # Environment variables
  environment:
variables:
      ENV: production
      DB_HOST: db.example.com

  tags:
    Environment: production
    Application: orders-processor
```

**Apply:**

```bash
kubectl apply -f lambda.yaml
```

**Check Status:**

```bash
kubectl get lambdafunctions
kubectl describe lambdafunction e2e-hello-function
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource for authentication

  AWSProvider resource name

Unique Lambda function name in AWS (between 1 and 64 characters)

  **Example:**

  ```yaml
  functionName: my-processor-function
  ```

Function runtime environment

  **Options:**
  - `python3.11`, `python3.10`, `python3.9`
  - `nodejs20.x`, `nodejs18.x`
  - `go1.x`
  - `ruby3.3`, `ruby3.2`
  - `java21`, `java17`, `java11`
  - `dotnet8`, `dotnet6`
  - `provided.al2` (custom runtimes)

Function that Lambda invokes in the code file

  **Format:** `filename.function-name`:

  ```yaml
  handler: index.handler      # Python/Node: index file, handler function
  handler: main.main          # Go: main package, main function
  handler: Main               # Java: Main class with handler method
  ```

Lambda function code

  Inline code (for small functions)

**Example:**

      ```yaml
      code:
        zipFile: |
          def handler(event, context):
              return 'Hello'
      ```

S3 bucket name containing the code

ZIP file key in S3 bucket

IAM Role ARN that Lambda assumes during execution

  **Example:**

  ```yaml
  role: arn:aws:iam::123456789012:role/lambda-execution-role
  ```

  The role MUST have trust policy for Lambda:
  ```json
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
  ```

### Optional Fields

Memory allocated to the function (MB)

  **Range:** 128 - 10,240 MB (in 1 MB increments):

  ```yaml
  memorySize: 512  # 512 MB RAM
  ```

  **Note:** CPU is allocated proportionally to memory:
  - 128 MB = 0.08 vCPU
  - 1,769 MB = 1 full vCPU
  - 10,240 MB = 6 vCPU

Maximum execution time in seconds

  **Range:** 3 - 900 seconds (15 minutes):

  ```yaml
  timeout: 60  # 1 minute
  ```

Environment variables accessible to the code

  Key-value pairs of environment variables

**Example:**

      ```yaml
      environment:
        variables:
          DATABASE_URL: postgres://user:pass@db.example.com/mydb
          ENVIRONMENT: production
          DEBUG: "false"
      ```

Configuration to run Lambda inside a VPC

  List of security group IDs

**Example:**

      ```yaml
      vpcConfig:
        securityGroupIds:
          - sg-0123456789abcdef0
      ```

List of subnet IDs

**Example:**

      ```yaml
      vpcConfig:
        subnetIds:
          - subnet-0123456789abcdef0
          - subnet-0123456789abcdef1
      ```

ARNs of Lambda Layers containing shared libraries

  **Example:**

  ```yaml
  layers:
- arn:aws:lambda:us-east-1:123456789012:layer:my-dependencies:1
- arn:aws:lambda:us-east-1:123456789012:layer:custom-utilities:2
  ```

Dead Letter Queue configuration for failures

  SQS queue ARN or SNS topic for failed messages

**Example:**

      ```yaml
      deadLetterConfig:
        targetArn: arn:aws:sqs:us-east-1:123456789012:my-dlq
      ```

Temporary space available in `/tmp` (MB)

  **Range:** 512 - 10,240 MB:

  ```yaml
  ephemeralStorage: 1024  # 1 GB in /tmp
  ```

Maximum number of concurrent executions

  **Example:**

  ```yaml
  reservedConcurrentExecutions: 100
  ```

  Useful for controlling costs or avoiding throttling of dependencies.

Processor architecture

  **Options:**
  - `x86_64` (default)
  - `arm64` (AWS Graviton, cheaper)

  **Example:**

  ```yaml
  architectures:
- arm64
  ```

Key-value pairs to tag the function

  **Example:**

  ```yaml
  tags:
    Application: order-processor
    Team: backend
    Environment: production
    CostCenter: engineering
  ```

What happens to the function when the CR is deleted

  **Options:**
  - `Delete`: Function is deleted from AWS
  - `Retain`: Function remains in AWS but unmanaged

## Status Fields

After the function is created, the following status fields are populated:

Full Lambda function ARN

  ```
  arn:aws:lambda:us-east-1:123456789012:function:my-function
  ```

Lambda function name

Published function version (e.g., `$LATEST`, `1`, `2`, etc)

Current function state
  - `Pending`: Function is being created
  - `Active`: Function is active and ready to use
  - `Inactive`: Function has been deactivated
  - `Failed`: Creation failed

Last modification timestamp

  ```
  2025-11-22T20:30:15Z
  ```

Code size in bytes

Allocated memory in MB

Timeout in seconds

`true` when the function is active and ready for invocation

Timestamp of last synchronization with AWS

## Examples

### Basic Lambda - Hello World

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: hello-world
  namespace: default
spec:
  providerRef:
    name: production-aws

  functionName: hello-world

  runtime: python3.11
  handler: index.handler

  code:
zipFile: |
      import json
      def handler(event, context):
          return {
              'statusCode': 200,
              'body': json.dumps({
                  'message': 'Hello, World!',
                  'input': event
              })
          }

  role: arn:aws:iam::123456789012:role/lambda-execution-role
  memorySize: 128
  timeout: 10

  tags:
    Environment: production
    Type: demo
```

### Lambda with S3 Trigger

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: s3-image-processor
  namespace: default
spec:
  providerRef:
    name: production-aws

  functionName: s3-image-processor

  runtime: python3.11
  handler: processor.handler

  code:
s3Bucket: my-lambda-code-bucket
s3Key: image-processor.zip

  role: arn:aws:iam::123456789012:role/lambda-s3-execution-role

  memorySize: 512
  timeout: 60
  ephemeralStorage: 2048

  environment:
variables:
      OUTPUT_BUCKET: processed-images
      LOG_LEVEL: INFO

  tags:
    Application: image-processing
    Environment: production
```

### Lambda with VPC and Database Access

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: db-query-function
  namespace: default
spec:
  providerRef:
    name: production-aws

  functionName: db-query-function

  runtime: python3.11
  handler: database.query_handler

  code:
zipFile: |
      import json
      import psycopg2
      import os

      def query_handler(event, context):
          conn = psycopg2.connect(
              host=os.getenv('DB_HOST'),
              user=os.getenv('DB_USER'),
              password=os.getenv('DB_PASSWORD'),
              database=os.getenv('DB_NAME')
          )
          cursor = conn.cursor()
          cursor.execute('SELECT * FROM users LIMIT 10')
          users = cursor.fetchall()
          cursor.close()
          conn.close()

          return {
              'statusCode': 200,
              'body': json.dumps({'users': users})
          }

  role: arn:aws:iam::123456789012:role/lambda-db-role

  memorySize: 256
  timeout: 30

  # Lambda runs inside VPC
  vpcConfig:
    securityGroupIds:
      - sg-0123456789abcdef0
subnetIds:
      - subnet-0123456789abcdef0
      - subnet-0123456789abcdef1

  environment:
variables:
      DB_HOST: postgres.internal.example.com
      DB_NAME: production
      DB_USER: lambda_user
      # DB_PASSWORD via Secrets Manager, not environment variables!

  tags:
    Application: user-service
    Environment: production
```

### Lambda with Layers and Dependencies

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

  functionName: api-handler

  runtime: python3.11
  handler: api.handler

  code:
zipFile: |
      import json
      import requests  # Comes from layer
      from utilities import format_response  # Comes from layer

      def handler(event, context):
          response = requests.get('https://api.example.com/data')
          formatted = format_response(response.json())
          return formatted

  # Layers contain shared libraries
  layers:
- arn:aws:lambda:us-east-1:123456789012:layer:python-dependencies:2
- arn:aws:lambda:us-east-1:123456789012:layer:custom-utilities:1

  role: arn:aws:iam::123456789012:role/lambda-api-role

  memorySize: 256
  timeout: 15

  tags:
    Application: api
    Environment: production
```

### Lambda with Dead Letter Queue (DLQ)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: async-processor
  namespace: default
spec:
  providerRef:
    name: production-aws

  functionName: async-processor

  runtime: python3.11
  handler: processor.handler

  code:
zipFile: |
      import json
      def handler(event, context):
          # Process queue messages
          try:
              process_record(event)
              return {'statusCode': 200}
          except Exception as e:
              # If it fails, message goes to DLQ
              print(f"Error: {str(e)}")
              raise

  # Queue for failed messages
  deadLetterConfig:
targetArn: arn:aws:sqs:us-east-1:123456789012:failed-messages

  role: arn:aws:iam::123456789012:role/lambda-processor-role

  memorySize: 512
  timeout: 300  # 5 minutes for heavy processing

  reservedConcurrentExecutions: 10  # Maximum 10 concurrent executions

  environment:
variables:
      QUEUE_NAME: async-jobs
      RETRY_COUNT: "3"

  tags:
    Application: async-processing
    Environment: production
```

### Lambda with Reserved Concurrency

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: critical-handler
  namespace: default
spec:
  providerRef:
    name: production-aws

  functionName: critical-handler

  runtime: nodejs20.x
  handler: index.handler

  code:
zipFile: |
      exports.handler = async (event) => {
          return {
              statusCode: 200,
              body: JSON.stringify('Critical function executed')
          };
      };

  role: arn:aws:iam::123456789012:role/lambda-critical-role

  memorySize: 1024
  timeout: 60

  # Reserved Concurrency guarantees capacity
  reservedConcurrentExecutions: 100

  # ARM is cheaper than x86
  architectures:
- arm64

  tags:
    Application: critical-service
    CostOptimization: arm64-graviton
    Environment: production
```

## Verification

### Check Function Status

**Command:**

```bash
# List all Lambda functions
kubectl get lambdafunctions

# Get detailed information
kubectl get lambdafunction db-query-function -o yaml

# Watch creation in real-time
kubectl get lambdafunction my-function -w
```

### Verify in AWS

**AWS CLI:**

```bash
# List Lambda functions
aws lambda list-functions

# Get function details
aws lambda get-function \
      --function-name my-function \
      --query 'Configuration' \
      --output json

# Invoke function for testing
aws lambda invoke \
      --function-name my-function \
      --payload '{"key": "value"}' \
      response.json && cat response.json

# View recent logs
aws logs tail /aws/lambda/my-function --follow
```

**LocalStack:**

```bash
# For testing with LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws lambda list-functions

# Invoke function
aws lambda invoke \
      --function-name my-function \
      --payload '{"test": true}' \
      response.json

# View logs
aws logs tail /aws/lambda/my-function --follow
```

### Expected Output

**Example:**

```yaml
status:
  functionArn: arn:aws:lambda:us-east-1:123456789012:function:my-function
  functionName: my-function
  version: $LATEST
  state: Active
  lastModified: "2025-11-22T20:30:15.000+0000"
  codeSize: 2048
  memorySize: 256
  timeout: 30
  ready: true
  lastSyncTime: "2025-11-22T20:30:15Z"
```

## Troubleshooting

### Timeout errors - Function takes too long

**Symptoms:** Invoked functions terminate with timeout error

**Common causes:**
1. Timeout configured too short for the operation
2. Slow database connections
3. Cold start (first invocation)
4. Heavy processing without parallelization
5. Network latency in VPC

**Solutions:**
```bash
# Increase timeout (maximum 900 seconds)
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"timeout":60}}'

# Increase memory (improves CPU)
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"memorySize":512}}'

# Use Reserved Concurrency to avoid cold starts
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"reservedConcurrentExecutions":10}}'

# View logs to identify bottleneck
aws logs tail /aws/lambda/my-function --follow
```

### Out of memory - Function without enough memory

**Symptoms:** Error `Process exited before completing request` or `Memory exceeded`

**Common causes:**
1. Insufficient allocated memory
2. Memory leak in code
3. Large unoptimized dependencies
4. Unlimited cache in function

**Solutions:**
```bash
# Increase allocated memory
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"memorySize":1024}}'

# Optimize dependencies - remove unused libraries
# Use layer for dependencies
# Use Lambda Layers for shared code
```

### Cold starts - First invocation slow

**Symptoms:** First invocation is slow, subsequent invocations are fast

**Causes:**
1. Lambda needs to create new container
2. Heavy imports/initializations
3. Database connections at initialization

**Solutions:**
```bash
# Use Reserved Concurrency to keep containers active
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"reservedConcurrentExecutions":10}}'

# Provisioned Concurrency (additional cost but no cold starts)
# Not available in this CR, configure in AWS Console if needed

# Optimized code - initialize outside handler
# ✅ GOOD - Initialize once when container is created
import db
connection = db.connect()  # Once

def handler(event, context):
        return connection.query()  # Reuse connection

# ❌ BAD - Initialize on every invocation
def handler(event, context):
        connection = db.connect()  # Every invocation!
        return connection.query()
```
  
### VPC networking issues - Slow VPC connection

**Symptoms:** Lambda in VPC has high latency or cannot connect to database

**Common causes:**
1. Security group doesn't allow outbound connections
2. NAT Gateway not configured for private subnets
3. Route tables not configured correctly
4. DNS doesn't resolve within VPC

**Solutions:**
```bash
# Verify security group allows outbound traffic
aws ec2 describe-security-groups --group-ids sg-xxx

# Should have egress rule like:
# Protocol: TCP, Port: 5432, CIDR: 10.0.0.0/16

# Verify ENI created correctly
aws ec2 describe-network-interfaces \
      --filters "Name=description,Values=*lambda*"

# Increase timeout - VPC can be slower
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"timeout":60}}'

# Increase ephemeral storage if using /tmp
kubectl patch lambdafunction my-function --type merge \
      -p '{"spec":{"ephemeralStorage":1024}}'
```

### Permission denied - IAM Role error

**Symptoms:** `User: arn:aws:iam::xxx is not authorized` or `AccessDenied`

**Causes:**
1. Incorrect Role ARN
2. Role doesn't have necessary permissions
3. Role doesn't allow Lambda to assume it

**Solutions:**
```bash
# Verify role exists
aws iam get-role --role-name lambda-execution-role

# Verify trust policy (assume role)
aws iam get-role --role-name lambda-execution-role \
      --query 'Role.AssumeRolePolicyDocument'

# MUST contain:
{
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
}

# Add necessary permissions
# For S3:
aws iam attach-role-policy \
      --role-name lambda-execution-role \
      --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess

# For DynamoDB:
aws iam attach-role-policy \
      --role-name lambda-execution-role \
      --policy-arn arn:aws:iam::aws:policy/AmazonDynamoDBReadOnlyAccess

# Check Lambda logs for specific error
aws logs tail /aws/lambda/my-function --follow
```

### DLQ not receiving messages - Dead Letter Queue doesn't work

**Symptoms:** Function fails but message doesn't reach DLQ

**Causes:**
1. Incorrect DLQ Target ARN
2. Role doesn't have permission to send to DLQ
3. DLQ (SQS/SNS) doesn't exist

**Solutions:**
```bash
# Verify DLQ exists
aws sqs get-queue-attributes \
      --queue-url https://queue.amazonaws.com/123456789012/failed-queue \
      --attribute-names All

# Role MUST have SendMessage permission
aws iam get-role-policy \
      --role-name lambda-role \
      --policy-name inline-policy

# Must contain:
{
      "Effect": "Allow",
      "Action": [
        "sqs:SendMessage",
        "sns:Publish"
      ],
      "Resource": "arn:aws:sqs:*:*:*"
}

# Update DLQ
kubectl patch lambdafunction my-function --type merge -p \
      '{"spec":{"deadLetterConfig":{"targetArn":"arn:aws:sqs:us-east-1:123456789012:new-dlq"}}}'
```

### Functions don't appear in kubectl

**Symptoms:** Created AWS function directly, doesn't appear in kubectl

**Cause:** Only functions created via infra-operator appear in kubectl

**Solutions:**
1. Create Lambda via infra-operator CR
2. Or import existing function via Kubernetes secret/configmap
3. Or manage via AWS console and integrate with application

**Command:**

```bash
# Check functions in cluster
kubectl get lambdafunctions

# If empty, none were created via operator
# Create a new one via CR
```
  
## Best Practices

:::note Best Practices

- **Right-size memory allocation** — Memory affects CPU proportionally (1,769 MB = 1 full vCPU), test to find optimal size, use CloudWatch to monitor MaxMemoryUsed
- **Use Lambda Layers** — Share libraries across functions, reduce code size, easier dependency versioning, better code organization
- **Configure Dead Letter Queues** — Handle failures gracefully, prevent data loss, monitor DLQ for issues, set up alerts when messages arrive
- **Enable X-Ray tracing** — Identify bottlenecks, visualize dependencies, trace across multiple services (requires xray:* IAM permission)
- **Use environment variables wisely** — Good for configuration, different values per environment, easy to change without redeploy
- **Never store secrets in environment variables** — Use AWS Secrets Manager for credentials, automatic rotation, access auditing, native Lambda integration
- **Set Reserved Concurrency** — Guarantees capacity, protects dependencies from throttling, limits costs (default max 1000)
- **Use Versions and Aliases** — Publish versions for production, use aliases for staging/prod, enables easy rollback and canary deployments
- **Keep functions small and focused** — Single responsibility, separate handler logic, easier to test, smaller functions are faster and cheaper

:::

## Common Usage Patterns

### REST API with API Gateway + Lambda

**Example:**

```yaml
# Lambda processes HTTP requests
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: rest-api-handler
spec:
  functionName: rest-api-handler
  runtime: nodejs20.x
  handler: api.handler
  code:
zipFile: |
      exports.handler = async (event) => {
          return {
              statusCode: 200,
              headers: {'Content-Type': 'application/json'},
              body: JSON.stringify({
                  path: event.path,
                  method: event.httpMethod,
                  body: JSON.parse(event.body || '{}')
              })
          };
      };
  role: arn:aws:iam::123456789012:role/lambda-api-role
```

### S3 Event Processing

**Example:**

```yaml
# Lambda processes S3 file uploads
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: s3-processor
spec:
  functionName: s3-processor
  runtime: python3.11
  handler: processor.handler
  code:
zipFile: |
      import json
      import boto3

      s3 = boto3.client('s3')

      def handler(event, context):
          # event contains S3 notification
          bucket = event['Records'][0]['s3']['bucket']['name']
          key = event['Records'][0]['s3']['object']['key']

          obj = s3.get_object(Bucket=bucket, Key=key)
          content = obj['Body'].read()

          # Process content
          return {'statusCode': 200}
  role: arn:aws:iam::123456789012:role/lambda-s3-role
```

### SQS Consumer Pattern

**Example:**

```yaml
# Lambda processes SQS queue messages
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: sqs-consumer
spec:
  functionName: sqs-consumer
  runtime: python3.11
  handler: consumer.handler
  timeout: 300
  code:
zipFile: |
      def handler(event, context):
          for record in event['Records']:
              message_id = record['messageId']
              body = record['body']

              try:
                  process_message(body)
              except Exception as e:
                  # Failure - move to DLQ
                  raise e
  deadLetterConfig:
targetArn: arn:aws:sqs:us-east-1:123456789012:dlq
  role: arn:aws:iam::123456789012:role/lambda-sqs-role
```

### Stream Processing (DynamoDB/Kinesis)

**Example:**

```yaml
# Lambda processes modified DynamoDB items
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
metadata:
  name: stream-processor
spec:
  functionName: stream-processor
  runtime: python3.11
  handler: stream.handler
  code:
zipFile: |
      def handler(event, context):
          for record in event['Records']:
              # record contains: eventName, eventSource, dynamodb
              event_type = record['eventName']  # INSERT, MODIFY, REMOVE
              new_image = record['dynamodb'].get('NewImage', {})

              if event_type == 'INSERT':
                  process_new_item(new_image)
              elif event_type == 'MODIFY':
                  update_analytics(new_image)
  role: arn:aws:iam::123456789012:role/lambda-dynamodb-role
```

## Related Resources

- [API Gateway](/services/compute/api-gateway)

  - [S3](/services/storage/s3)

  - [SQS](/services/messaging/sqs)

  - [DynamoDB](/services/database/dynamodb)

  - [IAM Role](/services/security/iam)
---