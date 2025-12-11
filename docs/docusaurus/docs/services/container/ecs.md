---
title: 'ECS - Elastic Container Service'
description: 'Orchestrate Docker containers without managing servers'
sidebar_position: 2
---

Orchestrate Docker containers at scale with ECS Fargate (serverless) or EC2.

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

**IAM Policy - ECS (ecs-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:CreateCluster",
        "ecs:DeleteCluster",
        "ecs:DescribeClusters",
        "ecs:UpdateCluster",
        "ecs:UpdateClusterSettings",
        "ecs:PutClusterCapacityProviders",
        "ecs:TagResource",
        "ecs:UntagResource",
        "ecs:ListTagsForResource"
      ],
      "Resource": "*"
    }
  ]
}
```

**Create Role:**

```bash
# Create IAM Role
aws iam create-role \
  --role-name infra-operator-ecs-role \
  --assume-role-policy-document file://trust-policy.json

# Attach policy
aws iam put-role-policy \
  --role-name infra-operator-ecs-role \
  --policy-name ecs-policy \
  --policy-document file://ecs-policy.json

# Annotate Service Account in K8s
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-ecs-role
```
## Creating ECS Cluster

**Cluster Basic (Fargate):**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: fargate-cluster
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterName: my-fargate-cluster

  # Container Insights enabled by default
  settings:
    - name: containerInsights
      value: enabled

  tags:
    Name: fargate-cluster
    Environment: production
    Team: platform
```

**Cluster with Fargate + EC2:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: hybrid-cluster
  namespace: default
spec:
  providerRef:
    name: production-aws

  clusterName: hybrid-cluster

  # Capacity providers (Fargate + EC2)
  capacityProviders:
    - FARGATE
    - FARGATE_SPOT
    - my-ec2-capacity-provider

  # Default strategy (80% Fargate, 20% Fargate Spot)
  defaultCapacityProviderStrategy:
    - capacityProvider: FARGATE
      weight: 80
      base: 10  # Ensures 10 tasks on regular Fargate
    - capacityProvider: FARGATE_SPOT
      weight: 20

  settings:
    - name: containerInsights
      value: enabled

  tags:
    Name: hybrid-cluster
    Environment: production
```

**Cluster with ECS Exec:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: debug-cluster
  namespace: development
spec:
  providerRef:
    name: dev-aws

  clusterName: debug-cluster

  # Configuration for ECS Exec (debugging)
  configuration:
    executeCommandConfiguration:
      logging: OVERRIDE
      kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012
      logConfiguration:
        cloudWatchLogGroupName: /ecs/execute-command
        cloudWatchEncryptionEnabled: true

  settings:
    - name: containerInsights
      value: enabled

  tags:
    Name: debug-cluster
    Environment: development
```

**Check Status:**

```bash
# List clusters
kubectl get ecscluster

# View details
kubectl describe ecscluster fargate-cluster

# View cluster ARN
kubectl get ecscluster fargate-cluster -o jsonpath='{.status.clusterARN}'

# View statistics
kubectl get ecscluster fargate-cluster -o jsonpath='{.status}'
```
## Specification Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `clusterName` | string | ✅ | ECS cluster name (1-255 characters) |
| `capacityProviders` | []string | ❌ | List of capacity providers (`FARGATE`, `FARGATE_SPOT`, or custom) |
| `defaultCapacityProviderStrategy` | []object | ❌ | Default distribution strategy among providers |
| `settings` | []object | ❌ | Cluster settings (containerInsights) |
| `configuration` | object | ❌ | Advanced configuration (execute command) |
| `serviceConnectDefaults` | object | ❌ | Defaults for Service Connect |
| `tags` | map[string]string | ❌ | Custom tags for cluster |
| `deletionPolicy` | string | ❌ | Deletion policy: `Delete` (default) or `Retain` |

### Capacity Provider Strategy

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `capacityProvider` | string | ✅ | Capacity provider name |
| `weight` | int32 | ❌ | Relative weight (0-1000, default: 0) |
| `base` | int32 | ❌ | Minimum number of tasks on this provider |

### Execute Command Configuration

| Field | Type | Description |
|-------|------|-------------|
| `logging` | string | Logging type: `NONE`, `DEFAULT`, `OVERRIDE` |
| `kmsKeyId` | string | KMS key for log encryption |
| `logConfiguration.cloudWatchLogGroupName` | string | CloudWatch log group |
| `logConfiguration.cloudWatchEncryptionEnabled` | bool | Enable CloudWatch encryption |
| `logConfiguration.s3BucketName` | string | S3 bucket for logs |
| `logConfiguration.s3EncryptionEnabled` | bool | Enable S3 encryption |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `ready` | bool | If the cluster is active (`status` = "ACTIVE") |
| `clusterARN` | string | ARN of created cluster |
| `status` | string | State: `PROVISIONING`, `ACTIVE`, `DEPROVISIONING`, `FAILED`, `INACTIVE` |
| `registeredContainerInstancesCount` | int32 | Number of registered EC2 instances |
| `runningTasksCount` | int32 | Number of running tasks |
| `pendingTasksCount` | int32 | Number of pending tasks |
| `activeServicesCount` | int32 | Number of active services |
| `lastSyncTime` | time | Last synchronization with AWS |

## Use Cases

### Cluster Serverless (Fargate Only)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: serverless-cluster
  namespace: production
spec:
  providerRef:
    name: production-aws

  clusterName: serverless-prod

  capacityProviders:
    - FARGATE
    - FARGATE_SPOT

  defaultCapacityProviderStrategy:
    - capacityProvider: FARGATE
      weight: 70
      base: 5
    - capacityProvider: FARGATE_SPOT
      weight: 30

  settings:
    - name: containerInsights
      value: enabled

  tags:
    Type: serverless
    Environment: production
```

### Cluster for Development with Debugging

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: dev-cluster
  namespace: development
spec:
  providerRef:
    name: dev-aws

  clusterName: development

  configuration:
    executeCommandConfiguration:
      logging: OVERRIDE
      logConfiguration:
        cloudWatchLogGroupName: /ecs/dev/exec
        cloudWatchEncryptionEnabled: false

  settings:
    - name: containerInsights
      value: enabled

  tags:
    Environment: development
    Debug: enabled
```

## Troubleshooting

### Cluster does not become ACTIVE

### Check cluster state

**Command:**

```bash
kubectl describe ecscluster my-cluster
```

Look for:
- `Status: PROVISIONING` - Cluster still being created (normally 1-2 minutes)
- `Status: FAILED` - Creation failed (check operator logs)

### Check IAM permissions

**Command:**

```bash
# View operator logs
kubectl logs -n infra-operator-system -l control-plane=controller-manager

# Look for errors like "AccessDenied"
```

**Common error**: IAM role without ECS permissions
**Solution**: Add policy `ecs:*` to IRSA role

### Check capacity providers

**Command:**

```bash
# Check if capacity providers exist
aws ecs describe-capacity-providers \
  --capacity-providers my-ec2-capacity-provider
```

**Common error**: Capacity provider does not exist
**Solution**: Create capacity provider first or use only `FARGATE`/`FARGATE_SPOT`

### Tasks do not start

### Check capacity providers

**Command:**

```bash
# View cluster details
kubectl get ecscluster my-cluster -o yaml
```

If there are no `capacityProviders` configured:
```bash
kubectl patch ecscluster my-cluster --type=merge -p '{"spec":{"capacityProviders":["FARGATE"]}}'
```

### Check subnets and security groups

Fargate tasks need:
- Subnets with internet connectivity (or NAT Gateway)
- Security groups allowing necessary traffic

**Command:**

```bash
# Check task subnets
aws ecs describe-tasks --cluster my-cluster --tasks <task-id> \
  --query 'tasks[0].attachments[0].details'
```

### Check account limits

**Command:**

```bash
# Check Fargate vCPUs quota
aws service-quotas get-service-quota \
  --service-code fargate \
  --quota-code L-3032A538

# View running tasks
kubectl get ecscluster my-cluster -o jsonpath='{.status.runningTasksCount}'
```

If limit reached:
- Request quota increase via AWS Console
- Scale tasks horizontally

### Error deleting cluster

### Cluster with active services or tasks

**Command:**

```bash
# View tasks and services
kubectl get ecscluster my-cluster -o jsonpath='{.status}'
```

If there are tasks/services running:
```bash
# Stop all tasks
aws ecs list-tasks --cluster my-cluster | \
  xargs -I {} aws ecs stop-task --cluster my-cluster --task {}

# Delete all services
aws ecs list-services --cluster my-cluster | \
  xargs -I {} aws ecs delete-service --cluster my-cluster --service {} --force
```

Wait 1-2 minutes and try deleting again.

### Registered container instances

**Command:**

```bash
# View instances
kubectl get ecscluster my-cluster -o jsonpath='{.status.registeredContainerInstancesCount}'
```

If there are EC2 instances:
```bash
# Deregister instances
aws ecs list-container-instances --cluster my-cluster | \
  xargs -I {} aws ecs deregister-container-instance --cluster my-cluster --container-instance {} --force
```

## Deletion Policies

### Delete (Default)

When the CR is deleted, the ECS cluster is automatically deleted:

```yaml
spec:
  deletionPolicy: Delete  # Default
```

:::warning

The cluster can only be deleted if it has no tasks, services, or container instances.

:::

### Retain

The ECS cluster is kept even after deleting the CR:

```yaml
spec:
  deletionPolicy: Retain
```

**Use case**: Clusters with complex configurations or important data.

## Advanced Examples

### Cluster with Service Connect

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: service-mesh-cluster
spec:
  providerRef:
    name: production-aws

  clusterName: service-mesh

  serviceConnectDefaults:
    namespace: my-app-namespace

  settings:
    - name: containerInsights
      value: enabled

  tags:
    ServiceMesh: enabled
```

### Cluster with Complete Logging

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
metadata:
  name: audit-cluster
spec:
  providerRef:
    name: production-aws

  clusterName: audit-prod

  configuration:
    executeCommandConfiguration:
      logging: OVERRIDE
      kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/audit-key
      logConfiguration:
        cloudWatchLogGroupName: /ecs/audit/exec
        cloudWatchEncryptionEnabled: true
        s3BucketName: audit-logs-bucket
        s3EncryptionEnabled: true
        s3KeyPrefix: ecs-exec/

  settings:
    - name: containerInsights
      value: enabled

  tags:
    Compliance: sox
    Audit: enabled
```

## Next Steps

After creating the ECS cluster:

1. **Create Task Definitions** defining containers and resources
2. **Configure Services** to run tasks permanently
3. **Setup Auto Scaling** to scale automatically
4. **Configure Load Balancers** (ALB/NLB) to distribute traffic
5. **Implement Service Discovery** with AWS Cloud Map
6. **Configure CI/CD** for automated deployment

:::info

The operator manages only the cluster. Task definitions, services, and tasks must be managed separately.

:::

## Monitoring

### CloudWatch Container Insights

With `containerInsights: enabled`, you have access to:

- **Cluster metrics**: CPU, memory, network of cluster
- **Service metrics**: CPU, memory per service
- **Task metrics**: CPU, memory per task
- **Container metrics**: Metrics per container

**Command:**

```bash
# View metrics in CloudWatch
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ClusterName,Value=my-cluster \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-01T23:59:59Z \
  --period 3600 \
  --statistics Average
```

### ECS Exec (Debugging)

With execute command enabled, you can execute commands in running containers:

```bash
# Enable execute command in task definition
# Then, execute commands:
aws ecs execute-command \
  --cluster my-cluster \
  --task <task-id> \
  --container my-container \
  --interactive \
  --command "/bin/sh"
```

## Comparison: ECS vs EKS

| Aspect | ECS | EKS (Kubernetes) |
|---------|-----|------------------|
| **Complexity** | Simple | Complex |
| **Lock-in** | AWS only | Multi-cloud |
| **Cost** | Free (pay for Fargate/EC2) | $0.10/hour per cluster |
| **AWS Integration** | Native and deep | Via AWS Load Balancer Controller |
| **Community** | Smaller | Giant |
| **Learning curve** | Fast | Slow |

**Use ECS if**:
- You are 100% AWS
- Want simplicity
- Small/medium team

**Use EKS if**:
- Need multi-cloud
- Already using Kubernetes
- Want portability

## References

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [Fargate Pricing](https://aws.amazon.com/fargate/pricing/)
- [ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [Container Insights](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html)