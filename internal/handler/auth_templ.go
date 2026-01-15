// Package handler contains templ-based HTTP handlers for authentication.
//
// This file provides templ alternatives to the Go template-based auth handlers.
// These handlers can be used via parallel routes (e.g., /login-templ) for testing
// before migrating completely.
package handler

import (
	"net/http"

	authpages "github.com/DukeRupert/lukaut/internal/templ/pages/auth"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
)

// =============================================================================
// Templ Auth Handlers
// =============================================================================

// ShowLoginTempl renders the login page using templ.
// This is a parallel route for testing the templ migration.
func (h *AuthHandler) ShowLoginTempl(w http.ResponseWriter, r *http.Request) {
	// Check for success query params
	var flash *shared.Flash
	if r.URL.Query().Get("registered") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Account created successfully! Please sign in.",
		}
	} else if r.URL.Query().Get("reset") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Password reset successfully! Please sign in with your new password.",
		}
	} else if r.URL.Query().Get("logout") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "You have been signed out.",
		}
	}

	// Get return_to from query params
	returnTo := r.URL.Query().Get("return_to")

	data := authpages.LoginPageData{
		Form:      authpages.FormData{},
		Errors:    make(map[string]string),
		Flash:     flash,
		CSRFToken: "", // TODO: Set CSRF token
		ReturnTo:  returnTo,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := authpages.LoginPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render login templ page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ShowRegisterTempl renders the registration page using templ.
// This is a parallel route for testing the templ migration.
func (h *AuthHandler) ShowRegisterTempl(w http.ResponseWriter, r *http.Request) {
	// Get return_to from query params for post-registration redirect
	returnTo := r.URL.Query().Get("return_to")

	data := authpages.RegisterPageData{
		Form:               authpages.FormData{},
		Errors:             make(map[string]string),
		Flash:              nil,
		CSRFToken:          "", // TODO: Set CSRF token
		ReturnTo:           returnTo,
		InviteCodesEnabled: h.inviteValidator.IsEnabled(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := authpages.RegisterPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render register templ page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RegisterTemplRoutes registers the templ-based auth routes on the provided ServeMux.
// These routes are parallel to the existing routes for testing the migration.
//
// Routes registered:
// - GET /login-templ    -> ShowLoginTempl
// - GET /register-templ -> ShowRegisterTempl
//
// Usage in main.go:
//
//	authHandler.RegisterTemplRoutes(mux)
func (h *AuthHandler) RegisterTemplRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login-templ", h.ShowLoginTempl)
	mux.HandleFunc("GET /register-templ", h.ShowRegisterTempl)
}

// =============================================================================
// Helper functions for converting handler data to templ data types
// =============================================================================

// convertFlashToTempl converts the handler Flash type to the templ shared.Flash type.
func convertFlashToTempl(f *Flash) *shared.Flash {
	if f == nil {
		return nil
	}

	var flashType shared.FlashType
	switch f.Type {
	case "error":
		flashType = shared.FlashError
	case "success":
		flashType = shared.FlashSuccess
	case "warning":
		flashType = shared.FlashWarning
	case "info":
		flashType = shared.FlashInfo
	default:
		flashType = shared.FlashInfo
	}

	return &shared.Flash{
		Type:    flashType,
		Message: f.Message,
	}
}

// convertFormToTempl converts the handler form map to templ FormData.
func convertFormToTempl(form map[string]string) authpages.FormData {
	if form == nil {
		return authpages.FormData{}
	}
	return authpages.FormData{
		Email:       form["Email"],
		Name:        form["Name"],
		CompanyName: form["CompanyName"],
		InviteCode:  form["InviteCode"],
	}
}
