package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Request Logging Middleware Tests
// =============================================================================

func TestRequestLoggingMiddleware_LogsBasicInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log method
	if !strings.Contains(logOutput, "GET") {
		t.Errorf("log should contain method, got: %s", logOutput)
	}

	// Should log path
	if !strings.Contains(logOutput, "/dashboard") {
		t.Errorf("log should contain path, got: %s", logOutput)
	}

	// Should log status
	if !strings.Contains(logOutput, "200") {
		t.Errorf("log should contain status code, got: %s", logOutput)
	}

	// Should log duration
	if !strings.Contains(logOutput, "duration") {
		t.Errorf("log should contain duration, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_LogsClientIP(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log the real client IP from X-Forwarded-For
	if !strings.Contains(logOutput, "203.0.113.195") {
		t.Errorf("log should contain client IP from X-Forwarded-For, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_LogsErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("POST", "/api/action", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log 500 status
	if !strings.Contains(logOutput, "500") {
		t.Errorf("log should contain 500 status, got: %s", logOutput)
	}

	// Should use WARN or ERROR level for 5xx
	if !strings.Contains(logOutput, "level=WARN") && !strings.Contains(logOutput, "level=ERROR") {
		t.Errorf("5xx should log at WARN/ERROR level, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_LogsUserAgent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log user agent
	if !strings.Contains(logOutput, "Mozilla") || !strings.Contains(logOutput, "TestBrowser") {
		t.Errorf("log should contain user agent, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_DoesNotLogSensitiveQueryParams(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	// Request with sensitive query params
	req := httptest.NewRequest("GET", "/verify-email?token=secrettoken123", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should NOT log the token value
	if strings.Contains(logOutput, "secrettoken123") {
		t.Errorf("log should NOT contain sensitive token value, got: %s", logOutput)
	}

	// Should log the path (without sensitive params or with redacted params)
	if !strings.Contains(logOutput, "/verify-email") {
		t.Errorf("log should contain path, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_DoesNotLogPasswordResetToken(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/reset-password?token=abc123secret", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should NOT log the token value
	if strings.Contains(logOutput, "abc123secret") {
		t.Errorf("log should NOT contain password reset token, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_PassesRequestThrough(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("response body"))
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("POST", "/create", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("handler should have been called")
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	if rec.Header().Get("X-Custom") != "value" {
		t.Error("custom header should be preserved")
	}

	if rec.Body.String() != "response body" {
		t.Errorf("response body should be preserved, got: %s", rec.Body.String())
	}
}

func TestRequestLoggingMiddleware_CapturesWrittenStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/missing", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log 404 status
	if !strings.Contains(logOutput, "404") {
		t.Errorf("log should contain 404 status, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_ExcludesHealthCheck(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	// Health check should not be logged (too noisy)
	req := httptest.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should NOT log health checks
	if strings.Contains(logOutput, "/health") {
		t.Errorf("health check should not be logged, got: %s", logOutput)
	}
}

func TestRequestLoggingMiddleware_ExcludesMetrics(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := NewRequestLoggingMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should NOT log metrics endpoint (too noisy)
	if strings.Contains(logOutput, "/metrics") {
		t.Errorf("metrics endpoint should not be logged, got: %s", logOutput)
	}
}
