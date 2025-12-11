---
title: 'Drift Detection'
description: 'Automatic detection and reconciliation of infrastructure drifts'
sidebar_position: 4
---

# Drift Detection and Auto-Healing

## Overview

**Drift Detection** is a feature that automatically detects when the actual state of AWS resources differs from the desired state defined in your Kubernetes Custom Resources (CRs), even when the CRs have not been changed.

### What is Drift?

Drift occurs when infrastructure is modified outside the operator, such as:

- Manual changes via AWS Console
- Modifications via AWS CLI or other tools
- Changes made by other automation systems
- Accidental modifications by team members
- AWS service updates or migrations

### Why Drift Detection Matters

Without drift detection:
- Manual changes can break your infrastructure
- Compliance violations go unnoticed
- Configuration inconsistencies accumulate
- Debugging becomes harder over time
- GitOps workflows lose their single source of truth

With drift detection:
- Automatic detection of all changes
- Configurable auto-healing or alerts
- Detailed drift reports in resource status
- Kubernetes events for monitoring
- Maintains infrastructure consistency

## How It Works

![Drift Detection Flow](/img/diagrams/drift-detection-flow.svg)

### Detection Process

1. **Periodic Checks**: Controller checks for drift at configured intervals (default: 5 minutes)
2. **Comparison**: Compares desired state (CR) with actual state (AWS API)
3. **Classification**: Categorizes drifts by severity (low, medium, high)
4. **Filtering**: Applies ignore patterns and severity thresholds
5. **Action**: Auto-heals or alerts based on configuration

## Configuration

### Enable Drift Detection in AWSProvider

Configure drift detection at the provider level:

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: production-aws
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role

  # Drift Detection Configuration
  driftDetection:
    # Enable drift detection (default: true)
    enabled: true

    # Drift check interval (default: "5m")
    # Accepts: "1m", "5m", "15m", "1h", etc.
    checkInterval: "5m"

    # Auto-heal detected drifts (default: true)
    # true: automatically fixes by updating AWS
    # false: only alerts via events and status (alert-only mode)
    autoHeal: true

    # Fields to ignore in drift detection
    # Supports wildcards (*)
    ignoreFields:
      - "tags.aws:*"           # Ignore AWS-managed tags
      - "lastModified"         # Ignore modification timestamps
      - "status.*"             # Ignore status fields
      - "tags.LastUpdated"     # Ignore specific tags

    # Minimum severity level to report (default: "medium")
    # Options: "low", "medium", "high"
    # Only drifts at this level or above trigger reconciliation
    severityThreshold: "medium"
```

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable drift detection |
| `checkInterval` | string | `"5m"` | Frequency of drift checks |
| `autoHeal` | boolean | `true` | Automatically fix vs alert only |
| `ignoreFields` | []string | `["tags.aws:*", "lastModified", "status.*"]` | Field patterns to ignore |
| `severityThreshold` | string | `"medium"` | Minimum severity to act on |

## Drift Severity Levels

The operator automatically assigns severity levels based on the field that drifted:

### High Severity
Fields affecting security, networking, or critical functionality:
- Security groups
- IAM roles
- Encryption configurations
- Public access flags
- CIDR blocks
- Network configurations

**Example:**

```yaml
# High severity drift detected
status:
  driftDetected: true
  driftDetails:
    - field: "securityGroupIds[0]"
      expected: "sg-prod-123"
      actual: "sg-dev-456"
      severity: "high"
```

### Medium Severity
Fields affecting functionality but not security:
- Instance types
- Storage sizes
- Connection configurations
- Resource settings

**Example:**

```yaml
# Medium severity drift detected
status:
  driftDetected: true
  driftDetails:
    - field: "instanceType"
      expected: "t3.large"
      actual: "t3.medium"
      severity: "medium"
```

### Low Severity
Metadata and cosmetic fields:
- Tags (except security tags)
- Descriptions
- Names
- Non-critical metadata

**Example:**

```yaml
# Low severity drift detected
status:
  driftDetected: true
  driftDetails:
    - field: "tags.Description"
      expected: "Production VPC"
      actual: "Prod VPC"
      severity: "low"
```

## Checking Drift Status

### View Drift in Resource Status

**Command:**

```bash
kubectl get vpc production-vpc -o yaml
```

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  cidrBlock: "10.0.0.0/16"
  tags:
    Environment: production
status:
  ready: true
  vpcID: vpc-0123456789abcdef

  # Drift Detection Status
  driftDetected: true
  lastDriftCheck: "2024-11-23T10:30:00Z"
  driftDetails:
    - field: "tags.Environment"
      expected: "production"
      actual: "prod"
      severity: "low"
    - field: "enableDnsHostnames"
      expected: "true"
      actual: "false"
      severity: "medium"
```

### View Drift Events

**Command:**

```bash
kubectl get events --field-selector involvedObject.name=production-vpc
```

**Output:**

```
LAST SEEN   TYPE      REASON              MESSAGE
2m          Warning   DriftDetected       Detected 2 drift(s) for VPC vpc-123 (high: 0, medium: 1, low: 1)
2m          Warning   HighSeverityDrift   High severity drift in securityGroupIds: desired=sg-prod, actual=sg-dev
1m          Normal    DriftHealed         Auto-healed 2 drift(s) for VPC vpc-123
```

### Monitor Drift with kubectl

**Command:**

```bash
# Check all resources with drift
kubectl get vpc,elasticip,s3bucket -A -o json | \
  jq '.items[] | select(.status.driftDetected == true) | {name: .metadata.name, drifts: .status.driftDetails}'

# Count drifts by severity
kubectl get vpc -o json | \
  jq '[.items[].status.driftDetails[]?.severity] | group_by(.) | map({severity: .[0], count: length})'
```

## Auto-Healing

When `autoHeal: true`, the operator automatically fixes drifts:

### How Auto-Healing Works

1. **Detect Drift**: Operator detects difference between CR and AWS
2. **Evaluate**: Checks if drift matches ignore patterns or severity threshold
3. **Fix**: Updates AWS resource to match CR specification
4. **Verify**: Re-checks to confirm drift is resolved
5. **Log**: Creates Kubernetes event and updates status

### Example: Auto-Healing Tags

**Example:**

```yaml
# CR defines these tags
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: prod-vpc
spec:
  cidrBlock: "10.0.0.0/16"
  tags:
    Environment: production
    Team: platform
```

**Scenario:** Someone manually changes tags in AWS Console:
- `Environment: prod` (changed)
- `Team: platform` (no change)
- `Owner: john` (added)

**Auto-Healing Process:**
1. Operator detects drift in tags
2. Logs drift to status and events
3. **Automatically** updates AWS tags to match CR:
   - Restores `Environment: production`
   - Removes `Owner: john` (not in CR)
4. Updates status: `driftDetected: false`

### Healing Function Implementation

Each resource type has its own healing function:

**Go Code:**

```go
// Example: VPC healing function
func (r *VPCReconciler) healVPCDrift(ctx context.Context, drifts []DriftItem, vpcCR *VPC) error {
// Convert CR to domain model
v := mapper.CRToDomainVPC(vpcCR)

// Sync AWS state to match CR
if err := r.vpcUseCase.SyncVPC(ctx, v); err != nil {
        return fmt.Errorf("failed to heal drift: %w", err)
}

return nil
}
```

## Alert-Only Mode

When `autoHeal: false`, the operator only reports drifts without fixing them:

**Example:**

```yaml
driftDetection:
  enabled: true
  autoHeal: false  # Alert-only mode
  checkInterval: "5m"
```

**Behavior:**
- Detects all drifts
- Logs events
- Updates status with drift details
- DOES NOT modify AWS resources
- DOES NOT auto-heal

**Use Cases:**
- Production environments requiring manual approval
- Compliance auditing without auto-remediation
- Testing drift detection before enabling auto-healing
- Resources managed by multiple systems

## Ignore Patterns

### Field Path Patterns

Use ignore patterns to exclude fields from drift detection:

**Example:**

```yaml
driftDetection:
  ignoreFields:
    # Exact match
    - "lastModified"

    # Prefix with wildcard
    - "tags.aws:*"      # Ignore all tags starting with "aws:"
    - "status.*"        # Ignore all status fields

    # Specific tag
    - "tags.LastSync"   # Ignore this specific tag
```

### Common Ignore Patterns

**Example:**

```yaml
# AWS-managed fields
ignoreFields:
  - "tags.aws:*"
  - "lastModified"
  - "createdTime"
  - "status.*"

  # External system tags
  - "tags.terraform:*"
  - "tags.cloudformation:*"

  # Monitoring tags (externally managed)
  - "tags.LastBackup"
  - "tags.LastPatched"
  - "tags.MonitoringEnabled"
```

## Best Practices

:::note Best Practices

- **Start with Alert-Only mode** — Begin with autoHeal disabled to understand what changes occur before enabling auto-remediation
- **Use appropriate check intervals** — Production: 5-10m, Development: 1-2m, balance detection speed vs API costs
- **Configure severity thresholds** — Set autoHealSeverityThreshold to control which drifts trigger automatic fixes
- **Ignore external system tags** — Use ignoreFields to exclude AWS-managed tags (aws:*, cloudformation:*) that change externally
- **Monitor drift metrics** — Configure alerting on drift_detected_total and drift_healing_failed_total metrics

:::

## Troubleshooting

### Drift Not Detected

**Problem:** Drift exists but is not being detected

**Solutions:**
1. Check if drift detection is enabled:
   ```bash
   kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection.enabled}'
   ```

2. Check if check interval has passed:
   ```bash
   kubectl get vpc -o jsonpath='{.items[*].status.lastDriftCheck}'
   ```

3. Check if field is in ignore list:
   ```bash
   kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection.ignoreFields}'
   ```

### Auto-Healing Not Working

**Problem:** Drift detected but not fixing

**Solutions:**
1. Check if auto-healing is enabled:
   ```bash
   kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection.autoHeal}'
   ```

2. Check severity threshold:
   ```bash
   # If threshold is "high", medium/low severity drifts won't be auto-healed
   kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection.severityThreshold}'
   ```

3. Check operator logs:
   ```bash
   kubectl logs -n infra-operator-system deployment/infra-operator-controller-manager | grep drift
   ```

### False Positive Drifts

**Problem:** Legitimate external changes flagged as drift

**Solutions:**
1. Add fields to ignore list:
   ```yaml
   driftDetection:
     ignoreFields:
       - "tags.ManagedByExternal"
       - "specificField"
   ```

2. Lower severity threshold:
   ```yaml
   driftDetection:
     severityThreshold: "high"  # Ignore low/medium severity
   ```

## Examples

### Example 1: High-Security Production Setup

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: production-aws
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator

  driftDetection:
    enabled: true
    checkInterval: "5m"
    autoHeal: true
    severityThreshold: "high"  # Only auto-heal critical issues
    ignoreFields:
      - "tags.aws:*"
      - "tags.backup:*"
      - "lastModified"
```

### Example 2: Development with Full Auto-Healing

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: dev-aws
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator

  driftDetection:
    enabled: true
    checkInterval: "1m"        # Faster checks in dev
    autoHeal: true
    severityThreshold: "low"   # Auto-heal everything
    ignoreFields:
      - "tags.aws:*"
```

### Example 3: Audit Mode (No Auto-Healing)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: audit-aws
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator

  driftDetection:
    enabled: true
    checkInterval: "15m"
    autoHeal: false           # Alert only, no changes
    severityThreshold: "low"  # Report all drifts
    ignoreFields: []          # No ignores, detect everything
```

## Related Features

- [Status Management](/concepts/status-management) - How status is updated
- [Deletion Policies](/concepts/deletion-policies) - Resource cleanup
- [AWS Provider](/concepts/aws-provider) - Credential management

## API Reference

See the complete API specification in CRD documentation:
- [AWSProvider DriftDetectionConfig](/api-reference/awsprovider#driftdetectionconfig)
- [VPC DriftDetail](/api-reference/vpc#driftdetail)
- [Status Fields](/api-reference/common-status#drift-fields)
