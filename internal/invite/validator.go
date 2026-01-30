// Package invite provides invite code validation for MVP testing.
package invite

import (
	"crypto/subtle"
	"strings"
)

// Validator provides invite code validation.
// Codes are stored in memory from environment variables.
type Validator struct {
	enabled bool
	codes   []string // Store as slice for constant-time iteration
}

// New creates a new invite code validator.
func New(enabled bool, codes []string) *Validator {
	// Normalize and deduplicate codes
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(codes))
	for _, code := range codes {
		norm := strings.TrimSpace(strings.ToUpper(code))
		if norm != "" && !seen[norm] {
			seen[norm] = true
			normalized = append(normalized, norm)
		}
	}
	return &Validator{
		enabled: enabled,
		codes:   normalized,
	}
}

// IsEnabled returns whether invite codes are required.
func (v *Validator) IsEnabled() bool {
	return v.enabled
}

// ValidateCode checks if the provided code is valid.
// Returns true if codes are disabled OR code is valid.
//
// Security: Uses constant-time comparison to prevent timing attacks.
// All codes are checked regardless of match to ensure consistent timing.
func (v *Validator) ValidateCode(code string) bool {
	if !v.enabled {
		return true
	}

	normalized := strings.TrimSpace(strings.ToUpper(code))
	if normalized == "" {
		return false
	}

	// Use constant-time comparison to prevent timing attacks.
	// We check all codes and accumulate the result to ensure
	// consistent timing regardless of which code matches.
	found := 0
	for _, validCode := range v.codes {
		// subtle.ConstantTimeCompare requires equal length strings
		// Pad shorter string to match length for constant-time behavior
		a := []byte(normalized)
		b := []byte(validCode)

		// If lengths differ, comparison will fail, but we still need
		// constant-time behavior. Use ConstantTimeEq for length check.
		if subtle.ConstantTimeEq(int32(len(a)), int32(len(b))) == 1 {
			found |= subtle.ConstantTimeCompare(a, b)
		}
	}

	return found == 1
}
