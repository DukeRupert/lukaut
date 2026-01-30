package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Security Headers Middleware Tests
// =============================================================================

func TestSecurityHeadersMiddleware_SetsAllHeaders(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true) // isSecure = true

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"X-XSS-Protection", "1; mode=block"},
	}

	for _, tc := range tests {
		got := rec.Header().Get(tc.header)
		if got != tc.expected {
			t.Errorf("%s: expected %q, got %q", tc.header, tc.expected, got)
		}
	}
}

func TestSecurityHeadersMiddleware_HSTSInProduction(t *testing.T) {
	// Production (isSecure = true) should set HSTS
	mwProd := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mwProd.Handler(handler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected HSTS header in production, got empty")
	}
	if !strings.Contains(hsts, "max-age=") {
		t.Errorf("expected HSTS max-age, got %q", hsts)
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("expected HSTS includeSubDomains, got %q", hsts)
	}
}

func TestSecurityHeadersMiddleware_NoHSTSInDevelopment(t *testing.T) {
	// Development (isSecure = false) should NOT set HSTS
	mwDev := NewSecurityHeadersMiddleware(false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mwDev.Handler(handler).ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no HSTS header in development, got %q", hsts)
	}
}

func TestSecurityHeadersMiddleware_CSPHeader(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header, got empty")
	}

	// Check required CSP directives
	requiredDirectives := []string{
		"default-src",
		"script-src",
		"style-src",
		"img-src",
		"frame-ancestors 'none'",
	}

	for _, directive := range requiredDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP missing directive %q: %s", directive, csp)
		}
	}
}

func TestSecurityHeadersMiddleware_PermissionsPolicy(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("expected Permissions-Policy header, got empty")
	}

	// Should disable sensitive permissions
	if !strings.Contains(pp, "geolocation=()") {
		t.Errorf("expected geolocation disabled in Permissions-Policy: %s", pp)
	}
	if !strings.Contains(pp, "camera=()") {
		t.Errorf("expected camera disabled in Permissions-Policy: %s", pp)
	}
	if !strings.Contains(pp, "microphone=()") {
		t.Errorf("expected microphone disabled in Permissions-Policy: %s", pp)
	}
}

func TestSecurityHeadersMiddleware_PassesThroughRequests(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := mw.Handler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", rec.Body.String())
	}
}

func TestSecurityHeadersMiddleware_WorksWithPOST(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest("POST", "/api/data", strings.NewReader("test data"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	// Headers should still be set
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options header on POST request")
	}
}

func TestSecurityHeadersMiddleware_CSPAllowsHTMX(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// CSP should allow htmx from unpkg
	if !strings.Contains(csp, "unpkg.com") {
		t.Errorf("CSP should allow unpkg.com for htmx: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_CSPAllowsInlineStyles(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// CSP should allow inline styles for Tailwind
	if !strings.Contains(csp, "'unsafe-inline'") {
		t.Errorf("CSP should allow unsafe-inline for Tailwind styles: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_CSPAllowsR2Images(t *testing.T) {
	mw := NewSecurityHeadersMiddleware(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mw.Handler(handler).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// CSP should allow images from various sources (data:, https:, self)
	if !strings.Contains(csp, "img-src") {
		t.Errorf("CSP should have img-src directive: %s", csp)
	}
	// Should allow data: URIs for inline images
	if !strings.Contains(csp, "data:") {
		t.Errorf("CSP should allow data: URIs for images: %s", csp)
	}
}
