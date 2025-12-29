// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements client CRUD handlers for managing construction
// company clients.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// ClientListPageData contains data for the client list page.
type ClientListPageData struct {
	CurrentPath string          // Current URL path
	User        interface{}     // Authenticated user
	Clients     []domain.Client // List of clients
	Pagination  PaginationData  // Pagination information
	Flash       *Flash          // Flash message (if any)
	CSRFToken   string          // CSRF token for form protection
}

// ClientFormPageData contains data for the client create/edit form.
type ClientFormPageData struct {
	CurrentPath string            // Current URL path
	User        interface{}       // Authenticated user
	Client      *domain.Client    // Client being edited (nil for create)
	Form        map[string]string // Form field values
	Errors      map[string]string // Field-level validation errors
	Flash       *Flash            // Flash message (if any)
	IsEdit      bool              // true for edit, false for create
	CSRFToken   string            // CSRF token for form protection
}

// ClientShowPageData contains data for the client detail page.
type ClientShowPageData struct {
	CurrentPath string         // Current URL path
	User        interface{}    // Authenticated user
	Client      *domain.Client // Client details
	Flash       *Flash         // Flash message (if any)
	CSRFToken   string         // CSRF token for form protection
}

// =============================================================================
// Handler Configuration
// =============================================================================

// ClientHandler handles client-related HTTP requests.
type ClientHandler struct {
	clientService service.ClientService
	renderer      TemplateRenderer
	logger        *slog.Logger
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(
	clientService service.ClientService,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *ClientHandler {
	return &ClientHandler{
		clientService: clientService,
		renderer:      renderer,
		logger:        logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all client routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - GET  /clients          -> Index (list)
// - GET  /clients/new      -> New (create form)
// - POST /clients          -> Create
// - GET  /clients/{id}     -> Show
// - GET  /clients/{id}/edit -> Edit
// - PUT  /clients/{id}     -> Update
// - DELETE /clients/{id}   -> Delete
func (h *ClientHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /clients", requireUser(http.HandlerFunc(h.Index)))
	mux.Handle("GET /clients/new", requireUser(http.HandlerFunc(h.New)))
	mux.Handle("POST /clients", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /clients/{id}", requireUser(http.HandlerFunc(h.Show)))
	mux.Handle("GET /clients/{id}/edit", requireUser(http.HandlerFunc(h.Edit)))
	mux.Handle("PUT /clients/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /clients/{id}", requireUser(http.HandlerFunc(h.Delete)))
}

// =============================================================================
// GET /clients - List Clients
// =============================================================================

// Index displays a paginated list of clients.
func (h *ClientHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("index handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := int32(20)
	offset := int32((page - 1) * int(perPage))

	// Fetch clients
	result, err := h.clientService.List(r.Context(), domain.ListClientsParams{
		UserID: user.ID,
		Limit:  perPage,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list clients", "error", err, "user_id", user.ID)
		h.renderError(w, r, "Failed to load clients. Please try again.")
		return
	}

	// Build pagination data
	pagination := buildClientPaginationData(result)

	// Render template
	data := ClientListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Clients:     result.Clients,
		Pagination:  pagination,
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "clients/index", data)
}

// =============================================================================
// GET /clients/new - Show Create Form
// =============================================================================

// New displays the client creation form.
func (h *ClientHandler) New(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("new handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Render form with empty values
	data := ClientFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Client:      nil,
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      false,
	}

	h.renderer.RenderHTTP(w, "clients/new", data)
}

// =============================================================================
// POST /clients - Create Client
// =============================================================================

// Create processes the client creation form.
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("create handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderFormError(w, r, user, nil, nil, nil, "Invalid form submission.", false)
		return
	}

	// Extract form values
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	addressLine2 := strings.TrimSpace(r.FormValue("address_line2"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"name":          name,
		"email":         email,
		"phone":         phone,
		"address_line1": addressLine1,
		"address_line2": addressLine2,
		"city":          city,
		"state":         state,
		"postal_code":   postalCode,
		"notes":         notes,
	}

	// Create client
	params := domain.CreateClientParams{
		UserID:       user.ID,
		Name:         name,
		Email:        email,
		Phone:        phone,
		AddressLine1: addressLine1,
		AddressLine2: addressLine2,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Notes:        notes,
	}

	client, err := h.clientService.Create(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderFormError(w, r, user, formValues, nil, nil, domain.ErrorMessage(err), false)
		default:
			h.logger.Error("failed to create client", "error", err, "user_id", user.ID)
			h.renderFormError(w, r, user, formValues, nil, nil, "Failed to create client. Please try again.", false)
		}
		return
	}

	// Redirect to client detail page
	http.Redirect(w, r, fmt.Sprintf("/clients/%s", client.ID), http.StatusSeeOther)
}

// =============================================================================
// GET /clients/{id} - Show Client
// =============================================================================

// Show displays client details.
func (h *ClientHandler) Show(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("show handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse client ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Fetch client
	client, err := h.clientService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get client", "error", err, "client_id", id)
			h.renderError(w, r, "Failed to load client. Please try again.")
		}
		return
	}

	// Render template
	data := ClientShowPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Client:      client,
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "clients/show", data)
}

// =============================================================================
// GET /clients/{id}/edit - Show Edit Form
// =============================================================================

// Edit displays the client edit form.
func (h *ClientHandler) Edit(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("edit handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse client ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Fetch client
	client, err := h.clientService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get client", "error", err, "client_id", id)
			h.renderError(w, r, "Failed to load client. Please try again.")
		}
		return
	}

	// Populate form with client data
	formValues := map[string]string{
		"name":          client.Name,
		"email":         client.Email,
		"phone":         client.Phone,
		"address_line1": client.AddressLine1,
		"address_line2": client.AddressLine2,
		"city":          client.City,
		"state":         client.State,
		"postal_code":   client.PostalCode,
		"notes":         client.Notes,
	}

	// Render form
	data := ClientFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Client:      client,
		Form:        formValues,
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      true,
	}

	h.renderer.RenderHTTP(w, "clients/edit", data)
}

// =============================================================================
// PUT /clients/{id} - Update Client
// =============================================================================

// Update processes the client update form.
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("update handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse client ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderError(w, r, "Invalid form submission.")
		return
	}

	// Extract form values
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	addressLine2 := strings.TrimSpace(r.FormValue("address_line2"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"name":          name,
		"email":         email,
		"phone":         phone,
		"address_line1": addressLine1,
		"address_line2": addressLine2,
		"city":          city,
		"state":         state,
		"postal_code":   postalCode,
		"notes":         notes,
	}

	// Fetch current client for re-rendering on error
	client, err := h.clientService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Update client
	params := domain.UpdateClientParams{
		ID:           id,
		UserID:       user.ID,
		Name:         name,
		Email:        email,
		Phone:        phone,
		AddressLine1: addressLine1,
		AddressLine2: addressLine2,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Notes:        notes,
	}

	err = h.clientService.Update(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderFormError(w, r, user, formValues, nil, client, domain.ErrorMessage(err), true)
		case domain.ENOTFOUND:
			NotFoundResponse(w, r, h.logger)
		default:
			h.logger.Error("failed to update client", "error", err, "client_id", id)
			h.renderFormError(w, r, user, formValues, nil, client, "Failed to update client. Please try again.", true)
		}
		return
	}

	// Check if this is an htmx request
	if r.Header.Get("HX-Request") == "true" {
		// For htmx, redirect via HX-Redirect header
		w.Header().Set("HX-Redirect", fmt.Sprintf("/clients/%s", id))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Regular form submission - redirect
	http.Redirect(w, r, fmt.Sprintf("/clients/%s", id), http.StatusSeeOther)
}

// =============================================================================
// DELETE /clients/{id} - Delete Client
// =============================================================================

// Delete deletes a client.
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("delete handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse client ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Delete client
	err = h.clientService.Delete(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			NotFoundResponse(w, r, h.logger)
		case domain.EINVALID:
			// Client has associated sites
			h.logger.Warn("cannot delete client with sites", "client_id", id)
			h.renderError(w, r, domain.ErrorMessage(err))
		default:
			h.logger.Error("failed to delete client", "error", err, "client_id", id)
			h.renderError(w, r, "Failed to delete client. Please try again.")
		}
		return
	}

	// For htmx requests, redirect via HX-Redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/clients")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Regular request - redirect
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildClientPaginationData builds pagination data from a client list result.
func buildClientPaginationData(result *domain.ListClientsResult) PaginationData {
	currentPage := result.CurrentPage()
	totalPages := result.TotalPages()

	return PaginationData{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		PerPage:     int(result.Limit),
		Total:       int(result.Total),
		HasPrevious: result.HasPrevious(),
		HasNext:     result.HasMore(),
		PrevPage:    currentPage - 1,
		NextPage:    currentPage + 1,
	}
}

// renderError renders a generic error page.
func (h *ClientHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
	data := ClientListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Clients:     []domain.Client{},
		Pagination:  PaginationData{},
		Flash: &Flash{
			Type:    "error",
			Message: message,
		},
	}
	h.renderer.RenderHTTP(w, "clients/index", data)
}

// renderFormError re-renders the form with errors.
func (h *ClientHandler) renderFormError(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	formValues map[string]string,
	fieldErrors map[string]string,
	client *domain.Client,
	flashMessage string,
	isEdit bool,
) {
	if formValues == nil {
		formValues = make(map[string]string)
	}
	if fieldErrors == nil {
		fieldErrors = make(map[string]string)
	}

	var flash *Flash
	if flashMessage != "" {
		flash = &Flash{
			Type:    "error",
			Message: flashMessage,
		}
	}

	template := "clients/new"
	if isEdit {
		template = "clients/edit"
	}

	data := ClientFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Client:      client,
		Form:        formValues,
		Errors:      fieldErrors,
		Flash:       flash,
		IsEdit:      isEdit,
	}

	h.renderer.RenderHTTP(w, template, data)
}
