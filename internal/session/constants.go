// Package session provides shared session constants used by both
// the handler and middleware packages.
package session

const (
	// CookieName is the name of the cookie that stores the session token.
	CookieName = "lukaut_session"

	// CookiePath ensures the cookie is sent with all requests.
	CookiePath = "/"

	// CookieMaxAge sets the cookie expiration (7 days = 604800 seconds).
	// This should match SessionDuration in the user service.
	CookieMaxAge = 7 * 24 * 60 * 60
)
