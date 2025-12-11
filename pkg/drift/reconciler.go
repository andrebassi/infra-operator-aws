// Package drift implementa detecção e reconciliação de drift de infraestrutura.
//
// Detecta diferenças entre estado desejado e real, permitindo auto-healing.
package drift

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	inframetrics "infra-operator/pkg/metrics"
)

// Reconciler defines the interface for reconciling detected drifts.
type Reconciler interface {
	// ReconcileDrift takes action on detected drifts based on configuration
	ReconcileDrift(ctx context.Context, result *Result, resource runtime.Object) error

	// GetAction determines what action to take for a given drift
	GetAction(drift DriftItem) ReconciliationAction
}

// reconciler is the default implementation of Reconciler.
type reconciler struct {
	config        *Config
	eventRecorder record.EventRecorder
	healFunc      HealFunction
}

// HealFunction is a callback function that implements the actual healing logic
// for a specific resource type. It receives the drift items and should update
// the AWS resource to match the desired state.
type HealFunction func(ctx context.Context, drifts []DriftItem, resource runtime.Object) error

// ReconcilerOption allows customizing the reconciler behavior.
type ReconcilerOption func(*reconciler)

// WithEventRecorder sets the Kubernetes event recorder for recording drift events.
func WithEventRecorder(recorder record.EventRecorder) ReconcilerOption {
	return func(r *reconciler) {
		r.eventRecorder = recorder
	}
}

// WithHealFunction sets the healing function that performs actual AWS updates.
func WithHealFunction(healFunc HealFunction) ReconcilerOption {
	return func(r *reconciler) {
		r.healFunc = healFunc
	}
}

// NewReconciler creates a new drift reconciler with the given configuration and options.
func NewReconciler(config *Config, opts ...ReconcilerOption) Reconciler {
	if config == nil {
		config = DefaultConfig()
	}

	r := &reconciler{
		config:        config,
		eventRecorder: nil,
		healFunc:      nil,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReconcileDrift implements the Reconciler interface.
func (r *reconciler) ReconcileDrift(ctx context.Context, result *Result, resource runtime.Object) error {
	// Iniciar recording de métricas de drift detection
	metricsRecorder := inframetrics.NewDriftMetricsRecorder(result.ResourceType)
	defer metricsRecorder.RecordDetectionDuration()

	logger := log.FromContext(ctx)

	// No drift detected, nothing to do
	if !result.HasDrift {
		logger.V(1).Info("No drift detected", "resourceType", result.ResourceType, "resourceID", result.ResourceID)
		return nil
	}

	logger.Info("Drift detected",
		"resourceType", result.ResourceType,
		"resourceID", result.ResourceID,
		"driftCount", len(result.Drifts),
		"high", result.HighSeverityCount(),
		"medium", result.MediumSeverityCount(),
		"low", result.LowSeverityCount(),
	)

	// Categorize drifts by action
	driftsToHeal := []DriftItem{}
	driftsToAlert := []DriftItem{}

	for _, drift := range result.Drifts {
		action := r.GetAction(drift)

		// Record drift detected metric with appropriate severity
		severity := convertSeverityToMetricLabel(drift.Severity)
		metricsRecorder.RecordDriftDetected(severity)

		switch action {
		case ActionAutoHeal:
			driftsToHeal = append(driftsToHeal, drift)
			logger.Info("Drift will be auto-healed",
				"field", drift.Field,
				"severity", drift.Severity,
				"desired", drift.Desired,
				"actual", drift.Actual,
			)

		case ActionAlertOnly:
			driftsToAlert = append(driftsToAlert, drift)
			logger.Info("Drift detected (alert-only)",
				"field", drift.Field,
				"severity", drift.Severity,
				"desired", drift.Desired,
				"actual", drift.Actual,
			)

		case ActionIgnore:
			logger.V(1).Info("Drift ignored",
				"field", drift.Field,
			)
		}
	}

	// Record Kubernetes events
	if r.eventRecorder != nil {
		r.recordDriftEvents(resource, result, driftsToHeal, driftsToAlert)
	}

	// Perform auto-healing if configured and we have drifts to heal
	if len(driftsToHeal) > 0 && r.config.DefaultAction == ActionAutoHeal {
		if r.healFunc == nil {
			logger.Error(nil, "Auto-heal configured but no heal function provided",
				"resourceType", result.ResourceType,
				"resourceID", result.ResourceID,
			)
			metricsRecorder.RecordDriftHealingFailed("no_heal_function")
			return fmt.Errorf("auto-heal configured but no heal function provided")
		}

		logger.Info("Starting auto-heal",
			"resourceType", result.ResourceType,
			"resourceID", result.ResourceID,
			"driftsToHeal", len(driftsToHeal),
		)

		if err := r.healFunc(ctx, driftsToHeal, resource); err != nil {
			logger.Error(err, "Failed to auto-heal drift",
				"resourceType", result.ResourceType,
				"resourceID", result.ResourceID,
			)
			metricsRecorder.RecordDriftHealingFailed("heal_function_error")
			return fmt.Errorf("failed to auto-heal drift: %w", err)
		}

		logger.Info("Auto-heal completed successfully",
			"resourceType", result.ResourceType,
			"resourceID", result.ResourceID,
			"driftsHealed", len(driftsToHeal),
		)

		// Record successful healing in metrics
		metricsRecorder.RecordDriftHealed()

		// Record healing event
		if r.eventRecorder != nil {
			r.eventRecorder.Eventf(resource, "Normal", "DriftHealed",
				"Auto-healed %d drift(s) for %s %s",
				len(driftsToHeal), result.ResourceType, result.ResourceID)
		}
	}

	return nil
}

// convertSeverityToMetricLabel converts drift severity to metric label format.
func convertSeverityToMetricLabel(severity Severity) string {
	switch severity {
	case SeverityHigh:
		return inframetrics.DriftSeverityCritical
	case SeverityMedium:
		return inframetrics.DriftSeverityWarning
	case SeverityLow:
		return inframetrics.DriftSeverityInfo
	default:
		return inframetrics.DriftSeverityInfo
	}
}

// GetAction implements the Reconciler interface.
func (r *reconciler) GetAction(drift DriftItem) ReconciliationAction {
	// Check if field should be ignored
	if r.config.ShouldIgnoreField(drift.Field) {
		return ActionIgnore
	}

	// Check severity threshold
	if !r.shouldIncludeBySeverity(drift.Severity) {
		return ActionIgnore
	}

	// Return configured default action
	return r.config.DefaultAction
}

// recordDriftEvents records Kubernetes events for detected drifts.
func (r *reconciler) recordDriftEvents(resource runtime.Object, result *Result, driftsToHeal, driftsToAlert []DriftItem) {
	// Record warning event for detected drift
	if len(result.Drifts) > 0 {
		r.eventRecorder.Eventf(resource, "Warning", "DriftDetected",
			"Detected %d drift(s) for %s %s (high: %d, medium: %d, low: %d)",
			len(result.Drifts), result.ResourceType, result.ResourceID,
			result.HighSeverityCount(), result.MediumSeverityCount(), result.LowSeverityCount())
	}

	// Record detailed events for high severity drifts
	for _, drift := range result.Drifts {
		if drift.Severity == SeverityHigh {
			r.eventRecorder.Eventf(resource, "Warning", "HighSeverityDrift",
				"High severity drift in %s: desired=%v, actual=%v",
				drift.Field, drift.Desired, drift.Actual)
		}
	}

	// Record alert-only events
	if len(driftsToAlert) > 0 {
		r.eventRecorder.Eventf(resource, "Normal", "DriftAlertOnly",
			"Detected %d drift(s) in alert-only mode (no auto-heal)",
			len(driftsToAlert))
	}
}

// shouldIncludeBySeverity checks if a drift should be included based on severity threshold.
func (r *reconciler) shouldIncludeBySeverity(severity Severity) bool {
	severityOrder := map[Severity]int{
		SeverityLow:    1,
		SeverityMedium: 2,
		SeverityHigh:   3,
	}

	threshold := severityOrder[r.config.SeverityThreshold]
	current := severityOrder[severity]

	return current >= threshold
}

// DriftDetail is a simplified representation of drift for status updates.
type DriftDetail struct {
	// Field is the drifted field path
	Field string `json:"field"`

	// Expected is the string representation of the desired value
	Expected string `json:"expected"`

	// Actual is the string representation of the actual value
	Actual string `json:"actual"`

	// Severity is the drift severity level
	Severity string `json:"severity"`

	// DetectedAt is when this drift was first detected
	DetectedAt time.Time `json:"detectedAt"`
}

// ToDriftDetails converts DriftItems to DriftDetails for status updates.
func ToDriftDetails(drifts []DriftItem) []DriftDetail {
	details := make([]DriftDetail, len(drifts))
	for i, d := range drifts {
		details[i] = DriftDetail{
			Field:      d.Field,
			Expected:   fmt.Sprintf("%v", d.Desired),
			Actual:     fmt.Sprintf("%v", d.Actual),
			Severity:   string(d.Severity),
			DetectedAt: d.DetectedAt,
		}
	}
	return details
}

// HealerRegistry provides a registry of heal functions for different resource types.
type HealerRegistry struct {
	healers map[string]HealFunction
}

// NewHealerRegistry creates a new healer registry.
func NewHealerRegistry() *HealerRegistry {
	return &HealerRegistry{
		healers: make(map[string]HealFunction),
	}
}

// Register registers a heal function for a specific resource type.
func (hr *HealerRegistry) Register(resourceType string, healFunc HealFunction) {
	hr.healers[resourceType] = healFunc
}

// Get retrieves a heal function for a specific resource type.
func (hr *HealerRegistry) Get(resourceType string) (HealFunction, bool) {
	healFunc, ok := hr.healers[resourceType]
	return healFunc, ok
}

// CreateReconcilerForResource creates a reconciler for a specific resource type.
func (hr *HealerRegistry) CreateReconcilerForResource(
	resourceType string,
	config *Config,
	recorder record.EventRecorder,
) (Reconciler, error) {
	healFunc, ok := hr.Get(resourceType)
	if !ok {
		// Create a no-op reconciler for alert-only mode
		return NewReconciler(config, WithEventRecorder(recorder)), nil
	}

	return NewReconciler(config,
		WithEventRecorder(recorder),
		WithHealFunction(healFunc),
	), nil
}
