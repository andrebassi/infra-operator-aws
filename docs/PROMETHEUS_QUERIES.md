# Prometheus Queries - Quick Reference

Common PromQL queries for Infra Operator metrics.

## Reconciliation Metrics

### Reconciliation Rate
```promql
# Total reconciliation rate
sum(rate(infra_operator_reconcile_total[5m]))

# Per resource type
sum(rate(infra_operator_reconcile_total[5m])) by (resource_type)

# Success rate only
sum(rate(infra_operator_reconcile_total{result="success"}[5m])) by (resource_type)

# Error rate
sum(rate(infra_operator_reconcile_total{result="error"}[5m])) by (resource_type)
```

### Reconciliation Duration
```promql
# p50 (median) duration
histogram_quantile(0.50, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le, resource_type))

# p95 duration
histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le, resource_type))

# p99 duration
histogram_quantile(0.99, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le, resource_type))

# Average duration
rate(infra_operator_reconcile_duration_seconds_sum[5m]) / rate(infra_operator_reconcile_duration_seconds_count[5m])

# Slowest resource types (p95)
topk(5, histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le, resource_type)))
```

### Error Rate
```promql
# Overall error percentage
(sum(rate(infra_operator_reconcile_errors_total[5m])) / sum(rate(infra_operator_reconcile_total[5m]))) * 100

# Per resource type
(sum(rate(infra_operator_reconcile_errors_total[5m])) by (resource_type) / sum(rate(infra_operator_reconcile_total[5m])) by (resource_type)) * 100

# Error rate by error type
sum(rate(infra_operator_reconcile_errors_total[5m])) by (error_type)

# Most common errors
topk(10, sum(rate(infra_operator_reconcile_errors_total[5m])) by (resource_type, error_type))
```

---

## Resource Metrics

### Resource Counts
```promql
# Total managed resources
sum(infra_operator_resources_total)

# By resource type
sum(infra_operator_resources_total) by (resource_type)

# By status
sum(infra_operator_resources_total) by (status)

# Ready resources
sum(infra_operator_resources_total{status="ready"})

# Not ready resources
sum(infra_operator_resources_total{status="notready"})

# Pending resources (being created)
sum(infra_operator_resources_total{status="pending"})

# Resources being deleted
sum(infra_operator_resources_total{status="deleting"})

# Resources in error state
sum(infra_operator_resources_total{status="error"})
```

### Resource Health
```promql
# Percentage of ready resources
(sum(infra_operator_resources_total{status="ready"}) / sum(infra_operator_resources_total)) * 100

# Resource types with most not-ready resources
topk(5, sum(infra_operator_resources_total{status="notready"}) by (resource_type))
```

### Resource Age
```promql
# Age of resources in hours
(time() - infra_operator_resource_creation_timestamp_seconds) / 3600

# Oldest resources
topk(10, time() - infra_operator_resource_creation_timestamp_seconds)

# Average resource age
avg(time() - infra_operator_resource_creation_timestamp_seconds) / 3600
```

---

## AWS API Metrics

### API Call Rate
```promql
# Total AWS API calls per second
sum(rate(infra_operator_aws_api_calls_total[5m]))

# By service
sum(rate(infra_operator_aws_api_calls_total[5m])) by (service)

# By operation
sum(rate(infra_operator_aws_api_calls_total[5m])) by (service, operation)

# Top 10 most called operations
topk(10, sum(rate(infra_operator_aws_api_calls_total[5m])) by (service, operation))

# Success rate
sum(rate(infra_operator_aws_api_calls_total{result="success"}[5m])) by (service)

# Error rate
sum(rate(infra_operator_aws_api_calls_total{result="error"}[5m])) by (service)
```

### API Latency
```promql
# p50 API latency by service
histogram_quantile(0.50, sum(rate(infra_operator_aws_api_call_duration_seconds_bucket[5m])) by (le, service))

# p95 API latency by service
histogram_quantile(0.95, sum(rate(infra_operator_aws_api_call_duration_seconds_bucket[5m])) by (le, service))

# p99 API latency
histogram_quantile(0.99, sum(rate(infra_operator_aws_api_call_duration_seconds_bucket[5m])) by (le, service, operation))

# Average API latency
rate(infra_operator_aws_api_call_duration_seconds_sum[5m]) / rate(infra_operator_aws_api_call_duration_seconds_count[5m])

# Slowest operations (p95)
topk(10, histogram_quantile(0.95, sum(rate(infra_operator_aws_api_call_duration_seconds_bucket[5m])) by (le, service, operation)))
```

### API Errors
```promql
# Total error rate
sum(rate(infra_operator_aws_api_errors_total[5m]))

# By service
sum(rate(infra_operator_aws_api_errors_total[5m])) by (service)

# By error code
sum(rate(infra_operator_aws_api_errors_total[5m])) by (error_code)

# Error percentage
(sum(rate(infra_operator_aws_api_errors_total[5m])) / sum(rate(infra_operator_aws_api_calls_total[5m]))) * 100

# Most common error codes
topk(10, sum(rate(infra_operator_aws_api_errors_total[5m])) by (error_code))

# Services with highest error rates
topk(5, sum(rate(infra_operator_aws_api_errors_total[5m])) by (service))
```

### API Throttling
```promql
# Total throttling rate
sum(rate(infra_operator_aws_api_throttles_total[5m]))

# By service
sum(rate(infra_operator_aws_api_throttles_total[5m])) by (service)

# Throttled operations
sum(rate(infra_operator_aws_api_throttles_total[5m])) by (service, operation) > 0

# Services being throttled most
topk(5, sum(rate(infra_operator_aws_api_throttles_total[5m])) by (service))
```

---

## Drift Detection Metrics

### Drift Detection
```promql
# Total drift detected (rate)
sum(rate(infra_operator_drift_detected_total[5m]))

# By severity
sum(rate(infra_operator_drift_detected_total[5m])) by (severity)

# Critical drifts only
sum(rate(infra_operator_drift_detected_total{severity="critical"}[5m]))

# By resource type
sum(rate(infra_operator_drift_detected_total[5m])) by (resource_type)

# Drifts in last hour
sum(increase(infra_operator_drift_detected_total[1h])) by (resource_type, severity)

# Resource types with most drift
topk(5, sum(increase(infra_operator_drift_detected_total[24h])) by (resource_type))
```

### Drift Healing
```promql
# Healing rate
sum(rate(infra_operator_drift_healed_total[5m]))

# By resource type
sum(rate(infra_operator_drift_healed_total[5m])) by (resource_type)

# Healing success percentage
(sum(rate(infra_operator_drift_healed_total[5m])) / sum(rate(infra_operator_drift_detected_total[5m]))) * 100

# Healing failure rate
sum(rate(infra_operator_drift_healing_failures_total[5m])) by (resource_type)

# Healed vs detected
sum(rate(infra_operator_drift_healed_total[5m])) / sum(rate(infra_operator_drift_detected_total[5m]))
```

### Drift Detection Duration
```promql
# p95 detection duration
histogram_quantile(0.95, sum(rate(infra_operator_drift_detection_duration_seconds_bucket[5m])) by (le, resource_type))

# Average detection duration
rate(infra_operator_drift_detection_duration_seconds_sum[5m]) / rate(infra_operator_drift_detection_duration_seconds_count[5m])
```

---

## Finalizer Metrics

### Finalizer Duration
```promql
# p95 finalizer duration
histogram_quantile(0.95, sum(rate(infra_operator_finalizer_duration_seconds_bucket[5m])) by (le, resource_type))

# p99 finalizer duration
histogram_quantile(0.99, sum(rate(infra_operator_finalizer_duration_seconds_bucket[5m])) by (le, resource_type))

# Average finalizer duration
rate(infra_operator_finalizer_duration_seconds_sum[5m]) / rate(infra_operator_finalizer_duration_seconds_count[5m])

# Slowest finalizers
topk(5, histogram_quantile(0.95, sum(rate(infra_operator_finalizer_duration_seconds_bucket[5m])) by (le, resource_type)))
```

### Finalizer Errors
```promql
# Finalizer error rate
sum(rate(infra_operator_finalizer_errors_total[5m]))

# By resource type
sum(rate(infra_operator_finalizer_errors_total[5m])) by (resource_type)

# By error type
sum(rate(infra_operator_finalizer_errors_total[5m])) by (error_type)
```

---

## Provider Metrics

### Provider Status
```promql
# All providers ready (1 = ready, 0 = not ready)
infra_operator_provider_ready

# Not ready providers
infra_operator_provider_ready == 0

# Ready providers
infra_operator_provider_ready == 1

# Percentage of providers ready
avg(infra_operator_provider_ready) * 100
```

### Credential Rotations
```promql
# Rotation rate
sum(rate(infra_operator_provider_credential_rotations_total[24h]))

# By provider
sum(rate(infra_operator_provider_credential_rotations_total[24h])) by (provider_name)

# Total rotations in last 30 days
sum(increase(infra_operator_provider_credential_rotations_total[30d])) by (provider_name)
```

---

## Workqueue Metrics

### Queue Depth
```promql
# Current queue depth
infra_operator_workqueue_depth

# By resource type
infra_operator_workqueue_depth by (resource_type)

# Deep queues (potential backlog)
infra_operator_workqueue_depth > 100

# Average queue depth
avg(infra_operator_workqueue_depth)
```

### Queue Throughput
```promql
# Items added to queue per second
sum(rate(infra_operator_workqueue_adds_total[5m]))

# By resource type
sum(rate(infra_operator_workqueue_adds_total[5m])) by (resource_type)
```

---

## SLI/SLO Queries

### Availability SLI
```promql
# 99.9% availability target (error budget)
1 - (sum(rate(infra_operator_reconcile_errors_total[30d])) / sum(rate(infra_operator_reconcile_total[30d])))

# Remaining error budget (percentage)
(1 - (sum(rate(infra_operator_reconcile_errors_total[30d])) / sum(rate(infra_operator_reconcile_total[30d])))) - 0.999
```

### Latency SLI
```promql
# 95% of reconciliations complete in <10s
histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le)) < 10
```

### Throughput SLI
```promql
# Process at least 10 reconciliations per minute
sum(rate(infra_operator_reconcile_total[1m])) > 10
```

---

## Alerting Queries

### Critical Alerts

```promql
# Operator appears down (no reconciliations in 10m)
sum(rate(infra_operator_reconcile_total[5m])) == 0

# Provider not ready
infra_operator_provider_ready == 0

# High error rate (>10%)
(sum(rate(infra_operator_reconcile_errors_total[5m])) / sum(rate(infra_operator_reconcile_total[5m]))) > 0.1

# Critical drift detected
sum(increase(infra_operator_drift_detected_total{severity="critical"}[10m])) > 0
```

### Warning Alerts

```promql
# Slow reconciliation (p95 >30s)
histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le)) > 30

# AWS API errors
sum(rate(infra_operator_aws_api_errors_total[5m])) by (service, operation) > 1

# Throttling detected
sum(rate(infra_operator_aws_api_throttles_total[5m])) > 0.1

# Resources not ready for extended period
sum(infra_operator_resources_total{status="notready"}) > 5

# Slow finalizers
histogram_quantile(0.95, sum(rate(infra_operator_finalizer_duration_seconds_bucket[5m])) by (le)) > 60
```

---

## Dashboard Queries

### Overview Panel
```promql
# Total resources
sum(infra_operator_resources_total)

# Ready percentage
(sum(infra_operator_resources_total{status="ready"}) / sum(infra_operator_resources_total)) * 100

# Reconciliation rate
sum(rate(infra_operator_reconcile_total[5m]))

# Error rate
(sum(rate(infra_operator_reconcile_errors_total[5m])) / sum(rate(infra_operator_reconcile_total[5m]))) * 100
```

### Performance Panel
```promql
# p50, p95, p99 latencies
histogram_quantile(0.50, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le))
histogram_quantile(0.99, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le))
```

### AWS Cost Estimation
```promql
# Total AWS API calls (for cost estimation)
sum(increase(infra_operator_aws_api_calls_total[24h]))

# By service (to identify expensive services)
sum(increase(infra_operator_aws_api_calls_total[24h])) by (service)
```

---

## Recording Rules

Optimize performance by pre-computing common queries:

```yaml
groups:
  - name: infra_operator_recording_rules
    interval: 30s
    rules:
    # Reconciliation rate per resource type
    - record: infra_operator:reconcile_total:rate5m
      expr: sum(rate(infra_operator_reconcile_total[5m])) by (resource_type)

    # Error rate percentage
    - record: infra_operator:error_rate:percent
      expr: (sum(rate(infra_operator_reconcile_errors_total[5m])) / sum(rate(infra_operator_reconcile_total[5m]))) * 100

    # p95 reconciliation duration
    - record: infra_operator:reconcile_duration:p95
      expr: histogram_quantile(0.95, sum(rate(infra_operator_reconcile_duration_seconds_bucket[5m])) by (le, resource_type))

    # AWS API call rate
    - record: infra_operator:aws_api_calls:rate5m
      expr: sum(rate(infra_operator_aws_api_calls_total[5m])) by (service)

    # Resources ready percentage
    - record: infra_operator:resources_ready:percent
      expr: (sum(infra_operator_resources_total{status="ready"}) / sum(infra_operator_resources_total)) * 100
```

Use recording rules in queries:
```promql
# Use pre-computed rate instead of calculating each time
infra_operator:reconcile_total:rate5m

# Compare to threshold
infra_operator:error_rate:percent > 5
```

---

## Tips

1. **Use `rate()` for counters**: Always use `rate()` or `increase()` with counter metrics
2. **Use appropriate time windows**: `[5m]` for real-time, `[1h]` for trends, `[24h]` for daily patterns
3. **Use `by` clause for grouping**: Break down metrics by labels for better insights
4. **Use `topk()` for top-N queries**: Find worst offenders quickly
5. **Use histogram_quantile for percentiles**: p50, p95, p99 are more useful than averages
6. **Create recording rules**: Pre-compute frequently used queries for better performance
7. **Test queries in Prometheus UI**: Use the Graph tab to visualize before adding to dashboards
