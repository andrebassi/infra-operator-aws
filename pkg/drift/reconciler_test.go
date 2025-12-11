// Package drift implementa detecção e reconciliação de drift de infraestrutura.
//
// Detecta diferenças entre estado desejado e real, permitindo auto-healing.
package drift

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockEventRecorder implements a simple event recorder for testing.
type mockEventRecorder struct {
	events []string
}

func (m *mockEventRecorder) Event(object interface{}, eventtype, reason, message string) {
	m.events = append(m.events, eventtype+":"+reason+":"+message)
}

func (m *mockEventRecorder) Eventf(object interface{}, eventtype, reason, messageFmt string, args ...interface{}) {
	// Simple implementation without formatting for tests
	m.events = append(m.events, eventtype+":"+reason)
}

func (m *mockEventRecorder) AnnotatedEventf(object interface{}, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.events = append(m.events, eventtype+":"+reason)
}

// TestReconciler_NoDrift verifies behavior when no drift is detected.
func TestReconciler_NoDrift(t *testing.T) {
	config := DefaultConfig()
	recorder := &mockEventRecorder{}
	reconciler := NewReconciler(config, WithEventRecorder(recorder))

	result := &Result{
		HasDrift:     false,
		Drifts:       []DriftItem{},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(recorder.events) != 0 {
		t.Errorf("Expected no events when no drift, got %d events", len(recorder.events))
	}
}

// TestReconciler_AlertOnly verifies alert-only mode.
func TestReconciler_AlertOnly(t *testing.T) {
	config := &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAlertOnly, // Alert only, no auto-heal
		IgnoreFields:      []string{},
		SeverityThreshold: SeverityLow,
	}
	recorder := &mockEventRecorder{}
	reconciler := NewReconciler(config, WithEventRecorder(recorder))

	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{
				Field:      "Tags.Environment",
				Desired:    "prod",
				Actual:     "dev",
				Severity:   SeverityMedium,
				Message:    "Tag mismatch",
				DetectedAt: time.Now(),
			},
		},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, struct{}{})

	if err != nil {
		t.Errorf("Expected no error in alert-only mode, got: %v", err)
	}

	// Should have recorded events
	if len(recorder.events) == 0 {
		t.Errorf("Expected events to be recorded in alert-only mode")
	}

	// Verify drift detected event
	foundDriftEvent := false
	for _, event := range recorder.events {
		if containsSubstr(event, "DriftDetected") {
			foundDriftEvent = true
			break
		}
	}

	if !foundDriftEvent {
		t.Errorf("Expected DriftDetected event to be recorded")
	}
}

// TestReconciler_AutoHeal verifies auto-heal mode.
func TestReconciler_AutoHeal(t *testing.T) {
	config := &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{},
		SeverityThreshold: SeverityLow,
	}
	recorder := &mockEventRecorder{}

	// Track if heal function was called
	healCalled := false
	healFunc := func(ctx context.Context, drifts []DriftItem, resource interface{}) error {
		healCalled = true
		if len(drifts) == 0 {
			t.Errorf("Expected drifts to be passed to heal function")
		}
		return nil
	}

	reconciler := NewReconciler(config,
		WithEventRecorder(recorder),
		WithHealFunction(healFunc),
	)

	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{
				Field:      "CidrBlock",
				Desired:    "10.0.0.0/16",
				Actual:     "10.1.0.0/16",
				Severity:   SeverityHigh,
				Message:    "CIDR block changed",
				DetectedAt: time.Now(),
			},
		},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, struct{}{})

	if err != nil {
		t.Errorf("Expected no error in auto-heal mode, got: %v", err)
	}

	if !healCalled {
		t.Errorf("Expected heal function to be called")
	}

	// Should have recorded healing event
	foundHealEvent := false
	for _, event := range recorder.events {
		if containsSubstr(event, "DriftHealed") {
			foundHealEvent = true
			break
		}
	}

	if !foundHealEvent {
		t.Errorf("Expected DriftHealed event to be recorded, events: %v", recorder.events)
	}
}

// TestReconciler_AutoHealError verifies error handling in heal function.
func TestReconciler_AutoHealError(t *testing.T) {
	config := DefaultConfig()
	config.DefaultAction = ActionAutoHeal
	recorder := &mockEventRecorder{}

	healFunc := func(ctx context.Context, drifts []DriftItem, resource interface{}) error {
		return errors.New("heal failed")
	}

	reconciler := NewReconciler(config,
		WithEventRecorder(recorder),
		WithHealFunction(healFunc),
	)

	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{Field: "test", Desired: "a", Actual: "b", Severity: SeverityMedium},
		},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, struct{}{})

	if err == nil {
		t.Errorf("Expected error when heal function fails")
	}

	if !containsSubstr(err.Error(), "heal") {
		t.Errorf("Expected error message to mention healing, got: %v", err)
	}
}

// TestReconciler_NoHealFunction verifies error when heal function is missing.
func TestReconciler_NoHealFunction(t *testing.T) {
	config := DefaultConfig()
	config.DefaultAction = ActionAutoHeal
	reconciler := NewReconciler(config) // No heal function

	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{Field: "test", Desired: "a", Actual: "b", Severity: SeverityMedium},
		},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, struct{}{})

	if err == nil {
		t.Errorf("Expected error when heal function is not provided")
	}
}

// TestReconciler_SeverityFiltering verifies that low severity drifts can be ignored.
func TestReconciler_SeverityFiltering(t *testing.T) {
	config := &Config{
		Enabled:           true,
		CheckInterval:     5 * time.Minute,
		DefaultAction:     ActionAutoHeal,
		IgnoreFields:      []string{},
		SeverityThreshold: SeverityHigh, // Only high severity
	}

	healCalled := false
	healFunc := func(ctx context.Context, drifts []DriftItem, resource interface{}) error {
		healCalled = true
		// Should only receive high severity drifts
		for _, d := range drifts {
			if d.Severity != SeverityHigh {
				t.Errorf("Expected only high severity drifts, got %s", d.Severity)
			}
		}
		return nil
	}

	reconciler := NewReconciler(config, WithHealFunction(healFunc))

	result := &Result{
		HasDrift: true,
		Drifts: []DriftItem{
			{Field: "tags", Desired: "a", Actual: "b", Severity: SeverityLow},    // Should be ignored
			{Field: "desc", Desired: "x", Actual: "y", Severity: SeverityMedium}, // Should be ignored
			{Field: "sg", Desired: "1", Actual: "2", Severity: SeverityHigh},     // Should trigger heal
		},
		CheckedAt:    time.Now(),
		ResourceType: "VPC",
		ResourceID:   "vpc-123",
	}

	err := reconciler.ReconcileDrift(context.Background(), result, struct{}{})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !healCalled {
		t.Errorf("Expected heal to be called for high severity drift")
	}
}

// TestGetAction verifies action determination logic.
func TestGetAction(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		drift          DriftItem
		expectedAction ReconciliationAction
	}{
		{
			name: "auto-heal by default",
			config: &Config{
				DefaultAction:     ActionAutoHeal,
				IgnoreFields:      []string{},
				SeverityThreshold: SeverityLow,
			},
			drift:          DriftItem{Field: "test", Severity: SeverityMedium},
			expectedAction: ActionAutoHeal,
		},
		{
			name: "ignore based on field pattern",
			config: &Config{
				DefaultAction:     ActionAutoHeal,
				IgnoreFields:      []string{"tags.*"},
				SeverityThreshold: SeverityLow,
			},
			drift:          DriftItem{Field: "tags.Environment", Severity: SeverityMedium},
			expectedAction: ActionIgnore,
		},
		{
			name: "ignore based on severity threshold",
			config: &Config{
				DefaultAction:     ActionAutoHeal,
				IgnoreFields:      []string{},
				SeverityThreshold: SeverityHigh,
			},
			drift:          DriftItem{Field: "test", Severity: SeverityLow},
			expectedAction: ActionIgnore,
		},
		{
			name: "alert-only mode",
			config: &Config{
				DefaultAction:     ActionAlertOnly,
				IgnoreFields:      []string{},
				SeverityThreshold: SeverityLow,
			},
			drift:          DriftItem{Field: "test", Severity: SeverityHigh},
			expectedAction: ActionAlertOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := NewReconciler(tt.config)
			action := reconciler.GetAction(tt.drift)

			if action != tt.expectedAction {
				t.Errorf("Expected action %s, got %s", tt.expectedAction, action)
			}
		})
	}
}

// TestToDriftDetails verifies conversion to drift details.
func TestToDriftDetails(t *testing.T) {
	drifts := []DriftItem{
		{
			Field:      "Name",
			Desired:    "test-vpc",
			Actual:     "prod-vpc",
			Severity:   SeverityHigh,
			DetectedAt: time.Now(),
		},
		{
			Field:      "Tags.Environment",
			Desired:    "production",
			Actual:     "development",
			Severity:   SeverityMedium,
			DetectedAt: time.Now(),
		},
	}

	details := ToDriftDetails(drifts)

	if len(details) != 2 {
		t.Errorf("Expected 2 drift details, got %d", len(details))
	}

	if details[0].Field != "Name" {
		t.Errorf("Expected field 'Name', got '%s'", details[0].Field)
	}

	if details[0].Expected != "test-vpc" {
		t.Errorf("Expected expected value 'test-vpc', got '%s'", details[0].Expected)
	}

	if details[0].Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", details[0].Severity)
	}
}

// TestHealerRegistry verifies healer registry functionality.
func TestHealerRegistry(t *testing.T) {
	registry := NewHealerRegistry()

	// Register a healer
	vpcHealFunc := func(ctx context.Context, drifts []DriftItem, resource interface{}) error {
		return nil
	}
	registry.Register("VPC", vpcHealFunc)

	// Retrieve healer
	healFunc, ok := registry.Get("VPC")
	if !ok {
		t.Errorf("Expected to find registered healer for VPC")
	}
	if healFunc == nil {
		t.Errorf("Expected non-nil heal function")
	}

	// Try to get non-existent healer
	_, ok = registry.Get("NonExistent")
	if ok {
		t.Errorf("Expected not to find healer for non-registered type")
	}
}

// TestHealerRegistry_CreateReconciler verifies reconciler creation from registry.
func TestHealerRegistry_CreateReconciler(t *testing.T) {
	registry := NewHealerRegistry()
	config := DefaultConfig()

	// Register a healer
	healFunc := func(ctx context.Context, drifts []DriftItem, resource interface{}) error {
		return nil
	}
	registry.Register("VPC", healFunc)

	// Create reconciler for registered type
	reconciler, err := registry.CreateReconcilerForResource("VPC", config, nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if reconciler == nil {
		t.Errorf("Expected non-nil reconciler")
	}

	// Create reconciler for non-registered type (should still work, alert-only)
	reconciler2, err := registry.CreateReconcilerForResource("Unknown", config, nil)
	if err != nil {
		t.Errorf("Expected no error for unknown type, got: %v", err)
	}
	if reconciler2 == nil {
		t.Errorf("Expected non-nil reconciler even for unknown type")
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
