// Package service contains the business logic layer.
//
// This file implements the inspection service for managing construction
// site safety inspections.
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
	"github.com/DukeRupert/lukaut/internal/metrics"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// =============================================================================
// JobEnqueuer Interface (defined here to avoid circular imports with worker)
// =============================================================================

// JobEnqueuer abstracts job enqueueing operations for use by services.
// This interface is satisfied by worker.JobEnqueuer.
type JobEnqueuer interface {
	// EnqueueAnalyzeInspection enqueues a job to analyze an inspection's images.
	EnqueueAnalyzeInspection(ctx context.Context, inspectionID, userID uuid.UUID) (repository.Job, error)

	// EnqueueGenerateReport enqueues a job to generate a report for an inspection.
	EnqueueGenerateReport(ctx context.Context, inspectionID, userID uuid.UUID, format, recipientEmail string) (repository.Job, error)
}

// =============================================================================
// Interface Definition
// =============================================================================

// InspectionService defines the interface for inspection-related operations.
//
// This interface enables:
// - Mocking in unit tests
// - Clear contract definition for handlers
// - Potential future implementations with different backends
type InspectionService interface {
	// Create creates a new inspection.
	// Returns domain.EINVALID for validation errors.
	// Returns domain.ENOTFOUND if client_id is provided but client doesn't exist or belong to user.
	Create(ctx context.Context, params domain.CreateInspectionParams) (*domain.Inspection, error)

	// GetByID retrieves an inspection by ID and user ID (for authorization).
	// Returns domain.ENOTFOUND if inspection does not exist or doesn't belong to user.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Inspection, error)

	// List retrieves a paginated list of inspections for a user.
	// Returns empty result if user has no inspections.
	List(ctx context.Context, params domain.ListInspectionsParams) (*domain.ListInspectionsResult, error)

	// Update updates an existing inspection.
	// Returns domain.ENOTFOUND if inspection does not exist or doesn't belong to user.
	// Returns domain.EINVALID for validation errors or if inspection is not editable.
	Update(ctx context.Context, params domain.UpdateInspectionParams) error

	// Delete deletes an inspection by ID.
	// Returns domain.ENOTFOUND if inspection does not exist or doesn't belong to user.
	// This cascades to delete all associated photos and violations.
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// UpdateStatus updates the status of an inspection.
	// Returns domain.ENOTFOUND if inspection does not exist or doesn't belong to user.
	// Returns domain.EINVALID if status transition is invalid.
	UpdateStatus(ctx context.Context, params domain.UpdateInspectionStatusParams) error

	// GetAnalysisStatus returns the computed analysis status for an inspection.
	// Returns domain.ENOTFOUND if inspection does not exist or doesn't belong to user.
	GetAnalysisStatus(ctx context.Context, inspectionID, userID uuid.UUID) (*domain.AnalysisStatus, error)

	// StartAnalysis transitions an inspection to analyzing status.
	// No-op if the inspection is already analyzing (supports job retries).
	// Returns domain.EINVALID if the current status does not allow this transition.
	StartAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error

	// CompleteAnalysis transitions an inspection from analyzing to review status.
	// Returns domain.EINVALID if the current status does not allow this transition.
	CompleteAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error

	// TriggerAnalysis enqueues a job to analyze an inspection's images.
	// Returns domain.EINVALID if the inspection cannot be analyzed.
	TriggerAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error

	// HasPendingAnalysisJob checks if there is a pending or running analysis job for the inspection.
	HasPendingAnalysisJob(ctx context.Context, inspectionID uuid.UUID) (bool, error)
}

// =============================================================================
// Implementation
// =============================================================================

// inspectionService implements the InspectionService interface.
type inspectionService struct {
	queries      *repository.Queries
	jobEnqueuer  JobEnqueuer
	quotaService QuotaService
	logger       *slog.Logger
}

// NewInspectionService creates a new InspectionService.
//
// Parameters:
// - queries: Repository queries for database access
// - jobEnqueuer: Job enqueuer for background jobs (can be nil if job enqueueing not needed)
// - quotaService: Quota service for rate limiting (can be nil to disable quota checks)
// - logger: Structured logger for operation logging
//
// Example usage:
//
//	inspectionService := service.NewInspectionService(repo, jobEnqueuer, quotaService, logger)
func NewInspectionService(
	queries *repository.Queries,
	jobEnqueuer JobEnqueuer,
	quotaService QuotaService,
	logger *slog.Logger,
) InspectionService {
	return &inspectionService{
		queries:      queries,
		jobEnqueuer:  jobEnqueuer,
		quotaService: quotaService,
		logger:       logger,
	}
}

// =============================================================================
// Create
// =============================================================================

// Create creates a new inspection.
func (s *inspectionService) Create(ctx context.Context, params domain.CreateInspectionParams) (*domain.Inspection, error) {
	const op = "inspection.create"

	// Validate parameters
	if err := s.validateCreateParams(params); err != nil {
		return nil, err
	}

	// If client_id is provided, verify it exists and belongs to the user
	if params.ClientID != nil {
		_, err := s.queries.GetClientByIDAndUserID(ctx, repository.GetClientByIDAndUserIDParams{
			ID:     *params.ClientID,
			UserID: params.UserID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.NotFound(op, "client", params.ClientID.String())
			}
			return nil, domain.Internal(err, op, "failed to verify client")
		}
	}

	// Create the inspection
	row, err := s.queries.CreateInspection(ctx, repository.CreateInspectionParams{
		UserID:            params.UserID,
		ClientID:          domain.ToNullUUID(params.ClientID),
		Title:             params.Title,
		Status:            string(domain.InspectionStatusDraft),
		InspectionDate:    params.InspectionDate,
		WeatherConditions: domain.ToNullString(params.WeatherConditions),
		Temperature:       domain.ToNullString(params.Temperature),
		InspectorNotes:    domain.ToNullString(params.InspectorNotes),
		AddressLine1:      params.AddressLine1,
		AddressLine2:      domain.ToNullString(params.AddressLine2),
		City:              params.City,
		State:             params.State,
		PostalCode:        params.PostalCode,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to create inspection")
	}

	// Convert to domain type
	inspection := s.rowToInspection(row)

	s.logger.Info("inspection created",
		"inspection_id", inspection.ID,
		"user_id", params.UserID,
		"title", params.Title,
	)
	metrics.InspectionsCreated.Inc()

	return inspection, nil
}

// validateCreateParams validates inspection creation parameters.
func (s *inspectionService) validateCreateParams(params domain.CreateInspectionParams) error {
	const op = "inspection.validate"

	// Title is required and must be 1-200 characters
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return domain.Invalid(op, "title is required")
	}
	if len(title) > 200 {
		return domain.Invalid(op, "title must be 200 characters or less")
	}

	// Address fields are required
	if strings.TrimSpace(params.AddressLine1) == "" {
		return domain.Invalid(op, "address is required")
	}
	if strings.TrimSpace(params.City) == "" {
		return domain.Invalid(op, "city is required")
	}
	if strings.TrimSpace(params.State) == "" {
		return domain.Invalid(op, "state is required")
	}
	if strings.TrimSpace(params.PostalCode) == "" {
		return domain.Invalid(op, "postal code is required")
	}

	// Inspection date cannot be more than 1 year in the future
	oneYearFromNow := time.Now().AddDate(1, 0, 0)
	if params.InspectionDate.After(oneYearFromNow) {
		return domain.Invalid(op, "inspection date cannot be more than 1 year in the future")
	}

	return nil
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves an inspection by ID.
func (s *inspectionService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Inspection, error) {
	const op = "inspection.get"

	// Get inspection with client information
	row, err := s.queries.GetInspectionWithClientByIDAndUserID(ctx, repository.GetInspectionWithClientByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "inspection", id.String())
		}
		return nil, domain.Internal(err, op, "failed to get inspection")
	}

	// Convert to domain type
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	inspection := &domain.Inspection{
		ID:                row.ID,
		UserID:            row.UserID,
		ClientID:          nullUUIDToPtr(row.ClientID),
		Title:             row.Title,
		Status:            domain.InspectionStatus(row.Status),
		InspectionDate:    row.InspectionDate,
		WeatherConditions: domain.NullStringValue(row.WeatherConditions),
		Temperature:       domain.NullStringValue(row.Temperature),
		InspectorNotes:    domain.NullStringValue(row.InspectorNotes),
		AddressLine1:      row.AddressLine1,
		AddressLine2:      domain.NullStringValue(row.AddressLine2),
		City:              row.City,
		State:             row.State,
		PostalCode:        row.PostalCode,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		ClientName:        row.ClientName,
	}

	return inspection, nil
}

// =============================================================================
// List
// =============================================================================

// List retrieves a paginated list of inspections.
func (s *inspectionService) List(ctx context.Context, params domain.ListInspectionsParams) (*domain.ListInspectionsResult, error) {
	const op = "inspection.list"

	// Get total count
	total, err := s.queries.CountInspectionsByUserID(ctx, params.UserID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count inspections")
	}

	// Get paginated results
	rows, err := s.queries.ListInspectionsWithClientByUserID(ctx, repository.ListInspectionsWithClientByUserIDParams{
		UserID: params.UserID,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list inspections")
	}

	// Convert to domain types
	inspections := make([]domain.Inspection, 0, len(rows))
	for _, row := range rows {
		createdAt := time.Time{}
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time
		}
		updatedAt := time.Time{}
		if row.UpdatedAt.Valid {
			updatedAt = row.UpdatedAt.Time
		}

		inspections = append(inspections, domain.Inspection{
			ID:             row.ID,
			UserID:         row.UserID,
			ClientID:       nullUUIDToPtr(row.ClientID),
			Title:          row.Title,
			Status:         domain.InspectionStatus(row.Status),
			InspectionDate: row.InspectionDate,
			AddressLine1:   row.AddressLine1,
			AddressLine2:   domain.NullStringValue(row.AddressLine2),
			City:           row.City,
			State:          row.State,
			PostalCode:     row.PostalCode,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			ClientName:     row.ClientName,
			ViolationCount: int(row.ViolationCount),
		})
	}

	return &domain.ListInspectionsResult{
		Inspections: inspections,
		Total:       total,
		Limit:       params.Limit,
		Offset:      params.Offset,
	}, nil
}

// =============================================================================
// Update
// =============================================================================

// Update updates an existing inspection.
func (s *inspectionService) Update(ctx context.Context, params domain.UpdateInspectionParams) error {
	const op = "inspection.update"

	// Validate parameters
	if err := s.validateUpdateParams(params); err != nil {
		return err
	}

	// Get existing inspection to verify it's editable
	existing, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "inspection", params.ID.String())
		}
		return domain.Internal(err, op, "failed to get inspection")
	}

	// Check if inspection is editable
	status := domain.InspectionStatus(existing.Status)
	if status == domain.InspectionStatusAnalyzing {
		return domain.Invalid(op, "cannot edit inspection while analysis is in progress")
	}

	// If client_id is provided, verify it exists and belongs to the user
	if params.ClientID != nil {
		_, err := s.queries.GetClientByIDAndUserID(ctx, repository.GetClientByIDAndUserIDParams{
			ID:     *params.ClientID,
			UserID: params.UserID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.NotFound(op, "client", params.ClientID.String())
			}
			return domain.Internal(err, op, "failed to verify client")
		}
	}

	// Update the inspection
	err = s.queries.UpdateInspectionByIDAndUserID(ctx, repository.UpdateInspectionByIDAndUserIDParams{
		ID:                params.ID,
		UserID:            params.UserID,
		Title:             params.Title,
		ClientID:          domain.ToNullUUID(params.ClientID),
		InspectionDate:    params.InspectionDate,
		WeatherConditions: domain.ToNullString(params.WeatherConditions),
		Temperature:       domain.ToNullString(params.Temperature),
		InspectorNotes:    domain.ToNullString(params.InspectorNotes),
		AddressLine1:      params.AddressLine1,
		AddressLine2:      domain.ToNullString(params.AddressLine2),
		City:              params.City,
		State:             params.State,
		PostalCode:        params.PostalCode,
	})
	if err != nil {
		return domain.Internal(err, op, "failed to update inspection")
	}

	s.logger.Info("inspection updated",
		"inspection_id", params.ID,
		"user_id", params.UserID,
	)

	return nil
}

// validateUpdateParams validates inspection update parameters.
func (s *inspectionService) validateUpdateParams(params domain.UpdateInspectionParams) error {
	const op = "inspection.validate"

	// Title is required and must be 1-200 characters
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return domain.Invalid(op, "title is required")
	}
	if len(title) > 200 {
		return domain.Invalid(op, "title must be 200 characters or less")
	}

	// Address fields are required
	if strings.TrimSpace(params.AddressLine1) == "" {
		return domain.Invalid(op, "address is required")
	}
	if strings.TrimSpace(params.City) == "" {
		return domain.Invalid(op, "city is required")
	}
	if strings.TrimSpace(params.State) == "" {
		return domain.Invalid(op, "state is required")
	}
	if strings.TrimSpace(params.PostalCode) == "" {
		return domain.Invalid(op, "postal code is required")
	}

	// Inspection date cannot be more than 1 year in the future
	oneYearFromNow := time.Now().AddDate(1, 0, 0)
	if params.InspectionDate.After(oneYearFromNow) {
		return domain.Invalid(op, "inspection date cannot be more than 1 year in the future")
	}

	return nil
}

// =============================================================================
// Delete
// =============================================================================

// Delete deletes an inspection.
func (s *inspectionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const op = "inspection.delete"

	// Verify inspection exists and belongs to user
	_, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "inspection", id.String())
		}
		return domain.Internal(err, op, "failed to get inspection")
	}

	// Delete the inspection (cascades to photos, violations, etc.)
	err = s.queries.DeleteInspectionByIDAndUserID(ctx, repository.DeleteInspectionByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return domain.Internal(err, op, "failed to delete inspection")
	}

	s.logger.Info("inspection deleted",
		"inspection_id", id,
		"user_id", userID,
	)

	return nil
}

// =============================================================================
// UpdateStatus
// =============================================================================

// UpdateStatus updates the status of an inspection.
func (s *inspectionService) UpdateStatus(ctx context.Context, params domain.UpdateInspectionStatusParams) error {
	const op = "inspection.update_status"

	// Validate new status
	if !params.Status.IsValid() {
		return domain.Invalid(op, fmt.Sprintf("invalid status: %s", params.Status))
	}

	// Get existing inspection
	existing, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "inspection", params.ID.String())
		}
		return domain.Internal(err, op, "failed to get inspection")
	}

	// Check if status transition is valid
	currentStatus := domain.InspectionStatus(existing.Status)
	if !currentStatus.CanTransitionTo(params.Status) {
		return domain.Invalid(op, fmt.Sprintf("cannot transition from %s to %s", currentStatus, params.Status))
	}

	// Update status
	err = s.queries.UpdateInspectionStatusByIDAndUserID(ctx, repository.UpdateInspectionStatusByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
		Status: string(params.Status),
	})
	if err != nil {
		return domain.Internal(err, op, "failed to update inspection status")
	}

	s.logger.Info("inspection status updated",
		"inspection_id", params.ID,
		"user_id", params.UserID,
		"old_status", currentStatus,
		"new_status", params.Status,
	)

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToInspection converts a repository inspection row to a domain Inspection.
func (s *inspectionService) rowToInspection(row repository.Inspection) *domain.Inspection {
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	return &domain.Inspection{
		ID:                row.ID,
		UserID:            row.UserID,
		ClientID:          nullUUIDToPtr(row.ClientID),
		Title:             row.Title,
		Status:            domain.InspectionStatus(row.Status),
		InspectionDate:    row.InspectionDate,
		WeatherConditions: domain.NullStringValue(row.WeatherConditions),
		Temperature:       domain.NullStringValue(row.Temperature),
		InspectorNotes:    domain.NullStringValue(row.InspectorNotes),
		AddressLine1:      row.AddressLine1,
		AddressLine2:      domain.NullStringValue(row.AddressLine2),
		City:              row.City,
		State:             row.State,
		PostalCode:        row.PostalCode,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

// =============================================================================
// GetAnalysisStatus
// =============================================================================

// GetAnalysisStatus returns the computed analysis status for an inspection.
func (s *inspectionService) GetAnalysisStatus(ctx context.Context, inspectionID, userID uuid.UUID) (*domain.AnalysisStatus, error) {
	const op = "inspection.get_analysis_status"

	inspection, err := s.GetByID(ctx, inspectionID, userID)
	if err != nil {
		return nil, err
	}

	pendingCount, err := s.queries.CountPendingImagesByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count pending images")
	}

	totalCount, err := s.queries.CountImagesByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count images")
	}

	violationCount, err := s.queries.CountViolationsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count violations")
	}

	hasPendingJob, err := s.queries.HasPendingAnalysisJob(ctx, inspectionID.String())
	if err != nil {
		return nil, domain.Internal(err, op, "failed to check pending job")
	}

	canAnalyze, message := inspection.DetermineAnalysisAction(pendingCount, totalCount, hasPendingJob)

	return &domain.AnalysisStatus{
		InspectionID:   inspectionID,
		Status:         inspection.Status,
		CanAnalyze:     canAnalyze,
		IsAnalyzing:    hasPendingJob,
		HasImages:      totalCount > 0,
		PendingImages:  pendingCount,
		TotalImages:    totalCount,
		AnalyzedImages: totalCount - pendingCount,
		ViolationCount: violationCount,
		Message:        message,
		PollingEnabled: hasPendingJob,
	}, nil
}

// =============================================================================
// StartAnalysis
// =============================================================================

// StartAnalysis transitions an inspection to analyzing status.
func (s *inspectionService) StartAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error {
	const op = "inspection.start_analysis"

	inspection, err := s.GetByID(ctx, inspectionID, userID)
	if err != nil {
		return err
	}

	// Already analyzing — no-op for idempotent job retries
	if inspection.Status == domain.InspectionStatusAnalyzing {
		return nil
	}

	if err := inspection.TransitionTo(domain.InspectionStatusAnalyzing); err != nil {
		return domain.Invalid(op, err.Error())
	}

	if err := s.queries.UpdateInspectionStatusByIDAndUserID(ctx, repository.UpdateInspectionStatusByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
		Status: string(domain.InspectionStatusAnalyzing),
	}); err != nil {
		return domain.Internal(err, op, "failed to update inspection status")
	}

	s.logger.Info("inspection analysis started",
		"inspection_id", inspectionID,
		"user_id", userID,
		"old_status", inspection.Status,
	)

	return nil
}

// =============================================================================
// CompleteAnalysis
// =============================================================================

// CompleteAnalysis transitions an inspection from analyzing to review status.
func (s *inspectionService) CompleteAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error {
	const op = "inspection.complete_analysis"

	inspection, err := s.GetByID(ctx, inspectionID, userID)
	if err != nil {
		return err
	}

	if err := inspection.TransitionTo(domain.InspectionStatusReview); err != nil {
		return domain.Invalid(op, err.Error())
	}

	if err := s.queries.UpdateInspectionStatusByIDAndUserID(ctx, repository.UpdateInspectionStatusByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
		Status: string(domain.InspectionStatusReview),
	}); err != nil {
		return domain.Internal(err, op, "failed to update inspection status")
	}

	s.logger.Info("inspection analysis completed",
		"inspection_id", inspectionID,
		"user_id", userID,
	)

	return nil
}

// nullUUIDToPtr converts a uuid.NullUUID to a *uuid.UUID.
func nullUUIDToPtr(nu uuid.NullUUID) *uuid.UUID {
	if !nu.Valid {
		return nil
	}
	return &nu.UUID
}

// =============================================================================
// TriggerAnalysis
// =============================================================================

// TriggerAnalysis enqueues a job to analyze an inspection's images.
func (s *inspectionService) TriggerAnalysis(ctx context.Context, inspectionID, userID uuid.UUID) error {
	const op = "inspection.trigger_analysis"

	if s.jobEnqueuer == nil {
		return domain.Internal(nil, op, "job enqueuer not configured")
	}

	// Check quota if quota service is configured
	if s.quotaService != nil {
		// Get user's subscription tier
		user, err := s.queries.GetUserByID(ctx, userID)
		if err != nil {
			return domain.Internal(err, op, "failed to get user")
		}

		// Determine tier: use subscription tier if active, otherwise free
		tier := domain.SubscriptionTierFree
		status := domain.SubscriptionStatus(domain.NullStringValue(user.SubscriptionStatus))
		if status == domain.SubscriptionStatusActive || status == domain.SubscriptionStatusTrialing {
			tierStr := domain.NullStringValue(user.SubscriptionTier)
			if tierStr != "" {
				tier = domain.SubscriptionTier(tierStr)
			}
		}

		if err := s.quotaService.CheckAnalysisQuota(ctx, userID, tier); err != nil {
			return err
		}
	}

	_, err := s.jobEnqueuer.EnqueueAnalyzeInspection(ctx, inspectionID, userID)
	if err != nil {
		return domain.Internal(err, op, "failed to enqueue analysis job")
	}

	s.logger.Info("Analysis job enqueued",
		"inspection_id", inspectionID,
		"user_id", userID,
	)

	return nil
}

// =============================================================================
// HasPendingAnalysisJob
// =============================================================================

// HasPendingAnalysisJob checks if there is a pending or running analysis job.
func (s *inspectionService) HasPendingAnalysisJob(ctx context.Context, inspectionID uuid.UUID) (bool, error) {
	const op = "inspection.has_pending_analysis_job"

	hasPending, err := s.queries.HasPendingAnalysisJob(ctx, inspectionID.String())
	if err != nil {
		return false, domain.Internal(err, op, "failed to check pending analysis job")
	}

	return hasPending, nil
}
