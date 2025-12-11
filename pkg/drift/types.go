// Package drift provides drift detection and reconciliation capabilities for AWS resources.
//
// Drift occurs when the actual state of AWS resources differs from the desired state
// defined in Kubernetes Custom Resources, even when the CR hasn't changed. This can happen
// when resources are modified directly through AWS Console, CLI, or other tools.
package drift

import (
	"fmt"
	"time"
)

// Severity represents the severity level of a detected drift.
type Severity string

const (
	// SeverityHigh indicates critical drift that affects functionality or security
	SeverityHigh Severity = "high"

	// SeverityMedium indicates important drift that should be addressed
	SeverityMedium Severity = "medium"

	// SeverityLow indicates minor drift that has minimal impact
	SeverityLow Severity = "low"
)

// DriftItem represents a single detected drift between desired and actual state.
type DriftItem struct {
	// Field is the name of the field that has drifted (e.g., "tags.Environment")
	Field string

	// Desired is the expected value from the Kubernetes CR
	Desired interface{}

	// Actual is the current value in AWS
	Actual interface{}

	// Severity indicates the impact level of this drift
	Severity Severity

	// Message provides additional context about the drift
	Message string

	// DetectedAt is the timestamp when this drift was first detected
	DetectedAt time.Time
}

// String returns a human-readable representation of the drift item.
func (d *DriftItem) String() string {
	return fmt.Sprintf("[%s] %s: desired=%v, actual=%v - %s",
		d.Severity, d.Field, d.Desired, d.Actual, d.Message)
}

// ReconciliationAction defines what action to take when drift is detected.
type ReconciliationAction string

const (
	// ActionAutoHeal automatically corrects the drift by updating AWS to match the CR
	ActionAutoHeal ReconciliationAction = "auto-heal"

	// ActionAlertOnly only records the drift in status and events, no automatic correction
	ActionAlertOnly ReconciliationAction = "alert-only"

	// ActionIgnore ignores the drift completely (used for allow-listed fields)
	ActionIgnore ReconciliationAction = "ignore"
)

// Config contains configuration for drift detection and reconciliation.
type Config struct {
	// Enabled determines if drift detection is active
	Enabled bool

	// CheckInterval is the duration between drift checks
	CheckInterval time.Duration

	// DefaultAction is the action to take when drift is detected
	DefaultAction ReconciliationAction

	// IgnoreFields is a list of field paths that are allowed to drift
	// Examples: "tags.aws:*", "lastModified", "status.*"
	IgnoreFields []string

	// SeverityThreshold - only drifts at or above this severity trigger reconciliation
	SeverityThreshold Severity
}

// DefaultConfig returns a sensible default configuration for drift detection.
func DefaultConfig() *Config {
	return &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{"lastModified", "status.*", "tags.aws:*"},
		SeverityThreshold: SeverityMedium,
	}
}

// ShouldIgnoreField checks if a field should be ignored based on ignore patterns.
func (c *Config) ShouldIgnoreField(field string) bool {
	if c.IgnoreFields == nil {
		return false
	}

	for _, pattern := range c.IgnoreFields {
		if matchPattern(field, pattern) {
			return true
		}
	}
	return false
}

// matchPattern performs simple wildcard matching (* and ?).
func matchPattern(s, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == s {
		return true
	}

	// Simple prefix matching for patterns like "tags.*" or "aws:*"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// Result contains the results of a drift detection run.
type Result struct {
	// HasDrift indicates if any drift was detected
	HasDrift bool

	// Drifts is the list of detected drift items
	Drifts []DriftItem

	// CheckedAt is when the drift check was performed
	CheckedAt time.Time

	// ResourceType identifies the type of resource checked (e.g., "VPC", "ElasticIP")
	ResourceType string

	// ResourceID is the AWS resource identifier
	ResourceID string
}

// HighSeverityCount returns the number of high severity drifts.
func (r *Result) HighSeverityCount() int {
	count := 0
	for _, d := range r.Drifts {
		if d.Severity == SeverityHigh {
			count++
		}
	}
	return count
}

// MediumSeverityCount returns the number of medium severity drifts.
func (r *Result) MediumSeverityCount() int {
	count := 0
	for _, d := range r.Drifts {
		if d.Severity == SeverityMedium {
			count++
		}
	}
	return count
}

// LowSeverityCount returns the number of low severity drifts.
func (r *Result) LowSeverityCount() int {
	count := 0
	for _, d := range r.Drifts {
		if d.Severity == SeverityLow {
			count++
		}
	}
	return count
}

// String returns a summary of the drift detection result.
func (r *Result) String() string {
	if !r.HasDrift {
		return fmt.Sprintf("No drift detected for %s %s", r.ResourceType, r.ResourceID)
	}
	return fmt.Sprintf("Detected %d drift(s) for %s %s (high: %d, medium: %d, low: %d)",
		len(r.Drifts), r.ResourceType, r.ResourceID,
		r.HighSeverityCount(), r.MediumSeverityCount(), r.LowSeverityCount())
}
