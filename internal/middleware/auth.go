// Package middleware contains HTTP middleware for the Lukaut application.
//
// Middleware functions follow the standard Go pattern of wrapping http.Handler.
// They are designed to be composed using a middleware stack approach.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/handler"
	"github.com/DukeRupert/lukaut/internal/service"
)

// =============================================================================
// Configuration Constants
// =============================================================================

const (
	// SessionCookieName is the name of the cookie that stores the session token.
	// Using a descriptive name helps with debugging while not revealing
	// implementation details.
	SessionCookieName = "lukaut_session"

	// SessionCookiePath ensures the cookie is sent with all requests.
	SessionCookiePath = "/"

	// SessionCookieMaxAge sets the cookie expiration.
	// This should match SessionDuration in the user service.
	// 7 days = 604800 seconds
	SessionCookieMaxAge = 7 * 24 * 60 * 60
)

// =============================================================================
// Context Keys
// =============================================================================

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserContextKey is the key used to store the authenticated user in context.
	// Use GetUser(ctx) to retrieve the user from the context.
	userContextKey contextKey = "user"
)

// =============================================================================
// Context Helpers
// =============================================================================

// GetUser retrieves the authenticated user from the request context.
//
// Returns nil if no user is authenticated (request passed through WithUser
// but no valid session was found).
//
// Usage:
//
//	user := middleware.GetUser(r.Context())
//	if user == nil {
//	    // Handle unauthenticated request
//	}
func GetUser(ctx context.Context) *domain.User {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

// setUser stores a user in the request context.
func setUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// =============================================================================
// Auth Middleware Configuration
// =============================================================================

// AuthMiddleware provides authentication middleware functionality.
//
// This struct holds dependencies needed by auth middleware functions.
// Create one instance and use its methods as middleware.
type AuthMiddleware struct {
	userService service.UserService
	logger      *slog.Logger
	isSecure    bool // Whether to set Secure flag on cookies (true in production)
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
//
// Parameters:
// - userService: Service for user and session operations
// - logger: Structured logger for auth events
// - isSecure: Set to true in production to enable Secure cookie flag
func NewAuthMiddleware(userService service.UserService, logger *slog.Logger, isSecure bool) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
		logger:      logger,
		isSecure:    isSecure,
	}
}

// =============================================================================
// WithUser Middleware
// =============================================================================

// WithUser is middleware that attempts to load the user from the session cookie.
//
// This middleware:
// 1. Checks for a session cookie
// 2. If found, validates the session and loads the user
// 3. Stores the user in the request context
// 4. Continues to the next handler regardless of authentication status
//
// Use this middleware on routes that work both authenticated and unauthenticated
// (e.g., home page shows different content for logged-in users).
//
// The user can be retrieved in handlers using:
//
//	user := middleware.GetUser(r.Context())
//
// Flow:
//
//	Request -> WithUser -> Handler
//	           |
//	           +-> Read cookie
//	           +-> Validate session (if cookie exists)
//	           +-> Set user in context (if valid)
//	           +-> Call next handler (always)
func (m *AuthMiddleware) WithUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			// No cookie found - continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Validate session and get user
		user, err := m.userService.GetBySessionToken(r.Context(), cookie.Value)
		if err != nil {
			// Invalid or expired session - clear the cookie and continue
			clearSessionCookie(w, m.isSecure)
			next.ServeHTTP(w, r)
			return
		}

		// Set user in context
		ctx := setUser(r.Context(), user)
		r = r.WithContext(ctx)

		// Call next handler with user in context
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// RequireUser Middleware
// =============================================================================

// RequireUser is middleware that requires an authenticated user.
//
// This middleware:
// 1. Checks if a user is present in the context (set by WithUser)
// 2. If not authenticated, redirects to login (HTML) or returns 401 (JSON)
// 3. If authenticated, continues to the next handler
//
// IMPORTANT: This middleware must be used AFTER WithUser in the middleware chain.
//
// Usage:
//
//	// Correct - WithUser runs first, then RequireUser
//	mux.Handle("GET /dashboard", authMw.WithUser(authMw.RequireUser(dashboardHandler)))
//
//	// Or with a middleware stack function
//	stack := func(h http.Handler) http.Handler {
//	    return authMw.WithUser(authMw.RequireUser(h))
//	}
//
// Flow:
//
//	Request -> WithUser -> RequireUser -> Handler
//	                       |
//	                       +-> Check context for user
//	                       +-> If no user: redirect to /login (or 401 for API)
//	                       +-> If user exists: call next handler
func (m *AuthMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get user from context
		// - Use GetUser(r.Context())

		// TODO: If no user, handle unauthenticated request
		// - Check if request accepts JSON (API request)
		// - If JSON: return 401 with JSON error body
		// - If HTML: redirect to /login with return URL parameter
		//   Example: /login?return_to=/dashboard

		// TODO: Call next handler (user is authenticated)
		// - next.ServeHTTP(w, r)

		// Placeholder implementation
		user := GetUser(r.Context())
		if user == nil {
			// Check if this is an API request
			if isAPIRequest(r) {
				handler.UnauthorizedResponse(w, r, m.logger)
				return
			}

			// HTML request - redirect to login
			// Include return_to parameter for post-login redirect
			returnTo := r.URL.Path
			if r.URL.RawQuery != "" {
				returnTo += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, "/login?return_to="+returnTo, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// RequireEmailVerified Middleware
// =============================================================================

// RequireEmailVerified is middleware that requires the user's email to be verified.
//
// This middleware:
// 1. Assumes a user is already in context (use after RequireUser)
// 2. Checks if the user's email is verified
// 3. If not verified, redirects to verification reminder page
// 4. If verified, continues to the next handler
//
// IMPORTANT: Use this AFTER RequireUser in the middleware chain.
//
// Usage:
//
//	mux.Handle("GET /dashboard",
//	    authMw.WithUser(
//	        authMw.RequireUser(
//	            authMw.RequireEmailVerified(dashboardHandler))))
func (m *AuthMiddleware) RequireEmailVerified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context (should exist because RequireUser ran first)
		user := GetUser(r.Context())
		if user == nil {
			// This shouldn't happen if RequireUser is used before this middleware
			m.logger.Error("RequireEmailVerified called without user in context")
			if isAPIRequest(r) {
				handler.UnauthorizedResponse(w, r, m.logger)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}

		// Check if email is verified
		if !user.EmailVerified {
			if isAPIRequest(r) {
				// Return 403 for API requests
				err := domain.Forbidden("", "Email verification required")
				handler.ErrorResponse(w, r, m.logger, err)
				return
			}

			// Redirect to verification reminder page for HTML requests
			http.Redirect(w, r, "/verify-email-reminder", http.StatusSeeOther)
			return
		}

		// Email is verified - continue to next handler
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// RequireActiveSubscription Middleware
// =============================================================================

// RequireActiveSubscription is middleware that requires an active subscription.
//
// This middleware:
// 1. Assumes a user is already in context (use after RequireUser)
// 2. Checks if the user has an active subscription or is trialing
// 3. If not active, redirects to billing page or returns 402 Payment Required
// 4. If active, continues to the next handler
//
// IMPORTANT: Use this AFTER RequireUser in the middleware chain.
//
// Usage:
//
//	mux.Handle("POST /inspections/{id}/analyze",
//	    authMw.WithUser(
//	        authMw.RequireUser(
//	            authMw.RequireActiveSubscription(analyzeHandler))))
func (m *AuthMiddleware) RequireActiveSubscription(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context (should exist because RequireUser ran first)
		user := GetUser(r.Context())
		if user == nil {
			// This shouldn't happen if RequireUser is used before this middleware
			m.logger.Error("RequireActiveSubscription called without user in context")
			if isAPIRequest(r) {
				handler.UnauthorizedResponse(w, r, m.logger)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}

		// Check if subscription is active
		if !user.IsActive() {
			if isAPIRequest(r) {
				// Return 402 Payment Required for API requests
				err := domain.Errorf(domain.EPAYMENT, "", "Active subscription required")
				handler.ErrorResponse(w, r, m.logger, err)
				return
			}

			// Redirect to billing page for HTML requests
			http.Redirect(w, r, "/settings/billing?upgrade=1", http.StatusSeeOther)
			return
		}

		// Subscription is active - continue to next handler
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// Cookie Helpers
// =============================================================================

// SetSessionCookie sets the session cookie on the response.
//
// Cookie Settings:
// - HttpOnly: true - Prevents JavaScript access (XSS protection)
// - Secure: configurable - Set true in production (HTTPS only)
// - SameSite: Lax - Prevents CSRF while allowing normal navigation
// - Path: / - Cookie sent with all requests
// - MaxAge: 7 days - Matches session duration
//
// Parameters:
// - w: Response writer to set cookie on
// - token: Raw session token (64-char hex string)
// - isSecure: Whether to set Secure flag (true in production)
func SetSessionCookie(w http.ResponseWriter, token string, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     SessionCookiePath,
		MaxAge:   SessionCookieMaxAge,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie removes the session cookie from the client.
//
// This is done by setting MaxAge to -1, which tells the browser to delete
// the cookie immediately.
func clearSessionCookie(w http.ResponseWriter, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     SessionCookiePath,
		MaxAge:   -1, // Delete immediately
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie is the exported version for use in logout handlers.
func ClearSessionCookie(w http.ResponseWriter, isSecure bool) {
	clearSessionCookie(w, isSecure)
}

// =============================================================================
// Request Helpers
// =============================================================================

// isAPIRequest determines if the request expects a JSON response.
//
// This is used to decide whether to redirect (HTML) or return JSON errors (API).
//
// Checks:
// 1. Accept header contains application/json
// 2. Content-Type is application/json
// 3. URL path starts with /api/
// 4. HX-Request header is NOT present (htmx wants HTML)
func isAPIRequest(r *http.Request) bool {
	// htmx requests want HTML fragments
	if r.Header.Get("HX-Request") == "true" {
		return false
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return true
	}

	// Check URL path (API routes)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}

	return false
}

// =============================================================================
// Middleware Stack Helpers
// =============================================================================

// Stack composes multiple middleware functions into a single middleware.
//
// Middleware is applied in the order provided, meaning the first middleware
// in the slice is the outermost (runs first on request, last on response).
//
// Example:
//
//	stack := Stack(loggingMw, authMw.WithUser, authMw.RequireUser)
//	mux.Handle("GET /dashboard", stack(dashboardHandler))
//
// This is equivalent to:
//
//	mux.Handle("GET /dashboard",
//	    loggingMw(authMw.WithUser(authMw.RequireUser(dashboardHandler))))
func Stack(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// =============================================================================
// Additional Middleware (Future Implementation)
// =============================================================================

// TODO: Implement CSRF protection middleware
// This should:
// - Generate CSRF token and store in session/cookie
// - Validate CSRF token on POST/PUT/DELETE requests
// - Provide template function to embed token in forms
// - Use double-submit cookie pattern or synchronizer token pattern

// TODO: Implement rate limiting middleware
// This should:
// - Track requests per IP address
// - Use token bucket or sliding window algorithm
// - Return 429 Too Many Requests when limit exceeded
// - Include Retry-After header in response
// - Consider storing in Redis for distributed rate limiting

// TODO: Implement request logging middleware
// This should:
// - Log request method, path, status code, duration
// - Include request ID for correlation
// - Log user ID if authenticated
// - Use structured logging (slog)

// =============================================================================
// Compile-time checks
// =============================================================================

// Ensure middleware functions have correct signature
var (
	_ func(http.Handler) http.Handler = (&AuthMiddleware{}).WithUser
	_ func(http.Handler) http.Handler = (&AuthMiddleware{}).RequireUser
	_ func(http.Handler) http.Handler = (&AuthMiddleware{}).RequireEmailVerified
	_ func(http.Handler) http.Handler = (&AuthMiddleware{}).RequireActiveSubscription
)

// =============================================================================
// Example Usage
// =============================================================================

/*
Example of how to use this middleware in main.go:

func main() {
    // ... initialize dependencies ...

    // Create auth middleware
    authMw := middleware.NewAuthMiddleware(userService, logger, cfg.Env != "development")

    // Create middleware stacks for different route groups
    publicStack := Stack(loggingMw)
    authStack := Stack(loggingMw, authMw.WithUser, authMw.RequireUser)
    paidStack := Stack(loggingMw, authMw.WithUser, authMw.RequireUser, authMw.RequireActiveSubscription)

    // Register routes
    mux := http.NewServeMux()

    // Public routes (no auth required)
    mux.Handle("GET /", publicStack(homeHandler))
    mux.Handle("GET /login", publicStack(loginPageHandler))
    mux.Handle("POST /login", publicStack(loginHandler))

    // Authenticated routes
    mux.Handle("GET /dashboard", authStack(dashboardHandler))
    mux.Handle("GET /settings", authStack(settingsHandler))

    // Paid feature routes
    mux.Handle("POST /inspections/{id}/analyze", paidStack(analyzeHandler))
    mux.Handle("POST /inspections/{id}/reports", paidStack(generateReportHandler))
}
*/
