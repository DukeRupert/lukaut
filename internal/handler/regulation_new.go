// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements templ-based regulation handlers.
package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/templ/pages/regulations"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/google/uuid"
)

// =============================================================================
// Templ-based Regulation Handlers
// =============================================================================

// IndexTempl displays the regulation browse/search page using templ.
func (h *RegulationHandler) IndexTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("index handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse query parameters
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	category := r.URL.Query().Get("category")
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := int32(20)
	offset := int32((page - 1) * int(perPage))

	// Fetch categories for dropdown
	categories, err := h.repo.ListAllCategories(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch categories", "error", err)
		categories = []string{} // Continue with empty list
	}

	// Determine whether to search or browse
	var regs []RegulationSummary
	var total int64

	if query != "" {
		// Search mode
		regs, total, err = h.searchRegulations(r.Context(), query, perPage, offset)
		if err != nil {
			h.logger.Error("failed to search regulations", "error", err, "query", query)
			h.renderIndexErrorTempl(w, r, user, "Failed to search regulations. Please try again.")
			return
		}
	} else {
		// Browse mode (optionally filtered by category)
		regs, total, err = h.browseRegulations(r.Context(), category, perPage, offset)
		if err != nil {
			h.logger.Error("failed to browse regulations", "error", err, "category", category)
			h.renderIndexErrorTempl(w, r, user, "Failed to load regulations. Please try again.")
			return
		}
	}

	// Build pagination data
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := regulations.PaginationData{
		CurrentPage: page,
		TotalPages:  totalPages,
		PerPage:     int(perPage),
		Total:       int(total),
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	// Convert to display types
	displayRegs := regulationsToDisplay(regs)

	data := regulations.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToRegulationDisplay(user),
		Regulations: displayRegs,
		Categories:  categories,
		Filter: regulations.FilterData{
			Query:    query,
			Category: category,
		},
		Pagination: pagination,
		Flash:      nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := regulations.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render regulations index", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// SearchTempl returns filtered regulation results as an htmx partial using templ.
func (h *RegulationHandler) SearchTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	category := r.URL.Query().Get("category")
	violationIDStr := r.URL.Query().Get("violation_id")
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := int32(20)
	offset := int32((page - 1) * int(perPage))

	// Parse optional violation_id for modal context
	violationID := ""
	if violationIDStr != "" {
		parsed, err := uuid.Parse(violationIDStr)
		if err == nil {
			violationID = parsed.String()
		}
	}

	// Determine whether to search or browse
	var regs []RegulationSummary
	var total int64
	var err error

	if query != "" {
		// Search mode
		regs, total, err = h.searchRegulations(r.Context(), query, perPage, offset)
	} else {
		// Browse mode (optionally filtered by category)
		regs, total, err = h.browseRegulations(r.Context(), category, perPage, offset)
	}

	if err != nil {
		h.logger.Error("failed to fetch regulations", "error", err, "query", query, "category", category)
		http.Error(w, "Failed to load regulations", http.StatusInternalServerError)
		return
	}

	// Build pagination data
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := regulations.PaginationData{
		CurrentPage: page,
		TotalPages:  totalPages,
		PerPage:     int(perPage),
		Total:       int(total),
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	// Determine empty message
	emptyMessage := "No regulations found."
	if query != "" {
		emptyMessage = fmt.Sprintf("No regulations found matching \"%s\".", query)
	} else if category != "" {
		emptyMessage = fmt.Sprintf("No regulations found in category \"%s\".", category)
	}

	// Convert to display types
	displayRegs := regulationsToDisplay(regs)

	data := regulations.SearchResultsData{
		Regulations: displayRegs,
		Filter: regulations.FilterData{
			Query:    query,
			Category: category,
		},
		Pagination:   pagination,
		ViolationID:  violationID,
		EmptyMessage: emptyMessage,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := regulations.SearchResultsPartial(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render search results", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetDetailTempl returns full regulation details as an htmx partial for a modal using templ.
func (h *RegulationHandler) GetDetailTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse regulation ID from path
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid regulation ID", http.StatusBadRequest)
		return
	}

	// Parse optional violation_id from query for "add to violation" button
	violationIDStr := r.URL.Query().Get("violation_id")
	violationID := ""
	var violationUUID *uuid.UUID
	if violationIDStr != "" {
		parsed, err := uuid.Parse(violationIDStr)
		if err == nil {
			violationUUID = &parsed
			violationID = parsed.String()
		}
	}

	// Fetch regulation details
	dbReg, err := h.repo.GetRegulationDetail(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Regulation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get regulation", "error", err, "regulation_id", id)
			http.Error(w, "Failed to load regulation", http.StatusInternalServerError)
		}
		return
	}

	// Convert to display type
	regulation := regulations.RegulationDetailDisplay{
		ID:             dbReg.ID.String(),
		StandardNumber: dbReg.StandardNumber,
		Title:          dbReg.Title,
		Category:       dbReg.Category,
		FullText:       dbReg.FullText,
	}

	// Handle nullable fields
	if dbReg.Subcategory.Valid {
		regulation.Subcategory = dbReg.Subcategory.String
	}
	if dbReg.Summary.Valid {
		regulation.Summary = dbReg.Summary.String
	}
	if dbReg.SeverityTypical.Valid {
		regulation.SeverityTypical = dbReg.SeverityTypical.String
	}
	if dbReg.ParentStandard.Valid {
		regulation.ParentStandard = dbReg.ParentStandard.String
	}
	if dbReg.EffectiveDate.Valid {
		regulation.EffectiveDate = dbReg.EffectiveDate.Time.Format("2006-01-02")
	}
	if dbReg.LastUpdated.Valid {
		regulation.LastUpdated = dbReg.LastUpdated.Time.Format("2006-01-02")
	}

	// Check if regulation is already linked to violation
	alreadyLinked := false
	if violationUUID != nil {
		// Verify user owns the violation
		violation, err := h.repo.GetViolationByID(r.Context(), *violationUUID)
		if err == nil {
			// Get inspection to verify ownership
			inspection, err := h.repo.GetInspectionByID(r.Context(), violation.InspectionID)
			if err == nil && inspection.UserID == user.ID {
				// Check if regulation is already linked
				_, err := h.repo.GetViolationRegulation(r.Context(), repository.GetViolationRegulationParams{
					ViolationID:  *violationUUID,
					RegulationID: id,
				})
				alreadyLinked = (err == nil)
			} else {
				// User doesn't own the violation, clear violation_id
				violationID = ""
			}
		} else {
			// Violation not found, clear violation_id
			violationID = ""
		}
	}

	data := regulations.DetailData{
		Regulation:    regulation,
		ViolationID:   violationID,
		AlreadyLinked: alreadyLinked,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := regulations.DetailPartial(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render regulation detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// Templ Route Registration
// =============================================================================

// RegisterTemplRoutes registers templ-based regulation routes on the provided ServeMux.
func (h *RegulationHandler) RegisterTemplRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /regulations", requireUser(http.HandlerFunc(h.IndexTempl)))
	mux.Handle("GET /regulations/search", requireUser(http.HandlerFunc(h.SearchTempl)))
	mux.Handle("GET /regulations/{id}", requireUser(http.HandlerFunc(h.GetDetailTempl)))
	// Keep violation linking routes as-is (they return text, not HTML)
	mux.Handle("POST /violations/{vid}/regulations/{rid}", requireUser(http.HandlerFunc(h.AddToViolation)))
	mux.Handle("DELETE /violations/{vid}/regulations/{rid}", requireUser(http.HandlerFunc(h.RemoveFromViolation)))
}

// =============================================================================
// Helper Functions
// =============================================================================

// domainUserToRegulationDisplay converts domain.User to regulations.UserDisplay.
func domainUserToRegulationDisplay(u *domain.User) *regulations.UserDisplay {
	if u == nil {
		return nil
	}
	return &regulations.UserDisplay{
		Name:               u.Name,
		Email:              u.Email,
		HasBusinessProfile: u.HasBusinessProfile(),
	}
}

// regulationsToDisplay converts a slice of RegulationSummary to regulations.RegulationDisplay.
func regulationsToDisplay(regs []RegulationSummary) []regulations.RegulationDisplay {
	display := make([]regulations.RegulationDisplay, len(regs))
	for i, r := range regs {
		display[i] = regulations.RegulationDisplay{
			ID:              r.ID.String(),
			StandardNumber:  r.StandardNumber,
			Title:           r.Title,
			Category:        r.Category,
			Subcategory:     r.Subcategory,
			Summary:         r.Summary,
			SeverityTypical: r.SeverityTypical,
			Rank:            r.Rank,
		}
	}
	return display
}

// renderIndexErrorTempl renders the index page with an error flash using templ.
func (h *RegulationHandler) renderIndexErrorTempl(w http.ResponseWriter, r *http.Request, user *domain.User, message string) {
	data := regulations.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToRegulationDisplay(user),
		Regulations: []regulations.RegulationDisplay{},
		Categories:  []string{},
		Filter:      regulations.FilterData{},
		Pagination:  regulations.PaginationData{},
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := regulations.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render regulations index error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
