// Package service contains the business logic layer.
//
// This file implements the violation service for managing OSHA violations
// identified during construction site safety inspections.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// =============================================================================
// Interface Definition
// =============================================================================

// ViolationService defines the interface for violation-related operations.
//
// All operations include authorization checks to ensure the user owns the
// inspection that contains the violation.
type ViolationService interface {
	// GetByID retrieves a violation by ID with authorization check.
	// Returns domain.ENOTFOUND if violation doesn't exist or user doesn't own the inspection.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Violation, error)

	// GetByIDWithRegulations retrieves a violation with its linked regulations.
	// Returns domain.ENOTFOUND if violation doesn't exist or user doesn't own the inspection.
	GetByIDWithRegulations(ctx context.Context, id, userID uuid.UUID) (*domain.Violation, []domain.ViolationRegulation, error)

	// ListByInspection retrieves all violations for an inspection.
	// Returns domain.ENOTFOUND if inspection doesn't exist or user doesn't own it.
	ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Violation, error)

	// Create creates a new manual violation.
	// Returns domain.EINVALID for validation errors.
	// Returns domain.ENOTFOUND if inspection doesn't exist or user doesn't own it.
	Create(ctx context.Context, params domain.CreateViolationParams) (*domain.Violation, error)

	// Update updates a violation's description, severity, and notes.
	// Returns domain.EINVALID for validation errors.
	// Returns domain.ENOTFOUND if violation doesn't exist or user doesn't own the inspection.
	Update(ctx context.Context, params domain.UpdateViolationParams) error

	// UpdateStatus updates a violation's review status (accept/reject).
	// Returns domain.EINVALID for invalid status.
	// Returns domain.ENOTFOUND if violation doesn't exist or user doesn't own the inspection.
	UpdateStatus(ctx context.Context, params domain.UpdateViolationStatusParams) error

	// Delete deletes a violation.
	// Returns domain.ENOTFOUND if violation doesn't exist or user doesn't own the inspection.
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// =============================================================================
// Implementation
// =============================================================================

// violationService implements the ViolationService interface.
type violationService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewViolationService creates a new ViolationService.
func NewViolationService(
	queries *repository.Queries,
	logger *slog.Logger,
) ViolationService {
	return &violationService{
		queries: queries,
		logger:  logger,
	}
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves a violation by ID with authorization check.
func (s *violationService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Violation, error) {
	const op = "violation.get"

	// Get violation with authorization check (joins to inspections to verify user ownership)
	row, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "violation", id.String())
		}
		return nil, domain.Internal(err, op, "failed to get violation")
	}

	violation := s.rowToViolation(row)
	return violation, nil
}

// =============================================================================
// GetByIDWithRegulations
// =============================================================================

// GetByIDWithRegulations retrieves a violation with its linked regulations.
func (s *violationService) GetByIDWithRegulations(ctx context.Context, id, userID uuid.UUID) (*domain.Violation, []domain.ViolationRegulation, error) {
	const op = "violation.get_with_regulations"

	// Get violation with authorization check
	violation, err := s.GetByID(ctx, id, userID)
	if err != nil {
		return nil, nil, err
	}

	// Get linked regulations
	regRows, err := s.queries.ListRegulationsByViolationID(ctx, id)
	if err != nil {
		return nil, nil, domain.Internal(err, op, "failed to get regulations")
	}

	regulations := make([]domain.ViolationRegulation, 0, len(regRows))
	for _, row := range regRows {
		// Parse relevance score (stored as text in DB for precision)
		relevanceScore := 0.0
		if row.RelevanceScore.Valid {
			// Try to parse the string as float, default to 0.0 on error
			fmt.Sscanf(row.RelevanceScore.String, "%f", &relevanceScore)
		}

		regulations = append(regulations, domain.ViolationRegulation{
			ViolationID:    id,
			RegulationID:   row.ID,
			RelevanceScore: relevanceScore,
			AIExplanation:  domain.NullStringValue(row.AiExplanation),
			IsPrimary:      row.IsPrimary.Valid && row.IsPrimary.Bool,
			StandardNumber: row.StandardNumber,
			Title:          row.Title,
			Category:       row.Category,
		})
	}

	return violation, regulations, nil
}

// =============================================================================
// ListByInspection
// =============================================================================

// ListByInspection retrieves all violations for an inspection.
func (s *violationService) ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Violation, error) {
	const op = "violation.list"

	// Verify user owns the inspection
	_, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "inspection", inspectionID.String())
		}
		return nil, domain.Internal(err, op, "failed to verify inspection ownership")
	}

	// Get violations
	rows, err := s.queries.ListViolationsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list violations")
	}

	violations := make([]domain.Violation, 0, len(rows))
	for _, row := range rows {
		violations = append(violations, *s.rowToViolation(row))
	}

	return violations, nil
}

// =============================================================================
// Create
// =============================================================================

// Create creates a new manual violation.
func (s *violationService) Create(ctx context.Context, params domain.CreateViolationParams) (*domain.Violation, error) {
	const op = "violation.create"

	// Validate parameters
	if err := s.validateCreateParams(params); err != nil {
		return nil, err
	}

	// Verify user owns the inspection
	_, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     params.InspectionID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "inspection", params.InspectionID.String())
		}
		return nil, domain.Internal(err, op, "failed to verify inspection ownership")
	}

	// If image_id is provided, verify it exists and belongs to this inspection
	if params.ImageID != nil {
		image, err := s.queries.GetImageByID(ctx, *params.ImageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.NotFound(op, "image", params.ImageID.String())
			}
			return nil, domain.Internal(err, op, "failed to verify image")
		}
		if image.InspectionID != params.InspectionID {
			return nil, domain.Invalid(op, "image does not belong to this inspection")
		}
	}

	// Get max sort order for this inspection
	maxSortOrder := int32(0)
	violations, err := s.queries.ListViolationsByInspectionID(ctx, params.InspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to get existing violations")
	}
	for _, v := range violations {
		if v.SortOrder.Valid && v.SortOrder.Int32 > maxSortOrder {
			maxSortOrder = v.SortOrder.Int32
		}
	}

	// Create the violation
	row, err := s.queries.CreateViolation(ctx, repository.CreateViolationParams{
		InspectionID:   params.InspectionID,
		ImageID:        domain.ToNullUUID(params.ImageID),
		Description:    params.Description,
		AiDescription:  sql.NullString{Valid: false}, // Manual violations have no AI description
		Confidence:     sql.NullString{Valid: false}, // Manual violations have no confidence
		BoundingBox:    pqtype.NullRawMessage{Valid: false}, // No bounding box for manual
		Status:         string(domain.ViolationStatusPending),
		Severity:       domain.ToNullString(string(params.Severity)),
		InspectorNotes: domain.ToNullString(params.InspectorNotes),
		SortOrder:      sql.NullInt32{Valid: true, Int32: maxSortOrder + 1},
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to create violation")
	}

	violation := s.rowToViolation(row)

	s.logger.Info("violation created",
		"violation_id", violation.ID,
		"inspection_id", params.InspectionID,
		"user_id", params.UserID,
	)

	return violation, nil
}

// validateCreateParams validates violation creation parameters.
func (s *violationService) validateCreateParams(params domain.CreateViolationParams) error {
	const op = "violation.validate"

	// Description is required and must be 1-1000 characters
	description := strings.TrimSpace(params.Description)
	if description == "" {
		return domain.Invalid(op, "description is required")
	}
	if len(description) > 1000 {
		return domain.Invalid(op, "description must be 1000 characters or less")
	}

	// Severity must be valid
	if !params.Severity.IsValid() {
		return domain.Invalid(op, fmt.Sprintf("invalid severity: %s", params.Severity))
	}

	return nil
}

// =============================================================================
// Update
// =============================================================================

// Update updates a violation's description, severity, and notes.
func (s *violationService) Update(ctx context.Context, params domain.UpdateViolationParams) error {
	const op = "violation.update"

	// Validate parameters
	if err := s.validateUpdateParams(params); err != nil {
		return err
	}

	// Verify violation exists and user owns the inspection
	_, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "violation", params.ID.String())
		}
		return domain.Internal(err, op, "failed to verify violation ownership")
	}

	// Update the violation
	err = s.queries.UpdateViolationDetails(ctx, repository.UpdateViolationDetailsParams{
		ID:             params.ID,
		Description:    params.Description,
		Severity:       domain.ToNullString(string(params.Severity)),
		InspectorNotes: domain.ToNullString(params.InspectorNotes),
	})
	if err != nil {
		return domain.Internal(err, op, "failed to update violation")
	}

	s.logger.Info("violation updated",
		"violation_id", params.ID,
		"user_id", params.UserID,
	)

	return nil
}

// validateUpdateParams validates violation update parameters.
func (s *violationService) validateUpdateParams(params domain.UpdateViolationParams) error {
	const op = "violation.validate"

	// Description is required and must be 1-1000 characters
	description := strings.TrimSpace(params.Description)
	if description == "" {
		return domain.Invalid(op, "description is required")
	}
	if len(description) > 1000 {
		return domain.Invalid(op, "description must be 1000 characters or less")
	}

	// Severity must be valid
	if !params.Severity.IsValid() {
		return domain.Invalid(op, fmt.Sprintf("invalid severity: %s", params.Severity))
	}

	return nil
}

// =============================================================================
// UpdateStatus
// =============================================================================

// UpdateStatus updates a violation's review status.
func (s *violationService) UpdateStatus(ctx context.Context, params domain.UpdateViolationStatusParams) error {
	const op = "violation.update_status"

	// Validate status
	if !params.Status.IsValid() {
		return domain.Invalid(op, fmt.Sprintf("invalid status: %s", params.Status))
	}

	// Verify violation exists and user owns the inspection
	_, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "violation", params.ID.String())
		}
		return domain.Internal(err, op, "failed to verify violation ownership")
	}

	// Update status
	err = s.queries.UpdateViolationStatus(ctx, repository.UpdateViolationStatusParams{
		ID:     params.ID,
		Status: string(params.Status),
	})
	if err != nil {
		return domain.Internal(err, op, "failed to update violation status")
	}

	s.logger.Info("violation status updated",
		"violation_id", params.ID,
		"user_id", params.UserID,
		"status", params.Status,
	)

	return nil
}

// =============================================================================
// Delete
// =============================================================================

// Delete deletes a violation.
func (s *violationService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const op = "violation.delete"

	// Verify violation exists and user owns the inspection
	_, err := s.queries.GetViolationByIDAndUserID(ctx, repository.GetViolationByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "violation", id.String())
		}
		return domain.Internal(err, op, "failed to verify violation ownership")
	}

	// Delete the violation (cascades to violation_regulations)
	err = s.queries.DeleteViolationByIDAndUserID(ctx, repository.DeleteViolationByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return domain.Internal(err, op, "failed to delete violation")
	}

	s.logger.Info("violation deleted",
		"violation_id", id,
		"user_id", userID,
	)

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToViolation converts a repository violation row to a domain Violation.
func (s *violationService) rowToViolation(row repository.Violation) *domain.Violation {
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	sortOrder := 0
	if row.SortOrder.Valid {
		sortOrder = int(row.SortOrder.Int32)
	}

	return &domain.Violation{
		ID:             row.ID,
		InspectionID:   row.InspectionID,
		ImageID:        domain.NullUUIDToPtr(row.ImageID),
		Description:    row.Description,
		AIDescription:  domain.NullStringValue(row.AiDescription),
		Confidence:     domain.ViolationConfidence(domain.NullStringValue(row.Confidence)),
		BoundingBox:    "", // Skip bounding box for now (pqtype.NullRawMessage)
		Status:         domain.ViolationStatus(row.Status),
		Severity:       domain.ViolationSeverity(domain.NullStringValue(row.Severity)),
		InspectorNotes: domain.NullStringValue(row.InspectorNotes),
		SortOrder:      sortOrder,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
