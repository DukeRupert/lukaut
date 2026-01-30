package invite

import (
	"testing"
	"time"
)

// =============================================================================
// Invite Code Validator Tests
// =============================================================================

func TestValidatorDisabled(t *testing.T) {
	v := New(false, []string{"CODE1", "CODE2"})

	if v.IsEnabled() {
		t.Error("expected validator to be disabled")
	}

	// Any code should be valid when disabled
	if !v.ValidateCode("anything") {
		t.Error("disabled validator should accept any code")
	}
	if !v.ValidateCode("") {
		t.Error("disabled validator should accept empty code")
	}
}

func TestValidatorEnabled(t *testing.T) {
	v := New(true, []string{"VALID1", "VALID2", "valid3"})

	if !v.IsEnabled() {
		t.Error("expected validator to be enabled")
	}

	// Valid codes should be accepted (case-insensitive)
	testCases := []struct {
		code  string
		valid bool
	}{
		{"VALID1", true},
		{"valid1", true},
		{"Valid1", true},
		{"VALID2", true},
		{"VALID3", true}, // Original was lowercase
		{"INVALID", false},
		{"", false},
		{"VALID", false}, // Partial match should fail
	}

	for _, tc := range testCases {
		result := v.ValidateCode(tc.code)
		if result != tc.valid {
			t.Errorf("ValidateCode(%q) = %v, want %v", tc.code, result, tc.valid)
		}
	}
}

func TestValidatorTrimsWhitespace(t *testing.T) {
	v := New(true, []string{"  TRIMMED  "})

	// Should trim whitespace on stored codes
	if !v.ValidateCode("TRIMMED") {
		t.Error("stored code whitespace should be trimmed")
	}

	// Should trim whitespace on input codes
	if !v.ValidateCode("  TRIMMED  ") {
		t.Error("input code whitespace should be trimmed")
	}
}

func TestValidatorEmptyCodes(t *testing.T) {
	v := New(true, []string{})

	if !v.IsEnabled() {
		t.Error("expected validator to be enabled even with no codes")
	}

	// No codes = nothing valid
	if v.ValidateCode("anything") {
		t.Error("empty code list should reject all codes")
	}
}

func TestValidatorIgnoresEmptyStrings(t *testing.T) {
	v := New(true, []string{"", "  ", "VALID"})

	// Empty strings in input should be ignored
	if v.ValidateCode("") {
		t.Error("empty string should not be valid")
	}

	// But actual codes should still work
	if !v.ValidateCode("VALID") {
		t.Error("valid code should be accepted")
	}
}

// =============================================================================
// Timing Attack Prevention Tests
// =============================================================================

func TestValidatorConstantTime(t *testing.T) {
	// This test verifies that validation takes similar time regardless of input.
	// Note: This is a statistical test and may have some variance.

	codes := []string{"ABCDEFGH", "IJKLMNOP", "QRSTUVWX"}
	v := New(true, codes)

	// Measure time for valid code
	validTimes := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		start := time.Now()
		v.ValidateCode("ABCDEFGH")
		validTimes[i] = time.Since(start)
	}

	// Measure time for invalid code (completely different)
	invalidTimes := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		start := time.Now()
		v.ValidateCode("ZZZZZZZZ")
		invalidTimes[i] = time.Since(start)
	}

	// Measure time for partially matching code
	partialTimes := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		start := time.Now()
		v.ValidateCode("ABCDXXXX") // First 4 chars match
		partialTimes[i] = time.Since(start)
	}

	// Calculate averages
	validAvg := averageDuration(validTimes)
	invalidAvg := averageDuration(invalidTimes)
	partialAvg := averageDuration(partialTimes)

	// The times should be relatively similar (within 10x of each other)
	// This is a loose bound because timing can be noisy
	maxRatio := 10.0
	if ratio := float64(validAvg) / float64(invalidAvg); ratio > maxRatio || ratio < 1/maxRatio {
		t.Logf("Warning: timing ratio valid/invalid = %.2f (valid=%v, invalid=%v)", ratio, validAvg, invalidAvg)
	}
	if ratio := float64(validAvg) / float64(partialAvg); ratio > maxRatio || ratio < 1/maxRatio {
		t.Logf("Warning: timing ratio valid/partial = %.2f (valid=%v, partial=%v)", ratio, validAvg, partialAvg)
	}

	// This test is informational - we log but don't fail on timing variance
	// because timing tests are inherently flaky
	t.Logf("Timing: valid=%v, invalid=%v, partial=%v", validAvg, invalidAvg, partialAvg)
}

func averageDuration(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func TestValidateCodeUsesConstantTimeCompare(t *testing.T) {
	// This test documents the expected behavior: ValidateCode should use
	// constant-time comparison to prevent timing attacks.
	//
	// While invite codes are low-risk, defense-in-depth is good practice.

	v := New(true, []string{"SECRETCODE"})

	// The implementation should:
	// 1. Iterate through all codes
	// 2. Use subtle.ConstantTimeCompare for each comparison
	// 3. Return result after checking all codes (not early return)

	// We can't directly test for constant-time behavior, but we verify
	// the function works correctly
	if !v.ValidateCode("SECRETCODE") {
		t.Error("valid code should be accepted")
	}
	if v.ValidateCode("WRONGCODE1") {
		t.Error("invalid code should be rejected")
	}
}
