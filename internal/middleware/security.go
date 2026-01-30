package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware adds HTTP security headers to all responses.
type SecurityHeadersMiddleware struct {
	isSecure bool // Whether to enable HTTPS-specific headers (true in production)
}

// NewSecurityHeadersMiddleware creates a new security headers middleware.
// Set isSecure to true in production to enable HSTS and other HTTPS-specific headers.
func NewSecurityHeadersMiddleware(isSecure bool) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{
		isSecure: isSecure,
	}
}

// Handler returns middleware that sets security headers on all responses.
func (m *SecurityHeadersMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking - deny all framing
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// XSS protection (legacy but still helpful for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// HSTS - only in production with HTTPS
		if m.isSecure {
			// max-age=31536000 = 1 year
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		// Configured for Lukaut's tech stack:
		// - htmx and Alpine.js from unpkg CDN
		// - Tailwind CSS with inline styles
		// - Images from R2 storage and data URIs
		csp := buildCSP()
		w.Header().Set("Content-Security-Policy", csp)

		// Permissions Policy - disable browser features we don't need
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// buildCSP constructs the Content-Security-Policy header value.
// This is configured specifically for Lukaut's tech stack.
func buildCSP() string {
	return "default-src 'self'; " +
		// Scripts: self + htmx/Alpine from unpkg + unsafe-inline for Alpine's x-data
		"script-src 'self' https://unpkg.com 'unsafe-inline'; " +
		// Styles: self + unsafe-inline for Tailwind's inline styles
		"style-src 'self' 'unsafe-inline'; " +
		// Images: self + data URIs + any HTTPS source (for R2/external images)
		"img-src 'self' data: https:; " +
		// Fonts: self only
		"font-src 'self'; " +
		// Connect: self only (for htmx AJAX calls)
		"connect-src 'self'; " +
		// Prevent framing by any site
		"frame-ancestors 'none'; " +
		// Restrict base URI to prevent base tag injection
		"base-uri 'self'; " +
		// Restrict form actions to self
		"form-action 'self'"
}
