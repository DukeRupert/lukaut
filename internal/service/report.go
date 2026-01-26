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

// ReportService defines operations for preparing report data.
type ReportService interface {
	// PrepareReportData aggregates all data needed for report generation.
	PrepareReportData(ctx context.Context, inspectionID, userID uuid.UUID) (*domain.ReportData, error)
}

// =============================================================================
// Implementation
// =============================================================================

type reportService struct {
	queries *repository.Queries
	storage storage.Storage
	logger  *slog.Logger
}

// NewReportService creates a new ReportService.
func NewReportService(
	queries *repository.Queries,
	storage storage.Storage,
	logger *slog.Logger,
) ReportService {
	return &reportService{
		queries: queries,
		storage: storage,
		logger:  logger,
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

	// Fetch confirmed violations
	violations, err := s.queries.ListConfirmedViolationsByInspectionID(ctx, inspectionID)
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
