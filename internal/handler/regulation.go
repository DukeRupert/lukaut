// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements regulation search and browse handlers for viewing
// OSHA regulations and linking them to violations.
package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// RegulationListPageData contains data for the regulation browse/search page.
type RegulationListPageData struct {
	CurrentPath string              // Current URL path
	User        interface{}         // Authenticated user
	Regulations []RegulationSummary // List of regulations
	Categories  []string            // Available categories
	Filter      RegulationFilter    // Current filter state
	Pagination  PaginationData      // Pagination information
	Flash       *Flash              // Flash message (if any)
	CSRFToken   string              // CSRF token for form protection
}

// RegulationFilter represents the current filter/search criteria.
type RegulationFilter struct {
	Query    string // Search query
	Category string // Selected category filter
}

// RegulationSummary represents a regulation in list view.
type RegulationSummary struct {
	ID              uuid.UUID
	StandardNumber  string
	Title           string
	Category        string
	Subcategory     string
	Summary         string
	SeverityTypical string
	Rank            float32 // Search relevance rank (only populated for search results)
}

// RegulationDetail represents full regulation details.
type RegulationDetail struct {
	ID              uuid.UUID
	StandardNumber  string
	Title           string
	Category        string
	Subcategory     string
	FullText        string
	Summary         string
	SeverityTypical string
	ParentStandard  string
	EffectiveDate   string
	LastUpdated     string
}

// RegulationSearchPartialData contains data for htmx search results partial.
type RegulationSearchPartialData struct {
	Regulations  []RegulationSummary // Search results
	Filter       RegulationFilter    // Current filter
	Pagination   PaginationData      // Pagination info
	ViolationID  *uuid.UUID          // Optional violation ID for modal context
	EmptyMessage string              // Message when no results
}

// RegulationDetailPartialData contains data for regulation detail modal.
type RegulationDetailPartialData struct {
	Regulation    RegulationDetail // Full regulation details
	ViolationID   *uuid.UUID       // Optional violation ID to enable add button
	AlreadyLinked bool             // Whether regulation is already linked to violation
}

// =============================================================================
// Handler Configuration
// =============================================================================

// RegulationHandler handles regulation-related HTTP requests.
type RegulationHandler struct {
	repo     *repository.Queries
	renderer TemplateRenderer
	logger   *slog.Logger
}

// NewRegulationHandler creates a new RegulationHandler.
func NewRegulationHandler(
	repo *repository.Queries,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *RegulationHandler {
	return &RegulationHandler{
		repo:     repo,
		renderer: renderer,
		logger:   logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all regulation routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - GET  /regulations           -> Index (browse/search page)
// - GET  /regulations/search    -> Search (htmx partial)
// - GET  /regulations/{id}      -> GetDetail (htmx modal)
// - POST /violations/{vid}/regulations/{rid} -> AddToViolation
// - DELETE /violations/{vid}/regulations/{rid} -> RemoveFromViolation
func (h *RegulationHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /regulations", requireUser(http.HandlerFunc(h.Index)))
	mux.Handle("GET /regulations/search", requireUser(http.HandlerFunc(h.Search)))
	mux.Handle("GET /regulations/{id}", requireUser(http.HandlerFunc(h.GetDetail)))
	mux.Handle("POST /violations/{vid}/regulations/{rid}", requireUser(http.HandlerFunc(h.AddToViolation)))
	mux.Handle("DELETE /violations/{vid}/regulations/{rid}", requireUser(http.HandlerFunc(h.RemoveFromViolation)))
}

// =============================================================================
// GET /regulations - Browse/Search Page
// =============================================================================

// Index displays the regulation browse and search page.
func (h *RegulationHandler) Index(w http.ResponseWriter, r *http.Request) {
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
	var regulations []RegulationSummary
	var total int64

	if query != "" {
		// Search mode
		regulations, total, err = h.searchRegulations(r.Context(), query, perPage, offset)
		if err != nil {
			h.logger.Error("failed to search regulations", "error", err, "query", query)
			h.renderError(w, r, user, "Failed to search regulations. Please try again.")
			return
		}
	} else {
		// Browse mode (optionally filtered by category)
		regulations, total, err = h.browseRegulations(r.Context(), category, perPage, offset)
		if err != nil {
			h.logger.Error("failed to browse regulations", "error", err, "category", category)
			h.renderError(w, r, user, "Failed to load regulations. Please try again.")
			return
		}
	}

	// Build pagination data
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := PaginationData{
		CurrentPage: page,
		TotalPages:  totalPages,
		PerPage:     int(perPage),
		Total:       int(total),
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	// Render page
	data := RegulationListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Regulations: regulations,
		Categories:  categories,
		Filter: RegulationFilter{
			Query:    query,
			Category: category,
		},
		Pagination: pagination,
		Flash:      nil,
	}

	h.renderer.RenderHTTP(w, "regulations/index", data)
}

// =============================================================================
// GET /regulations/search - HTMX Search Results
// =============================================================================

// Search returns filtered regulation results as an htmx partial.
func (h *RegulationHandler) Search(w http.ResponseWriter, r *http.Request) {
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
	var violationID *uuid.UUID
	if violationIDStr != "" {
		parsed, err := uuid.Parse(violationIDStr)
		if err == nil {
			violationID = &parsed
		}
	}

	// Determine whether to search or browse
	var regulations []RegulationSummary
	var total int64
	var err error

	if query != "" {
		// Search mode
		regulations, total, err = h.searchRegulations(r.Context(), query, perPage, offset)
	} else {
		// Browse mode (optionally filtered by category)
		regulations, total, err = h.browseRegulations(r.Context(), category, perPage, offset)
	}

	if err != nil {
		h.logger.Error("failed to fetch regulations", "error", err, "query", query, "category", category)
		http.Error(w, "Failed to load regulations", http.StatusInternalServerError)
		return
	}

	// Build pagination data
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := PaginationData{
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

	// Render partial
	data := RegulationSearchPartialData{
		Regulations: regulations,
		Filter: RegulationFilter{
			Query:    query,
			Category: category,
		},
		Pagination:   pagination,
		ViolationID:  violationID,
		EmptyMessage: emptyMessage,
	}

	h.renderer.RenderHTTP(w, "partials/regulation_results", data)
}

// =============================================================================
// GET /regulations/{id} - Regulation Detail Modal
// =============================================================================

// GetDetail returns full regulation details as an htmx partial for a modal.
func (h *RegulationHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
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
	var violationID *uuid.UUID
	if violationIDStr != "" {
		parsed, err := uuid.Parse(violationIDStr)
		if err == nil {
			violationID = &parsed
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

	// Convert to domain type
	regulation := RegulationDetail{
		ID:             dbReg.ID,
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
	if violationID != nil {
		// Verify user owns the violation
		violation, err := h.repo.GetViolationByID(r.Context(), *violationID)
		if err == nil {
			// Get inspection to verify ownership
			inspection, err := h.repo.GetInspectionByID(r.Context(), violation.InspectionID)
			if err == nil && inspection.UserID == user.ID {
				// Check if regulation is already linked
				_, err := h.repo.GetViolationRegulation(r.Context(), repository.GetViolationRegulationParams{
					ViolationID:  *violationID,
					RegulationID: id,
				})
				alreadyLinked = (err == nil)
			} else {
				// User doesn't own the violation, clear violation_id
				violationID = nil
			}
		} else {
			// Violation not found, clear violation_id
			violationID = nil
		}
	}

	// Render detail partial
	data := RegulationDetailPartialData{
		Regulation:    regulation,
		ViolationID:   violationID,
		AlreadyLinked: alreadyLinked,
	}

	h.renderer.RenderHTTP(w, "partials/regulation_detail", data)
}

// =============================================================================
// POST /violations/{vid}/regulations/{rid} - Add Regulation to Violation
// =============================================================================

// AddToViolation links a regulation to a violation.
func (h *RegulationHandler) AddToViolation(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation and regulation IDs from path
	vidStr := r.PathValue("vid")
	ridStr := r.PathValue("rid")

	vid, err := uuid.Parse(vidStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	rid, err := uuid.Parse(ridStr)
	if err != nil {
		http.Error(w, "Invalid regulation ID", http.StatusBadRequest)
		return
	}

	// Verify user owns the violation
	violation, err := h.repo.GetViolationByID(r.Context(), vid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get violation", "error", err, "violation_id", vid)
			http.Error(w, "Failed to load violation", http.StatusInternalServerError)
		}
		return
	}

	// Get inspection to verify ownership
	inspection, err := h.repo.GetInspectionByID(r.Context(), violation.InspectionID)
	if err != nil {
		h.logger.Error("failed to get inspection", "error", err, "inspection_id", violation.InspectionID)
		http.Error(w, "Failed to verify ownership", http.StatusInternalServerError)
		return
	}

	if inspection.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Verify regulation exists
	_, err = h.repo.GetRegulationByID(r.Context(), rid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Regulation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get regulation", "error", err, "regulation_id", rid)
			http.Error(w, "Failed to load regulation", http.StatusInternalServerError)
		}
		return
	}

	// Add regulation to violation
	_, err = h.repo.AddRegulationToViolation(r.Context(), repository.AddRegulationToViolationParams{
		ViolationID:  vid,
		RegulationID: rid,
		RelevanceScore: sql.NullString{
			String: "1.0", // Manual addition, full relevance
			Valid:  true,
		},
		AiExplanation: sql.NullString{
			String: "Manually added by inspector",
			Valid:  true,
		},
		IsPrimary: sql.NullBool{
			Bool:  false,
			Valid: true,
		},
	})

	if err != nil {
		h.logger.Error("failed to add regulation to violation", "error", err, "violation_id", vid, "regulation_id", rid)
		http.Error(w, "Failed to add regulation", http.StatusInternalServerError)
		return
	}

	h.logger.Info("regulation added to violation", "violation_id", vid, "regulation_id", rid, "user_id", user.ID)

	// Return success response
	// For htmx, we'll trigger a refresh of the regulation detail modal
	w.Header().Set("HX-Trigger", "regulation-linked")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Regulation added successfully"))
}

// =============================================================================
// DELETE /violations/{vid}/regulations/{rid} - Remove Regulation from Violation
// =============================================================================

// RemoveFromViolation unlinks a regulation from a violation.
func (h *RegulationHandler) RemoveFromViolation(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation and regulation IDs from path
	vidStr := r.PathValue("vid")
	ridStr := r.PathValue("rid")

	vid, err := uuid.Parse(vidStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	rid, err := uuid.Parse(ridStr)
	if err != nil {
		http.Error(w, "Invalid regulation ID", http.StatusBadRequest)
		return
	}

	// Verify user owns the violation
	violation, err := h.repo.GetViolationByID(r.Context(), vid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get violation", "error", err, "violation_id", vid)
			http.Error(w, "Failed to load violation", http.StatusInternalServerError)
		}
		return
	}

	// Get inspection to verify ownership
	inspection, err := h.repo.GetInspectionByID(r.Context(), violation.InspectionID)
	if err != nil {
		h.logger.Error("failed to get inspection", "error", err, "inspection_id", violation.InspectionID)
		http.Error(w, "Failed to verify ownership", http.StatusInternalServerError)
		return
	}

	if inspection.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Remove regulation from violation
	err = h.repo.RemoveRegulationFromViolation(r.Context(), repository.RemoveRegulationFromViolationParams{
		ViolationID:  vid,
		RegulationID: rid,
	})

	if err != nil {
		h.logger.Error("failed to remove regulation from violation", "error", err, "violation_id", vid, "regulation_id", rid)
		http.Error(w, "Failed to remove regulation", http.StatusInternalServerError)
		return
	}

	h.logger.Info("regulation removed from violation", "violation_id", vid, "regulation_id", rid, "user_id", user.ID)

	// Return success response
	w.Header().Set("HX-Trigger", "regulation-unlinked")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Regulation removed successfully"))
}

// =============================================================================
// Helper Functions
// =============================================================================

// searchRegulations performs a full-text search on regulations.
func (h *RegulationHandler) searchRegulations(ctx context.Context, query string, limit, offset int32) ([]RegulationSummary, int64, error) {
	// Get search results
	results, err := h.repo.SearchRegulationsWithOffset(ctx, repository.SearchRegulationsWithOffsetParams{
		WebsearchToTsquery: query,
		Limit:              limit,
		Offset:             offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search regulations: %w", err)
	}

	// Get total count
	total, err := h.repo.CountSearchResults(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	// Convert to summary type
	summaries := make([]RegulationSummary, len(results))
	for i, r := range results {
		summary := RegulationSummary{
			ID:             r.ID,
			StandardNumber: r.StandardNumber,
			Title:          r.Title,
			Category:       r.Category,
			Rank:           r.Rank,
		}
		if r.Subcategory.Valid {
			summary.Subcategory = r.Subcategory.String
		}
		if r.Summary.Valid {
			summary.Summary = r.Summary.String
		}
		if r.SeverityTypical.Valid {
			summary.SeverityTypical = r.SeverityTypical.String
		}
		summaries[i] = summary
	}

	return summaries, total, nil
}

// browseRegulations lists regulations optionally filtered by category.
func (h *RegulationHandler) browseRegulations(ctx context.Context, category string, limit, offset int32) ([]RegulationSummary, int64, error) {
	var categoryFilter sql.NullString
	if category != "" {
		categoryFilter = sql.NullString{String: category, Valid: true}
	}

	// Get regulations
	results, err := h.repo.ListRegulations(ctx, repository.ListRegulationsParams{
		Category: categoryFilter,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list regulations: %w", err)
	}

	// Get total count
	total, err := h.repo.CountRegulations(ctx, categoryFilter)
	if err != nil {
		return nil, 0, fmt.Errorf("count regulations: %w", err)
	}

	// Convert to summary type
	summaries := make([]RegulationSummary, len(results))
	for i, r := range results {
		summary := RegulationSummary{
			ID:             r.ID,
			StandardNumber: r.StandardNumber,
			Title:          r.Title,
			Category:       r.Category,
			Rank:           0, // Not applicable for browse
		}
		if r.Subcategory.Valid {
			summary.Subcategory = r.Subcategory.String
		}
		if r.Summary.Valid {
			summary.Summary = r.Summary.String
		}
		if r.SeverityTypical.Valid {
			summary.SeverityTypical = r.SeverityTypical.String
		}
		summaries[i] = summary
	}

	return summaries, total, nil
}

// renderError renders an error message to the user.
func (h *RegulationHandler) renderError(w http.ResponseWriter, r *http.Request, user *domain.User, message string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"User":        user,
		"Flash": &Flash{
			Type:    "error",
			Message: message,
		},
		"Regulations": []RegulationSummary{},
		"Categories":  []string{},
		"Filter":      RegulationFilter{},
		"Pagination":  PaginationData{},
	}
	h.renderer.RenderHTTP(w, "regulations/index", data)
}
