// Package metrics provides helpers for recording metrics in a consistent way.
//
// Recorders simplify metric collection by providing a fluent API and ensuring
// consistent labeling across the operator.
package metrics

import (
	"errors"
	"time"

	"github.com/aws/smithy-go"
)

// ============================================
// Reconciliation Metrics Recorder
// ============================================

// ReconcileMetricsRecorder helps record reconciliation metrics consistently.
// Usage:
//   recorder := metrics.NewReconcileMetricsRecorder("VPC")
//   defer func() {
//     if err != nil {
//       recorder.RecordError("sync_failed")
//     } else {
//       recorder.RecordSuccess()
//     }
//   }()
type ReconcileMetricsRecorder struct {
	resourceType string
	startTime    time.Time
}

// NewReconcileMetricsRecorder creates a new reconciliation metrics recorder.
// It automatically starts timing the operation.
func NewReconcileMetricsRecorder(resourceType string) *ReconcileMetricsRecorder {
	WorkqueueAdds.WithLabelValues(resourceType).Inc()
	return &ReconcileMetricsRecorder{
		resourceType: resourceType,
		startTime:    time.Now(),
	}
}

// RecordSuccess records a successful reconciliation.
// It increments the success counter and records the duration.
func (r *ReconcileMetricsRecorder) RecordSuccess() {
	duration := time.Since(r.startTime).Seconds()

	ReconcileTotal.WithLabelValues(r.resourceType, ResultSuccess).Inc()
	ReconcileDuration.WithLabelValues(r.resourceType).Observe(duration)
}

// RecordError records a failed reconciliation.
// It increments the error counter, records the duration, and tracks the error type.
func (r *ReconcileMetricsRecorder) RecordError(errorType string) {
	duration := time.Since(r.startTime).Seconds()

	ReconcileTotal.WithLabelValues(r.resourceType, ResultError).Inc()
	ReconcileDuration.WithLabelValues(r.resourceType).Observe(duration)
	ReconcileErrors.WithLabelValues(r.resourceType, errorType).Inc()
}

// RecordRequeue records a requeued reconciliation (neither success nor error).
// This happens when reconciliation needs to be retried later.
func (r *ReconcileMetricsRecorder) RecordRequeue() {
	ReconcileTotal.WithLabelValues(r.resourceType, ResultRequeue).Inc()
}

// ============================================
// Resource Status Metrics Recorder
// ============================================

// ResourceStatusRecorder helps track resource status changes.
type ResourceStatusRecorder struct {
	resourceType string
	resourceName string
}

// NewResourceStatusRecorder creates a new resource status recorder.
func NewResourceStatusRecorder(resourceType, resourceName string) *ResourceStatusRecorder {
	return &ResourceStatusRecorder{
		resourceType: resourceType,
		resourceName: resourceName,
	}
}

// SetReady marks a resource as ready and updates the gauge.
func (r *ResourceStatusRecorder) SetReady() {
	ResourcesTotal.WithLabelValues(r.resourceType, StatusReady).Inc()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusNotReady).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusPending).Dec()
}

// SetNotReady marks a resource as not ready.
func (r *ResourceStatusRecorder) SetNotReady() {
	ResourcesTotal.WithLabelValues(r.resourceType, StatusNotReady).Inc()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusReady).Dec()
}

// SetPending marks a resource as pending (being created).
func (r *ResourceStatusRecorder) SetPending() {
	ResourcesTotal.WithLabelValues(r.resourceType, StatusPending).Inc()
}

// SetDeleting marks a resource as being deleted.
func (r *ResourceStatusRecorder) SetDeleting() {
	ResourcesTotal.WithLabelValues(r.resourceType, StatusDeleting).Inc()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusReady).Dec()
}

// SetError marks a resource as in error state.
func (r *ResourceStatusRecorder) SetError() {
	ResourcesTotal.WithLabelValues(r.resourceType, StatusError).Inc()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusReady).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusNotReady).Dec()
}

// RecordCreation records when a resource was created.
func (r *ResourceStatusRecorder) RecordCreation() {
	ResourceCreationTime.WithLabelValues(r.resourceType, r.resourceName).SetToCurrentTime()
}

// Remove removes all metrics for this resource (called on deletion).
func (r *ResourceStatusRecorder) Remove() {
	// Decrement from whatever status it was in
	ResourcesTotal.WithLabelValues(r.resourceType, StatusReady).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusNotReady).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusPending).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusDeleting).Dec()
	ResourcesTotal.WithLabelValues(r.resourceType, StatusError).Dec()

	// Remove creation time metric
	ResourceCreationTime.DeleteLabelValues(r.resourceType, r.resourceName)
}

// ============================================
// AWS API Metrics Recorder
// ============================================

// AWSAPIMetricsRecorder helps record AWS API call metrics consistently.
// Usage:
//   recorder := metrics.NewAWSAPIMetricsRecorder("EC2", "AllocateAddress")
//   output, err := r.client.AllocateAddress(ctx, input)
//   if err != nil {
//     recorder.RecordError(err)
//     return err
//   }
//   recorder.RecordSuccess()
type AWSAPIMetricsRecorder struct {
	service   string
	operation string
	startTime time.Time
}

// NewAWSAPIMetricsRecorder creates a new AWS API metrics recorder.
// It automatically starts timing the API call.
//
// Common service names (use constants from metrics.go):
// - ServiceEC2, ServiceS3, ServiceRDS, ServiceELBv2, ServiceIAM, etc.
func NewAWSAPIMetricsRecorder(service, operation string) *AWSAPIMetricsRecorder {
	return &AWSAPIMetricsRecorder{
		service:   service,
		operation: operation,
		startTime: time.Now(),
	}
}

// RecordSuccess records a successful AWS API call.
func (a *AWSAPIMetricsRecorder) RecordSuccess() {
	duration := time.Since(a.startTime).Seconds()

	AWSAPICallsTotal.WithLabelValues(a.service, a.operation, ResultSuccess).Inc()
	AWSAPICallDuration.WithLabelValues(a.service, a.operation).Observe(duration)
}

// RecordError records a failed AWS API call.
// It extracts the AWS error code from the error and records it.
func (a *AWSAPIMetricsRecorder) RecordError(err error) {
	duration := time.Since(a.startTime).Seconds()

	errorCode := extractAWSErrorCode(err)

	AWSAPICallsTotal.WithLabelValues(a.service, a.operation, ResultError).Inc()
	AWSAPICallDuration.WithLabelValues(a.service, a.operation).Observe(duration)
	AWSAPIErrors.WithLabelValues(a.service, a.operation, errorCode).Inc()

	// Check for throttling errors
	if isThrottlingError(errorCode) {
		AWSAPIThrottles.WithLabelValues(a.service, a.operation).Inc()
	}
}

// extractAWSErrorCode extracts the AWS error code from an error.
// Returns "Unknown" if the error is not an AWS error.
func extractAWSErrorCode(err error) string {
	if err == nil {
		return "Unknown"
	}

	// Try to extract AWS SDK v2 error
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}

	// Fallback to error message if not an AWS error
	return "Unknown"
}

// isThrottlingError checks if an error code represents throttling.
func isThrottlingError(errorCode string) bool {
	throttlingCodes := []string{
		"Throttling",
		"ThrottlingException",
		"RequestLimitExceeded",
		"TooManyRequestsException",
		"ProvisionedThroughputExceededException",
		"RequestThrottled",
	}

	for _, code := range throttlingCodes {
		if errorCode == code {
			return true
		}
	}
	return false
}

// ============================================
// Drift Detection Metrics Recorder
// ============================================

// DriftMetricsRecorder helps record drift detection metrics.
type DriftMetricsRecorder struct {
	resourceType string
	startTime    time.Time
}

// NewDriftMetricsRecorder creates a new drift detection metrics recorder.
func NewDriftMetricsRecorder(resourceType string) *DriftMetricsRecorder {
	return &DriftMetricsRecorder{
		resourceType: resourceType,
		startTime:    time.Now(),
	}
}

// RecordDriftDetected records that a drift was detected.
// severity should be one of: DriftSeverityCritical, DriftSeverityWarning, DriftSeverityInfo
func (d *DriftMetricsRecorder) RecordDriftDetected(severity string) {
	DriftDetectedTotal.WithLabelValues(d.resourceType, severity).Inc()
}

// RecordDriftHealed records that a drift was successfully healed.
func (d *DriftMetricsRecorder) RecordDriftHealed() {
	DriftHealedTotal.WithLabelValues(d.resourceType).Inc()
}

// RecordDriftHealingFailed records that drift healing failed.
func (d *DriftMetricsRecorder) RecordDriftHealingFailed(errorType string) {
	DriftHealingFailures.WithLabelValues(d.resourceType, errorType).Inc()
}

// RecordDetectionDuration records how long drift detection took.
func (d *DriftMetricsRecorder) RecordDetectionDuration() {
	duration := time.Since(d.startTime).Seconds()
	DriftDetectionDuration.WithLabelValues(d.resourceType).Observe(duration)
}

// ============================================
// Finalizer Metrics Recorder
// ============================================

// FinalizerMetricsRecorder helps record finalizer execution metrics.
type FinalizerMetricsRecorder struct {
	resourceType string
	startTime    time.Time
}

// NewFinalizerMetricsRecorder creates a new finalizer metrics recorder.
func NewFinalizerMetricsRecorder(resourceType string) *FinalizerMetricsRecorder {
	return &FinalizerMetricsRecorder{
		resourceType: resourceType,
		startTime:    time.Now(),
	}
}

// RecordSuccess records successful finalizer execution.
func (f *FinalizerMetricsRecorder) RecordSuccess() {
	duration := time.Since(f.startTime).Seconds()
	FinalizerDuration.WithLabelValues(f.resourceType).Observe(duration)
}

// RecordError records failed finalizer execution.
func (f *FinalizerMetricsRecorder) RecordError(errorType string) {
	duration := time.Since(f.startTime).Seconds()
	FinalizerDuration.WithLabelValues(f.resourceType).Observe(duration)
	FinalizerErrors.WithLabelValues(f.resourceType, errorType).Inc()
}

// ============================================
// Provider Metrics Recorder
// ============================================

// ProviderMetricsRecorder helps record provider status metrics.
type ProviderMetricsRecorder struct {
	providerName string
	region       string
}

// NewProviderMetricsRecorder creates a new provider metrics recorder.
func NewProviderMetricsRecorder(providerName, region string) *ProviderMetricsRecorder {
	return &ProviderMetricsRecorder{
		providerName: providerName,
		region:       region,
	}
}

// SetReady marks the provider as ready.
func (p *ProviderMetricsRecorder) SetReady() {
	ProviderReady.WithLabelValues(p.providerName, p.region).Set(1)
}

// SetNotReady marks the provider as not ready.
func (p *ProviderMetricsRecorder) SetNotReady() {
	ProviderReady.WithLabelValues(p.providerName, p.region).Set(0)
}

// RecordCredentialRotation records a credential rotation event.
func (p *ProviderMetricsRecorder) RecordCredentialRotation() {
	ProviderCredentialRotations.WithLabelValues(p.providerName).Inc()
}

// ============================================
// Workqueue Metrics Helpers
// ============================================

// RecordWorkqueueDepth records the current depth of a resource's work queue.
func RecordWorkqueueDepth(resourceType string, depth int) {
	WorkqueueDepth.WithLabelValues(resourceType).Set(float64(depth))
}

// ============================================
// Cache Metrics Helpers
// ============================================

// RecordCacheHit records a successful cache lookup.
func RecordCacheHit(resourceType string) {
	CacheHits.WithLabelValues(resourceType).Inc()
}

// RecordCacheMiss records a failed cache lookup.
func RecordCacheMiss(resourceType string) {
	CacheMisses.WithLabelValues(resourceType).Inc()
}
