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
	renderer      TemplateRenderer
	logger        *slog.Logger
}

// NewSiteHandler creates a new SiteHandler.
func NewSiteHandler(
	siteService service.SiteService,
	clientService service.ClientService,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *SiteHandler {
	return &SiteHandler{
		siteService:   siteService,
		clientService: clientService,
		renderer:      renderer,
		logger:        logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all site routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - GET  /sites          -> Index (list)
// - GET  /sites/new      -> New (create form)
// - POST /sites          -> Create
// - GET  /sites/{id}/edit -> Edit
// - PUT  /sites/{id}     -> Update
// - DELETE /sites/{id}   -> Delete
func (h *SiteHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /sites", requireUser(http.HandlerFunc(h.Index)))
	mux.Handle("GET /sites/new", requireUser(http.HandlerFunc(h.New)))
	mux.Handle("POST /sites", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /sites/{id}/edit", requireUser(http.HandlerFunc(h.Edit)))
	mux.Handle("PUT /sites/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /sites/{id}", requireUser(http.HandlerFunc(h.Delete)))
}

// =============================================================================
// GET /sites - List Sites
// =============================================================================

// Index displays a list of all sites for the current user.
func (h *SiteHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("index handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch sites
	sites, err := h.siteService.List(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list sites", "error", err, "user_id", user.ID)
		h.renderError(w, r, "Failed to load sites. Please try again.")
		return
	}

	data := SiteListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Sites:       sites,
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "sites/index", data)
}

// =============================================================================
// GET /sites/new - Show Create Form
// =============================================================================

// New displays the site creation form.
func (h *SiteHandler) New(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("new handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	isModal := r.URL.Query().Get("modal") == "true"

	// Fetch clients for dropdown
	clients, err := h.clientService.ListAll(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list clients for site form", "error", err, "user_id", user.ID)
		clients = []domain.Client{} // Continue with empty list
	}

	data := SiteFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Site:        nil,
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      false,
		IsModal:     isModal,
		Clients:     clients,
	}

	if isModal {
		h.renderer.RenderPartial(w, "site_form", data)
	} else {
		h.renderer.RenderHTTP(w, "sites/new", data)
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
		h.renderer.RenderPartial(w, "site_select_option", site)
		return
	}

	// Redirect to sites list with success message
	http.Redirect(w, r, "/sites", http.StatusSeeOther)
}

// =============================================================================
// GET /sites/{id}/edit - Show Edit Form
// =============================================================================

// Edit displays the site edit form.
func (h *SiteHandler) Edit(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("edit handler called without authenticated user")
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

	// Fetch site
	site, err := h.siteService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
			return
		}
		h.logger.Error("failed to get site", "error", err, "site_id", id)
		h.renderError(w, r, "Failed to load site. Please try again.")
		return
	}

	// Fetch clients for dropdown
	clients, err := h.clientService.ListAll(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list clients for site form", "error", err, "user_id", user.ID)
		clients = []domain.Client{} // Continue with empty list
	}

	// Populate form values from site
	clientIDStr := ""
	if site.ClientID != nil {
		clientIDStr = site.ClientID.String()
	}
	formValues := map[string]string{
		"name":          site.Name,
		"address_line1": site.AddressLine1,
		"address_line2": site.AddressLine2,
		"city":          site.City,
		"state":         site.State,
		"postal_code":   site.PostalCode,
		"client_id":     clientIDStr,
		"notes":         site.Notes,
	}

	data := SiteFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Site:        site,
		Form:        formValues,
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      true,
		IsModal:     false,
		Clients:     clients,
	}

	h.renderer.RenderHTTP(w, "sites/edit", data)
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
		// For htmx requests, return error toast
		if r.Header.Get("HX-Request") == "true" {
			h.renderer.RenderHTTPWithToast(w, "sites/index", nil, ToastData{
				Type:    "error",
				Message: "Failed to delete site. Please try again.",
			})
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

// renderError renders a generic error page.
func (h *SiteHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
	data := SiteListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Sites:       nil,
		Flash: &Flash{
			Type:    "error",
			Message: message,
		},
	}
	h.renderer.RenderHTTP(w, "sites/index", data)
}

// renderFormError renders the form with a generic error.
func (h *SiteHandler) renderFormError(w http.ResponseWriter, r *http.Request, site *domain.Site, message string) {
	user := auth.GetUserFromRequest(r)

	template := "sites/new"
	isEdit := false
	if site != nil {
		template = "sites/edit"
		isEdit = true
	}

	// Fetch clients for dropdown
	clients, _ := h.clientService.ListAll(r.Context(), user.ID)

	data := SiteFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Site:        site,
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash: &Flash{
			Type:    "error",
			Message: message,
		},
		IsEdit:  isEdit,
		IsModal: false,
		Clients: clients,
	}
	h.renderer.RenderHTTP(w, template, data)
}

// renderFormWithErrors renders the form with validation errors.
func (h *SiteHandler) renderFormWithErrors(w http.ResponseWriter, r *http.Request, site *domain.Site, form map[string]string, errors map[string]string, isModal bool) {
	user := auth.GetUserFromRequest(r)

	template := "sites/new"
	isEdit := false
	if site != nil {
		template = "sites/edit"
		isEdit = true
	}

	// Fetch clients for dropdown
	clients, _ := h.clientService.ListAll(r.Context(), user.ID)

	data := SiteFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Site:        site,
		Form:        form,
		Errors:      errors,
		Flash:       nil,
		IsEdit:      isEdit,
		IsModal:     isModal,
		Clients:     clients,
	}

	if isModal {
		h.renderer.RenderPartial(w, "site_form", data)
	} else {
		h.renderer.RenderHTTP(w, template, data)
	}
}
