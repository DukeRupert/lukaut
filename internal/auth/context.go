// Package auth provides authentication context helpers.
//
// This package is designed to be imported by both middleware and handler
// packages without causing import cycles.
package auth

import (
	"context"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/domain"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey is the key used to store the authenticated user in context.
	userContextKey contextKey = "user"
)

// GetUser retrieves the authenticated user from the context.
//
// Returns nil if no user is authenticated.
//
// Usage:
//
//	user := auth.GetUser(r.Context())
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

// GetUserFromRequest retrieves the authenticated user from the request context.
//
// This is a convenience wrapper around GetUser that takes the request directly.
func GetUserFromRequest(r *http.Request) *domain.User {
	return GetUser(r.Context())
}

// SetUser stores a user in the context.
//
// This is typically called by authentication middleware after validating
// a session token.
func SetUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}
