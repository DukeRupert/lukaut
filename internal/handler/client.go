// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements client CRUD handlers for managing construction
// company clients.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/pages/clients"
	"github.com/DukeRupert/lukaut/internal/templ/partials"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
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
	logger        *slog.Logger
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(
	clientService service.ClientService,
	logger *slog.Logger,
) *ClientHandler {
	return &ClientHandler{
		clientService: clientService,
		logger:        logger,
	}
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
// GET /clients/quick-form - Quick Create Form Partial
// =============================================================================

// QuickCreateForm returns the quick client creation form partial.
func (h *ClientHandler) QuickCreateForm(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := partials.QuickClientFormData{
		Form:   partials.QuickClientFormValues{},
		Errors: make(map[string]string),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.QuickClientForm(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render quick client form", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /clients/quick - Quick Create Client
// =============================================================================

// QuickCreate creates a client via HTMX and returns updated select options.
func (h *ClientHandler) QuickCreate(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Company name is required"
	}
	if email != "" && !isValidEmail(email) {
		errors["email"] = "Invalid email address"
	}

	if len(errors) > 0 {
		// Re-render form with errors
		data := partials.QuickClientFormData{
			Form: partials.QuickClientFormValues{
				Name:  name,
				Email: email,
				Phone: phone,
			},
			Errors: errors,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := partials.QuickClientForm(data).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render form with errors", "error", err)
		}
		return
	}

	// Create client
	params := domain.CreateClientParams{
		UserID: user.ID,
		Name:   name,
		Email:  email,
		Phone:  phone,
	}

	client, err := h.clientService.Create(r.Context(), params)
	if err != nil {
		h.logger.Error("failed to create client", "error", err)
		errors["name"] = "Failed to create client. Please try again."
		data := partials.QuickClientFormData{
			Form: partials.QuickClientFormValues{
				Name:  name,
				Email: email,
				Phone: phone,
			},
			Errors: errors,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		partials.QuickClientForm(data).Render(r.Context(), w)
		return
	}

	// Fetch all clients for updated select
	clientsResult, err := h.clientService.List(r.Context(), domain.ListClientsParams{
		UserID: user.ID,
		Limit:  1000, // Get all clients for dropdown
		Offset: 0,
	})
	if err != nil {
		h.logger.Error("failed to list clients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Build select options
	options := make([]partials.ClientSelectOption, len(clientsResult.Clients))
	for i, c := range clientsResult.Clients {
		options[i] = partials.ClientSelectOption{
			ID:   c.ID.String(),
			Name: c.Name,
		}
	}

	// Return success response with OOB swap for select
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render success message that closes the form
	if err := partials.QuickClientSuccess(client.Name).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render success message", "error", err)
	}

	// Render the updated select (OOB swap)
	selectData := partials.ClientSelectOptionsData{
		Clients:    options,
		SelectedID: client.ID.String(), // Auto-select the new client
	}
	if err := partials.ClientSelectOptions(selectData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render select options", "error", err)
	}
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

// renderError renders a generic error page using templ.
func (h *ClientHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
	data := clients.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Clients:     []clients.DisplayClient{},
		Pagination:  clients.PaginationData{},
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := clients.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render clients index error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderFormError re-renders the form with errors using templ.
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

	var flash *shared.Flash
	if flashMessage != "" {
		flash = &shared.Flash{
			Type:    shared.FlashError,
			Message: flashMessage,
		}
	}

	data := clients.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Client:      domainClientToDetail(client),
		Form: clients.ClientFormValues{
			Name:         formValues["name"],
			Email:        formValues["email"],
			Phone:        formValues["phone"],
			AddressLine1: formValues["address_line1"],
			AddressLine2: formValues["address_line2"],
			City:         formValues["city"],
			State:        formValues["state"],
			PostalCode:   formValues["postal_code"],
			Notes:        formValues["notes"],
		},
		Errors: fieldErrors,
		Flash:  flash,
		IsEdit: isEdit,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var renderErr error
	if isEdit {
		renderErr = clients.EditPage(data).Render(r.Context(), w)
	} else {
		renderErr = clients.NewPage(data).Render(r.Context(), w)
	}
	if renderErr != nil {
		h.logger.Error("failed to render form error", "error", renderErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
