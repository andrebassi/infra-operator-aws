// Package drift implementa detecção e reconciliação de drift de infraestrutura.
//
// Detecta diferenças entre estado desejado e real, permitindo auto-healing.
package drift

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// Detector defines the interface for detecting drift between desired and actual state.
type Detector interface {
	// DetectDrift compares desired state (from CR) with actual state (from AWS)
	// and returns detected drifts. The resource parameter identifies the resource being checked.
	DetectDrift(ctx context.Context, desired, actual interface{}, resourceType, resourceID string) (*Result, error)

	// DetectFieldDrift checks a specific field for drift
	DetectFieldDrift(fieldPath string, desired, actual interface{}, severity Severity) *DriftItem
}

// detector is the default implementation of Detector.
type detector struct {
	config *Config
}

// NewDetector creates a new drift detector with the given configuration.
func NewDetector(config *Config) Detector {
	if config == nil {
		config = DefaultConfig()
	}
	return &detector{
		config: config,
	}
}

// DetectDrift implements the Detector interface.
func (d *detector) DetectDrift(ctx context.Context, desired, actual interface{}, resourceType, resourceID string) (*Result, error) {
	result := &Result{
		HasDrift:     false,
		Drifts:       []DriftItem{},
		CheckedAt:    time.Now(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	// Skip drift detection if disabled
	if !d.config.Enabled {
		return result, nil
	}

	// Detect drifts using reflection
	drifts, err := d.detectDriftRecursive("", desired, actual)
	if err != nil {
		return nil, fmt.Errorf("failed to detect drift: %w", err)
	}

	// Filter out ignored fields and apply severity threshold
	filteredDrifts := []DriftItem{}
	for _, drift := range drifts {
		if d.config.ShouldIgnoreField(drift.Field) {
			continue
		}
		if d.shouldIncludeBySeverity(drift.Severity) {
			filteredDrifts = append(filteredDrifts, drift)
		}
	}

	result.Drifts = filteredDrifts
	result.HasDrift = len(filteredDrifts) > 0

	return result, nil
}

// DetectFieldDrift implements the Detector interface for single field checking.
func (d *detector) DetectFieldDrift(fieldPath string, desired, actual interface{}, severity Severity) *DriftItem {
	if d.config.ShouldIgnoreField(fieldPath) {
		return nil
	}

	if !d.valuesEqual(desired, actual) {
		return &DriftItem{
			Field:      fieldPath,
			Desired:    desired,
			Actual:     actual,
			Severity:   severity,
			Message:    fmt.Sprintf("Field %s has drifted", fieldPath),
			DetectedAt: time.Now(),
		}
	}

	return nil
}

// detectDriftRecursive recursively compares two values and detects drifts.
func (d *detector) detectDriftRecursive(path string, desired, actual interface{}) ([]DriftItem, error) {
	drifts := []DriftItem{}

	// Handle nil cases
	if desired == nil && actual == nil {
		return drifts, nil
	}
	if desired == nil || actual == nil {
		drifts = append(drifts, DriftItem{
			Field:      path,
			Desired:    desired,
			Actual:     actual,
			Severity:   SeverityHigh,
			Message:    "Value is nil in one state but not the other",
			DetectedAt: time.Now(),
		})
		return drifts, nil
	}

	desiredValue := reflect.ValueOf(desired)
	actualValue := reflect.ValueOf(actual)

	// Types must match
	if desiredValue.Type() != actualValue.Type() {
		drifts = append(drifts, DriftItem{
			Field:      path,
			Desired:    desired,
			Actual:     actual,
			Severity:   SeverityHigh,
			Message:    fmt.Sprintf("Type mismatch: %v vs %v", desiredValue.Type(), actualValue.Type()),
			DetectedAt: time.Now(),
		})
		return drifts, nil
	}

	switch desiredValue.Kind() {
	case reflect.Struct:
		// Compare struct fields
		for i := 0; i < desiredValue.NumField(); i++ {
			fieldName := desiredValue.Type().Field(i).Name
			fieldPath := d.buildFieldPath(path, fieldName)

			desiredField := desiredValue.Field(i)
			actualField := actualValue.Field(i)

			// Skip unexported fields
			if !desiredField.CanInterface() {
				continue
			}

			fieldDrifts, err := d.detectDriftRecursive(
				fieldPath,
				desiredField.Interface(),
				actualField.Interface(),
			)
			if err != nil {
				return nil, err
			}
			drifts = append(drifts, fieldDrifts...)
		}

	case reflect.Map:
		// Compare map entries
		for _, key := range desiredValue.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			fieldPath := d.buildFieldPath(path, keyStr)

			desiredMapValue := desiredValue.MapIndex(key)
			actualMapValue := actualValue.MapIndex(key)

			if !actualMapValue.IsValid() {
				// Key exists in desired but not in actual
				drifts = append(drifts, DriftItem{
					Field:      fieldPath,
					Desired:    desiredMapValue.Interface(),
					Actual:     nil,
					Severity:   d.inferSeverity(path),
					Message:    "Key missing in actual state",
					DetectedAt: time.Now(),
				})
				continue
			}

			if !d.valuesEqual(desiredMapValue.Interface(), actualMapValue.Interface()) {
				drifts = append(drifts, DriftItem{
					Field:      fieldPath,
					Desired:    desiredMapValue.Interface(),
					Actual:     actualMapValue.Interface(),
					Severity:   d.inferSeverity(path),
					Message:    "Map value differs",
					DetectedAt: time.Now(),
				})
			}
		}

		// Check for keys in actual but not in desired
		for _, key := range actualValue.MapKeys() {
			if !desiredValue.MapIndex(key).IsValid() {
				keyStr := fmt.Sprintf("%v", key.Interface())
				fieldPath := d.buildFieldPath(path, keyStr)
				drifts = append(drifts, DriftItem{
					Field:      fieldPath,
					Desired:    nil,
					Actual:     actualValue.MapIndex(key).Interface(),
					Severity:   d.inferSeverity(path),
					Message:    "Extra key in actual state",
					DetectedAt: time.Now(),
				})
			}
		}

	case reflect.Slice, reflect.Array:
		// Compare slices/arrays element by element
		if desiredValue.Len() != actualValue.Len() {
			drifts = append(drifts, DriftItem{
				Field:      path,
				Desired:    fmt.Sprintf("length=%d", desiredValue.Len()),
				Actual:     fmt.Sprintf("length=%d", actualValue.Len()),
				Severity:   d.inferSeverity(path),
				Message:    "Slice/array length differs",
				DetectedAt: time.Now(),
			})
			return drifts, nil
		}

		for i := 0; i < desiredValue.Len(); i++ {
			fieldPath := fmt.Sprintf("%s[%d]", path, i)
			if !d.valuesEqual(desiredValue.Index(i).Interface(), actualValue.Index(i).Interface()) {
				drifts = append(drifts, DriftItem{
					Field:      fieldPath,
					Desired:    desiredValue.Index(i).Interface(),
					Actual:     actualValue.Index(i).Interface(),
					Severity:   d.inferSeverity(path),
					Message:    "Slice/array element differs",
					DetectedAt: time.Now(),
				})
			}
		}

	case reflect.Ptr:
		// Dereference pointers and recurse
		if desiredValue.IsNil() && actualValue.IsNil() {
			return drifts, nil
		}
		if desiredValue.IsNil() || actualValue.IsNil() {
			drifts = append(drifts, DriftItem{
				Field:      path,
				Desired:    desired,
				Actual:     actual,
				Severity:   SeverityMedium,
				Message:    "Pointer is nil in one state but not the other",
				DetectedAt: time.Now(),
			})
			return drifts, nil
		}
		return d.detectDriftRecursive(path, desiredValue.Elem().Interface(), actualValue.Elem().Interface())

	default:
		// Compare primitive values
		if !d.valuesEqual(desired, actual) {
			drifts = append(drifts, DriftItem{
				Field:      path,
				Desired:    desired,
				Actual:     actual,
				Severity:   d.inferSeverity(path),
				Message:    "Value differs",
				DetectedAt: time.Now(),
			})
		}
	}

	return drifts, nil
}

// valuesEqual compares two values for equality using deep comparison.
func (d *detector) valuesEqual(v1, v2 interface{}) bool {
	return reflect.DeepEqual(v1, v2)
}

// buildFieldPath constructs a field path from parent and field name.
func (d *detector) buildFieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// inferSeverity infers the severity level based on the field path.
// This is a simple heuristic - in production, you might want to configure this.
func (d *detector) inferSeverity(fieldPath string) Severity {
	// High severity fields (security, networking, critical config)
	highSeverityFields := []string{"securityGroup", "iamRole", "encryption", "public", "cidr"}
	for _, hs := range highSeverityFields {
		if contains(fieldPath, hs) {
			return SeverityHigh
		}
	}

	// Low severity fields (tags, descriptions, metadata)
	lowSeverityFields := []string{"tag", "description", "metadata", "name"}
	for _, ls := range lowSeverityFields {
		if contains(fieldPath, ls) {
			return SeverityLow
		}
	}

	// Default to medium severity
	return SeverityMedium
}

// shouldIncludeBySeverity checks if a drift should be included based on severity threshold.
func (d *detector) shouldIncludeBySeverity(severity Severity) bool {
	severityOrder := map[Severity]int{
		SeverityLow:    1,
		SeverityMedium: 2,
		SeverityHigh:   3,
	}

	threshold := severityOrder[d.config.SeverityThreshold]
	current := severityOrder[severity]

	return current >= threshold
}

// contains is a simple helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 (len(s) > len(substr) && (s[:len(substr)] == substr ||
		                            s[len(s)-len(substr):] == substr ||
		                            containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
