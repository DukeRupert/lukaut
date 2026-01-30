// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements regulation search and browse handlers for viewing
// OSHA regulations and linking them to violations.
package handler

import (
	"log/slog"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
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
	regulationService service.RegulationService
	violationService  service.ViolationService
	logger            *slog.Logger
}

// NewRegulationHandler creates a new RegulationHandler.
func NewRegulationHandler(
	regulationService service.RegulationService,
	violationService service.ViolationService,
	logger *slog.Logger,
) *RegulationHandler {
	return &RegulationHandler{
		regulationService: regulationService,
		violationService:  violationService,
		logger:            logger,
	}
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

	// Link regulation to violation (idempotent)
	err = h.regulationService.LinkToViolation(r.Context(), domain.LinkRegulationParams{
		ViolationID:  vid,
		RegulationID: rid,
		UserID:       user.ID,
	})
	if err != nil {
		h.handleServiceError(w, err, "link regulation")
		return
	}

	// Return success
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("HX-Trigger", "regulationLinked")
	w.WriteHeader(http.StatusCreated)
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

	// Unlink regulation from violation (idempotent)
	err = h.regulationService.UnlinkFromViolation(r.Context(), domain.UnlinkRegulationParams{
		ViolationID:  vid,
		RegulationID: rid,
		UserID:       user.ID,
	})
	if err != nil {
		h.handleServiceError(w, err, "unlink regulation")
		return
	}

	// Return success
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("HX-Trigger", "regulationUnlinked")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Regulation removed successfully"))
}

// =============================================================================
// Helper Functions - Regulation Queries
// =============================================================================

// searchRegulations performs a full-text search on regulations.
func (h *RegulationHandler) searchRegulations(r *http.Request, query string, limit, offset int32) ([]RegulationSummary, int64, error) {
	result, err := h.regulationService.Search(r.Context(), query, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Convert domain type to handler type
	summaries := make([]RegulationSummary, len(result.Regulations))
	for i, reg := range result.Regulations {
		summaries[i] = RegulationSummary{
			ID:              reg.ID,
			StandardNumber:  reg.StandardNumber,
			Title:           reg.Title,
			Category:        reg.Category,
			Subcategory:     reg.Subcategory,
			Summary:         reg.Summary,
			SeverityTypical: reg.SeverityTypical,
			Rank:            reg.Rank,
		}
	}

	return summaries, result.Total, nil
}

// browseRegulations lists regulations optionally filtered by category.
func (h *RegulationHandler) browseRegulations(r *http.Request, category string, limit, offset int32) ([]RegulationSummary, int64, error) {
	result, err := h.regulationService.Browse(r.Context(), category, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Convert domain type to handler type
	summaries := make([]RegulationSummary, len(result.Regulations))
	for i, reg := range result.Regulations {
		summaries[i] = RegulationSummary{
			ID:              reg.ID,
			StandardNumber:  reg.StandardNumber,
			Title:           reg.Title,
			Category:        reg.Category,
			Subcategory:     reg.Subcategory,
			Summary:         reg.Summary,
			SeverityTypical: reg.SeverityTypical,
			Rank:            reg.Rank,
		}
	}

	return summaries, result.Total, nil
}

// =============================================================================
// Error Handling
// =============================================================================

// handleServiceError converts domain errors to HTTP responses.
func (h *RegulationHandler) handleServiceError(w http.ResponseWriter, err error, operation string) {
	if domainErr, ok := err.(*domain.Error); ok {
		switch domainErr.Code {
		case domain.ENOTFOUND:
			http.Error(w, domainErr.Message, http.StatusNotFound)
		case domain.EFORBIDDEN:
			http.Error(w, "Forbidden", http.StatusForbidden)
		case domain.EINVALID:
			http.Error(w, domainErr.Message, http.StatusBadRequest)
		default:
			h.logger.Error("service error", "error", err, "operation", operation)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Error("unexpected error", "error", err, "operation", operation)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
