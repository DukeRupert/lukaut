// Package csrf provides CSRF protection using the double-submit cookie pattern.
//
// The double-submit cookie pattern works by:
// 1. Setting a random token in a cookie (not HttpOnly, so JS can read it)
// 2. Including the same token in forms as a hidden field
// 3. On POST, comparing the cookie value with the form value
//
// This is secure because:
// - Attackers can make the browser send cookies with cross-origin requests
// - But attackers cannot read/set cookies for our domain (same-origin policy)
// - So they cannot include the correct token in the form body
package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// =============================================================================
// Configuration Constants
// =============================================================================

const (
	// CookieName is the name of the CSRF token cookie.
	CookieName = "csrf_token"

	// FormFieldName is the name of the CSRF token form field.
	FormFieldName = "csrf_token"

	// TokenLength is the number of random bytes for the token (32 bytes = 256 bits).
	TokenLength = 32

	// CookieMaxAge is the lifetime of the CSRF cookie (1 hour).
	// This is shorter than session cookies since CSRF tokens should be refreshed.
	CookieMaxAge = 3600
)

// =============================================================================
// Token Generation
// =============================================================================

// GenerateToken generates a cryptographically secure random token.
//
// The token is 32 bytes of random data, base64 URL-encoded.
// This produces a 43-character string.
func GenerateToken() (string, error) {
	b := make([]byte, TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// MustGenerateToken generates a token or panics.
// Use this only during application startup or in contexts where
// crypto/rand failure would be catastrophic anyway.
func MustGenerateToken() string {
	token, err := GenerateToken()
	if err != nil {
		panic("csrf: failed to generate token: " + err.Error())
	}
	return token
}

// =============================================================================
// Token Validation
// =============================================================================

// ValidateToken compares the cookie token with the form token.
//
// Uses constant-time comparison to prevent timing attacks.
// Returns true if tokens match, false otherwise.
func ValidateToken(cookieToken, formToken string) bool {
	if cookieToken == "" || formToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(formToken)) == 1
}

// ValidateRequest validates the CSRF token from a request.
//
// It reads the token from:
// - Cookie: the csrf_token cookie
// - Form: the csrf_token form field (requires ParseForm to be called first)
//
// Returns true if the tokens match, false otherwise.
func ValidateRequest(r *http.Request) bool {
	// Get token from cookie
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}

	// Get token from form
	formToken := r.FormValue(FormFieldName)

	return ValidateToken(cookie.Value, formToken)
}

// =============================================================================
// Cookie Management
// =============================================================================

// SetCookie sets the CSRF token cookie on the response.
//
// Cookie settings:
// - HttpOnly: false - Must be readable by forms (not JS in our case, but forms need it)
// - Secure: configurable - true in production (HTTPS only)
// - SameSite: Strict - Maximum CSRF protection
// - Path: / - Available on all routes
// - MaxAge: 1 hour - Short lifetime for security
func SetCookie(w http.ResponseWriter, token string, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   CookieMaxAge,
		HttpOnly: false, // Must be accessible for form submission
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetTokenFromRequest retrieves the CSRF token from the request cookie.
// Returns empty string if cookie doesn't exist.
func GetTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// =============================================================================
// Handler Helpers
// =============================================================================

// EnsureToken ensures a CSRF token exists for the request.
// If a valid token cookie exists, it returns that token.
// Otherwise, it generates a new token, sets the cookie, and returns it.
//
// This is the main function handlers should use on GET requests.
func EnsureToken(w http.ResponseWriter, r *http.Request, isSecure bool) string {
	// Check for existing token
	existingToken := GetTokenFromRequest(r)
	if existingToken != "" {
		return existingToken
	}

	// Generate new token
	token, err := GenerateToken()
	if err != nil {
		// In the unlikely event of crypto failure, generate a less secure fallback
		// This should essentially never happen
		token = MustGenerateToken()
	}

	// Set cookie
	SetCookie(w, token, isSecure)

	return token
}

// RefreshToken generates a new CSRF token and sets it in the response cookie.
// Use this after successful form submissions to prevent token reuse.
func RefreshToken(w http.ResponseWriter, isSecure bool) string {
	token, err := GenerateToken()
	if err != nil {
		token = MustGenerateToken()
	}
	SetCookie(w, token, isSecure)
	return token
}
