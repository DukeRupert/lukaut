// Package invite provides invite code validation for MVP testing.
package invite

import "strings"

// Validator provides invite code validation.
// Codes are stored in memory from environment variables.
type Validator struct {
	enabled bool
	codes   map[string]bool // Set for O(1) lookup
}

// New creates a new invite code validator.
func New(enabled bool, codes []string) *Validator {
	codeSet := make(map[string]bool)
	for _, code := range codes {
		normalized := strings.TrimSpace(strings.ToUpper(code))
		if normalized != "" {
			codeSet[normalized] = true
		}
	}
	return &Validator{
		enabled: enabled,
		codes:   codeSet,
	}
}

// IsEnabled returns whether invite codes are required.
func (v *Validator) IsEnabled() bool {
	return v.enabled
}

// ValidateCode checks if the provided code is valid.
// Returns true if codes are disabled OR code is valid.
func (v *Validator) ValidateCode(code string) bool {
	if !v.enabled {
		return true
	}
	normalized := strings.TrimSpace(strings.ToUpper(code))
	return v.codes[normalized]
}
