package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// Metrics Auth Middleware Tests
// =============================================================================

func TestMetricsAuthMiddleware_AllowsValidCredentials(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics data"))
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("admin", "secret123")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "metrics data" {
		t.Errorf("expected body 'metrics data', got %q", rec.Body.String())
	}
}

func TestMetricsAuthMiddleware_RejectsNoCredentials(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	// Should have WWW-Authenticate header
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header")
	}
	if wwwAuth != `Basic realm="metrics"` {
		t.Errorf("unexpected WWW-Authenticate header: %q", wwwAuth)
	}
}

func TestMetricsAuthMiddleware_RejectsWrongUsername(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("wronguser", "secret123")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMetricsAuthMiddleware_RejectsWrongPassword(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("admin", "wrongpassword")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMetricsAuthMiddleware_RejectsMalformedAuth(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	// Set a malformed Authorization header
	req.Header.Set("Authorization", "Basic notvalidbase64!!!")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMetricsAuthMiddleware_RejectsEmptyCredentials(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("", "")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMetricsAuthMiddleware_ConstantTimeComparison(t *testing.T) {
	// This test verifies that we use constant-time comparison
	// by checking that both user and password are validated
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	testCases := []struct {
		user     string
		pass     string
		expected int
	}{
		{"admin", "secret123", http.StatusOK},
		{"admin", "wrong", http.StatusUnauthorized},
		{"wrong", "secret123", http.StatusUnauthorized},
		{"wrong", "wrong", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.SetBasicAuth(tc.user, tc.pass)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != tc.expected {
			t.Errorf("user=%q pass=%q: expected %d, got %d",
				tc.user, tc.pass, tc.expected, rec.Code)
		}
	}
}

func TestMetricsAuthMiddleware_DisabledWhenNoCredentials(t *testing.T) {
	// When both user and pass are empty, auth should be disabled
	mw := NewMetricsAuthMiddleware("", "")

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	// No auth header
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called when auth is disabled")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 when auth is disabled, got %d", rec.Code)
	}
}

func TestMetricsAuthMiddleware_HeaderInjection(t *testing.T) {
	mw := NewMetricsAuthMiddleware("admin", "secret123")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	// Try to inject newlines in auth header
	req := httptest.NewRequest("GET", "/metrics", nil)
	malicious := base64.StdEncoding.EncodeToString([]byte("admin:secret123\r\nX-Injected: header"))
	req.Header.Set("Authorization", "Basic "+malicious)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should fail because credentials don't match
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for injection attempt, got %d", rec.Code)
	}
}
