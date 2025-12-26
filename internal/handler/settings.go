// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements settings handlers for user profile and password management.
package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
)

// SettingsHandler handles settings-related HTTP requests.
//
// Routes handled:
// - GET  /settings          -> ShowProfile
// - POST /settings/profile  -> UpdateProfile
// - GET  /settings/password -> ShowPassword
// - POST /settings/password -> ChangePassword
type SettingsHandler struct {
	userService service.UserService
	renderer    TemplateRenderer
	logger      *slog.Logger
}

// NewSettingsHandler creates a new SettingsHandler with the required dependencies.
func NewSettingsHandler(
	userService service.UserService,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *SettingsHandler {
	return &SettingsHandler{
		userService: userService,
		renderer:    renderer,
		logger:      logger,
	}
}

// SettingsPageData contains data for settings pages.
type SettingsPageData struct {
	CurrentPath string
	User        *domain.User
	CSRFToken   string
	Form        map[string]string
	Errors      map[string]string
	Flash       *Flash
	ActiveTab   string // "profile" or "password"
}

// =============================================================================
// GET /settings - Show Profile Form
// =============================================================================

// ShowProfile renders the profile settings form.
func (h *SettingsHandler) ShowProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check for success flash from query param
	var flash *Flash
	if r.URL.Query().Get("updated") == "1" {
		flash = &Flash{
			Type:    "success",
			Message: "Profile updated successfully.",
		}
	}

	data := SettingsPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		CSRFToken:   "",
		Form: map[string]string{
			"Name":        user.Name,
			"CompanyName": user.CompanyName,
			"Phone":       user.Phone,
		},
		Errors:    make(map[string]string),
		Flash:     flash,
		ActiveTab: "profile",
	}

	h.renderer.RenderHTTP(w, "settings/profile", data)
}

// =============================================================================
// POST /settings/profile - Update Profile
// =============================================================================

// UpdateProfile processes the profile update form submission.
func (h *SettingsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderProfileError(w, r, user, nil, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Extract and normalize form values
	name := strings.TrimSpace(r.FormValue("name"))
	companyName := strings.TrimSpace(r.FormValue("company_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"Name":        name,
		"CompanyName": companyName,
		"Phone":       phone,
	}

	// Validate form fields
	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Name is required"
	} else if len(name) > 255 {
		errors["name"] = "Name must be 255 characters or less"
	}

	if len(companyName) > 255 {
		errors["company_name"] = "Company name must be 255 characters or less"
	}

	if len(phone) > 50 {
		errors["phone"] = "Phone must be 50 characters or less"
	}

	// If validation errors, re-render form
	if len(errors) > 0 {
		h.renderProfileError(w, r, user, formValues, errors, nil)
		return
	}

	// Call UserService.UpdateProfile
	err := h.userService.UpdateProfile(r.Context(), domain.ProfileUpdateParams{
		UserID:      user.ID,
		Name:        name,
		CompanyName: companyName,
		Phone:       phone,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderProfileError(w, r, user, formValues, nil, &Flash{
				Type:    "error",
				Message: domain.ErrorMessage(err),
			})
		default:
			h.logger.Error("profile update failed", "error", err, "user_id", user.ID)
			h.renderProfileError(w, r, user, formValues, nil, &Flash{
				Type:    "error",
				Message: "Failed to update profile. Please try again later.",
			})
		}
		return
	}

	// Log successful update
	h.logger.Info("user profile updated", "user_id", user.ID)

	// Redirect with success message
	http.Redirect(w, r, "/settings?updated=1", http.StatusSeeOther)
}

// renderProfileError re-renders the profile form with errors.
func (h *SettingsHandler) renderProfileError(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	formValues map[string]string,
	errors map[string]string,
	flash *Flash,
) {
	if formValues == nil {
		formValues = map[string]string{
			"Name":        user.Name,
			"CompanyName": user.CompanyName,
			"Phone":       user.Phone,
		}
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := SettingsPageData{
		CurrentPath: "/settings",
		User:        user,
		CSRFToken:   "",
		Form:        formValues,
		Errors:      errors,
		Flash:       flash,
		ActiveTab:   "profile",
	}

	h.renderer.RenderHTTP(w, "settings/profile", data)
}

// =============================================================================
// GET /settings/password - Show Password Form
// =============================================================================

// ShowPassword renders the password change form.
func (h *SettingsHandler) ShowPassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check for success flash from query param
	var flash *Flash
	if r.URL.Query().Get("changed") == "1" {
		flash = &Flash{
			Type:    "success",
			Message: "Password changed successfully.",
		}
	}

	data := SettingsPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		CSRFToken:   "",
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       flash,
		ActiveTab:   "password",
	}

	h.renderer.RenderHTTP(w, "settings/password", data)
}

// =============================================================================
// POST /settings/password - Change Password
// =============================================================================

// ChangePassword processes the password change form submission.
func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderPasswordError(w, r, user, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Extract form values
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate form fields
	errors := make(map[string]string)

	if currentPassword == "" {
		errors["current_password"] = "Current password is required"
	}

	if newPassword == "" {
		errors["new_password"] = "New password is required"
	} else if len(newPassword) < 8 {
		errors["new_password"] = "Password must be at least 8 characters"
	}

	if confirmPassword == "" {
		errors["confirm_password"] = "Please confirm your new password"
	} else if newPassword != confirmPassword {
		errors["confirm_password"] = "Passwords do not match"
	}

	// If validation errors, re-render form
	if len(errors) > 0 {
		h.renderPasswordError(w, r, user, errors, nil)
		return
	}

	// Call UserService.ChangePassword
	err := h.userService.ChangePassword(r.Context(), domain.PasswordChangeParams{
		UserID:          user.ID,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EUNAUTHORIZED:
			errors["current_password"] = "Current password is incorrect"
			h.renderPasswordError(w, r, user, errors, nil)
		case domain.EINVALID:
			h.renderPasswordError(w, r, user, nil, &Flash{
				Type:    "error",
				Message: domain.ErrorMessage(err),
			})
		default:
			h.logger.Error("password change failed", "error", err, "user_id", user.ID)
			h.renderPasswordError(w, r, user, nil, &Flash{
				Type:    "error",
				Message: "Failed to change password. Please try again later.",
			})
		}
		return
	}

	// Log successful password change
	h.logger.Info("user password changed", "user_id", user.ID)

	// Password change invalidates all sessions, so redirect to login
	http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
}

// renderPasswordError re-renders the password form with errors.
func (h *SettingsHandler) renderPasswordError(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	errors map[string]string,
	flash *Flash,
) {
	if errors == nil {
		errors = make(map[string]string)
	}

	data := SettingsPageData{
		CurrentPath: "/settings/password",
		User:        user,
		CSRFToken:   "",
		Form:        make(map[string]string), // Never re-populate password fields
		Errors:      errors,
		Flash:       flash,
		ActiveTab:   "password",
	}

	h.renderer.RenderHTTP(w, "settings/password", data)
}

// =============================================================================
// Route Registration Helper
// =============================================================================

// RegisterRoutes registers all settings routes on the provided ServeMux.
func (h *SettingsHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /settings", requireUser(http.HandlerFunc(h.ShowProfile)))
	mux.Handle("POST /settings/profile", requireUser(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("GET /settings/password", requireUser(http.HandlerFunc(h.ShowPassword)))
	mux.Handle("POST /settings/password", requireUser(http.HandlerFunc(h.ChangePassword)))
}
