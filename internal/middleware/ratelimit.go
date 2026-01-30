package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Rate Limiter
// =============================================================================

// RateLimiter tracks request counts per key with a sliding window.
type RateLimiter struct {
	maxAttempts int
	window      time.Duration
	logger      *slog.Logger

	mu      sync.RWMutex
	entries map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(maxAttempts int, window time.Duration, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		logger:      logger,
		entries:     make(map[string]*rateLimitEntry),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given key should be allowed.
// Returns true if allowed, false if rate limited.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[key]

	if !exists {
		// First request from this key
		rl.entries[key] = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// Check if window has expired
	if now.Sub(entry.windowStart) > rl.window {
		// Reset window
		entry.count = 1
		entry.windowStart = now
		return true
	}

	// Check if under limit
	if entry.count < rl.maxAttempts {
		entry.count++
		return true
	}

	// Rate limited
	return false
}

// RecordFailure records a failed attempt without checking the limit.
// Used to track failed logins that should count against the limit.
func (rl *RateLimiter) RecordFailure(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[key]

	if !exists {
		rl.entries[key] = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return
	}

	// Check if window has expired
	if now.Sub(entry.windowStart) > rl.window {
		entry.count = 1
		entry.windowStart = now
		return
	}

	entry.count++
}

// Reset clears the rate limit for a key (e.g., after successful login).
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, key)
}

// TimeUntilReset returns how long until the rate limit resets for a key.
func (rl *RateLimiter) TimeUntilReset(key string) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.entries[key]
	if !exists {
		return 0
	}

	elapsed := time.Since(entry.windowStart)
	if elapsed >= rl.window {
		return 0
	}

	return rl.window - elapsed
}

// cleanup periodically removes expired entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.entries {
			if now.Sub(entry.windowStart) > rl.window {
				delete(rl.entries, key)
			}
		}
		rl.mu.Unlock()
	}
}

// =============================================================================
// Rate Limit Middleware
// =============================================================================

// RateLimitMiddleware wraps a rate limiter for use as HTTP middleware.
type RateLimitMiddleware struct {
	limiter *RateLimiter
	logger  *slog.Logger
}

// NewRateLimitMiddleware creates a new rate limit middleware.
func NewRateLimitMiddleware(limiter *RateLimiter, logger *slog.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
	}
}

// Limit returns middleware that rate limits requests.
func (m *RateLimitMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if !m.limiter.Allow(clientIP) {
			m.logger.Warn("rate limit exceeded",
				"ip", clientIP,
				"path", r.URL.Path,
				"method", r.Method,
			)

			retryAfter := int(m.limiter.TimeUntilReset(clientIP).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

			if isAPIRequest(r) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
				})
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Too Many Requests</title></head>
<body>
<h1>Too Many Requests</h1>
<p>You have made too many requests. Please wait a moment and try again.</p>
</body>
</html>`))
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// Auth Rate Limiter (combined limiter for auth endpoints)
// =============================================================================

// AuthRateLimiter provides rate limiting for authentication endpoints
// with different limits for different actions.
type AuthRateLimiter struct {
	loginLimiter         *RateLimiter
	registerLimiter      *RateLimiter
	passwordResetLimiter *RateLimiter
	logger               *slog.Logger
}

// NewAuthRateLimiter creates rate limiters for auth endpoints with sensible defaults.
// - Login: 5 attempts per 15 minutes
// - Register: 3 attempts per hour
// - Password reset: 3 attempts per hour
func NewAuthRateLimiter(logger *slog.Logger) *AuthRateLimiter {
	return &AuthRateLimiter{
		loginLimiter:         NewRateLimiter(5, 15*time.Minute, logger),
		registerLimiter:      NewRateLimiter(3, time.Hour, logger),
		passwordResetLimiter: NewRateLimiter(3, time.Hour, logger),
		logger:               logger,
	}
}

// LimitLogin returns middleware for rate limiting login attempts.
func (a *AuthRateLimiter) LimitLogin(next http.Handler) http.Handler {
	mw := NewRateLimitMiddleware(a.loginLimiter, a.logger)
	return mw.Limit(next)
}

// LimitRegister returns middleware for rate limiting registration attempts.
func (a *AuthRateLimiter) LimitRegister(next http.Handler) http.Handler {
	mw := NewRateLimitMiddleware(a.registerLimiter, a.logger)
	return mw.Limit(next)
}

// LimitPasswordReset returns middleware for rate limiting password reset requests.
func (a *AuthRateLimiter) LimitPasswordReset(next http.Handler) http.Handler {
	mw := NewRateLimitMiddleware(a.passwordResetLimiter, a.logger)
	return mw.Limit(next)
}

// RecordFailedLogin records a failed login attempt for the given IP.
// Call this when login fails to make failed attempts count against the limit.
func (a *AuthRateLimiter) RecordFailedLogin(ip string) {
	a.loginLimiter.RecordFailure(ip)
}

// ResetLogin clears the rate limit for an IP after successful login.
func (a *AuthRateLimiter) ResetLogin(ip string) {
	a.loginLimiter.Reset(ip)
}

// =============================================================================
// Helpers
// =============================================================================

// getClientIP extracts the client IP from the request, considering proxy headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (most common proxy header)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: client, proxy1, proxy2
		// The first one is the original client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	// Check X-Real-IP (nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}

	return ip
}
