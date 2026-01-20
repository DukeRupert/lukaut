// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements templ-based client CRUD handlers.
package handler

import (
	"net/http"
	"strconv"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/templ/pages/clients"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/google/uuid"
)

// =============================================================================
// Templ-based Client Handlers
// =============================================================================

// IndexTempl displays a paginated list of clients using templ.
func (h *ClientHandler) IndexTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
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
		h.renderIndexErrorTempl(w, r, user, "Failed to load clients. Please try again.")
		return
	}

	// Transform to display types
	displayClients := make([]clients.DisplayClient, len(result.Clients))
	for i, c := range result.Clients {
		displayClients[i] = clients.DisplayClient{
			ID:        c.ID.String(),
			Name:      c.Name,
			Email:     c.Email,
			Phone:     c.Phone,
			InspectionCount: c.InspectionCount,
		}
	}

	// Build pagination data
	pagination := clients.PaginationData{
		CurrentPage: result.CurrentPage(),
		TotalPages:  result.TotalPages(),
		PerPage:     int(result.Limit),
		Total:       int(result.Total),
		HasPrevious: result.HasPrevious(),
		HasNext:     result.HasMore(),
		PrevPage:    result.CurrentPage() - 1,
		NextPage:    result.CurrentPage() + 1,
	}

	data := clients.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Clients:     displayClients,
		Pagination:  pagination,
		Flash:       nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := clients.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render clients index", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewTempl displays the client creation form using templ.
func (h *ClientHandler) NewTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := clients.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Client:      nil,
		Form:        clients.ClientFormValues{},
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      false,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := clients.NewPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render new client page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ShowTempl displays client details using templ.
func (h *ClientHandler) ShowTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	client, err := h.clientService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get client", "error", err, "client_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load client. Please try again.")
		}
		return
	}

	data := clients.ShowPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Client:      domainClientToDetail(client),
		Flash:       nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := clients.ShowPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render client show page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// EditTempl displays the client edit form using templ.
func (h *ClientHandler) EditTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	client, err := h.clientService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get client", "error", err, "client_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load client. Please try again.")
		}
		return
	}

	data := clients.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToClientDisplay(user),
		Client:      domainClientToDetail(client),
		Form: clients.ClientFormValues{
			Name:         client.Name,
			Email:        client.Email,
			Phone:        client.Phone,
			AddressLine1: client.AddressLine1,
			AddressLine2: client.AddressLine2,
			City:         client.City,
			State:        client.State,
			PostalCode:   client.PostalCode,
			Notes:        client.Notes,
		},
		Errors: make(map[string]string),
		Flash:  nil,
		IsEdit: true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := clients.EditPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render edit client page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// Templ Route Registration
// =============================================================================

// RegisterTemplRoutes registers templ-based client routes on the provided ServeMux.
func (h *ClientHandler) RegisterTemplRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /clients", requireUser(http.HandlerFunc(h.IndexTempl)))
	mux.Handle("GET /clients/new", requireUser(http.HandlerFunc(h.NewTempl)))
	mux.Handle("POST /clients", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /clients/{id}", requireUser(http.HandlerFunc(h.ShowTempl)))
	mux.Handle("GET /clients/{id}/edit", requireUser(http.HandlerFunc(h.EditTempl)))
	mux.Handle("PUT /clients/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /clients/{id}", requireUser(http.HandlerFunc(h.Delete)))
}

// =============================================================================
// Helper Functions
// =============================================================================

// domainUserToClientDisplay converts domain.User to clients.UserDisplay
func domainUserToClientDisplay(u *domain.User) *clients.UserDisplay {
	if u == nil {
		return nil
	}
	return &clients.UserDisplay{
		Name:               u.Name,
		Email:              u.Email,
		HasBusinessProfile: u.HasBusinessProfile(),
	}
}

// domainClientToDetail converts domain.Client to clients.ClientDetail
func domainClientToDetail(c *domain.Client) *clients.ClientDetail {
	if c == nil {
		return nil
	}
	hasAddress := c.AddressLine1 != "" || c.AddressLine2 != "" || c.City != "" || c.State != "" || c.PostalCode != ""
	return &clients.ClientDetail{
		ID:           c.ID.String(),
		Name:         c.Name,
		Email:        c.Email,
		Phone:        c.Phone,
		AddressLine1: c.AddressLine1,
		AddressLine2: c.AddressLine2,
		City:         c.City,
		State:        c.State,
		PostalCode:   c.PostalCode,
		Notes:        c.Notes,
		InspectionCount:    c.InspectionCount,
		HasAddress:   hasAddress,
	}
}

// renderIndexErrorTempl renders the index page with an error flash
func (h *ClientHandler) renderIndexErrorTempl(w http.ResponseWriter, r *http.Request, user *domain.User, message string) {
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
