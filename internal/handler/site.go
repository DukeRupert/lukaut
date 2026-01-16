// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements site CRUD handlers for managing construction site locations.
package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/pages/sites"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// SiteListPageData contains data for the sites list page.
type SiteListPageData struct {
	CurrentPath string        // Current URL path
	User        interface{}   // Authenticated user
	Sites       []domain.Site // List of sites
	Flash       *Flash        // Flash message (if any)
	CSRFToken   string        // CSRF token for form protection
}

// SiteFormPageData contains data for the site create/edit form.
type SiteFormPageData struct {
	CurrentPath string            // Current URL path
	User        interface{}       // Authenticated user
	Site        *domain.Site      // Site being edited (nil for create)
	Form        map[string]string // Form field values
	Errors      map[string]string // Field-level validation errors
	Flash       *Flash            // Flash message (if any)
	IsEdit      bool              // true for edit, false for create
	IsModal     bool              // true if rendering in modal
	CSRFToken   string            // CSRF token for form protection
	Clients     []domain.Client   // Available clients for dropdown
}

// =============================================================================
// Handler Configuration
// =============================================================================

// SiteHandler handles site-related HTTP requests.
type SiteHandler struct {
	siteService   service.SiteService
	clientService service.ClientService
	logger        *slog.Logger
}

// NewSiteHandler creates a new SiteHandler.
func NewSiteHandler(
	siteService service.SiteService,
	clientService service.ClientService,
	logger *slog.Logger,
) *SiteHandler {
	return &SiteHandler{
		siteService:   siteService,
		clientService: clientService,
		logger:        logger,
	}
}

// =============================================================================
// POST /sites - Create Site
// =============================================================================

// Create processes the site creation form.
func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("create handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderFormError(w, r, nil, "Invalid form data")
		return
	}

	isModal := r.URL.Query().Get("modal") == "true"

	// Extract form values
	formValues := map[string]string{
		"name":          strings.TrimSpace(r.FormValue("name")),
		"address_line1": strings.TrimSpace(r.FormValue("address_line1")),
		"address_line2": strings.TrimSpace(r.FormValue("address_line2")),
		"city":          strings.TrimSpace(r.FormValue("city")),
		"state":         strings.TrimSpace(r.FormValue("state")),
		"postal_code":   strings.TrimSpace(r.FormValue("postal_code")),
		"client_id":     strings.TrimSpace(r.FormValue("client_id")),
		"notes":         strings.TrimSpace(r.FormValue("notes")),
	}

	// Validate
	errors := h.validateSiteForm(formValues)
	if len(errors) > 0 {
		h.renderFormWithErrors(w, r, nil, formValues, errors, isModal)
		return
	}

	// Parse client_id
	var clientID *uuid.UUID
	if formValues["client_id"] != "" {
		parsed, err := uuid.Parse(formValues["client_id"])
		if err == nil {
			clientID = &parsed
		}
	}

	// Create site
	site, err := h.siteService.Create(r.Context(), domain.CreateSiteParams{
		UserID:       user.ID,
		Name:         formValues["name"],
		AddressLine1: formValues["address_line1"],
		AddressLine2: formValues["address_line2"],
		City:         formValues["city"],
		State:        formValues["state"],
		PostalCode:   formValues["postal_code"],
		Notes:        formValues["notes"],
		ClientID:     clientID,
	})
	if err != nil {
		h.logger.Error("failed to create site", "error", err)
		code := domain.ErrorCode(err)
		if code == domain.EINVALID {
			errors["_form"] = domain.ErrorMessage(err)
			h.renderFormWithErrors(w, r, nil, formValues, errors, isModal)
			return
		}
		h.renderFormError(w, r, nil, "Failed to create site. Please try again.")
		return
	}

	h.logger.Info("site created", "site_id", site.ID, "user_id", user.ID)

	// If modal request, return the new site option for the dropdown
	if isModal {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "siteCreated")
		// Return an option element for the dropdown
		optionHTML := `<option value="` + site.ID.String() + `" selected>` + site.Name + `</option>`
		w.Write([]byte(optionHTML))
		return
	}

	// Redirect to sites list with success message
	http.Redirect(w, r, "/sites", http.StatusSeeOther)
}

// =============================================================================
// PUT /sites/{id} - Update Site
// =============================================================================

// Update processes the site update form.
func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("update handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse site ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("invalid site ID", "id", idStr, "error", err)
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}

	// Fetch site first to verify ownership
	site, err := h.siteService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
			return
		}
		h.logger.Error("failed to get site for update", "error", err, "site_id", id)
		h.renderError(w, r, "Failed to load site. Please try again.")
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderFormError(w, r, site, "Invalid form data")
		return
	}

	// Extract form values
	formValues := map[string]string{
		"name":          strings.TrimSpace(r.FormValue("name")),
		"address_line1": strings.TrimSpace(r.FormValue("address_line1")),
		"address_line2": strings.TrimSpace(r.FormValue("address_line2")),
		"city":          strings.TrimSpace(r.FormValue("city")),
		"state":         strings.TrimSpace(r.FormValue("state")),
		"postal_code":   strings.TrimSpace(r.FormValue("postal_code")),
		"client_id":     strings.TrimSpace(r.FormValue("client_id")),
		"notes":         strings.TrimSpace(r.FormValue("notes")),
	}

	// Validate
	errors := h.validateSiteForm(formValues)
	if len(errors) > 0 {
		h.renderFormWithErrors(w, r, site, formValues, errors, false)
		return
	}

	// Parse client_id
	var clientID *uuid.UUID
	if formValues["client_id"] != "" {
		parsed, err := uuid.Parse(formValues["client_id"])
		if err == nil {
			clientID = &parsed
		}
	}

	// Update site
	err = h.siteService.Update(r.Context(), domain.UpdateSiteParams{
		ID:           id,
		UserID:       user.ID,
		Name:         formValues["name"],
		AddressLine1: formValues["address_line1"],
		AddressLine2: formValues["address_line2"],
		City:         formValues["city"],
		State:        formValues["state"],
		PostalCode:   formValues["postal_code"],
		Notes:        formValues["notes"],
		ClientID:     clientID,
	})
	if err != nil {
		h.logger.Error("failed to update site", "error", err, "site_id", id)
		code := domain.ErrorCode(err)
		if code == domain.EINVALID {
			errors["_form"] = domain.ErrorMessage(err)
			h.renderFormWithErrors(w, r, site, formValues, errors, false)
			return
		}
		h.renderFormError(w, r, site, "Failed to update site. Please try again.")
		return
	}

	h.logger.Info("site updated", "site_id", id, "user_id", user.ID)

	// Redirect to sites list
	http.Redirect(w, r, "/sites", http.StatusSeeOther)
}

// =============================================================================
// DELETE /sites/{id} - Delete Site
// =============================================================================

// Delete removes a site.
func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("delete handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse site ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Error("invalid site ID", "id", idStr, "error", err)
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}

	// Delete site
	err = h.siteService.Delete(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			// Already deleted or doesn't exist, redirect to list
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
			return
		}
		h.logger.Error("failed to delete site", "error", err, "site_id", id)
		// For htmx requests, return error toast via HX-Trigger
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Trigger", `{"showToast": {"type": "error", "message": "Failed to delete site. Please try again."}}`)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}

	h.logger.Info("site deleted", "site_id", id, "user_id", user.ID)

	// For htmx requests, trigger a refresh
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/sites")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/sites", http.StatusSeeOther)
}

// =============================================================================
// Helper Methods
// =============================================================================

// validateSiteForm validates the site form fields.
func (h *SiteHandler) validateSiteForm(form map[string]string) map[string]string {
	errors := make(map[string]string)

	if form["name"] == "" {
		errors["name"] = "Site name is required"
	} else if len(form["name"]) > 200 {
		errors["name"] = "Site name must be 200 characters or less"
	}

	if form["address_line1"] == "" {
		errors["address_line1"] = "Address is required"
	}

	if form["city"] == "" {
		errors["city"] = "City is required"
	}

	if form["state"] == "" {
		errors["state"] = "State is required"
	}

	if form["postal_code"] == "" {
		errors["postal_code"] = "Postal code is required"
	}

	return errors
}

// renderError renders a generic error page using templ.
func (h *SiteHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
	data := sites.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Sites:       []sites.DisplaySite{},
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sites.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render sites index error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderFormError renders the form with a generic error using templ.
func (h *SiteHandler) renderFormError(w http.ResponseWriter, r *http.Request, site *domain.Site, message string) {
	user := auth.GetUserFromRequest(r)

	isEdit := site != nil

	// Fetch clients for dropdown
	clientList, _ := h.clientService.ListAll(r.Context(), user.ID)

	data := sites.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Site:        domainSiteToDetail(site),
		Form:        sites.SiteFormValues{},
		Errors:      make(map[string]string),
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
		IsEdit:  isEdit,
		Clients: domainClientsToOptions(clientList),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var renderErr error
	if isEdit {
		renderErr = sites.EditPage(data).Render(r.Context(), w)
	} else {
		renderErr = sites.NewPage(data).Render(r.Context(), w)
	}
	if renderErr != nil {
		h.logger.Error("failed to render form error", "error", renderErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderFormWithErrors renders the form with validation errors using templ.
func (h *SiteHandler) renderFormWithErrors(w http.ResponseWriter, r *http.Request, site *domain.Site, form map[string]string, errors map[string]string, isModal bool) {
	user := auth.GetUserFromRequest(r)

	isEdit := site != nil

	// Fetch clients for dropdown
	clientList, _ := h.clientService.ListAll(r.Context(), user.ID)

	// Convert form map to SiteFormValues
	siteForm := sites.SiteFormValues{
		Name:         form["name"],
		AddressLine1: form["address_line1"],
		AddressLine2: form["address_line2"],
		City:         form["city"],
		State:        form["state"],
		PostalCode:   form["postal_code"],
		ClientID:     form["client_id"],
		Notes:        form["notes"],
	}

	data := sites.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Site:        domainSiteToDetail(site),
		Form:        siteForm,
		Errors:      errors,
		Flash:       nil,
		IsEdit:      isEdit,
		Clients:     domainClientsToOptions(clientList),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var renderErr error
	if isEdit {
		renderErr = sites.EditPage(data).Render(r.Context(), w)
	} else {
		renderErr = sites.NewPage(data).Render(r.Context(), w)
	}
	if renderErr != nil {
		h.logger.Error("failed to render form with errors", "error", renderErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
