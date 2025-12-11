// Package drift implementa detecção e reconciliação de drift de infraestrutura.
//
// Detecta diferenças entre estado desejado e real, permitindo auto-healing.
package drift

import (
	"context"
	"testing"
	"time"
)

// TestDetector_NoDrift verifies that no drift is detected when states match.
func TestDetector_NoDrift(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	type Resource struct {
		Name string
		Tags map[string]string
		Size int
	}

	desired := Resource{
		Name: "test-resource",
		Tags: map[string]string{"env": "prod"},
		Size: 100,
	}

	actual := Resource{
		Name: "test-resource",
		Tags: map[string]string{"env": "prod"},
		Size: 100,
	}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.HasDrift {
		t.Errorf("Expected no drift, but got drift detected")
	}

	if len(result.Drifts) != 0 {
		t.Errorf("Expected 0 drifts, got %d", len(result.Drifts))
	}
}

// TestDetector_SimpleDrift verifies that simple value drifts are detected.
func TestDetector_SimpleDrift(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	type Resource struct {
		Name string
		Size int
	}

	desired := Resource{Name: "test", Size: 100}
	actual := Resource{Name: "test", Size: 200}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.HasDrift {
		t.Errorf("Expected drift to be detected")
	}

	if len(result.Drifts) == 0 {
		t.Fatalf("Expected at least one drift item")
	}

	// Verify the drift is in the Size field
	found := false
	for _, d := range result.Drifts {
		if d.Field == "Size" {
			found = true
			if d.Desired != 100 {
				t.Errorf("Expected desired value 100, got %v", d.Desired)
			}
			if d.Actual != 200 {
				t.Errorf("Expected actual value 200, got %v", d.Actual)
			}
		}
	}

	if !found {
		t.Errorf("Expected drift in Size field, but not found")
	}
}

// TestDetector_MapDrift verifies that map drifts are detected.
func TestDetector_MapDrift(t *testing.T) {
	config := DefaultConfig()
	config.IgnoreFields = []string{} // Don't ignore any fields for this test
	detector := NewDetector(config)

	type Resource struct {
		Tags map[string]string
	}

	desired := Resource{
		Tags: map[string]string{
			"env":  "prod",
			"team": "platform",
		},
	}

	actual := Resource{
		Tags: map[string]string{
			"env":  "dev", // Different value
			"team": "platform",
			"cost": "high", // Extra key
		},
	}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.HasDrift {
		t.Errorf("Expected drift to be detected")
	}

	// Should detect:
	// 1. Tags.env value differs
	// 2. Tags.cost exists in actual but not desired
	if len(result.Drifts) < 2 {
		t.Errorf("Expected at least 2 drifts, got %d", len(result.Drifts))
	}
}

// TestDetector_IgnoreFields verifies that configured fields are ignored.
func TestDetector_IgnoreFields(t *testing.T) {
	config := &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{"Tags.*", "LastModified"},
		SeverityThreshold: SeverityLow,
	}
	detector := NewDetector(config)

	type Resource struct {
		Name         string
		Tags         map[string]string
		LastModified string
	}

	desired := Resource{
		Name:         "test",
		Tags:         map[string]string{"env": "prod"},
		LastModified: "2024-01-01",
	}

	actual := Resource{
		Name:         "test",
		Tags:         map[string]string{"env": "dev"}, // Different, but should be ignored
		LastModified: "2024-01-02",                    // Different, but should be ignored
	}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should not detect drift because all different fields are ignored
	if result.HasDrift {
		t.Errorf("Expected no drift (all fields ignored), but got %d drifts: %v",
			len(result.Drifts), result.Drifts)
	}
}

// TestDetector_SeverityThreshold verifies that severity filtering works.
func TestDetector_SeverityThreshold(t *testing.T) {
	// Configure to only report high severity drifts
	config := &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{},
		SeverityThreshold: SeverityHigh,
	}
	detector := NewDetector(config)

	type Resource struct {
		Tags        map[string]string // Low severity (contains "tag")
		SecurityGrp string            // High severity (contains "security")
	}

	desired := Resource{
		Tags:        map[string]string{"env": "prod"},
		SecurityGrp: "sg-123",
	}

	actual := Resource{
		Tags:        map[string]string{"env": "dev"}, // Low severity drift
		SecurityGrp: "sg-456",                        // High severity drift
	}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only detect high severity drift (SecurityGrp), not Tags
	if !result.HasDrift {
		t.Errorf("Expected high severity drift to be detected")
	}

	if result.HighSeverityCount() == 0 {
		t.Errorf("Expected at least one high severity drift")
	}

	// Tags drift should be filtered out due to severity threshold
	for _, d := range result.Drifts {
		if d.Severity != SeverityHigh {
			t.Errorf("Expected only high severity drifts, got %s: %v", d.Severity, d)
		}
	}
}

// TestDetector_NilValues verifies handling of nil values.
func TestDetector_NilValues(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	type Resource struct {
		Name string
		Ptr  *string
	}

	str1 := "value1"
	desired := Resource{Name: "test", Ptr: &str1}
	actual := Resource{Name: "test", Ptr: nil}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.HasDrift {
		t.Errorf("Expected drift to be detected (nil pointer)")
	}
}

// TestDetector_SliceDrift verifies that slice drifts are detected.
func TestDetector_SliceDrift(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	type Resource struct {
		Items []string
	}

	desired := Resource{Items: []string{"a", "b", "c"}}
	actual := Resource{Items: []string{"a", "b", "d"}} // Last element differs

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.HasDrift {
		t.Errorf("Expected drift to be detected in slice")
	}
}

// TestDetector_DisabledConfig verifies that disabled detector returns no drift.
func TestDetector_DisabledConfig(t *testing.T) {
	config := &Config{
		Enabled:           false,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{},
		SeverityThreshold: SeverityLow,
	}
	detector := NewDetector(config)

	type Resource struct {
		Name string
	}

	desired := Resource{Name: "test1"}
	actual := Resource{Name: "test2"}

	result, err := detector.DetectDrift(context.Background(), desired, actual, "Resource", "res-123")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should not detect drift when disabled
	if result.HasDrift {
		t.Errorf("Expected no drift when detector is disabled")
	}
}

// TestDetectFieldDrift verifies single field drift detection.
func TestDetectFieldDrift(t *testing.T) {
	config := DefaultConfig()
	detector := NewDetector(config)

	// Test with different values
	drift := detector.DetectFieldDrift("TestField", "desired", "actual", SeverityMedium)
	if drift == nil {
		t.Errorf("Expected drift to be detected")
	} else {
		if drift.Field != "TestField" {
			t.Errorf("Expected field 'TestField', got '%s'", drift.Field)
		}
		if drift.Severity != SeverityMedium {
			t.Errorf("Expected severity Medium, got %s", drift.Severity)
		}
	}

	// Test with same values
	noDrift := detector.DetectFieldDrift("TestField", "same", "same", SeverityMedium)
	if noDrift != nil {
		t.Errorf("Expected no drift for identical values")
	}

	// Test with ignored field
	config.IgnoreFields = []string{"Ignored*"}
	detector = NewDetector(config)
	ignoredDrift := detector.DetectFieldDrift("IgnoredField", "val1", "val2", SeverityMedium)
	if ignoredDrift != nil {
		t.Errorf("Expected no drift for ignored field")
	}
}

// TestResult_SeverityCounts verifies severity count methods.
func TestResult_SeverityCounts(t *testing.T) {
	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{Field: "f1", Severity: SeverityHigh},
			{Field: "f2", Severity: SeverityHigh},
			{Field: "f3", Severity: SeverityMedium},
			{Field: "f4", Severity: SeverityLow},
			{Field: "f5", Severity: SeverityLow},
			{Field: "f6", Severity: SeverityLow},
		},
		CheckedAt:    time.Now(),
		ResourceType: "Test",
		ResourceID:   "test-123",
	}

	if result.HighSeverityCount() != 2 {
		t.Errorf("Expected 2 high severity drifts, got %d", result.HighSeverityCount())
	}

	if result.MediumSeverityCount() != 1 {
		t.Errorf("Expected 1 medium severity drift, got %d", result.MediumSeverityCount())
	}

	if result.LowSeverityCount() != 3 {
		t.Errorf("Expected 3 low severity drifts, got %d", result.LowSeverityCount())
	}
}

// TestDriftItem_String verifies string representation.
func TestDriftItem_String(t *testing.T) {
	drift := DriftItem{
		Field:      "TestField",
		Desired:    "expected",
		Actual:     "current",
		Severity:   SeverityHigh,
		Message:    "Test message",
		DetectedAt: time.Now(),
	}

	str := drift.String()
	if str == "" {
		t.Errorf("Expected non-empty string representation")
	}

	// Should contain key information
	if !containsSubstring(str, "TestField") {
		t.Errorf("Expected string to contain field name")
	}
	if !containsSubstring(str, "high") {
		t.Errorf("Expected string to contain severity")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
