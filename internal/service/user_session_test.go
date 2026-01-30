package service

import (
	"testing"
	"time"
)

// =============================================================================
// Session Duration Configuration Tests
// =============================================================================

func TestDefaultSessionDuration(t *testing.T) {
	// Default session duration should be 24 hours (changed from 7 days)
	expected := 24 * time.Hour
	if DefaultSessionDuration != expected {
		t.Errorf("expected default session duration %v, got %v", expected, DefaultSessionDuration)
	}
}

func TestSessionDurationIsConfigurable(t *testing.T) {
	// Verify that session duration can be configured via UserServiceConfig
	testCases := []struct {
		name     string
		duration time.Duration
	}{
		{"1 hour", 1 * time.Hour},
		{"12 hours", 12 * time.Hour},
		{"24 hours", 24 * time.Hour},
		{"7 days", 7 * 24 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := UserServiceConfig{
				SessionDuration: tc.duration,
			}
			if cfg.SessionDuration != tc.duration {
				t.Errorf("expected session duration %v, got %v", tc.duration, cfg.SessionDuration)
			}
		})
	}
}

func TestSessionDurationMinimum(t *testing.T) {
	// Session duration should have a minimum of 15 minutes for security
	minDuration := 15 * time.Minute

	testCases := []struct {
		name      string
		input     time.Duration
		shouldUse time.Duration
	}{
		{"below minimum uses minimum", 5 * time.Minute, minDuration},
		{"at minimum uses input", 15 * time.Minute, 15 * time.Minute},
		{"above minimum uses input", 1 * time.Hour, 1 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeSessionDuration(tc.input)
			if result != tc.shouldUse {
				t.Errorf("expected %v, got %v", tc.shouldUse, result)
			}
		})
	}
}

func TestSessionDurationMaximum(t *testing.T) {
	// Session duration should have a maximum of 30 days
	maxDuration := 30 * 24 * time.Hour

	testCases := []struct {
		name      string
		input     time.Duration
		shouldUse time.Duration
	}{
		{"below maximum uses input", 7 * 24 * time.Hour, 7 * 24 * time.Hour},
		{"at maximum uses input", 30 * 24 * time.Hour, 30 * 24 * time.Hour},
		{"above maximum uses maximum", 60 * 24 * time.Hour, maxDuration},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeSessionDuration(tc.input)
			if result != tc.shouldUse {
				t.Errorf("expected %v, got %v", tc.shouldUse, result)
			}
		})
	}
}

func TestSessionDurationZeroUsesDefault(t *testing.T) {
	// Zero duration should use the default
	result := normalizeSessionDuration(0)
	if result != DefaultSessionDuration {
		t.Errorf("expected default %v for zero input, got %v", DefaultSessionDuration, result)
	}
}
