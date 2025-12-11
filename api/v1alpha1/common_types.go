// Package v1alpha1 contém os tipos de API compartilhados.
package v1alpha1

// DriftDetail represents a detected difference between desired and actual state
type DriftDetail struct {
	// Field is the path to the drifted field
	Field string `json:"field"`

	// Expected is the expected value from the CR
	Expected string `json:"expected"`

	// Actual is the current value in AWS
	Actual string `json:"actual"`

	// Severity indicates the impact level: "low", "medium", "high"
	Severity string `json:"severity,omitempty"`
}
