package middleware

import (
	"crypto/subtle"
	"net/http"
)

// MetricsAuthMiddleware provides basic authentication for the metrics endpoint.
type MetricsAuthMiddleware struct {
	username string
	password string
	enabled  bool
}

// NewMetricsAuthMiddleware creates a new metrics auth middleware.
// If both username and password are empty, authentication is disabled.
func NewMetricsAuthMiddleware(username, password string) *MetricsAuthMiddleware {
	return &MetricsAuthMiddleware{
		username: username,
		password: password,
		enabled:  username != "" || password != "",
	}
}

// Handler returns middleware that requires basic authentication.
func (m *MetricsAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, pass through
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			m.unauthorized(w)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(m.username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(m.password)) == 1

		if !userMatch || !passMatch {
			m.unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// unauthorized sends a 401 response with WWW-Authenticate header.
func (m *MetricsAuthMiddleware) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
