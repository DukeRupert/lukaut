package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// RequestLoggingMiddleware logs HTTP requests with timing and status information.
type RequestLoggingMiddleware struct {
	logger *slog.Logger
}

// NewRequestLoggingMiddleware creates a new request logging middleware.
func NewRequestLoggingMiddleware(logger *slog.Logger) *RequestLoggingMiddleware {
	return &RequestLoggingMiddleware{
		logger: logger,
	}
}

// Handler returns middleware that logs all HTTP requests.
func (m *RequestLoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for noisy endpoints
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get client IP
		clientIP := getClientIP(r)

		// Sanitize path to remove sensitive query params
		safePath := sanitizePath(r.URL.Path, r.URL.RawQuery)

		// Build log attributes
		attrs := []any{
			"method", r.Method,
			"path", safePath,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"ip", clientIP,
			"user_agent", r.UserAgent(),
		}

		// Log at appropriate level based on status code
		if wrapped.statusCode >= 500 {
			m.logger.Warn("request", attrs...)
		} else {
			m.logger.Info("request", attrs...)
		}
	})
}

// shouldSkip returns true for paths that should not be logged (too noisy).
func (m *RequestLoggingMiddleware) shouldSkip(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/static/", // Static assets
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}

	return false
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// sanitizePath removes sensitive query parameters from the path for logging.
func sanitizePath(path, rawQuery string) string {
	if rawQuery == "" {
		return path
	}

	// List of sensitive query parameter names to redact
	sensitiveParams := []string{
		"token",
		"code",
		"key",
		"secret",
		"password",
		"api_key",
		"apikey",
		"access_token",
		"refresh_token",
	}

	// Parse and redact sensitive params
	parts := strings.Split(rawQuery, "&")
	var safeParts []string

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.ToLower(kv[0])
		isSensitive := false
		for _, sensitive := range sensitiveParams {
			if key == sensitive {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			safeParts = append(safeParts, kv[0]+"=[REDACTED]")
		} else {
			safeParts = append(safeParts, part)
		}
	}

	if len(safeParts) == 0 {
		return path
	}

	return path + "?" + strings.Join(safeParts, "&")
}
