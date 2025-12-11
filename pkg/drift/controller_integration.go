// Package drift implementa detecção e reconciliação de drift de infraestrutura.
//
// Detecta diferenças entre estado desejado e real, permitindo auto-healing.
package drift

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

// ControllerHelper provides utilities for integrating drift detection into controllers.
type ControllerHelper struct {
	detector      Detector
	reconciler    Reconciler
	config        *Config
	eventRecorder record.EventRecorder
}

// NewControllerHelper creates a new controller helper for drift detection.
func NewControllerHelper(
	config *Config,
	eventRecorder record.EventRecorder,
	healFunc HealFunction,
) *ControllerHelper {
	detector := NewDetector(config)
	reconciler := NewReconciler(config,
		WithEventRecorder(eventRecorder),
		WithHealFunction(healFunc),
	)

	return &ControllerHelper{
		detector:      detector,
		reconciler:    reconciler,
		config:        config,
		eventRecorder: eventRecorder,
	}
}

// CheckAndReconcileDrift performs drift detection and reconciliation for a resource.
// Returns true if drift was detected, false otherwise.
func (h *ControllerHelper) CheckAndReconcileDrift(
	ctx context.Context,
	desired, actual interface{},
	resourceType, resourceID string,
	resource runtime.Object,
) (bool, error) {
	logger := log.FromContext(ctx)

	// Skip if drift detection is disabled
	if !h.config.Enabled {
		return false, nil
	}

	// Detect drift
	result, err := h.detector.DetectDrift(ctx, desired, actual, resourceType, resourceID)
	if err != nil {
		logger.Error(err, "Failed to detect drift",
			"resourceType", resourceType,
			"resourceID", resourceID,
		)
		return false, err
	}

	// No drift detected
	if !result.HasDrift {
		logger.V(1).Info("No drift detected",
			"resourceType", resourceType,
			"resourceID", resourceID,
		)
		return false, nil
	}

	// Drift detected - log it
	logger.Info("Drift detected",
		"resourceType", resourceType,
		"resourceID", resourceID,
		"driftCount", len(result.Drifts),
		"high", result.HighSeverityCount(),
		"medium", result.MediumSeverityCount(),
		"low", result.LowSeverityCount(),
	)

	// Reconcile drift (alert and/or auto-heal)
	if err := h.reconciler.ReconcileDrift(ctx, result, resource); err != nil {
		logger.Error(err, "Failed to reconcile drift",
			"resourceType", resourceType,
			"resourceID", resourceID,
		)
		return true, err
	}

	return true, nil
}

// GetDriftDetailsForStatus converts drift items to status-friendly format.
func GetDriftDetailsForStatus(drifts []DriftItem) []infrav1alpha1.DriftDetail {
	if len(drifts) == 0 {
		return nil
	}

	details := make([]infrav1alpha1.DriftDetail, len(drifts))
	for i, d := range drifts {
		details[i] = infrav1alpha1.DriftDetail{
			Field:    d.Field,
			Expected: formatValue(d.Desired),
			Actual:   formatValue(d.Actual),
			Severity: string(d.Severity),
		}
	}
	return details
}

// formatValue formats a value for display in status.
func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	return limitString(v, 100)
}

// limitString converts a value to string and limits its length.
func limitString(v interface{}, maxLen int) string {
	s := ""
	switch val := v.(type) {
	case string:
		s = val
	default:
		s = truncateString(v, maxLen)
	}

	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func truncateString(v interface{}, maxLen int) string {
	// Simple implementation - in production you might want a better formatter
	result := ""
	switch val := v.(type) {
	case string:
		result = val
	case int, int32, int64, uint, uint32, uint64:
		result = anyToString(val)
	case bool:
		if val {
			result = "true"
		} else {
			result = "false"
		}
	default:
		result = anyToString(val)
	}

	if len(result) > maxLen {
		return result[:maxLen]
	}
	return result
}

func anyToString(v interface{}) string {
	// Fallback string conversion
	return limitedSprintf(v, 100)
}

func limitedSprintf(v interface{}, max int) string {
	// Very basic implementation to avoid importing fmt in a loop
	// In production, use fmt.Sprintf with length checking
	result := ""
	switch val := v.(type) {
	case string:
		result = val
	case int:
		result = intToString(val)
	case bool:
		if val {
			result = "true"
		} else {
			result = "false"
		}
	default:
		result = "<complex>"
	}

	if len(result) > max {
		return result[:max]
	}
	return result
}

func intToString(n int) string {
	// Simple int to string without importing strconv
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// ConfigFromProvider extracts drift detection configuration from an AWSProvider.
func ConfigFromProvider(provider *infrav1alpha1.AWSProvider) *Config {
	// Use default config if drift detection is not configured
	if provider.Spec.DriftDetection == nil {
		defaultCfg := DefaultConfig()
		defaultCfg.Enabled = false // Disabled by default if not configured
		return defaultCfg
	}

	driftCfg := provider.Spec.DriftDetection

	// Parse check interval
	checkInterval := 5 * time.Minute
	if driftCfg.CheckInterval != "" {
		if parsed, err := time.ParseDuration(driftCfg.CheckInterval); err == nil {
			checkInterval = parsed
		}
	}

	// Parse severity threshold
	severity := SeverityMedium
	switch driftCfg.SeverityThreshold {
	case "low":
		severity = SeverityLow
	case "high":
		severity = SeverityHigh
	default:
		severity = SeverityMedium
	}

	// Determine default action
	action := ActionAlertOnly
	if driftCfg.AutoHeal {
		action = ActionAutoHeal
	}

	return &Config{
		Enabled:           driftCfg.Enabled,
		CheckInterval:     checkInterval,
		DefaultAction:     action,
		IgnoreFields:      driftCfg.IgnoreFields,
		SeverityThreshold: severity,
	}
}

// ShouldCheckDrift determines if it's time to check for drift based on last check time.
func ShouldCheckDrift(lastCheck *metav1.Time, interval time.Duration) bool {
	if lastCheck == nil {
		return true
	}

	elapsed := time.Since(lastCheck.Time)
	return elapsed >= interval
}
