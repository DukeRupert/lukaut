// Package service contains the business logic layer.
//
// This file implements the report service for preparing report data
// from inspection, violation, and user records.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/google/uuid"
)

// =============================================================================
// Interface Definition
// =============================================================================

// ReportService defines operations for managing and generating reports.
type ReportService interface {
	// PrepareReportData aggregates all data needed for report generation.
	PrepareReportData(ctx context.Context, inspectionID, userID uuid.UUID) (*domain.ReportData, error)

	// GetByID retrieves a report by ID with user authorization.
	// Returns domain.ENOTFOUND if report doesn't exist or doesn't belong to user.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Report, error)

	// ListByInspection returns all reports for an inspection owned by the user.
	ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Report, error)

	// TriggerGeneration enqueues a job to generate a report.
	// Returns domain.EINVALID if generation cannot proceed.
	TriggerGeneration(ctx context.Context, inspectionID, userID uuid.UUID, format, recipientEmail string) error
}

// =============================================================================
// Implementation
// =============================================================================

type reportService struct {
	queries      *repository.Queries
	storage      storage.Storage
	jobEnqueuer  JobEnqueuer
	quotaService QuotaService
	logger       *slog.Logger
}

// NewReportService creates a new ReportService.
func NewReportService(
	queries *repository.Queries,
	storage storage.Storage,
	jobEnqueuer JobEnqueuer,
	quotaService QuotaService,
	logger *slog.Logger,
) ReportService {
	return &reportService{
		queries:      queries,
		storage:      storage,
		jobEnqueuer:  jobEnqueuer,
		quotaService: quotaService,
		logger:       logger,
	}
}

// =============================================================================
// PrepareReportData
// =============================================================================

// PrepareReportData aggregates all data needed for report generation.
func (s *reportService) PrepareReportData(ctx context.Context, inspectionID, userID uuid.UUID) (*domain.ReportData, error) {
	// Fetch user with business profile
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	// Fetch inspection with client info
	inspection, err := s.queries.GetInspectionWithClientByIDAndUserID(ctx, repository.GetInspectionWithClientByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch inspection with client: %w", err)
	}

	// Fetch client details if inspection has client_id
	var clientName, clientEmail, clientPhone string
	if inspection.ClientID.Valid {
		client, err := s.queries.GetClientByID(ctx, inspection.ClientID.UUID)
		if err == nil {
			clientName = client.Name
			clientEmail = domain.NullStringValue(client.Email)
			clientPhone = domain.NullStringValue(client.Phone)
		} else {
			s.logger.Warn("Failed to fetch client for inspection",
				"inspection_id", inspectionID,
				"client_id", inspection.ClientID.UUID,
				"error", err,
			)
		}
	}

	// Fetch confirmed violations (with authorization check for defense in depth)
	violations, err := s.queries.ListConfirmedViolationsByInspectionIDAndUserID(ctx, repository.ListConfirmedViolationsByInspectionIDAndUserIDParams{
		InspectionID: inspectionID,
		UserID:       userID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch confirmed violations: %w", err)
	}

	// Build report violations with regulations
	reportViolations := make([]domain.ReportViolation, 0, len(violations))
	for i, v := range violations {
		// Fetch regulations for this violation
		regs, err := s.queries.ListRegulationsByViolationID(ctx, v.ID)
		if err != nil {
			s.logger.Warn("Failed to fetch regulations for violation",
				"violation_id", v.ID,
				"error", err,
			)
			regs = nil
		}

		// Convert regulations to report format
		reportRegs := make([]domain.ReportRegulation, 0, len(regs))
		for _, r := range regs {
			relevanceScore := 0.0
			if r.RelevanceScore.Valid {
				relevanceScore = r.RelevanceScore.Float64
			}

			reportRegs = append(reportRegs, domain.ReportRegulation{
				StandardNumber: r.StandardNumber,
				Title:          r.Title,
				Category:       r.Category,
				FullText:       r.FullText,
				IsPrimary:      r.IsPrimary.Valid && r.IsPrimary.Bool,
				RelevanceScore: relevanceScore,
			})
		}

		// Get thumbnail URL if violation has image
		thumbnailURL := ""
		if v.ImageID.Valid {
			img, err := s.queries.GetImageByID(ctx, v.ImageID.UUID)
			if err == nil && img.ThumbnailKey.Valid {
				url, err := s.storage.URL(ctx, img.ThumbnailKey.String, time.Hour)
				if err == nil {
					thumbnailURL = url
				} else {
					s.logger.Warn("Failed to generate thumbnail URL",
						"image_id", v.ImageID.UUID,
						"error", err,
					)
				}
			}
		}

		reportViolations = append(reportViolations, domain.ReportViolation{
			Number:         i + 1,
			Description:    v.Description,
			Severity:       domain.ViolationSeverity(domain.NullStringValue(v.Severity)),
			InspectorNotes: domain.NullStringValue(v.InspectorNotes),
			ThumbnailURL:   thumbnailURL,
			Regulations:    reportRegs,
		})
	}

	// Format inspector address
	inspectorAddress := formatAddress(
		domain.NullStringValue(user.BusinessAddressLine1),
		domain.NullStringValue(user.BusinessAddressLine2),
		domain.NullStringValue(user.BusinessCity),
		domain.NullStringValue(user.BusinessState),
		domain.NullStringValue(user.BusinessPostalCode),
	)

	// Build report data using business profile if available, falling back to user data
	inspectorName := domain.NullStringValue(user.BusinessName)
	if inspectorName == "" {
		inspectorName = user.Name
	}

	inspectorEmail := domain.NullStringValue(user.BusinessEmail)
	if inspectorEmail == "" {
		inspectorEmail = user.Email
	}

	return &domain.ReportData{
		// Inspector info
		InspectorName:    inspectorName,
		InspectorCompany: domain.NullStringValue(user.BusinessName),
		InspectorLicense: domain.NullStringValue(user.BusinessLicenseNumber),
		InspectorEmail:   inspectorEmail,
		InspectorPhone:   domain.NullStringValue(user.BusinessPhone),
		InspectorAddress: inspectorAddress,
		InspectorLogoURL: domain.NullStringValue(user.BusinessLogoUrl),

		// Inspection details
		InspectionID:      inspection.ID,
		InspectionTitle:   inspection.Title,
		InspectionDate:    inspection.InspectionDate,
		WeatherConditions: domain.NullStringValue(inspection.WeatherConditions),
		Temperature:       domain.NullStringValue(inspection.Temperature),
		InspectorNotes:    domain.NullStringValue(inspection.InspectorNotes),

		// Location info
		SiteName:       inspection.Title,
		SiteAddress:    inspection.AddressLine1,
		SiteCity:       inspection.City,
		SiteState:      inspection.State,
		SitePostalCode: inspection.PostalCode,

		// Client info
		ClientName:  clientName,
		ClientEmail: clientEmail,
		ClientPhone: clientPhone,

		// Violations
		Violations: reportViolations,

		// Metadata
		GeneratedAt: time.Now(),
	}, nil
}

// formatAddress combines address components into a formatted string.
func formatAddress(line1, line2, city, state, postal string) string {
	if line1 == "" {
		return ""
	}

	addr := line1
	if line2 != "" {
		addr += "\n" + line2
	}
	if city != "" || state != "" || postal != "" {
		addr += "\n"
		if city != "" {
			addr += city
		}
		if state != "" {
			if city != "" {
				addr += ", "
			}
			addr += state
		}
		if postal != "" {
			addr += " " + postal
		}
	}
	return addr
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves a report by ID with user authorization.
func (s *reportService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Report, error) {
	const op = "report.get_by_id"

	report, err := s.queries.GetReportByIDAndUserID(ctx, repository.GetReportByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return nil, domain.NotFound(op, "report", id.String())
	}

	return s.repoReportToDomain(report), nil
}

// =============================================================================
// ListByInspection
// =============================================================================

// ListByInspection returns all reports for an inspection owned by the user.
func (s *reportService) ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Report, error) {
	const op = "report.list_by_inspection"

	repoReports, err := s.queries.ListReportsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list reports")
	}

	// Filter to only reports owned by this user
	var reports []domain.Report
	for _, r := range repoReports {
		if r.UserID == userID {
			reports = append(reports, *s.repoReportToDomain(r))
		}
	}

	return reports, nil
}

// =============================================================================
// TriggerGeneration
// =============================================================================

// TriggerGeneration enqueues a job to generate a report.
func (s *reportService) TriggerGeneration(ctx context.Context, inspectionID, userID uuid.UUID, format, recipientEmail string) error {
	const op = "report.trigger_generation"

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

		if err := s.quotaService.CheckReportQuota(ctx, userID, tier); err != nil {
			return err
		}
	}

	_, err := s.jobEnqueuer.EnqueueGenerateReport(ctx, inspectionID, userID, format, recipientEmail)
	if err != nil {
		return domain.Internal(err, op, "failed to enqueue report generation job")
	}

	s.logger.Info("Report generation job enqueued",
		"inspection_id", inspectionID,
		"user_id", userID,
		"format", format,
	)

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// repoReportToDomain converts a repository Report to a domain Report.
func (s *reportService) repoReportToDomain(r repository.Report) *domain.Report {
	var pdfKey, docxKey string
	if r.PdfStorageKey.Valid {
		pdfKey = r.PdfStorageKey.String
	}
	if r.DocxStorageKey.Valid {
		docxKey = r.DocxStorageKey.String
	}

	generatedAt := time.Time{}
	if r.GeneratedAt.Valid {
		generatedAt = r.GeneratedAt.Time
	}

	return &domain.Report{
		ID:             r.ID,
		InspectionID:   r.InspectionID,
		UserID:         r.UserID,
		PDFStorageKey:  pdfKey,
		DOCXStorageKey: docxKey,
		ViolationCount: int(r.ViolationCount),
		GeneratedAt:    generatedAt,
	}
}
