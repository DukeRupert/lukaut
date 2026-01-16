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

	"github.com/DukeRupert/lukaut/internal/auth"
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
	repo   *repository.Queries
	logger *slog.Logger
}

// NewRegulationHandler creates a new RegulationHandler.
func NewRegulationHandler(
	repo *repository.Queries,
	logger *slog.Logger,
) *RegulationHandler {
	return &RegulationHandler{
		repo:   repo,
		logger: logger,
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

	// Verify user owns the violation
	violation, err := h.repo.GetViolationByID(r.Context(), vid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get violation", "error", err, "violation_id", vid)
			http.Error(w, "Failed to verify violation", http.StatusInternalServerError)
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
	_, err = h.repo.GetRegulationDetail(r.Context(), rid)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Regulation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get regulation", "error", err, "regulation_id", rid)
			http.Error(w, "Failed to verify regulation", http.StatusInternalServerError)
		}
		return
	}

	// Check if already linked
	_, err = h.repo.GetViolationRegulation(r.Context(), repository.GetViolationRegulationParams{
		ViolationID:  vid,
		RegulationID: rid,
	})
	if err == nil {
		// Already linked
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Regulation already linked"))
		return
	}

	// Link regulation to violation (as non-primary by default)
	_, err = h.repo.CreateViolationRegulation(r.Context(), repository.CreateViolationRegulationParams{
		ViolationID:    vid,
		RegulationID:   rid,
		RelevanceScore: sql.NullString{String: "1.0", Valid: true},
		AiExplanation:  sql.NullString{String: "Manually added by inspector", Valid: true},
		IsPrimary:      sql.NullBool{Bool: false, Valid: true},
	})
	if err != nil {
		h.logger.Error("failed to link regulation to violation", "error", err, "violation_id", vid, "regulation_id", rid)
		http.Error(w, "Failed to add regulation", http.StatusInternalServerError)
		return
	}

	h.logger.Info("regulation linked to violation",
		"violation_id", vid,
		"regulation_id", rid,
		"user_id", user.ID,
	)

	// Return success
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("HX-Trigger", "regulationLinked")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Regulation added successfully"))
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
			http.Error(w, "Failed to verify violation", http.StatusInternalServerError)
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

	// Remove the link
	err = h.repo.RemoveRegulationFromViolation(r.Context(), repository.RemoveRegulationFromViolationParams{
		ViolationID:  vid,
		RegulationID: rid,
	})
	if err != nil {
		h.logger.Error("failed to unlink regulation from violation", "error", err, "violation_id", vid, "regulation_id", rid)
		http.Error(w, "Failed to remove regulation", http.StatusInternalServerError)
		return
	}

	h.logger.Info("regulation unlinked from violation",
		"violation_id", vid,
		"regulation_id", rid,
		"user_id", user.ID,
	)

	// Return success
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("HX-Trigger", "regulationUnlinked")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Regulation removed successfully"))
}

// =============================================================================
// Helper Functions - Regulation Queries
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
