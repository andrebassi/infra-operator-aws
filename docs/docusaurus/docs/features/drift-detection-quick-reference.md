# Drift Detection - Quick Reference

## Quick Start (30 seconds)

**Example:**

```yaml
# Enable drift detection in your AWSProvider
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: my-provider
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/operator
  driftDetection:
    enabled: true
    autoHeal: true
```

## Common Commands

**Command:**

```bash
# Check if drift is detected
kubectl get vpc,s3bucket,rds -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.driftDetected}{"\n"}{end}'

# View drift details
kubectl get vpc my-vpc -o jsonpath='{.status.driftDetails}' | jq .

# View drift events
kubectl get events --field-selector involvedObject.name=my-vpc | grep Drift

# Count resources with drift
kubectl get vpc -o json | jq '[.items[] | select(.status.driftDetected == true)] | length'
```

## Configuration Cheat Sheet

| Mode | Config | Use Case |
|------|--------|----------|
| **Production** | `autoHeal: true`<br>`severityThreshold: "high"`<br>`checkInterval: "5m"` | Auto-heal only critical issues |
| **Staging** | `autoHeal: true`<br>`severityThreshold: "medium"`<br>`checkInterval: "5m"` | Auto-heal most issues |
| **Development** | `autoHeal: true`<br>`severityThreshold: "low"`<br>`checkInterval: "1m"` | Auto-heal everything, fast |
| **Audit Only** | `autoHeal: false`<br>`checkInterval: "15m"` | Alert only, no changes |

## Common Ignore Patterns

**Example:**

```yaml
driftDetection:
  ignoreFields:
    # AWS-managed tags
    - "tags.aws:*"

    # Timestamps
    - "lastModified"
    - "createdTime"

    # External tools
    - "tags.terraform:*"
    - "tags.backup:*"

    # Monitoring
    - "tags.LastBackup"
    - "tags.MonitoringEnabled"
```

## Severity Levels

| Severity | Auto-Assigned To | Example Fields |
|----------|------------------|----------------|
| **High** | Security, networking | `securityGroupIds`, `iamRole`, `encryption`, `cidrBlock` |
| **Medium** | Functionality | `instanceType`, `storageSize`, `timeout` |
| **Low** | Metadata | `tags.*`, `description`, `name` |

## Troubleshooting One-Liners

**Command:**

```bash
# Why isn't drift being detected?
kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection}'

# When was the last drift check?
kubectl get vpc my-vpc -o jsonpath='{.status.lastDriftCheck}'

# What fields are being ignored?
kubectl get awsprovider -o jsonpath='{.items[*].spec.driftDetection.ignoreFields}'

# View operator logs for drift
kubectl logs -n infra-operator-system deployment/infra-operator-controller-manager | grep -i drift

# Get all high severity drifts
kubectl get vpc,s3bucket -o json | jq '.items[].status.driftDetails[] | select(.severity == "high")'
```

## Integration Template

**Example:**

```go
// Add to your controller's Reconcile method after sync

// 1. Get provider
provider := &infrav1alpha1.AWSProvider{}
r.Get(ctx, client.ObjectKey{Name: cr.Spec.ProviderRef.Name, Namespace: cr.Namespace}, provider)

// 2. Get config
driftConfig := drift.ConfigFromProvider(provider)

// 3. Check if time to check drift
if drift.ShouldCheckDrift(cr.Status.LastDriftCheck, driftConfig.CheckInterval) {
// 4. Get actual state
actual, _ := repo.Get(ctx, resourceID)

// 5. Create heal function
healFunc := func(ctx context.Context, drifts []drift.DriftItem, resource interface{}) error {
        return useCase.Sync(ctx, desired)
}

// 6. Create helper
helper := drift.NewControllerHelper(driftConfig, r.Scheme, healFunc)

// 7. Detect and reconcile
hasDrift, _ := helper.CheckAndReconcileDrift(ctx, desired, actual, "ResourceType", resourceID, cr)

// 8. Update status
now := metav1.Now()
cr.Status.LastDriftCheck = &now
cr.Status.DriftDetected = hasDrift

if hasDrift {
        detector := drift.NewDetector(driftConfig)
        result, _ := detector.DetectDrift(ctx, desired, actual, "ResourceType", resourceID)
        cr.Status.DriftDetails = drift.GetDriftDetailsForStatus(result.Drifts)
}
}
```

## Status Fields to Add to CRDs

**Example:**

```go
type MyResourceStatus struct {
// ... existing fields ...

// Drift detection
DriftDetected  bool                         `json:"driftDetected,omitempty"`
DriftDetails   []infrav1alpha1.DriftDetail  `json:"driftDetails,omitempty"`
LastDriftCheck *metav1.Time                 `json:"lastDriftCheck,omitempty"`
}
```

## Files Modified/Created

```
pkg/drift/
├── types.go                      # Core types and config
├── detector.go                   # Drift detection logic
├── reconciler.go                 # Reconciliation logic
├── controller_integration.go     # Helper for controllers
├── detector_test.go              # Detector tests
└── reconciler_test.go            # Reconciler tests

api/v1alpha1/
├── awsprovider_types.go          # Added DriftDetectionConfig
├── vpc_types.go                  # Added drift status fields
└── elasticip_types.go            # Added drift status fields

controllers/
├── vpc_controller_drift_example.go       # VPC integration example
└── elasticip_controller_drift_example.go # ElasticIP integration example

docs/features/
└── drift-detection.mdx           # Complete documentation
```

## Next Steps

1. **Enable in AWSProvider**: Add `driftDetection` config
2. **Update CRDs**: Run `make manifests generate`
3. **Deploy operator**: `kubectl apply -f config/...`
4. **Monitor**: Watch events and status
5. **Tune**: Adjust `ignoreFields` and `severityThreshold` as needed

---

For full documentation, see [drift-detection.md](./drift-detection.md)
