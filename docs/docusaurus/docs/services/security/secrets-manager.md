---
title: 'Secrets Manager - Secrets Management'
description: 'Secure management of credentials and secrets in the cloud'
sidebar_position: 4
---

Store, manage, and rotate secrets securely using AWS Secrets Manager with KMS encryption, automatic rotation, and complete auditing.

## Prerequisites: AWSProvider Configuration

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

**IAM Policy - Secrets Manager (secrets-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:CreateSecret",
        "secretsmanager:DeleteSecret",
        "secretsmanager:DescribeSecret",
        "secretsmanager:UpdateSecret",
        "secretsmanager:PutSecretValue",
        "secretsmanager:GetSecretValue",
        "secretsmanager:TagResource",
        "secretsmanager:UntagResource",
        "secretsmanager:RotateSecret",
        "secretsmanager:RestoreSecret"
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
  --role-name infra-operator-secrets-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator Secrets Manager management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-secrets-role \
  --policy-name SecretsManagement \
  --policy-document file://secrets-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-secrets-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-secrets-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

AWS Secrets Manager provides centralized management of secrets (credentials, API keys, tokens, certificates) with:

- **Secure Storage**: Encryption at rest with KMS
- **Automatic Rotation**: Rotate secrets without manual intervention
- **Access Control**: IAM policies and resource-based policies
- **Complete Auditing**: Logging of all operations
- **Versioning**: Maintains history of secret versions
- **Multi-Region Replication**: Automatic disaster recovery
- **Service Integration**: RDS, Aurora, Redshift, DocumentDB
- **Recovery Window**: Recovery window for safe deletion
- **Transparent Encryption**: KMS encryption with AWS Managed or Customer Master Key
- **Secret Attachment**: Automatic binding with AWS resources

**Status**: ✅ Ready for Production
**LocalStack**: ✅ Community

## Quick Start

**Database Password:**

```yaml
# Kubernetes Secret containing the secret value
apiVersion: v1
kind: Secret
metadata:
  name: db-password-source
  namespace: default
type: Opaque
stringData:
  password: "MyS3cr3tP@ssw0rd123!"

---
# Secrets Manager Secret referencing the Kubernetes Secret
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: e2e-db-password
  namespace: default
spec:
  providerRef:
    name: localstack
  secretName: e2e/db/password
  description: "Database password for E2E testing"
  secretStringRef:
    name: db-password-source
    key: password
  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing
  deletionPolicy: Delete
  recoveryWindowInDays: 7
```

**API Key:**

```yaml
# Kubernetes Secret for API key
apiVersion: v1
kind: Secret
metadata:
  name: api-key-source
  namespace: default
type: Opaque
stringData:
  apikey: "sk-1234567890abcdefghijklmnopqrstuvwxyz"

---
# Secrets Manager Secret with immediate deletion
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: e2e-api-key
  namespace: default
spec:
  providerRef:
    name: localstack
  secretName: e2e/api/key
  description: "API key for external service - E2E testing"
  secretStringRef:
    name: api-key-source
    key: apikey
  tags:
    environment: test
    managed-by: infra-operator
    service: api
    purpose: e2e-testing
  deletionPolicy: Delete
  recoveryWindowInDays: 0
```

**Production Secret:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: db-credentials
  namespace: default
spec:
  providerRef:
    name: production-aws
  secretName: prod/database/credentials
  description: Database credentials for production
  secretString: |
    {
      "username": "admin",
      "password": "MySecurePassword123!",
      "host": "db.example.com",
      "port": 5432
    }
  kmsKeyId: alias/aws/secretsmanager
  tags:
    Environment: production
```

**Apply:**

```bash
kubectl apply -f secrets-manager.yaml
```

**Verify:**

```bash
kubectl get secretsmanagersecrets
kubectl describe secretsmanagersecret db-credentials
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource

  AWSProvider resource name

Secret name in AWS Secrets Manager. Use namespacing with `/` for organization.

  **Examples:**
  - `prod/database/credentials`
  - `dev/api/keys`
  - `staging/third-party/webhooks`

  **Rules:**
  - Maximum 512 characters
  - Supports characters: A-Z a-z 0-9 /_+=.@-
  - Case-sensitive

Secret string (plain text or JSON). Maximum 65536 characters.

  **JSON Example:**
  ```json
  {
    "username": "admin",
    "password": "secure-password",
    "host": "db.example.com",
    "port": 5432
  }
  ```

  **Note:** Use `secretString` for text or `secretBinary` for binary data (base64).

### Optional Fields

Clear description of the secret's purpose

  **Examples:**
  - "Database credentials for production PostgreSQL"
  - "API keys for third-party payment service"
  - "SSL certificate for domain.com"

Binary data in base64 (alternative to secretString)

  **Example:**

  ```yaml
  secretBinary: aGVsbG8gd29ybGQgYmluYXJ5IGRhdGE=
  ```

KMS key ID for encryption. If not provided, uses AWS Managed Key.

  **Accepted formats:**
  - ARN: `arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012`
  - Key ID: `12345678-1234-1234-1234-123456789012`
  - Alias: `alias/my-key`
  - Special alias: `alias/aws/secretsmanager` (default)

  **Recommendation:** Use customer-managed KMS key for compliance

If `true`, enables automatic secret rotation

  **Details:**
  - Requires `rotationLambdaArn` and `rotationSchedule`
  - Lambda function rotates the secret automatically
  - Creates new secret version with each rotation

ARN of the Lambda function that rotates the secret

  **Example:**
  ```
  arn:aws:lambda:us-east-1:123456789012:function:rotate-secret
  ```

  **Required if:** `rotationEnabled: true`

Rotation schedule configuration

  **Example:**

  ```yaml
  rotationSchedule:
    automaticallyAfterDays: 30
    duration: 3  # hours
  ```

  **Fields:**
  - `automaticallyAfterDays`: Rotate every N days (30-180)
  - `duration`: Hours allowed to complete rotation

Replicate secret in multiple regions for disaster recovery

  **Example:**

  ```yaml
  replicaRegions:
    - region: us-west-2
      kmsKeyId: arn:aws:kms:us-west-2:123456789012:key/...
    - region: eu-west-1
      kmsKeyId: arn:aws:kms:eu-west-1:123456789012:key/...
  ```

Key-value pairs for organization and access control

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: myapp
    Team: platform
    CostCenter: engineering
    Compliance: pci-dss
  ```

Days for recovery window before deleting secret (7-30 days)

  **Details:**
  - During this period, secret can be recovered
  - After the period, deletion is permanent
  - Default: 30 days (maximum protection)

What happens to the secret when the CR is deleted

  **Options:**
  - `ScheduleDeletion`: Schedule deletion with recovery window (default, safe)
  - `Delete`: Delete immediately
  - `Retain`: Keep secret in AWS but without management

## Status Fields

After the secret is created, the following status fields are populated:

Complete secret ARN (e.g., `arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/database/credentials-abc123`)

Current secret version ID

Whether automatic rotation is enabled

Last rotation date (ISO 8601)

Scheduled date for next rotation

`true` when the secret is created and ready for use

Last synchronization timestamp with AWS

## Examples

### Database Credentials (JSON)

Secret with database credentials in JSON format:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: prod-postgres-credentials
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: prod/database/postgresql/credentials

  description: PostgreSQL database credentials for production environment

  # Credentials in JSON format
  secretString: |
    {
      "username": "postgres_admin",
      "password": "Tr0picálC0l0rsSecure!@#2025",
      "host": "prod-db.example.com",
      "port": 5432,
      "dbname": "production_db",
      "engine": "postgresql"
    }

  # Use customer-managed KMS key
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012

  tags:
    Environment: production
    Application: main-app
    Team: database
    Compliance: sox-compliant

  recoveryWindowInDays: 30
```

### API Keys and Tokens

Secret with multiple API keys:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: api-keys
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: prod/api/third-party-keys

  description: Third-party API keys for payment and notification services

  secretString: |
    {
      "stripe_api_key": "sk_live_abc123def456ghi789jkl",
      "stripe_webhook_secret": "whsec_abc123def456",
      "sendgrid_api_key": "SG.abc123def456",
      "github_token": "ghp_abc123def456",
      "slack_webhook": "https://hooks.slack.com/services/..."
    }

  kmsKeyId: alias/aws/secretsmanager

  tags:
    Environment: production
    Type: api-keys
    Owner: platform-team
```

### SSL/TLS Certificates

Secret to store certificate and private key:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: ssl-certificate
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: prod/certificates/domain-com

  description: SSL/TLS certificate and private key for domain.com

  secretString: |
    {
      "certificate": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
      "certificate_chain": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
      "expiration_date": "2026-12-31"
    }

  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/cert-key-id

  tags:
    Environment: production
    Domain: domain.com
    Type: ssl-certificate
    ExpiryDate: "2026-12-31"
```

### Secret with Automatic Rotation

Secret configured with Lambda for automatic rotation:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: rotating-password
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: prod/database/rotating/password

  description: Database password rotated automatically monthly

  secretString: |
    {
      "username": "app_user",
      "password": "CurrentPassword2025!@#"
    }

  # KMS encryption
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/rotation-key

  # Enable automatic rotation
  rotationEnabled: true

  # Lambda that rotates the secret
  rotationLambdaArn: arn:aws:lambda:us-east-1:123456789012:function:rds-password-rotator

  # Rotation schedule
  rotationSchedule:
    automaticallyAfterDays: 30  # Rotate every 30 days
    duration: 3  # Complete in up to 3 hours
    scheduleExpression: "rate(30 days)"

  tags:
    Environment: production
    Rotation: monthly
    Type: database-password
```

### Multi-Region Replicated Secret

Secret replicated for disaster recovery:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: global-secret
  namespace: default
spec:
  providerRef:
    name: production-aws

  secretName: prod/global/app-master-key

  description: Application master encryption key replicated globally

  secretString: |
    {
      "master_key": "85ecf8c5-ac6c-4ab2-b1f0-7cd3c7e8a9b0",
      "key_version": "1"
    }

  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/primary-key

  # Replicate in multiple regions
  replicaRegions:
    - region: us-west-2
      kmsKeyId: arn:aws:kms:us-west-2:123456789012:key/west-key
    - region: eu-west-1
      kmsKeyId: arn:aws:kms:eu-west-1:123456789012:key/eu-key

  tags:
    Environment: production
    HA: enabled
    ReplicationEnabled: "true"
```

## Verification

### Check Status via kubectl

**Command:**

```bash
# List all secrets
kubectl get secretsmanagersecrets

# Get detailed information
kubectl get secretsmanagersecret db-credentials -o yaml

# Describe secret with events
kubectl describe secretsmanagersecret db-credentials

# Watch creation in real-time
kubectl get secretsmanagersecret db-credentials -w

# View only status
kubectl get secretsmanagersecret db-credentials -o jsonpath='{.status}'
```

### Check on AWS

**AWS CLI:**

```bash
# List secrets
aws secretsmanager list-secrets --region us-east-1

# Get secret details
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials \
      --region us-east-1 \
      --output json | jq '.'

# Retrieve secret value
aws secretsmanager get-secret-value \
      --secret-id prod/database/credentials \
      --region us-east-1

# View secret versions
aws secretsmanager list-secret-version-ids \
      --secret-id prod/database/credentials \
      --region us-east-1

# View rotation configuration
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials \
      --query 'RotationRules'

# View secret replication
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials \
      --query 'AddedToRegionDate,ReplicationStatus'

# Test access to secret
aws secretsmanager get-secret-value \
      --secret-id prod/database/credentials \
      --version-id <version-id>
```

**LocalStack:**

```bash
# Point to LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

# List secrets
aws secretsmanager list-secrets

# Describe secret
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials

# Retrieve value
aws secretsmanager get-secret-value \
      --secret-id prod/database/credentials
```

### Expected Output

**Example:**

```yaml
status:
  secretArn: arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/database/credentials-abc123
  versionId: a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6
  rotationEnabled: true
  lastRotatedDate: "2025-11-22T20:18:08Z"
  nextRotationDate: "2025-12-22T20:18:08Z"
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### Secret Does Not Rotate Automatically

**Symptoms:** `lastRotatedDate` doesn't change, rotation doesn't execute

**Common Causes:**
1. `rotationEnabled: false` (disabled by default)
2. Invalid Lambda ARN or insufficient permissions
3. Lambda function has execution error

**Solutions:**
```bash
# Check rotation configuration
kubectl get secretsmanagersecret rotating-password -o yaml | grep -A 5 rotation

# View rotation status in AWS
aws secretsmanager describe-secret \
      --secret-id prod/database/rotating/password \
      --query 'RotationRules'

# Check Lambda logs
aws logs tail /aws/lambda/rds-password-rotator --follow

# Check rotation events
aws secretsmanager get-secret-value \
      --secret-id prod/database/rotating/password \
      --query 'VersionIdsToStages'

# Force manual rotation
aws secretsmanager rotate-secret \
      --secret-id prod/database/rotating/password \
      --rotation-lambda-arn arn:aws:lambda:us-east-1:123456789012:function:rotator

# Check Lambda permissions
aws secretsmanager describe-secret \
      --secret-id prod/database/rotating/password \
      --query 'RotationEnabled,RotationLambdaARN,RotationRules'
```

### Access Denied When Reading Secret

**Error:** `AccessDeniedException` or `UserNotAuthorizedForSecretException`

**Causes:**
1. IAM policy doesn't allow `secretsmanager:GetSecretValue`
2. Resource-based policy on secret blocks access
3. KMS key policy doesn't allow decrypt

**Solutions:**
```bash
# Check IAM permissions
aws iam get-user-policy --user-name <user> --policy-name <policy>

# Check secret policy (if configured)
aws secretsmanager get-resource-policy \
      --secret-id prod/database/credentials

# Check KMS permissions
aws kms describe-key --key-id <kms-key-id> \
      --query 'KeyMetadata.KeyState'

# Add IAM permission
aws iam put-user-policy --user-name <user> \
      --policy-name secretsmanager-access \
      --policy-document file://policy.json
```

**Example policy.json:**
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret"
    ],
    "Resource": "arn:aws:secretsmanager:*:123456789012:secret:prod/*"
  }, {
    "Effect": "Allow",
    "Action": [
      "kms:Decrypt",
      "kms:DescribeKey"
    ],
    "Resource": "arn:aws:kms:*:123456789012:key/*"
  }]
}
```
  
### Lambda Fails to Rotate

**Symptoms:** Scheduled rotation fails, Lambda produces error

**Causes:**
1. Lambda doesn't have permission to update secret
2. Lambda can't connect to database
3. Function can't authenticate with AWS

**Solutions:**
```bash
# Check Lambda logs
aws logs tail /aws/lambda/rds-password-rotator --follow

# View error details
aws secretsmanager describe-secret \
      --secret-id prod/database/rotating/password \
      --query 'LastFailedDate,LastFailedSyncErrorCode'

# Check Lambda IAM role
aws iam get-role --role-name lambda-rotation-role

# Add necessary permissions
aws iam attach-role-policy \
      --role-name lambda-rotation-role \
      --policy-arn arn:aws:iam::aws:policy/SecretsManagerReadWrite

# Test rotation manually
aws secretsmanager rotate-secret \
      --secret-id prod/database/rotating/password \
      --rotation-lambda-arn arn:aws:lambda:us-east-1:123456789012:function:rotator
```

### Secret Replication Not Working

**Symptoms:** Secret doesn't appear in replicated regions, replication status error

**Causes:**
1. KMS keys in replica regions don't exist or lack permission
2. Unsupported regions
3. Replication limit reached

**Solutions:**
```bash
# Check replication status
aws secretsmanager describe-secret \
      --secret-id prod/global/app-master-key \
      --region us-east-1 \
      --query 'ReplicationStatus'

# Check KMS keys in replica regions
aws kms list-keys --region us-west-2
aws kms list-keys --region eu-west-1

# Check permissions in replica regions
aws kms describe-key \
      --key-id <kms-key-id> \
      --region us-west-2

# If error, create replica secret manually
aws secretsmanager replicate-secret-to-regions \
      --secret-id prod/global/app-master-key \
      --add-replica-regions Region=us-west-2,KmsKeyId=<key-id>
```

### High Costs (secret has multiple versions)

**Symptoms:** Unexpected charges, many versions accumulated

**Cause:** Many secret versions created by continuous rotations

**Solutions:**
```bash
# View secret versions
aws secretsmanager list-secret-version-ids \
      --secret-id prod/database/credentials

# Delete old versions (manually)
aws secretsmanager delete-secret \
      --secret-id prod/database/credentials \
      --version-id <old-version-id>

# Or use retention policy for auto-cleanup
# (note: Secrets Manager only keeps current staging versions)

# Limit rotations (increase days between rotations)
kubectl patch secretsmanagersecret rotating-password \
      --type merge \
      -p '{"spec":{"rotationSchedule":{"automaticallyAfterDays":90}}}'
```

### Secret Doesn't Appear After Creating CR

**Symptoms:** `ready: false`, secret doesn't appear in AWS

**Causes:**
1. AWSProvider not configured or not Ready
2. Insufficient permission (CreateSecret)
3. Secret name already exists in another account

**Solutions:**
```bash
# Check AWSProvider status
kubectl describe awsprovider production-aws

# View operator logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100 | grep -i secret

# Check if secret exists in AWS
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials 2>&1

# Check CR events
kubectl describe secretsmanagersecret db-credentials

# Try forcing synchronization
kubectl annotate secretsmanagersecret db-credentials \
      force-sync="$(date +%s)" --overwrite
```

### Cannot Delete Secret (in scheduled deletion)

**Symptoms:** Secret remains in "PendingDeletion" indefinitely

**Cause:** Recovery window hasn't expired yet

**Solutions:**
```bash
# View scheduled deletion date
aws secretsmanager describe-secret \
      --secret-id prod/database/credentials \
      --query 'DeletedDate'

# Cancel deletion if changed mind
aws secretsmanager restore-secret \
      --secret-id prod/database/credentials

# Force immediate deletion
aws secretsmanager delete-secret \
      --secret-id prod/database/credentials \
      --force-delete-without-recovery

# Via kubectl (use deletionPolicy: Delete)
kubectl patch secretsmanagersecret db-credentials \
      --type merge \
      -p '{"spec":{"deletionPolicy":"Delete"}}'
```

## Best Practices

:::note Best Practices

- **Use customer-managed KMS keys** — Enables granular auditing via CloudTrail, more control than AWS-managed
- **Enable automatic rotation** — Configure rotation schedules for database credentials
- **Version secrets properly** — Use staging labels to manage secret versions during rotation
- **Never log secret values** — Use *** in logs instead of actual values
- **Tag secrets consistently** — Environment, application, rotation-enabled tags for governance

:::

## Architecture Patterns

### Pattern: RDS Integration

Secrets Manager with automatic rotation integrated with RDS:

```yaml
# 1. Create secret with RDS credentials
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: rds-auto-rotate
spec:
  providerRef:
    name: production-aws

  secretName: prod/rds/postgres/password

  description: PostgreSQL password rotated by RDS integration

  secretString: |
    {
      "username": "postgres",
      "password": "InitialPassword123!@#"
    }

  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/rds-key

  # Automatic rotation via Lambda
  rotationEnabled: true
  rotationLambdaArn: arn:aws:lambda:us-east-1:123456789012:function:SecretsManager-postgres-rotate
  rotationSchedule:
    automaticallyAfterDays: 30

---
# 2. RDS uses this secret for credentials
# (Configured via console or RDS IaC)
```

### Pattern: Secrets per Application

Organize secrets by application with namespacing:

```yaml
---
# App A: Database
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: app-a-db
spec:
  secretName: prod/app-a/database/credentials
  secretString: |
    {
      "host": "db-a.example.com",
      "username": "app_a_user"
    }
  tags:
    Application: app-a

---
# App A: API Keys
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: app-a-api-keys
spec:
  secretName: prod/app-a/api/keys
  secretString: |
    {
      "stripe_key": "sk_live_..."
    }
  tags:
    Application: app-a

---
# App B: Database
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: app-b-db
spec:
  secretName: prod/app-b/database/credentials
  tags:
    Application: app-b
```

### Pattern: Secrets with Multi-Region HA

Global configuration with replication:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: global-master-key
spec:
  providerRef:
    name: production-aws

  secretName: prod/global/encryption-key

  secretString: |
    {
      "master_key": "key-value-here"
    }

  # Automatic replication
  replicaRegions:
    - region: us-west-2
      kmsKeyId: arn:aws:kms:us-west-2:123456789012:key/west
    - region: eu-west-1
      kmsKeyId: arn:aws:kms:eu-west-1:123456789012:key/eu
    - region: ap-southeast-1
      kmsKeyId: arn:aws:kms:ap-southeast-1:123456789012:key/ap

  tags:
    HA: enabled
    GlobalReplication: "true"
```

## Service Integration

### RDS Password Rotation

RDS supports automatic password rotation via Lambda:

```bash
# AWS automatically creates Lambda function for RDS rotation
# Secret must be in the format:
# {"username": "...", "password": "..."}

# Configure secret for RDS (via AWS Console or CLI)
aws secretsmanager put-secret-attachments \
  --secret-id prod/rds/postgres/password \
  --secret-binary arn:aws:rds:<region>:account:db:instance-name
```

### Kubernetes ExternalSecrets Operator

Use External Secrets Operator to sync with Kubernetes:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets-sa

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-store
    kind: SecretStore
  target:
    name: db-credentials
    creationPolicy: Owner
  data:
  - secretKey: username
    remoteRef:
      key: prod/database/credentials
      property: username
  - secretKey: password
    remoteRef:
      key: prod/database/credentials
      property: password
```

## Use Cases

### 1. Database Credentials

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: prod-postgres
spec:
  secretName: prod/postgres/admin
  secretString: |
    {
      "host": "prod-db.example.com",
      "port": 5432,
      "username": "admin",
      "password": "SecurePassword123!"
    }
```

### 2. Third-Party API Keys

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: stripe-keys
spec:
  secretName: prod/stripe/api-keys
  secretString: |
    {
      "public_key": "pk_live_...",
      "secret_key": "sk_live_..."
    }
```

### 3. SSL/TLS Certificates

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
metadata:
  name: tls-cert
spec:
  secretName: prod/tls/domain-com
  secretString: |
    {
      "certificate": "-----BEGIN CERTIFICATE-----...",
      "private_key": "-----BEGIN PRIVATE KEY-----...",
      "expiration": "2026-12-31"
    }
```

## Comparison with Alternatives

| Aspect | Secrets Manager | Parameter Store | Kubernetes Secrets |
|--------|-----------------|-----------------|-------------------|
| **Encryption** | KMS mandatory | KMS optional | Base64 only |
| **Rotation** | Automatic via Lambda | Manual | Manual |
| **Versioning** | Automatic | No versioning | No versioning |
| **Auditing** | Complete CloudTrail | Basic CloudTrail | etcd logs |
| **Cost** | $0.40/month per secret | Free up to 100 | Free (cluster) |
| **Replication** | Automatic multi-region | Manual | Manual |
| **Compliance** | SOC2, PCI-DSS ready | Limited | Limited |

## Related Resources

- [KMS Key](/services/security/kms)

  - [IAM](/services/security/iam)

  - [RDS](/services/database/rds)

  - [Lambda](/services/compute/lambda)
---