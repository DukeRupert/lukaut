// Package service contains the business logic layer.
//
// This file implements the regulation service for searching OSHA regulations
// and linking them to violations.
package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// =============================================================================
// Interface Definition
// =============================================================================

// RegulationService defines the interface for regulation-related operations.
//
// Regulations are public reference data (OSHA standards). Authorization checks
// are only required for operations that modify violation-regulation links.
type RegulationService interface {
	// GetByID retrieves a regulation by ID.
	// Returns domain.ENOTFOUND if regulation doesn't exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Regulation, error)

	// Search performs full-text search on regulations.
	// Returns paginated results with total count.
	Search(ctx context.Context, query string, limit, offset int32) (*domain.RegulationSearchResult, error)

	// Browse lists regulations with optional category filter.
	// Returns paginated results with total count.
	Browse(ctx context.Context, category string, limit, offset int32) (*domain.RegulationSearchResult, error)

	// ListCategories returns all unique regulation categories.
	ListCategories(ctx context.Context) ([]string, error)

	// LinkToViolation links a regulation to a violation.
	// Idempotent: succeeds silently if already linked.
	// Returns domain.ENOTFOUND if violation or regulation doesn't exist.
	// Returns domain.EFORBIDDEN if user doesn't own the violation's inspection.
	LinkToViolation(ctx context.Context, params domain.LinkRegulationParams) error

	// UnlinkFromViolation removes a regulation link from a violation.
	// Idempotent: succeeds silently if not linked.
	// Returns domain.ENOTFOUND if violation doesn't exist.
	// Returns domain.EFORBIDDEN if user doesn't own the violation's inspection.
	UnlinkFromViolation(ctx context.Context, params domain.UnlinkRegulationParams) error

	// IsLinkedToViolation checks if a regulation is linked to a violation.
	IsLinkedToViolation(ctx context.Context, violationID, regulationID uuid.UUID) (bool, error)
}

// =============================================================================
// Implementation
// =============================================================================

// regulationService implements the RegulationService interface.
type regulationService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewRegulationService creates a new RegulationService.
func NewRegulationService(
	queries *repository.Queries,
	logger *slog.Logger,
) RegulationService {
	return &regulationService{
		queries: queries,
		logger:  logger,
	}
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves a regulation by ID.
func (s *regulationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Regulation, error) {
	const op = "regulation.get"

	row, err := s.queries.GetRegulationDetail(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "regulation", id.String())
		}
		return nil, domain.Internal(err, op, "failed to get regulation")
	}

	return s.detailRowToRegulation(row), nil
}

// =============================================================================
// Search
// =============================================================================

// Search performs full-text search on regulations.
func (s *regulationService) Search(ctx context.Context, query string, limit, offset int32) (*domain.RegulationSearchResult, error) {
	const op = "regulation.search"

	// Get search results
	results, err := s.queries.SearchRegulationsWithOffset(ctx, repository.SearchRegulationsWithOffsetParams{
		WebsearchToTsquery: query,
		Limit:              limit,
		Offset:             offset,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to search regulations")
	}

	// Get total count
	total, err := s.queries.CountSearchResults(ctx, query)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count search results")
	}

	// Convert to domain type
	summaries := make([]domain.RegulationSummary, len(results))
	for i, r := range results {
		summaries[i] = domain.RegulationSummary{
			ID:              r.ID,
			StandardNumber:  r.StandardNumber,
			Title:           r.Title,
			Category:        r.Category,
			Subcategory:     domain.NullStringValue(r.Subcategory),
			Summary:         domain.NullStringValue(r.Summary),
			SeverityTypical: domain.NullStringValue(r.SeverityTypical),
			Rank:            r.Rank,
		}
	}

	return &domain.RegulationSearchResult{
		Regulations: summaries,
		Total:       total,
	}, nil
}

// =============================================================================
// Browse
// =============================================================================

// Browse lists regulations with optional category filter.
func (s *regulationService) Browse(ctx context.Context, category string, limit, offset int32) (*domain.RegulationSearchResult, error) {
	const op = "regulation.browse"

	var categoryFilter sql.NullString
	if category != "" {
		categoryFilter = sql.NullString{String: category, Valid: true}
	}

	// Get regulations
	results, err := s.queries.ListRegulations(ctx, repository.ListRegulationsParams{
		Category: categoryFilter,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list regulations")
	}

	// Get total count
	total, err := s.queries.CountRegulations(ctx, categoryFilter)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count regulations")
	}

	// Convert to domain type
	summaries := make([]domain.RegulationSummary, len(results))
	for i, r := range results {
		summaries[i] = domain.RegulationSummary{
			ID:              r.ID,
			StandardNumber:  r.StandardNumber,
			Title:           r.Title,
			Category:        r.Category,
			Subcategory:     domain.NullStringValue(r.Subcategory),
			Summary:         domain.NullStringValue(r.Summary),
			SeverityTypical: domain.NullStringValue(r.SeverityTypical),
			Rank:            0, // Not applicable for browse
		}
	}

	return &domain.RegulationSearchResult{
		Regulations: summaries,
		Total:       total,
	}, nil
}

// =============================================================================
// ListCategories
// =============================================================================

// ListCategories returns all unique regulation categories.
func (s *regulationService) ListCategories(ctx context.Context) ([]string, error) {
	const op = "regulation.list_categories"

	categories, err := s.queries.ListAllCategories(ctx)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list categories")
	}

	return categories, nil
}

// =============================================================================
// LinkToViolation
// =============================================================================

// LinkToViolation links a regulation to a violation.
// Idempotent: succeeds silently if already linked.
func (s *regulationService) LinkToViolation(ctx context.Context, params domain.LinkRegulationParams) error {
	const op = "regulation.link"

	// Verify user owns the violation (via inspection)
	_, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     params.ViolationID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "violation", params.ViolationID.String())
		}
		return domain.Internal(err, op, "failed to verify violation ownership")
	}

	// Verify regulation exists
	_, err = s.queries.GetRegulationByID(ctx, params.RegulationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "regulation", params.RegulationID.String())
		}
		return domain.Internal(err, op, "failed to verify regulation exists")
	}

	// Check if already linked (idempotent)
	_, err = s.queries.GetViolationRegulation(ctx, repository.GetViolationRegulationParams{
		ViolationID:  params.ViolationID,
		RegulationID: params.RegulationID,
	})
	if err == nil {
		// Already linked - success (idempotent)
		s.logger.Debug("regulation already linked to violation",
			"violation_id", params.ViolationID,
			"regulation_id", params.RegulationID,
		)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Internal(err, op, "failed to check existing link")
	}

	// Set defaults
	relevanceScore := params.RelevanceScore
	if relevanceScore == 0 {
		relevanceScore = 1.0
	}
	explanation := params.Explanation
	if explanation == "" {
		explanation = "Manually added by inspector"
	}

	// Create the link
	_, err = s.queries.CreateViolationRegulation(ctx, repository.CreateViolationRegulationParams{
		ViolationID:    params.ViolationID,
		RegulationID:   params.RegulationID,
		RelevanceScore: sql.NullFloat64{Float64: relevanceScore, Valid: true},
		AiExplanation:  sql.NullString{String: explanation, Valid: true},
		IsPrimary:      sql.NullBool{Bool: params.IsPrimary, Valid: true},
	})
	if err != nil {
		return domain.Internal(err, op, "failed to link regulation to violation")
	}

	s.logger.Info("regulation linked to violation",
		"violation_id", params.ViolationID,
		"regulation_id", params.RegulationID,
		"user_id", params.UserID,
	)

	return nil
}

// =============================================================================
// UnlinkFromViolation
// =============================================================================

// UnlinkFromViolation removes a regulation link from a violation.
// Idempotent: succeeds silently if not linked.
func (s *regulationService) UnlinkFromViolation(ctx context.Context, params domain.UnlinkRegulationParams) error {
	const op = "regulation.unlink"

	// Verify user owns the violation (via inspection)
	_, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     params.ViolationID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "violation", params.ViolationID.String())
		}
		return domain.Internal(err, op, "failed to verify violation ownership")
	}

	// Remove the link (idempotent - no error if not found)
	err = s.queries.RemoveRegulationFromViolation(ctx, repository.RemoveRegulationFromViolationParams{
		ViolationID:  params.ViolationID,
		RegulationID: params.RegulationID,
	})
	if err != nil {
		return domain.Internal(err, op, "failed to unlink regulation from violation")
	}

	s.logger.Info("regulation unlinked from violation",
		"violation_id", params.ViolationID,
		"regulation_id", params.RegulationID,
		"user_id", params.UserID,
	)

	return nil
}

// =============================================================================
// IsLinkedToViolation
// =============================================================================

// IsLinkedToViolation checks if a regulation is linked to a violation.
func (s *regulationService) IsLinkedToViolation(ctx context.Context, violationID, regulationID uuid.UUID) (bool, error) {
	const op = "regulation.is_linked"

	_, err := s.queries.GetViolationRegulation(ctx, repository.GetViolationRegulationParams{
		ViolationID:  violationID,
		RegulationID: regulationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, domain.Internal(err, op, "failed to check link")
	}

	return true, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// detailRowToRegulation converts a repository regulation detail row to a domain Regulation.
func (s *regulationService) detailRowToRegulation(row repository.GetRegulationDetailRow) *domain.Regulation {
	return &domain.Regulation{
		ID:              row.ID,
		StandardNumber:  row.StandardNumber,
		Title:           row.Title,
		Category:        row.Category,
		Subcategory:     domain.NullStringValue(row.Subcategory),
		FullText:        row.FullText,
		Summary:         domain.NullStringValue(row.Summary),
		SeverityTypical: domain.NullStringValue(row.SeverityTypical),
		ParentStandard:  domain.NullStringValue(row.ParentStandard),
		EffectiveDate:   domain.NullTimeValue(row.EffectiveDate),
		LastUpdated:     domain.NullTimeValue(row.LastUpdated),
	}
}
