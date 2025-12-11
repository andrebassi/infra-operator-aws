// Package metrics provides Prometheus metrics for observability of the Infra Operator.
//
// This package exposes comprehensive metrics about:
// - Reconciliation performance and errors
// - Resource counts and status
// - AWS API call performance and errors
// - Drift detection and healing
// - Finalizer execution duration
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ============================================
	// Reconciliation Metrics
	// ============================================

	// ReconcileTotal tracks the total number of reconciliations per resource type and result.
	// Labels: resource_type (VPC, Subnet, etc.), result (success, error, requeue)
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_reconcile_total",
			Help: "Total number of reconciliations per resource type and result",
		},
		[]string{"resource_type", "result"},
	)

	// ReconcileDuration tracks the duration of reconciliation operations in seconds.
	// Labels: resource_type (VPC, Subnet, etc.)
	// Buckets: Default Prometheus buckets (.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10)
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "infra_operator_reconcile_duration_seconds",
			Help:    "Duration of reconciliation operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type"},
	)

	// ReconcileErrors tracks the total number of reconciliation errors.
	// Labels: resource_type (VPC, Subnet, etc.), error_type (get_failed, sync_failed, status_update_failed, etc.)
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors by type",
		},
		[]string{"resource_type", "error_type"},
	)

	// ============================================
	// Resource Metrics
	// ============================================

	// ResourcesTotal tracks the current number of resources by type and status.
	// Labels: resource_type (VPC, Subnet, etc.), status (ready, notready, pending, deleting)
	// Type: Gauge (can increase and decrease)
	ResourcesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "infra_operator_resources_total",
			Help: "Current number of managed resources by type and status",
		},
		[]string{"resource_type", "status"},
	)

	// ResourceCreationTime tracks when resources were created (Unix timestamp).
	// Labels: resource_type, resource_name
	ResourceCreationTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "infra_operator_resource_creation_timestamp_seconds",
			Help: "Unix timestamp when the resource was created",
		},
		[]string{"resource_type", "resource_name"},
	)

	// ============================================
	// AWS API Metrics
	// ============================================

	// AWSAPICallsTotal tracks the total number of AWS API calls.
	// Labels: service (EC2, S3, RDS, etc.), operation (AllocateAddress, CreateBucket, etc.), result (success, error)
	AWSAPICallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_aws_api_calls_total",
			Help: "Total number of AWS API calls by service, operation, and result",
		},
		[]string{"service", "operation", "result"},
	)

	// AWSAPICallDuration tracks the duration of AWS API calls in seconds.
	// Labels: service (EC2, S3, RDS, etc.), operation (AllocateAddress, CreateBucket, etc.)
	// Buckets: Custom buckets optimized for API latency (.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10)
	AWSAPICallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "infra_operator_aws_api_call_duration_seconds",
			Help:    "Duration of AWS API calls in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "operation"},
	)

	// AWSAPIErrors tracks the total number of AWS API errors.
	// Labels: service (EC2, S3, RDS, etc.), operation (AllocateAddress, CreateBucket, etc.), error_code (InvalidParameterValue, ResourceNotFound, etc.)
	AWSAPIErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_aws_api_errors_total",
			Help: "Total number of AWS API errors by service, operation, and error code",
		},
		[]string{"service", "operation", "error_code"},
	)

	// AWSAPIThrottles tracks the number of AWS API throttling events.
	// Labels: service (EC2, S3, RDS, etc.), operation (AllocateAddress, CreateBucket, etc.)
	AWSAPIThrottles = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_aws_api_throttles_total",
			Help: "Total number of AWS API throttling events (rate limit exceeded)",
		},
		[]string{"service", "operation"},
	)

	// ============================================
	// Drift Detection Metrics
	// ============================================

	// DriftDetectedTotal tracks the total number of configuration drifts detected.
	// Labels: resource_type (VPC, Subnet, etc.), severity (critical, warning, info)
	DriftDetectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_drift_detected_total",
			Help: "Total number of configuration drifts detected by severity",
		},
		[]string{"resource_type", "severity"},
	)

	// DriftHealedTotal tracks the total number of drifts that were automatically healed.
	// Labels: resource_type (VPC, Subnet, etc.)
	DriftHealedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_drift_healed_total",
			Help: "Total number of configuration drifts automatically healed",
		},
		[]string{"resource_type"},
	)

	// DriftHealingFailures tracks the number of failed drift healing attempts.
	// Labels: resource_type (VPC, Subnet, etc.), error_type
	DriftHealingFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_drift_healing_failures_total",
			Help: "Total number of failed drift healing attempts",
		},
		[]string{"resource_type", "error_type"},
	)

	// DriftDetectionDuration tracks how long drift detection takes.
	// Labels: resource_type (VPC, Subnet, etc.)
	DriftDetectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "infra_operator_drift_detection_duration_seconds",
			Help:    "Duration of drift detection operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type"},
	)

	// ============================================
	// Finalizer Metrics
	// ============================================

	// FinalizerDuration tracks the duration of finalizer execution (cleanup) in seconds.
	// Labels: resource_type (VPC, Subnet, etc.)
	FinalizerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "infra_operator_finalizer_duration_seconds",
			Help:    "Duration of finalizer execution (resource cleanup) in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type"},
	)

	// FinalizerErrors tracks the total number of finalizer errors.
	// Labels: resource_type (VPC, Subnet, etc.), error_type
	FinalizerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_finalizer_errors_total",
			Help: "Total number of finalizer execution errors",
		},
		[]string{"resource_type", "error_type"},
	)

	// ============================================
	// Provider Metrics
	// ============================================

	// ProviderReady tracks if an AWS Provider is ready (1) or not ready (0).
	// Labels: provider_name, region
	ProviderReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "infra_operator_provider_ready",
			Help: "Indicates if an AWS Provider is ready (1) or not (0)",
		},
		[]string{"provider_name", "region"},
	)

	// ProviderCredentialRotations tracks credential rotation events.
	// Labels: provider_name
	ProviderCredentialRotations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_provider_credential_rotations_total",
			Help: "Total number of AWS credential rotations",
		},
		[]string{"provider_name"},
	)

	// ============================================
	// Queue Metrics (Controller Runtime)
	// ============================================

	// WorkqueueDepth tracks the current depth of work queues.
	// Labels: resource_type
	WorkqueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "infra_operator_workqueue_depth",
			Help: "Current depth of the work queue for each resource type",
		},
		[]string{"resource_type"},
	)

	// WorkqueueAdds tracks items added to work queues.
	// Labels: resource_type
	WorkqueueAdds = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_workqueue_adds_total",
			Help: "Total number of items added to work queue",
		},
		[]string{"resource_type"},
	)

	// ============================================
	// Cache Metrics (Future)
	// ============================================

	// CacheHits tracks successful cache lookups.
	// Labels: resource_type
	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_cache_hits_total",
			Help: "Total number of successful cache lookups",
		},
		[]string{"resource_type"},
	)

	// CacheMisses tracks failed cache lookups.
	// Labels: resource_type
	CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "infra_operator_cache_misses_total",
			Help: "Total number of failed cache lookups",
		},
		[]string{"resource_type"},
	)
)

// init registers all metrics with the controller-runtime metrics registry.
// This function is called automatically when the package is imported.
func init() {
	// Register reconciliation metrics
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileDuration,
		ReconcileErrors,
	)

	// Register resource metrics
	metrics.Registry.MustRegister(
		ResourcesTotal,
		ResourceCreationTime,
	)

	// Register AWS API metrics
	metrics.Registry.MustRegister(
		AWSAPICallsTotal,
		AWSAPICallDuration,
		AWSAPIErrors,
		AWSAPIThrottles,
	)

	// Register drift detection metrics
	metrics.Registry.MustRegister(
		DriftDetectedTotal,
		DriftHealedTotal,
		DriftHealingFailures,
		DriftDetectionDuration,
	)

	// Register finalizer metrics
	metrics.Registry.MustRegister(
		FinalizerDuration,
		FinalizerErrors,
	)

	// Register provider metrics
	metrics.Registry.MustRegister(
		ProviderReady,
		ProviderCredentialRotations,
	)

	// Register queue metrics
	metrics.Registry.MustRegister(
		WorkqueueDepth,
		WorkqueueAdds,
	)

	// Register cache metrics
	metrics.Registry.MustRegister(
		CacheHits,
		CacheMisses,
	)
}

// Common error types for standardized error_type labels
const (
	ErrorTypeGetFailed          = "get_failed"
	ErrorTypeSyncFailed         = "sync_failed"
	ErrorTypeStatusUpdateFailed = "status_update_failed"
	ErrorTypeFinalizerFailed    = "finalizer_failed"
	ErrorTypeProviderFailed     = "provider_failed"
	ErrorTypeValidationFailed   = "validation_failed"
	ErrorTypeAWSAPIFailed       = "aws_api_failed"
	ErrorTypeDriftHealingFailed = "drift_healing_failed"
)

// Common AWS service names for standardized service labels
const (
	ServiceEC2           = "EC2"
	ServiceS3            = "S3"
	ServiceRDS           = "RDS"
	ServiceELB           = "ELB"
	ServiceELBv2         = "ELBv2"
	ServiceIAM           = "IAM"
	ServiceSecretsManager = "SecretsManager"
	ServiceKMS           = "KMS"
	ServiceLambda        = "Lambda"
	ServiceAPIGateway    = "APIGateway"
	ServiceCloudFront    = "CloudFront"
	ServiceRoute53       = "Route53"
	ServiceACM           = "ACM"
	ServiceDynamoDB      = "DynamoDB"
	ServiceSQS           = "SQS"
	ServiceSNS           = "SNS"
	ServiceECR           = "ECR"
	ServiceECS           = "ECS"
	ServiceEKS           = "EKS"
	ServiceElastiCache   = "ElastiCache"
)

// Common drift severity levels
const (
	DriftSeverityCritical = "critical"
	DriftSeverityWarning  = "warning"
	DriftSeverityInfo     = "info"
)

// Common resource status values
const (
	StatusReady    = "ready"
	StatusNotReady = "notready"
	StatusPending  = "pending"
	StatusDeleting = "deleting"
	StatusError    = "error"
)

// Common reconciliation results
const (
	ResultSuccess = "success"
	ResultError   = "error"
	ResultRequeue = "requeue"
)
