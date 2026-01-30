package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// =============================================================================
// RateLimiter Tests
// =============================================================================

func TestNewRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(5, time.Minute, logger)

	if rl == nil {
		t.Fatal("expected rate limiter to be created")
	}
	if rl.maxAttempts != 5 {
		t.Errorf("expected maxAttempts=5, got %d", rl.maxAttempts)
	}
	if rl.window != time.Minute {
		t.Errorf("expected window=1m, got %v", rl.window)
	}
}

func TestRateLimiter_Allow_UnderLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(5, time.Minute, logger)

	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_Allow_AtLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(5, time.Minute, logger)

	// Use up all 5 attempts
	for i := 0; i < 5; i++ {
		rl.Allow("192.168.1.1")
	}

	// 6th request should be denied
	if rl.Allow("192.168.1.1") {
		t.Error("6th request should be denied")
	}
}

func TestRateLimiter_Allow_DifferentIPs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(2, time.Minute, logger)

	// IP 1 uses its limit
	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")
	if rl.Allow("192.168.1.1") {
		t.Error("IP 1 should be rate limited")
	}

	// IP 2 should still have its own limit
	if !rl.Allow("192.168.1.2") {
		t.Error("IP 2 should not be rate limited")
	}
	if !rl.Allow("192.168.1.2") {
		t.Error("IP 2 should still not be rate limited")
	}
	if rl.Allow("192.168.1.2") {
		t.Error("IP 2 should now be rate limited")
	}
}

func TestRateLimiter_Allow_WindowExpiry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// Use a very short window for testing
	rl := NewRateLimiter(2, 50*time.Millisecond, logger)

	// Use up the limit
	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")
	if rl.Allow("192.168.1.1") {
		t.Error("should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("192.168.1.1") {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiter_RecordFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(5, time.Minute, logger)

	// Record failures (simulating failed login attempts)
	for i := 0; i < 5; i++ {
		rl.RecordFailure("192.168.1.1")
	}

	// Should now be blocked
	if rl.Allow("192.168.1.1") {
		t.Error("should be blocked after 5 failures")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(2, time.Minute, logger)

	// Use up limit
	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.1")
	if rl.Allow("192.168.1.1") {
		t.Error("should be rate limited")
	}

	// Reset the IP
	rl.Reset("192.168.1.1")

	// Should be allowed again
	if !rl.Allow("192.168.1.1") {
		t.Error("should be allowed after reset")
	}
}

// =============================================================================
// RateLimitMiddleware Tests
// =============================================================================

func TestRateLimitMiddleware_AllowsRequests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(5, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := mw.Limit(handler)

	// First request should pass
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_BlocksAfterLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(2, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// Make requests until blocked
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 2 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 2 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_RetryAfterHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(1, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// First request passes
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Second request is rate limited
	req = httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestRateLimitMiddleware_HTMLResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(1, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// Use up the limit
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Second request - HTML response
	req = httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Accept", "text/html")
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %s", contentType)
	}
}

func TestRateLimitMiddleware_JSONResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(1, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// Use up the limit
	req := httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Second request - JSON response
	req = httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Accept", "application/json")
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json content type, got %s", contentType)
	}
}

func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(2, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// Requests with X-Forwarded-For header (behind proxy)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:12345" // Proxy IP
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 2 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 2 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_XRealIP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rl := NewRateLimiter(2, time.Minute, logger)
	mw := NewRateLimitMiddleware(rl, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.Limit(handler)

	// Requests with X-Real-IP header (nginx proxy)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:12345" // Proxy IP
		req.Header.Set("X-Real-IP", "203.0.113.195")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 2 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 2 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

// =============================================================================
// AuthRateLimiter Tests (Combined auth endpoint limiter)
// =============================================================================

func TestNewAuthRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	if arl == nil {
		t.Fatal("expected auth rate limiter to be created")
	}
	if arl.loginLimiter == nil {
		t.Error("expected login limiter to be created")
	}
	if arl.registerLimiter == nil {
		t.Error("expected register limiter to be created")
	}
	if arl.passwordResetLimiter == nil {
		t.Error("expected password reset limiter to be created")
	}
}

func TestAuthRateLimiter_Login(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := arl.LimitLogin(handler)

	// Default login limit is 5 per 15 minutes
	// Make 6 requests, 6th should be blocked
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 5 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 5 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

func TestAuthRateLimiter_Register(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := arl.LimitRegister(handler)

	// Default register limit is 3 per hour
	// Make 4 requests, 4th should be blocked
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("POST", "/register", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 3 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 3 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

func TestAuthRateLimiter_PasswordReset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := arl.LimitPasswordReset(handler)

	// Default password reset limit is 3 per hour
	// Make 4 requests, 4th should be blocked
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("POST", "/forgot-password", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if i < 3 && rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
		if i == 3 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
		}
	}
}

func TestAuthRateLimiter_RecordFailedLogin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	// Record 5 failed logins
	for i := 0; i < 5; i++ {
		arl.RecordFailedLogin("192.168.1.1")
	}

	// Next login attempt should be blocked
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := arl.LimitLogin(handler)

	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after failed logins, got %d", rec.Code)
	}
}

func TestAuthRateLimiter_ResetOnSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	arl := NewAuthRateLimiter(logger)

	// Record some failed logins
	for i := 0; i < 3; i++ {
		arl.RecordFailedLogin("192.168.1.1")
	}

	// Simulate successful login - reset the counter
	arl.ResetLogin("192.168.1.1")

	// Should be able to make 5 more attempts
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := arl.LimitLogin(handler)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200 after reset, got %d", i+1, rec.Code)
		}
	}
}
