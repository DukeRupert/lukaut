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
	"github.com/DukeRupert/lukaut/internal/templ/pages/settings"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
)

// SettingsHandler handles settings-related HTTP requests.
//
// Routes handled:
// - GET  /settings          -> ShowProfileTempl
// - POST /settings/profile  -> UpdateProfile
// - GET  /settings/password -> ShowPasswordTempl
// - POST /settings/password -> ChangePassword
type SettingsHandler struct {
	userService service.UserService
	logger      *slog.Logger
}

// NewSettingsHandler creates a new SettingsHandler with the required dependencies.
func NewSettingsHandler(
	userService service.UserService,
	logger *slog.Logger,
) *SettingsHandler {
	return &SettingsHandler{
		userService: userService,
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
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"Name":  name,
		"Phone": phone,
	}

	// Validate form fields
	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Name is required"
	} else if len(name) > 255 {
		errors["name"] = "Name must be 255 characters or less"
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
		UserID: user.ID,
		Name:   name,
		Phone:  phone,
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

// renderProfileError re-renders the profile form with errors using templ.
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
			"Name":  user.Name,
			"Phone": user.Phone,
		}
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	var templFlash *shared.Flash
	if flash != nil {
		templFlash = &shared.Flash{
			Type:    shared.FlashType(flash.Type),
			Message: flash.Message,
		}
	}

	data := settings.ProfilePageData{
		CurrentPath: "/settings",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Form: settings.ProfileFormData{
			Name:  formValues["Name"],
			Phone: formValues["Phone"],
		},
		Errors:    errors,
		Flash:     templFlash,
		ActiveTab: settings.TabProfile,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.ProfilePage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render profile error page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

// renderPasswordError re-renders the password form with errors using templ.
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

	var templFlash *shared.Flash
	if flash != nil {
		templFlash = &shared.Flash{
			Type:    shared.FlashType(flash.Type),
			Message: flash.Message,
		}
	}

	data := settings.PasswordPageData{
		CurrentPath: "/settings/password",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Errors:      errors,
		Flash:       templFlash,
		ActiveTab:   settings.TabPassword,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.PasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render password error page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /settings/business - Update Business
// =============================================================================

// UpdateBusiness processes the business settings form submission.
func (h *SettingsHandler) UpdateBusiness(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderBusinessError(w, r, user, nil, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Extract and normalize form values
	businessName := strings.TrimSpace(r.FormValue("business_name"))
	businessEmail := strings.TrimSpace(r.FormValue("business_email"))
	businessPhone := strings.TrimSpace(r.FormValue("business_phone"))
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	addressLine2 := strings.TrimSpace(r.FormValue("address_line2"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	licenseNumber := strings.TrimSpace(r.FormValue("license_number"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"BusinessName":          businessName,
		"BusinessEmail":         businessEmail,
		"BusinessPhone":         businessPhone,
		"BusinessAddressLine1":  addressLine1,
		"BusinessAddressLine2":  addressLine2,
		"BusinessCity":          city,
		"BusinessState":         state,
		"BusinessPostalCode":    postalCode,
		"BusinessLicenseNumber": licenseNumber,
		"BusinessLogoURL":       user.BusinessLogoURL, // Preserve existing logo
	}

	// Validate form fields
	errors := make(map[string]string)

	if len(businessName) > 255 {
		errors["business_name"] = "Business name must be 255 characters or less"
	}

	if len(businessEmail) > 255 {
		errors["business_email"] = "Business email must be 255 characters or less"
	}

	if len(businessPhone) > 50 {
		errors["business_phone"] = "Business phone must be 50 characters or less"
	}

	if len(addressLine1) > 255 {
		errors["address_line1"] = "Street address must be 255 characters or less"
	}

	if len(licenseNumber) > 100 {
		errors["license_number"] = "License number must be 100 characters or less"
	}

	// If validation errors, re-render form
	if len(errors) > 0 {
		h.renderBusinessError(w, r, user, formValues, errors, nil)
		return
	}

	// Call UserService.UpdateBusinessProfile
	err := h.userService.UpdateBusinessProfile(r.Context(), domain.BusinessProfileUpdateParams{
		UserID:        user.ID,
		BusinessName:  businessName,
		BusinessEmail: businessEmail,
		BusinessPhone: businessPhone,
		AddressLine1:  addressLine1,
		AddressLine2:  addressLine2,
		City:          city,
		State:         state,
		PostalCode:    postalCode,
		LicenseNumber: licenseNumber,
		LogoURL:       user.BusinessLogoURL, // Preserve existing logo
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderBusinessError(w, r, user, formValues, nil, &Flash{
				Type:    "error",
				Message: domain.ErrorMessage(err),
			})
		default:
			h.logger.Error("business profile update failed", "error", err, "user_id", user.ID)
			h.renderBusinessError(w, r, user, formValues, nil, &Flash{
				Type:    "error",
				Message: "Failed to update business information. Please try again later.",
			})
		}
		return
	}

	// Log successful update
	h.logger.Info("user business profile updated", "user_id", user.ID)

	// Redirect with success message
	http.Redirect(w, r, "/settings/business?updated=1", http.StatusSeeOther)
}

// renderBusinessError re-renders the business form with errors using templ.
func (h *SettingsHandler) renderBusinessError(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	formValues map[string]string,
	errors map[string]string,
	flash *Flash,
) {
	if formValues == nil {
		formValues = map[string]string{
			"BusinessName":          user.BusinessName,
			"BusinessEmail":         user.BusinessEmail,
			"BusinessPhone":         user.BusinessPhone,
			"BusinessAddressLine1":  user.BusinessAddressLine1,
			"BusinessAddressLine2":  user.BusinessAddressLine2,
			"BusinessCity":          user.BusinessCity,
			"BusinessState":         user.BusinessState,
			"BusinessPostalCode":    user.BusinessPostalCode,
			"BusinessLicenseNumber": user.BusinessLicenseNumber,
			"BusinessLogoURL":       user.BusinessLogoURL,
		}
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	var templFlash *shared.Flash
	if flash != nil {
		templFlash = &shared.Flash{
			Type:    shared.FlashType(flash.Type),
			Message: flash.Message,
		}
	}

	data := settings.BusinessPageData{
		CurrentPath: "/settings/business",
		CSRFToken:   "",
		User:        domainUserToDisplay(user),
		Form: settings.BusinessFormData{
			BusinessName:          formValues["BusinessName"],
			BusinessEmail:         formValues["BusinessEmail"],
			BusinessPhone:         formValues["BusinessPhone"],
			BusinessAddressLine1:  formValues["BusinessAddressLine1"],
			BusinessAddressLine2:  formValues["BusinessAddressLine2"],
			BusinessCity:          formValues["BusinessCity"],
			BusinessState:         formValues["BusinessState"],
			BusinessPostalCode:    formValues["BusinessPostalCode"],
			BusinessLicenseNumber: formValues["BusinessLicenseNumber"],
			BusinessLogoURL:       formValues["BusinessLogoURL"],
		},
		Errors:    errors,
		Flash:     templFlash,
		ActiveTab: settings.TabBusiness,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.BusinessPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render business error page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
