// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements templ-based site CRUD handlers.
package handler

import (
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/templ/pages/sites"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/google/uuid"
)

// =============================================================================
// Templ-based Site Handlers
// =============================================================================

// IndexTempl displays a list of all sites using templ.
func (h *SiteHandler) IndexTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch sites
	siteList, err := h.siteService.List(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list sites", "error", err, "user_id", user.ID)
		h.renderIndexErrorTempl(w, r, user, "Failed to load sites. Please try again.")
		return
	}

	// Transform to display types
	displaySites := make([]sites.DisplaySite, len(siteList))
	for i, s := range siteList {
		displaySites[i] = sites.DisplaySite{
			ID:               s.ID.String(),
			Name:             s.Name,
			CityStateZip:     s.CityStateZip(),
			LinkedClientName: s.LinkedClientName,
		}
		if s.ClientID != nil {
			displaySites[i].LinkedClientID = s.ClientID.String()
		}
	}

	data := sites.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Sites:       displaySites,
		Flash:       nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sites.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render sites index", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewTempl displays the site creation form using templ.
func (h *SiteHandler) NewTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch clients for dropdown
	clientList, err := h.clientService.ListAll(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list clients for site form", "error", err, "user_id", user.ID)
		clientList = []domain.Client{}
	}

	data := sites.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Site:        nil,
		Form:        sites.SiteFormValues{},
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      false,
		Clients:     domainClientsToOptions(clientList),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sites.NewPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render new site page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// EditTempl displays the site edit form using templ.
func (h *SiteHandler) EditTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}

	site, err := h.siteService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
		} else {
			h.logger.Error("failed to get site", "error", err, "site_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load site. Please try again.")
		}
		return
	}

	// Fetch clients for dropdown
	clientList, err := h.clientService.ListAll(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list clients for site form", "error", err, "user_id", user.ID)
		clientList = []domain.Client{}
	}

	clientIDStr := ""
	if site.ClientID != nil {
		clientIDStr = site.ClientID.String()
	}

	data := sites.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToSiteDisplay(user),
		Site:        domainSiteToDetail(site),
		Form: sites.SiteFormValues{
			Name:         site.Name,
			AddressLine1: site.AddressLine1,
			AddressLine2: site.AddressLine2,
			City:         site.City,
			State:        site.State,
			PostalCode:   site.PostalCode,
			ClientID:     clientIDStr,
			Notes:        site.Notes,
		},
		Errors:  make(map[string]string),
		Flash:   nil,
		IsEdit:  true,
		Clients: domainClientsToOptions(clientList),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sites.EditPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render edit site page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// Templ Route Registration
// =============================================================================

// RegisterTemplRoutes registers templ-based site routes on the provided ServeMux.
func (h *SiteHandler) RegisterTemplRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /sites", requireUser(http.HandlerFunc(h.IndexTempl)))
	mux.Handle("GET /sites/new", requireUser(http.HandlerFunc(h.NewTempl)))
	mux.Handle("POST /sites", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /sites/{id}/edit", requireUser(http.HandlerFunc(h.EditTempl)))
	mux.Handle("PUT /sites/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /sites/{id}", requireUser(http.HandlerFunc(h.Delete)))
}

// =============================================================================
// Helper Functions
// =============================================================================

// domainUserToSiteDisplay converts domain.User to sites.UserDisplay
func domainUserToSiteDisplay(u *domain.User) *sites.UserDisplay {
	if u == nil {
		return nil
	}
	return &sites.UserDisplay{
		Name:  u.Name,
		Email: u.Email,
	}
}

// domainSiteToDetail converts domain.Site to sites.SiteDetail
func domainSiteToDetail(s *domain.Site) *sites.SiteDetail {
	if s == nil {
		return nil
	}
	clientIDStr := ""
	if s.ClientID != nil {
		clientIDStr = s.ClientID.String()
	}
	return &sites.SiteDetail{
		ID:           s.ID.String(),
		Name:         s.Name,
		AddressLine1: s.AddressLine1,
		AddressLine2: s.AddressLine2,
		City:         s.City,
		State:        s.State,
		PostalCode:   s.PostalCode,
		Notes:        s.Notes,
		ClientID:     clientIDStr,
	}
}

// domainClientsToOptions converts domain.Client slice to sites.ClientOption slice
func domainClientsToOptions(clients []domain.Client) []sites.ClientOption {
	options := make([]sites.ClientOption, len(clients))
	for i, c := range clients {
		options[i] = sites.ClientOption{
			ID:   c.ID.String(),
			Name: c.Name,
		}
	}
	return options
}

// renderIndexErrorTempl renders the index page with an error flash
func (h *SiteHandler) renderIndexErrorTempl(w http.ResponseWriter, r *http.Request, user *domain.User, message string) {
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
