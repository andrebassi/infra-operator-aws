---
title: 'KMS Key - Key Management'
description: 'Create and manage encryption keys with AWS KMS'
sidebar_position: 3
---

Create and manage symmetric and asymmetric encryption keys on AWS with support for automatic rotation, granular access control, and complete auditing.

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

**IAM Policy - KMS (kms-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Action": [
        "kms:CreateKey",
        "kms:DescribeKey",
        "kms:GetKeyPolicy",
        "kms:PutKeyPolicy",
        "kms:EnableKeyRotation",
        "kms:DisableKeyRotation",
        "kms:GetKeyRotationStatus",
        "kms:ScheduleKeyDeletion",
        "kms:CancelKeyDeletion",
        "kms:TagResource",
        "kms:UntagResource",
        "kms:ListResourceTags",
        "kms:EnableKey",
        "kms:DisableKey"
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
  --role-name infra-operator-kms-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator KMS management"

# 4. Create and attach policy
aws iam put-role-policy \
  --role-name infra-operator-kms-role \
  --policy-name KMSManagement \
  --policy-document file://kms-policy.json

# 5. Get Role ARN
aws iam get-role \
  --role-name infra-operator-kms-role \
  --query 'Role.Arn' \
  --output text
```

**Annotate Operator ServiceAccount:**

```bash
# Add annotation to operator ServiceAccount
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-kms-role
```
:::note

Replace `123456789012` with your AWS Account ID and `EXAMPLED539D4633E53DE1B71EXAMPLE` with your OIDC provider ID.

:::

## Overview

AWS Key Management Service (KMS) provides centralized management of encryption keys with:

- **Symmetric Keys**: For fast encryption/decryption (ENCRYPT_DECRYPT)
- **Asymmetric Keys**: For digital signatures and verification (SIGN_VERIFY)
- **Automatic Rotation**: Rotate keys annually without losing access to old data
- **Complete Auditing**: Logging of all key operations
- **Access Control**: Granular Key Policies with IAM policies
- **Multi-Region**: Key replication between regions for disaster recovery
- **HSM Backing**: Secure storage in Hardware Security Modules (CloudHSM)

## Quick Start

**Symmetric Key:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: e2e-symmetric-key
  namespace: default
spec:
  providerRef:
    name: localstack
  description: "Symmetric encryption key for E2E testing"
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true
  enabled: true
  tags:
    environment: test
    managed-by: infra-operator
    purpose: e2e-testing
  deletionPolicy: Delete
  pendingWindowInDays: 7
```

**Asymmetric RSA Key:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: e2e-rsa-signing-key
  namespace: default
spec:
  providerRef:
    name: localstack
  description: "RSA key for digital signatures - E2E testing"
  keyUsage: SIGN_VERIFY
  keySpec: RSA_2048
  enabled: true
  tags:
    environment: test
    managed-by: infra-operator
    key-type: signing
    purpose: e2e-testing
  deletionPolicy: Delete
  pendingWindowInDays: 7
```

**Production Key:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: app-encryption-key
  namespace: default
spec:
  providerRef:
    name: production-aws
  description: Encryption key for application data
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true
  tags:
    Environment: production
    Application: myapp
  deletionPolicy: Retain
```

**Apply:**

```bash
kubectl apply -f kms.yaml
```

**Check Status:**

```bash
kubectl get kmskeys
kubectl describe kmskey e2e-symmetric-key
```
## Configuration Reference

### Required Fields

Reference to AWSProvider resource

  AWSProvider resource name

Clear description of the key and its purpose

  **Examples:**
  - "Encryption key for S3 buckets"
  - "Production database encryption"
  - "Lambda function secrets"

Type of operation allowed with the key

  **Options:**
  - `ENCRYPT_DECRYPT`: Encryption/decryption (symmetric keys)
  - `SIGN_VERIFY`: Digital signature (asymmetric keys)

Technical specification of the key (also accepted as `customerMasterKeySpec` for compatibility)

  **Symmetric Options:**
  - `SYMMETRIC_DEFAULT`: Default symmetric key (recommended for most cases)

  **Asymmetric Options (RSA):**
  - `RSA_2048`: 2048-bit RSA key
  - `RSA_3072`: 3072-bit RSA key
  - `RSA_4096`: 4096-bit RSA key

  **Asymmetric Options (ECC):**
  - `ECC_NIST_P256`: NIST P-256 curve
  - `ECC_NIST_P384`: NIST P-384 curve
  - `ECC_NIST_P521`: NIST P-521 curve

### Optional Fields

JSON policy defining key access permissions

  **Note:** If not provided, default allows access to account root

  **Example:**
  ```json
  {
"Version": "2012-10-17",
"Statement": [{
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {"AWS": "arn:aws:iam::ACCOUNT_ID:root"},
      "Action": "kms:*",
      "Resource": "*"
}, {
      "Sid": "Allow services",
      "Effect": "Allow",
      "Principal": {"Service": ["s3.amazonaws.com", "rds.amazonaws.com"]},
      "Action": ["kms:Decrypt", "kms:GenerateDataKey"],
      "Resource": "*"
}]
  }
  ```

If `true`, automatically rotates the key every year

  **Recommendation:** Enable for production
  **Cost:** No additional cost
  **Compatibility:** Symmetric keys only

If `true`, replicates the key to multiple regions (disaster recovery)

  **Cost:** 1x additional cost per replication region
  **Use cases:** High availability and disaster recovery

Allow key creation even if policy could block your access

  **Warning:** Use only if you know what you're doing

Key-value pairs for organization, billing, and control

  **Example:**

  ```yaml
  tags:
    Environment: production
    Application: myapp
    Team: platform
    CostCenter: engineering
    BackupRequired: "true"
  ```

What happens to the key when the CR is deleted

  **Options:**
  - `ScheduleKeyDeletion`: Schedule deletion in 7-30 days (default, safe)
  - `DisableKey`: Only disables, keeps data accessible
  - `ForceDelete`: Delete immediately (DANGEROUS - data will become inaccessible)

## Status Fields

After the key is created, the following status fields are populated:

Unique key ID (e.g., `a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6`)

Complete key ARN (e.g., `arn:aws:kms:us-east-1:123456789012:key/a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6`)

Current key state:
  - `Enabled`: Key is operational
  - `Disabled`: Key has been disabled (can be re-enabled)
  - `PendingDeletion`: Scheduled for deletion
  - `Unavailable`: Not available (temporary error)

Key creation timestamp (ISO 8601)

Last automatic rotation timestamp (if enabled)

Whether automatic rotation is enabled

`true` when key is Enabled and ready for use

Last synchronization timestamp with AWS

## Examples

### Symmetric Key for S3 Encryption

Key for encrypting S3 buckets:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: s3-encryption-key
spec:
  providerRef:
    name: production-aws
  description: Encryption key for S3 buckets
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true

  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM User Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }, {
        "Sid": "Allow S3 to use the key",
        "Effect": "Allow",
        "Principal": {"Service": "s3.amazonaws.com"},
        "Action": [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ],
        "Resource": "*"
      }]
    }

  tags:
    Purpose: s3-encryption
    Environment: production

  deletionPolicy: ScheduleKeyDeletion
```

### Asymmetric Key for Digital Signatures

RSA key for digital message signing:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: signing-key-rsa
spec:
  providerRef:
    name: production-aws
  description: RSA key for digital signatures and verification
  keyUsage: SIGN_VERIFY
  keySpec: RSA_2048

  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }, {
        "Sid": "Allow Lambda to sign",
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": [
          "kms:Sign",
          "kms:Verify"
        ],
        "Resource": "*"
      }]
    }

  tags:
    Purpose: digital-signatures
    Service: auth-service

  deletionPolicy: ScheduleKeyDeletion
```

### Multi-Region Key for Disaster Recovery

Key replicated across multiple regions:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: dr-replication-key
spec:
  providerRef:
    name: production-aws
  description: Multi-region key for disaster recovery
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT

  # Enable multi-region
  multiRegion: true
  enableKeyRotation: true

  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }, {
        "Sid": "Allow replication across regions",
        "Effect": "Allow",
        "Principal": {"Service": "kms.amazonaws.com"},
        "Action": "kms:ReplicateKey",
        "Resource": "*"
      }]
    }

  tags:
    Purpose: dr-replication
    Environment: production
    BackupRequired: "true"

  deletionPolicy: ScheduleKeyDeletion
```

### Key with Restrictive Policy (Least Privilege)

Key with access restricted to specific service only:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: lambda-secrets-key
spec:
  providerRef:
    name: production-aws
  description: Encryption key for Lambda secrets
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true

  # Very restrictive policy - Lambda only
  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }, {
        "Sid": "Allow Lambda function to decrypt",
        "Effect": "Allow",
        "Principal": {
          "AWS": "arn:aws:iam::123456789012:role/lambda-execution-role"
        },
        "Action": [
          "kms:Decrypt",
          "kms:DescribeKey"
        ],
        "Resource": "*"
      }]
    }

  tags:
    Service: lambda
    Purpose: secrets-encryption

  deletionPolicy: ScheduleKeyDeletion
```

### Key with Grants for Lambda

Key using Grants instead of Key Policy (more secure pattern):

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: app-grants-key
spec:
  providerRef:
    name: production-aws
  description: Key using grants for Lambda access
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true

  # Default policy (allows root only)
  keyPolicy: |
    {
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }]
    }

  # Grants for granular access
  grants:
  - name: lambda-decrypt-grant
    granteeRoleArn: arn:aws:iam::123456789012:role/lambda-execution-role
    operations:
    - Decrypt
    - DescribeKey

  - name: rds-encrypt-grant
    granteeRoleArn: arn:aws:iam::123456789012:role/rds-service-role
    operations:
    - Encrypt
    - Decrypt
    - GenerateDataKey
    - DescribeKey

  tags:
    Environment: production
    AccessModel: grants

  deletionPolicy: ScheduleKeyDeletion
```

## Verification

### Check Status via kubectl

**Command:**

```bash
# List all keys
kubectl get kmskeys

# Get detailed information
kubectl get kmskey app-encryption-key -o yaml

# Watch creation in real-time
kubectl get kmskey app-encryption-key -w

# View only status fields
kubectl get kmskey app-encryption-key -o jsonpath='{.status}'

# Describe key with events
kubectl describe kmskey app-encryption-key
```

### Check on AWS

**AWS CLI:**

```bash
# Describe key
aws kms describe-key \
      --key-id arn:aws:kms:us-east-1:123456789012:key/a1b2c3d4 \
      --region us-east-1 \
      --output json | jq '.KeyMetadata'

# List aliases
aws kms list-aliases --region us-east-1

# Create alias for key
aws kms create-alias \
      --alias-name alias/app-encryption \
      --target-key-id a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6 \
      --region us-east-1

# View key policy
aws kms get-key-policy \
      --key-id a1b2c3d4 \
      --policy-name default \
      --region us-east-1

# Check if rotation is enabled
aws kms get-key-rotation-status \
      --key-id a1b2c3d4 \
      --region us-east-1

# List grants for a key
aws kms list-grants \
      --key-id a1b2c3d4 \
      --region us-east-1
```
  
**LocalStack:**

```bash
# Point to LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

# Describe key
aws kms describe-key \
      --key-id a1b2c3d4 \
      --output json | jq '.KeyMetadata'

# List keys
aws kms list-keys

# List aliases
aws kms list-aliases
```

### Expected Output

**Example:**

```yaml
status:
  keyId: a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6
  keyArn: arn:aws:kms:us-east-1:123456789012:key/a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6
  keyState: Enabled
  creationDate: "2025-11-22T20:18:08Z"
  lastRotationDate: "2025-11-22T20:18:08Z"
  rotationEnabled: true
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Troubleshooting

### Key Creation Failure

**Symptoms:** `status.ready: false`, key doesn't appear in AWS

**Common Causes:**
1. AWSProvider not configured correctly
2. Insufficient credentials (`kms:CreateKey` permission)
3. Key limit reached in account

**Solutions:**
```bash
# Check AWSProvider status
kubectl describe awsprovider production-aws

# View controller logs
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      -f | grep -i kms

# Check events
kubectl describe kmskey app-encryption-key

# Count existing keys
aws kms list-keys | jq '.Keys | length'
```
  
### Access Denied When Using Key

**Error:** `UserNotAuthorizedForKeyException` or `InvalidStateException`

**Causes:**
1. Key Policy doesn't allow the operation
2. IAM role without kms:Decrypt or kms:GenerateDataKey permission
3. Key is Disabled or PendingDeletion

**Solutions:**
```bash
# Check key state
aws kms describe-key --key-id a1b2c3d4 \
      --query 'KeyMetadata.KeyState'

# View key policy
aws kms get-key-policy \
      --key-id a1b2c3d4 \
      --policy-name default | jq '.'

# Re-enable if disabled
aws kms enable-key --key-id a1b2c3d4

# Add permission via IAM policy
# (alternative to modifying key policy)
aws iam put-role-policy \
      --role-name lambda-execution-role \
      --policy-name kms-decrypt \
      --policy-document file://kms-policy.json
```

**Example kms-policy.json:**
```json
{
      "Version": "2012-10-17",
      "Statement": [{
        "Effect": "Allow",
        "Action": [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ],
        "Resource": "arn:aws:kms:*:123456789012:key/a1b2c3d4"
      }]
}
```
  
### Rotation Not Working

**Symptoms:** `lastRotationDate` doesn't change, rotation not automatic

**Causes:**
1. `enableKeyRotation: false` (default)
2. Asymmetric key (automatic rotation only for symmetric)
3. Key in PendingDeletion state

**Solutions:**
```bash
# Enable rotation (via patch)
kubectl patch kmskey app-encryption-key --type='json' \
      -p='[{"op": "replace", "path": "/spec/enableKeyRotation", "value":true}]'

# Check if rotation is enabled
aws kms get-key-rotation-status --key-id a1b2c3d4

# Rotate manually (creates new version)
aws kms rotate-key --key-id a1b2c3d4

# View rotation history
aws kms list-key-rotations --key-id a1b2c3d4
```
  
### Overly Permissive Key Policy

**Symptoms:** Compliance violation, security issue

**Causes:**
1. Actions: `kms:*` (allows everything)
2. Principal: `*` (allows anyone)
3. Resource: `*` (more specific is better)

**Solutions:**
```bash
# Get current policy
aws kms get-key-policy \
      --key-id a1b2c3d4 \
      --policy-name default > policy.json

# Edit policy.json with least privilege

# Apply new policy
aws kms put-key-policy \
      --key-id a1b2c3d4 \
      --policy-name default \
      --policy file://policy.json

# Verify new policy
aws kms get-key-policy \
      --key-id a1b2c3d4 \
      --policy-name default | jq '.'
```

**Example policy with least privilege:**
```json
{
      "Version": "2012-10-17",
      "Statement": [{
        "Sid": "Enable IAM Permissions",
        "Effect": "Allow",
        "Principal": {"AWS": "arn:aws:iam::123456789012:root"},
        "Action": "kms:*",
        "Resource": "*"
      }, {
        "Sid": "Allow specific service",
        "Effect": "Allow",
        "Principal": {"Service": "s3.amazonaws.com"},
        "Action": [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ],
        "Resource": "*",
        "Condition": {
          "StringEquals": {
            "aws:SourceAccount": "123456789012"
          }
        }
      }]
}
```
  
### High Costs (~$1/month per key)

**Symptoms:** AWS account with unexpected KMS costs

**Possible causes:**
1. Many keys created and not used
2. Multi-region enabled unnecessarily
3. Old keys not deleted

**Solutions:**
```bash
# List all keys with dates
aws kms list-keys | jq '.Keys[] | {KeyId, KeyArn}' | while read line; do
      KEY_ID=$(echo $line | jq -r '.KeyId')
      aws kms describe-key --key-id $KEY_ID | jq '.KeyMetadata | {KeyId, CreationDate, KeyState, MultiRegion}'
done

# Disable unused keys (without deleting)
aws kms disable-key --key-id a1b2c3d4

# Schedule deletion (to clean up old keys)
aws kms schedule-key-deletion \
      --key-id a1b2c3d4 \
      --pending-window-in-days 7

# Remove multi-region replication if not needed
# Note: can't be removed, but don't use multiRegion: true for new keys
```

### Deletion Stuck or Unexpectedly Cancels

**Symptoms:** ScheduleKeyDeletion stays pending, or policy doesn't allow deletion

**Causes:**
1. Another service using the key (S3, RDS, etc)
2. Grants not revoked
3. Key policy doesn't allow DeleteKey

**Solutions:**
```bash
# Check if there's data encrypted with the key
# (AWS S3, RDS, Secrets Manager, etc might be using it)

# List grants (usage authorities)
aws kms list-grants --key-id a1b2c3d4

# Revoke grants if necessary
aws kms retire-grant --key-id a1b2c3d4 --grant-token <token>

# View deletion schedule
aws kms describe-key --key-id a1b2c3d4 \
      --query 'KeyMetadata.DeletionDate'

# Cancel scheduled deletion
aws kms cancel-key-deletion --key-id a1b2c3d4

# Force delete (ONLY if absolutely necessary)
# WARNING: Encrypted data will become inaccessible!
# Use only with deletionPolicy: ForceDelete in spec
```
  
## Best Practices

:::note Best Practices

- **Enable automatic key rotation** — AWS rotates every 365 days automatically with enableKeyRotation
- **Use separate keys per environment** — Isolate dev/staging/prod encryption keys
- **Apply least-privilege key policies** — Only grant kms:Encrypt/Decrypt to services that need it
- **Enable CloudTrail logging** — Audit all key usage for compliance
- **Use aliases for key management** — Easier to reference and rotate keys without updating applications

:::

## Architecture Patterns

### Pattern: Envelope Encryption (Recommended)

Use KMS to encrypt data keys, not data directly:

```yaml
# 1. KMS key for Data Key Encryption Key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: dkek-master-key
spec:
  providerRef:
    name: production-aws
  description: Data Key Encryption Key (DKEK) for envelope encryption
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  enableKeyRotation: true
  deletionPolicy: ScheduleKeyDeletion

# 2. Use in S3:
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: encrypted-bucket
spec:
  providerRef:
    name: production-aws
  bucketName: my-encrypted-data
  encryption:
algorithm: aws:kms
keyArn: arn:aws:kms:us-east-1:123456789012:key/a1b2c3d4

# 3. Lambda encrypts data locally, then sends to S3 already encrypted
```

### Pattern: Key per Service

Separate keys by service for access isolation:

```yaml
# Database key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: rds-encryption-key
spec:
  description: KMS key for RDS PostgreSQL
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  keyPolicy: |
{
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "rds.amazonaws.com"},
        "Action": ["kms:Decrypt", "kms:GenerateDataKey"],
        "Resource": "*"
      }]
}

---
# S3 key (different for isolation)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: s3-encryption-key
spec:
  description: KMS key for S3 buckets
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  keyPolicy: |
{
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "s3.amazonaws.com"},
        "Action": ["kms:Decrypt", "kms:GenerateDataKey"],
        "Resource": "*"
      }]
}
```

### Pattern: Multi-Region for HA

Key replicated to multiple regions:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: global-ha-key
spec:
  providerRef:
    name: production-aws
  description: Multi-region key for global HA
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
  multiRegion: true  # Automatically replicated
  enableKeyRotation: true

  tags:
    Tier: critical
    HA: enabled
```

## Use Cases

### S3 Bucket Encryption

**Example:**

```yaml
# KMS key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: s3-key
spec:
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT

---
# S3 bucket using the key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: encrypted-data
spec:
  encryption:
algorithm: aws:kms
keyArn: <keyArn from KMSKey>
```

### RDS Database Encryption

**Example:**

```yaml
# KMS key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: rds-encryption-key
spec:
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT

---
# RDS using the key (in RDSInstance CR)
spec:
  storageEncrypted: true
  kmsKeyId: <keyId from KMSKey>
```

### EBS Volume Encryption

**Example:**

```yaml
# KMS key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: ebs-encryption-key
spec:
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
```

### Secrets Manager Encryption

**Example:**

```yaml
# KMS key
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: secrets-key
spec:
  keyUsage: ENCRYPT_DECRYPT
  keySpec: SYMMETRIC_DEFAULT
```

### Digital Signature (SIGN_VERIFY)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
metadata:
  name: signing-key
spec:
  keyUsage: SIGN_VERIFY
  keySpec: RSA_2048
```

## Related Resources

- [S3 Bucket](/services/storage/s3)

  - [RDS](/services/database/rds)

  - [Secrets Manager](/services/security/secrets-manager)

  - [IAM](/services/security/iam)

  - [CloudTrail](/services/logging/cloudtrail)