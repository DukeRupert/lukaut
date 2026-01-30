package service

import (
	"strings"
	"testing"

	"github.com/DukeRupert/lukaut/internal/domain"
)

// =============================================================================
// Password Validation Tests
// =============================================================================

func TestValidatePassword_MinimumLength(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		valid    bool
	}{
		{"too short - 7 chars", "Abcdef1", false},
		{"minimum - 8 chars", "Abcdef12", true},
		{"longer - 12 chars", "Abcdefgh1234", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if tc.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected error for short password")
			}
		})
	}
}

func TestValidatePassword_MaximumLength(t *testing.T) {
	// 72 is the bcrypt limit
	longPassword := strings.Repeat("Aa1", 24) // 72 chars
	tooLong := strings.Repeat("Aa1", 25)      // 75 chars

	if err := validatePassword(longPassword); err != nil {
		t.Errorf("72 char password should be valid: %v", err)
	}

	if err := validatePassword(tooLong); err == nil {
		t.Error("73+ char password should be invalid")
	}
}

func TestValidatePassword_RequiresLetter(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		valid    bool
	}{
		{"numbers only", "12345678", false},
		{"symbols only", "!@#$%^&*", false},
		{"numbers and symbols", "1234!@#$", false},
		{"has lowercase", "xmqr1234", true},
		{"has uppercase", "XMQR1234", true},
		{"has both cases", "XmQr1234", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if tc.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected error for password without letters")
			}
		})
	}
}

func TestValidatePassword_RequiresNumber(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		valid    bool
	}{
		{"letters only", "Abcdefgh", false},
		{"letters and symbols", "Abcd!@#$", false},
		{"has number", "Abcdefg1", true},
		{"multiple numbers", "Xyz98765", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if tc.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("expected error for password without numbers")
			}
		})
	}
}

func TestValidatePassword_CommonPasswords(t *testing.T) {
	// Common passwords should be rejected even if they meet other requirements
	commonPasswords := []string{
		"Password1",
		"Qwerty123",
		"Letmein1",
		"Welcome1",
		"Admin123",
	}

	for _, pw := range commonPasswords {
		t.Run(pw, func(t *testing.T) {
			err := validatePassword(pw)
			if err == nil {
				t.Errorf("common password %q should be rejected", pw)
			}
		})
	}
}

func TestValidatePassword_ValidPasswords(t *testing.T) {
	validPasswords := []string{
		"MyS3cur3Pass",
		"Th1sIsF1ne!",
		"C0mpl3xP@ss",
		"Randoms7ring",
		"N0tC0mmon!",
	}

	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			err := validatePassword(pw)
			if err != nil {
				t.Errorf("password %q should be valid: %v", pw, err)
			}
		})
	}
}

func TestValidatePassword_ErrorMessages(t *testing.T) {
	testCases := []struct {
		name            string
		password        string
		errorContains   string
	}{
		{"too short", "Ab1", "at least 8"},
		{"no letter", "12345678", "letter"},
		{"no number", "Abcdefgh", "number"},
		{"common password", "Password1", "common"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if err == nil {
				t.Fatal("expected error")
			}

			msg := domain.ErrorMessage(err)
			if !strings.Contains(strings.ToLower(msg), strings.ToLower(tc.errorContains)) {
				t.Errorf("error message %q should contain %q", msg, tc.errorContains)
			}
		})
	}
}
