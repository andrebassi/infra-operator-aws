---
title: 'Installation'
description: 'Install the Infra Operator via Helm Chart'
---

## Prerequisites

Before installing Infra Operator, make sure you have:

- **Kubernetes 1.28+** - Working Kubernetes cluster
- **Helm 3.x** - Kubernetes package manager
- **kubectl** - Kubernetes CLI configured
- **AWS Credentials** - AWS credentials or LocalStack for testing

:::tip

For local development, we recommend using **LocalStack** to emulate AWS services at no cost.

:::

## Helm Installation

### 1. Clone the Repository

**Command:**

```bash
git clone https://github.com/andrebassi/infra-operator-aws.git
cd infra-operator
```

### 2. Install via Helm

**Command:**

```bash
# Install with default values
helm install infra-operator ./chart \
  --namespace iop-system \
  --create-namespace

# Or with custom values
helm install infra-operator ./chart \
  --namespace iop-system \
  --create-namespace \
  --set image.tag=latest \
  --set replicaCount=2 \
  --set resources.requests.memory=256Mi
```

### 3. Verify Installation

**Command:**

```bash
# Check pods
kubectl get pods -n iop-system

# Should show something like:
# NAME                              READY   STATUS    RESTARTS   AGE
# infra-operator-5d4c7b9f8d-xxxxx   1/1     Running   0          30s

# Check installed CRDs (26 CRDs)
kubectl get crds | grep aws-infra-operator.runner.codes

# Check logs
kubectl logs -n iop-system deploy/infra-operator
```

<Check>
  If all pods are **Running** and all 26 CRDs are created, the installation was successful!
</Check>

## AWSProvider Configuration

Before creating AWS resources, you need to configure credentials:

### Option 1: IRSA (Recommended for EKS)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-provider
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role
```

### Option 2: Static Credentials

**Command:**

```bash
# Create secret with AWS credentials
kubectl create secret generic aws-credentials \
  --from-literal=aws-access-key-id=YOUR_ACCESS_KEY \
  --from-literal=aws-secret-access-key=YOUR_SECRET_KEY \
  -n iop-system
```

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-provider
spec:
  region: us-east-1
  credentials:
    secretRef:
      name: aws-credentials
```

### Option 3: LocalStack (Development)

**Command:**

```bash
# Install LocalStack
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=ec2,s3,dynamodb,sqs,sns,iam,secretsmanager,kms \
  localstack/localstack:latest
```

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack-provider
spec:
  region: us-east-1
  endpoint: http://localstack:4566
  credentials:
    secretRef:
      name: localstack-credentials
```

:::note

For LocalStack, use fake credentials: `test` / `test`

:::

## Apply the Provider

**Command:**

```bash
kubectl apply -f awsprovider.yaml

# Check status
kubectl get awsprovider aws-provider

# Should show:
# NAME           REGION      READY   AGE
# aws-provider   us-east-1   True    10s
```

## Testing with Samples

The repository includes **26 samples** ready in the `./samples/` folder, organized in the recommended creation order:

### Samples Structure

#### Networking (01-09)

**01-vpc.yaml** - Virtual Private Cloud:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
spec:
  cidrBlock: "10.10.0.0/16"          # Private IP range
  enableDnsSupport: true              # Enable DNS resolution
  enableDnsHostnames: true            # Enable DNS hostnames
```
Creates the AWS network foundation. All other network resources depend on the VPC.

**02-subnet.yaml** - Subnet within VPC:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
spec:
  vpcID: "vpc-xxx"                    # Reference to created VPC
  cidrBlock: "10.10.1.0/24"          # Sub-range of VPC
  availabilityZone: us-east-1a        # Specific AZ
  mapPublicIpOnLaunch: true          # Automatically assign public IP
```
Creates a public subnet to host internet-accessible resources.

**03-elastic-ip.yaml** - Static Public IP:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
spec:
  domain: vpc                         # IP for VPC use
```
Fixed public IP, used for NAT Gateways or EC2 instances.

**04-internet-gateway.yaml** - Internet Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
spec:
  vpcID: "vpc-xxx"                    # Attached to VPC
```
Allows VPC resources to access the internet and be accessed externally.

**05-nat-gateway.yaml** - NAT Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
spec:
  subnetID: "subnet-xxx"              # Public subnet
  allocationID: ""                    # Elastic IP allocation
```
Allows instances in private subnets to access the internet (outbound only).

**06-route-table.yaml** - Route Table:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
spec:
  vpcID: "vpc-xxx"
  routes:
    - destinationCIDR: 0.0.0.0/0      # Default route
      gatewayID: ""                    # Internet Gateway ID
```
Defines network traffic routes (e.g., to internet via IGW).

**07-security-group.yaml** - Virtual Firewall:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
spec:
  vpcID: "vpc-xxx"
  ingressRules:                       # Inbound traffic
    - ipProtocol: tcp
      fromPort: 80
      toPort: 80
      cidrIPv4: 0.0.0.0/0            # Allow HTTP from anywhere
  egressRules:                        # Outbound traffic
    - ipProtocol: "-1"                # All protocols
      cidrIPv4: 0.0.0.0/0            # To any destination
```
Controls network traffic allowed for AWS resources.

**08-alb.yaml** - Application Load Balancer (Layer 7):
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
spec:
  loadBalancerName: helm-test-alb
  scheme: internet-facing             # Public or internal
  subnets: ["subnet-xxx"]             # Multiple subnets (HA)
  ipAddressType: ipv4
```
Load balancer for HTTP/HTTPS with content-based routing.

**09-nlb.yaml** - Network Load Balancer (Layer 4):
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NLB
spec:
  loadBalancerName: helm-test-nlb
  scheme: internet-facing
  ipAddressType: ipv4
```
Load balancer for TCP/UDP with high performance and low latency.

#### Security (10-13)

**10-iam-role.yaml** - IAM Role:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
spec:
  roleName: helm-test-role
  assumeRolePolicyDocument: |         # Trust policy
{
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}
```
IAM role for Lambda functions to assume and access AWS resources.

**11-kms-key.yaml** - KMS Encryption Key:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
spec:
  description: "Helm test KMS key"
  keyUsage: ENCRYPT_DECRYPT           # Encryption/Decryption
```
Managed encryption key to protect sensitive data.

**12-secrets-manager.yaml** - Secrets Manager:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
spec:
  secretName: helm-test-secret
  secretString: '{"username":"admin","password":"secret123"}'
```
Stores credentials, API keys, and other secrets with automatic rotation.

**13-certificate.yaml** - ACM SSL/TLS Certificate:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
spec:
  domainName: example.com
  subjectAlternativeNames:            # SANs
- "*.example.com"                 # Wildcard
  validationMethod: DNS               # DNS or EMAIL
```
SSL/TLS certificate for HTTPS on ALB, CloudFront, API Gateway.

#### Storage & Database (14-18)

**14-s3.yaml** - S3 Bucket:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
spec:
  bucketName: helm-test-bucket
  versioning:
    enabled: true                     # Object versioning
  encryption:
    algorithm: AES256                 # Encryption at rest
```
Object storage for files, backups, data lakes, static hosting.

**15-ecr-repository.yaml** - Container Registry:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
spec:
  repositoryName: helm-test-repo
  imageTagMutability: MUTABLE
  imageScanningConfiguration:
    scanOnPush: true                  # Vulnerability scanning
```
Private registry for Docker/OCI images.

**16-dynamodb.yaml** - DynamoDB NoSQL Table:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
spec:
  tableName: helm-test-users
  hashKey:
    name: userID                      # Partition key
    type: S                           # String type
  billingMode: PAY_PER_REQUEST        # On-demand pricing
```
Serverless NoSQL database with high performance and automatic scaling.

**17-rds-instance.yaml** - RDS Relational Database:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
spec:
  dbInstanceIdentifier: helm-test-db
  dbInstanceClass: db.t3.micro        # Instance type
  engine: postgres                    # postgres, mysql, mariadb, etc
  engineVersion: "14.5"
  masterUsername: admin
  masterUserPassword: secret123
  allocatedStorage: 20                # GB
  storageType: gp2                    # SSD
```
Managed relational database (PostgreSQL, MySQL, etc) with automatic backups.

**18-elasticache.yaml** - ElastiCache Redis/Memcached:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
spec:
  cacheClusterID: helm-test-cache
  engine: redis                       # redis or memcached
  cacheNodeType: cache.t3.micro
  numCacheNodes: 1
```
Managed in-memory cache to improve application performance.

#### Messaging (19-20)

**19-sqs.yaml** - SQS Queue:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
spec:
  queueName: helm-test-queue
  messageRetentionPeriod: 345600      # 4 days (seconds)
  visibilityTimeout: 30               # 30 seconds
```
Message queue for asynchronous communication between services.

**20-sns.yaml** - SNS Topic:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
spec:
  topicName: helm-test-notifications
  displayName: "Helm Test Notifications"
```
Pub/Sub for notifications (email, SMS, Lambda, SQS, HTTP).

#### Compute (21-24)

**21-ec2-instance.yaml** - EC2 Virtual Machine:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EC2Instance
spec:
  imageID: ami-12345678               # Region AMI
  instanceType: t2.micro              # Instance type
  subnetID: "subnet-xxx"
  securityGroupIDs: []                # Security Groups
```
Virtual machine for general workloads, applications, servers.

**22-lambda.yaml** - Lambda Function:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
spec:
  functionName: helm-test-function
  runtime: python3.9                  # Runtime environment
  handler: lambda_function.handler    # Entry point
  role: arn:aws:iam::xxx:role/lambda-role
  code:
    zipFile: <base64>                 # Code in base64 ZIP
  timeout: 30                         # Timeout in seconds
  memorySize: 128                     # MB
```
Serverless function to run code without managing servers.

**23-ecs-cluster.yaml** - ECS Cluster:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
spec:
  clusterName: helm-test-ecs
```
Cluster to orchestrate Docker containers with Fargate or EC2.

**24-eks-cluster.yaml** - EKS Kubernetes Cluster:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
spec:
  clusterName: helm-test-eks
  version: "1.28"                     # Kubernetes version
  roleARN: arn:aws:iam::xxx:role/eks-cluster-role
  resourcesVpcConfig:
    subnetIDs: ["subnet-xxx"]
    endpointPublicAccess: true
    endpointPrivateAccess: false
```
Kubernetes cluster managed by AWS.

#### API & CDN (25-26)

**25-api-gateway.yaml** - API Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: APIGateway
spec:
  name: helm-test-api
  description: "Helm test API Gateway"
  protocolType: HTTP                  # HTTP, REST, or WebSocket
```
Gateway to create, publish, and manage REST/HTTP/WebSocket APIs.

**26-cloudfront.yaml** - CloudFront CDN:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: CloudFront
spec:
  comment: "Helm test CloudFront"
  enabled: true
  origins:
    - id: test-origin
      domainName: example.com
      customOriginConfig:
        httpPort: 80
        httpsPort: 443
        originProtocolPolicy: http-only
  defaultCacheBehavior:
    targetOriginId: test-origin
    viewerProtocolPolicy: allow-all
    allowedMethods: [GET, HEAD]
    cachedMethods: [GET, HEAD]
    forwardedValues:
      queryString: false              # Don't forward query strings
      cookies:
        forward: none                 # Don't forward cookies
```
Global CDN to distribute content with low latency and edge caching.

### Test a Specific Service

#### 1. VPC (Networking Foundation)

**Command:**

```bash
# Apply VPC
kubectl apply -f samples/01-vpc.yaml

# Check status
kubectl get vpc production-vpc

# See full details
kubectl describe vpc production-vpc
```

#### 2. S3 Bucket (Storage)

**Command:**

```bash
# Apply S3
kubectl apply -f samples/14-s3.yaml

# Check status
kubectl get s3bucket app-data

# Status should show:
# NAME       BUCKET NAME               READY   AGE
# app-data   mycompany-app-data-prod   True    30s
```

#### 3. DynamoDB (Database)

**Command:**

```bash
# Apply DynamoDB
kubectl apply -f samples/16-dynamodb.yaml

# Check status
kubectl get dynamodbtable users-table
```

### Test Complete Stack

To test multiple services at once:

```bash
# Apply Networking (VPC, Subnet, IGW, NAT)
kubectl apply -f samples/01-vpc.yaml
kubectl apply -f samples/02-subnet.yaml
kubectl apply -f samples/04-internet-gateway.yaml
kubectl apply -f samples/05-nat-gateway.yaml

# Wait for resources to become Ready
kubectl wait --for=condition=Ready vpc/production-vpc --timeout=60s
kubectl wait --for=condition=Ready subnet/public-subnet-1a --timeout=60s

# Apply Load Balancers
kubectl apply -f samples/08-alb.yaml
kubectl apply -f samples/09-nlb.yaml

# Check all resources
kubectl get vpc,subnet,internetgateway,natgateway,alb,nlb
```

## Recommended Creation Order

To create a complete AWS infrastructure, follow this order:

### 1. Networking Foundation

**Command:**

```bash
kubectl apply -f samples/01-vpc.yaml
kubectl apply -f samples/02-subnet.yaml
kubectl apply -f samples/03-elastic-ip.yaml
kubectl apply -f samples/04-internet-gateway.yaml
kubectl apply -f samples/05-nat-gateway.yaml
kubectl apply -f samples/06-route-table.yaml
kubectl apply -f samples/07-security-group.yaml
```

### 2. Load Balancing

**Command:**

```bash
kubectl apply -f samples/08-alb.yaml
kubectl apply -f samples/09-nlb.yaml
```

### 3. Security

**Command:**

```bash
kubectl apply -f samples/10-iam-role.yaml
kubectl apply -f samples/11-kms-key.yaml
kubectl apply -f samples/12-secrets-manager.yaml
kubectl apply -f samples/13-certificate.yaml
```

### 4. Storage & Database

**Command:**

```bash
kubectl apply -f samples/14-s3.yaml
kubectl apply -f samples/15-ecr-repository.yaml
kubectl apply -f samples/16-dynamodb.yaml
kubectl apply -f samples/17-rds-instance.yaml
kubectl apply -f samples/18-elasticache.yaml
```

### 5. Messaging

**Command:**

```bash
kubectl apply -f samples/19-sqs.yaml
kubectl apply -f samples/20-sns.yaml
```

### 6. Compute

**Command:**

```bash
kubectl apply -f samples/21-ec2-instance.yaml
kubectl apply -f samples/22-lambda.yaml
kubectl apply -f samples/23-ecs-cluster.yaml
kubectl apply -f samples/24-eks-cluster.yaml
```

### 7. API & CDN

**Command:**

```bash
kubectl apply -f samples/25-api-gateway.yaml
kubectl apply -f samples/26-cloudfront.yaml
```

## Check All Resources

**Command:**

```bash
# See all created resources
kubectl get awsprovider,vpc,subnet,elasticip,internetgateway,natgateway,routetable,securitygroup,alb,nlb

# See security resources
kubectl get iamrole,kmskey,secretsmanagersecret,certificate

# See storage and database
kubectl get s3bucket,ecrrepository,dynamodbtable,rdsinstance,elasticachecluster

# See messaging
kubectl get sqsqueue,snstopic

# See compute
kubectl get ec2instance,lambdafunction,ecscluster,ekscluster

# See API and CDN
kubectl get apigateway,cloudfront
```

## Useful Commands

### Check Resource Status

**Command:**

```bash
# See Ready status
kubectl get vpc production-vpc -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# See AWS Resource ID
kubectl get vpc production-vpc -o jsonpath='{.status.vpcId}'

# See all status conditions
kubectl get vpc production-vpc -o jsonpath='{.status.conditions}' | jq
```

### Delete Resources

**Command:**

```bash
# Delete a specific resource
kubectl delete vpc production-vpc

# Delete all resources of a type
kubectl delete vpc --all

# Delete all samples
kubectl delete -f samples/
```

:::warning

By default, deleting a CR also **deletes the AWS resource**. Use `deletionPolicy: Retain` to keep the AWS resource.

:::

## Uninstall

**Command:**

```bash
# Delete all created resources
kubectl delete -f samples/

# Uninstall Helm chart
helm uninstall infra-operator -n iop-system

# Delete CRDs (optional)
kubectl delete crd $(kubectl get crd | grep aws-infra-operator.runner.codes | awk '{print $1}')

# Delete namespace
kubectl delete namespace iop-system
```

## Next Steps

- [Networking](/services/networking/vpc)
- [Storage](/services/storage/s3)
- [Compute](/services/compute/ec2)
- [All Services](/services/networking/vpc)

## Troubleshooting

### Pods don't start

**Command:**

```bash
# See operator logs
kubectl logs -n iop-system deploy/infra-operator --tail=100

# See events
kubectl get events -n iop-system --sort-by='.lastTimestamp'
```

### AWSProvider not Ready

**Command:**

```bash
# Check credentials
kubectl describe awsprovider aws-provider

# Test credentials manually
aws sts get-caller-identity --region us-east-1
```

### Resources stuck in "NotReady"

**Command:**

```bash
# See reason
kubectl describe vpc production-vpc

# See events
kubectl get events --field-selector involvedObject.name=production-vpc
```

:::tip

For more troubleshooting details, see the specific documentation for each service.

:::
