// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements templ-based settings handlers for user profile and password management.
package handler

import (
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/templ/pages/settings"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
)

// =============================================================================
// Templ-based Settings Handlers
// =============================================================================

// ShowProfileTempl renders the profile settings form using templ.
func (h *SettingsHandler) ShowProfileTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check for success flash from query param
	var flash *shared.Flash
	if r.URL.Query().Get("updated") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Profile updated successfully.",
		}
	}

	data := settings.ProfilePageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "", // Add CSRF token if using CSRF middleware
		User:        domainUserToDisplay(user),
		Form: settings.ProfileFormData{
			Name:        user.Name,
			CompanyName: user.CompanyName,
			Phone:       user.Phone,
		},
		Errors:    make(map[string]string),
		Flash:     flash,
		ActiveTab: settings.TabProfile,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.ProfilePage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render profile page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderProfileErrorTempl re-renders the profile form with errors using templ.
func (h *SettingsHandler) renderProfileErrorTempl(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	form *settings.ProfileFormData,
	errors map[string]string,
	flash *shared.Flash,
) {
	if form == nil {
		form = &settings.ProfileFormData{
			Name:        user.Name,
			CompanyName: user.CompanyName,
			Phone:       user.Phone,
		}
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := settings.ProfilePageData{
		CurrentPath: "/settings",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Form:        *form,
		Errors:      errors,
		Flash:       flash,
		ActiveTab:   settings.TabProfile,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.ProfilePage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render profile page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ShowPasswordTempl renders the password change form using templ.
func (h *SettingsHandler) ShowPasswordTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check for success flash from query param
	var flash *shared.Flash
	if r.URL.Query().Get("changed") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Password changed successfully.",
		}
	}

	data := settings.PasswordPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Errors:      make(map[string]string),
		Flash:       flash,
		ActiveTab:   settings.TabPassword,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.PasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render password page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPasswordErrorTempl re-renders the password form with errors using templ.
func (h *SettingsHandler) renderPasswordErrorTempl(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	errors map[string]string,
	flash *shared.Flash,
) {
	if errors == nil {
		errors = make(map[string]string)
	}

	data := settings.PasswordPageData{
		CurrentPath: "/settings/password",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Errors:      errors,
		Flash:       flash,
		ActiveTab:   settings.TabPassword,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.PasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render password page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ShowBusinessTempl renders the business settings form using templ.
func (h *SettingsHandler) ShowBusinessTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check for success flash from query param
	var flash *shared.Flash
	if r.URL.Query().Get("updated") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Business information updated successfully.",
		}
	}

	data := settings.BusinessPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Form:        domainUserToBusinessForm(user),
		Errors:      make(map[string]string),
		Flash:       flash,
		ActiveTab:   settings.TabBusiness,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.BusinessPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render business page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderBusinessErrorTempl re-renders the business form with errors using templ.
func (h *SettingsHandler) renderBusinessErrorTempl(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	form *settings.BusinessFormData,
	errors map[string]string,
	flash *shared.Flash,
) {
	if form == nil {
		businessForm := domainUserToBusinessForm(user)
		form = &businessForm
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := settings.BusinessPageData{
		CurrentPath: "/settings/business",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Form:        *form,
		Errors:      errors,
		Flash:       flash,
		ActiveTab:   settings.TabBusiness,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.BusinessPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render business page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// Templ Route Registration
// =============================================================================

// RegisterTemplRoutes registers templ-based settings routes on the provided ServeMux.
func (h *SettingsHandler) RegisterTemplRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /settings", requireUser(http.HandlerFunc(h.ShowProfileTempl)))
	mux.Handle("POST /settings/profile", requireUser(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("GET /settings/password", requireUser(http.HandlerFunc(h.ShowPasswordTempl)))
	mux.Handle("POST /settings/password", requireUser(http.HandlerFunc(h.ChangePassword)))
	mux.Handle("GET /settings/business", requireUser(http.HandlerFunc(h.ShowBusinessTempl)))
	mux.Handle("POST /settings/business", requireUser(http.HandlerFunc(h.UpdateBusiness)))
}

// =============================================================================
// Helper Functions
// =============================================================================

// domainUserToDisplay converts a domain.User to settings.UserDisplay
func domainUserToDisplay(u *domain.User) *settings.UserDisplay {
	if u == nil {
		return nil
	}
	return &settings.UserDisplay{
		Name:  u.Name,
		Email: u.Email,
	}
}

// domainUserToBusinessForm converts a domain.User to settings.BusinessFormData
func domainUserToBusinessForm(u *domain.User) settings.BusinessFormData {
	return settings.BusinessFormData{
		BusinessName:          u.BusinessName,
		BusinessEmail:         u.BusinessEmail,
		BusinessPhone:         u.BusinessPhone,
		BusinessAddressLine1:  u.BusinessAddressLine1,
		BusinessAddressLine2:  u.BusinessAddressLine2,
		BusinessCity:          u.BusinessCity,
		BusinessState:         u.BusinessState,
		BusinessPostalCode:    u.BusinessPostalCode,
		BusinessLicenseNumber: u.BusinessLicenseNumber,
		BusinessLogoURL:       u.BusinessLogoURL,
	}
}

// flashToSharedFlash converts handler.Flash to shared.Flash
func flashToSharedFlash(f *Flash) *shared.Flash {
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
	default:
		flashType = shared.FlashInfo
	}
	return &shared.Flash{
		Type:    flashType,
		Message: f.Message,
	}
}
