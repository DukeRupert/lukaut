package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/email"
	"github.com/DukeRupert/lukaut/internal/report"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/DukeRupert/lukaut/internal/worker"
	"github.com/google/uuid"
)

// GenerateReportHandler processes jobs that generate PDF or DOCX reports.
// It creates a formatted report from inspection data and uploads it to storage.
type GenerateReportHandler struct {
	queries      *repository.Queries
	storage      storage.Storage
	emailService email.EmailService
	pdfGen       report.Generator
	docxGen      report.Generator
	logger       *slog.Logger
	baseURL      string
}

// NewGenerateReportHandler creates a new handler for report generation jobs.
func NewGenerateReportHandler(
	queries *repository.Queries,
	storage storage.Storage,
	emailService email.EmailService,
	logger *slog.Logger,
	baseURL string,
) *GenerateReportHandler {
	return &GenerateReportHandler{
		queries:      queries,
		storage:      storage,
		emailService: emailService,
		pdfGen:       report.NewPDFGenerator(),
		docxGen:      report.NewDOCXGenerator(),
		logger:       logger,
		baseURL:      baseURL,
	}
}

// Type returns the job type identifier.
func (h *GenerateReportHandler) Type() string {
	return worker.JobTypeGenerateReport
}

// Handle executes the report generation job.
func (h *GenerateReportHandler) Handle(ctx context.Context, payload []byte) error {
	// 1. Unmarshal the payload
	var p worker.GenerateReportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return worker.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	// 2. Validate format
	format := domain.ReportFormat(p.Format)
	if !format.IsValid() {
		return worker.NewPermanentError(fmt.Errorf("invalid format: %s (must be 'pdf' or 'docx')", p.Format))
	}

	h.logger.Info("Generating report",
		"inspection_id", p.InspectionID,
		"user_id", p.UserID,
		"format", p.Format,
	)

	// 3. Fetch and validate inspection
	inspection, err := h.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     p.InspectionID,
		UserID: p.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return worker.NewPermanentError(fmt.Errorf("inspection not found: %s", p.InspectionID))
		}
		return fmt.Errorf("fetch inspection: %w", err)
	}

	// 4. Verify inspection status allows report generation
	// Allow both 'completed' and 'review' status - inspectors may want to generate
	// preliminary reports during review
	if inspection.Status != domain.InspectionStatusCompleted.String() &&
		inspection.Status != domain.InspectionStatusReview.String() {
		return worker.NewPermanentError(fmt.Errorf(
			"inspection must be in 'review' or 'completed' status to generate report, got: %s",
			inspection.Status,
		))
	}

	// 5. Aggregate all report data
	reportData, err := h.aggregateReportData(ctx, p.InspectionID, p.UserID)
	if err != nil {
		return fmt.Errorf("aggregate report data: %w", err)
	}

	// 6. Select generator based on format
	var gen report.Generator
	if format == domain.ReportFormatPDF {
		gen = h.pdfGen
	} else {
		gen = h.docxGen
	}

	// 7. Generate report to buffer
	var buf bytes.Buffer
	bytesWritten, err := gen.Generate(ctx, reportData, &buf)
	if err != nil {
		return fmt.Errorf("generate %s: %w", format, err)
	}

	h.logger.Info("Report generated",
		"inspection_id", p.InspectionID,
		"format", format,
		"size_bytes", bytesWritten,
		"violation_count", len(reportData.Violations),
	)

	// 8. Upload to storage
	storageKey := storage.ReportKey(p.InspectionID, p.Format)
	err = h.storage.Put(ctx, storageKey, &buf, storage.PutOptions{
		ContentType: format.ContentType(),
		Overwrite:   true,
	})
	if err != nil {
		return fmt.Errorf("upload report to storage: %w", err)
	}

	// 9. Create report record in database
	createParams := repository.CreateReportParams{
		InspectionID:   p.InspectionID,
		UserID:         p.UserID,
		ViolationCount: int32(len(reportData.Violations)),
	}
	if format == domain.ReportFormatPDF {
		createParams.PdfStorageKey = domain.ToNullString(storageKey)
	} else {
		createParams.DocxStorageKey = domain.ToNullString(storageKey)
	}

	dbReport, err := h.queries.CreateReport(ctx, createParams)
	if err != nil {
		return fmt.Errorf("create report record: %w", err)
	}

	// 10. Send email notification (optional - don't fail job if email fails)
	if h.emailService != nil && reportData.InspectorEmail != "" {
		reportURL := fmt.Sprintf("%s/reports/%s", h.baseURL, dbReport.ID)
		if err := h.emailService.SendReportReadyEmail(
			ctx,
			reportData.InspectorEmail,
			reportData.InspectorName,
			reportURL,
		); err != nil {
			// Log error but don't fail the job - report was generated successfully
			h.logger.Error("Failed to send report ready email",
				"error", err,
				"user_id", p.UserID,
				"report_id", dbReport.ID,
			)
		} else {
			h.logger.Info("Report ready email sent",
				"user_id", p.UserID,
				"email", reportData.InspectorEmail,
			)
		}
	}

	h.logger.Info("Report generation completed",
		"report_id", dbReport.ID,
		"inspection_id", p.InspectionID,
		"storage_key", storageKey,
		"format", format,
	)

	return nil
}

// aggregateReportData fetches all data needed for report generation.
func (h *GenerateReportHandler) aggregateReportData(
	ctx context.Context,
	inspectionID uuid.UUID,
	userID uuid.UUID,
) (*domain.ReportData, error) {
	// Fetch user with business profile
	user, err := h.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	// Fetch inspection with site info
	inspection, err := h.queries.GetInspectionWithSiteByIDAndUserID(ctx, repository.GetInspectionWithSiteByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch inspection with site: %w", err)
	}

	// Fetch site details to get postal code and client
	var sitePostalCode string
	var clientName, clientEmail, clientPhone string

	if inspection.SiteID.Valid {
		site, err := h.queries.GetSiteByID(ctx, inspection.SiteID.UUID)
		if err == nil {
			sitePostalCode = site.PostalCode

			// Fetch client info if site has client_id
			if site.ClientID.Valid {
				client, err := h.queries.GetClientByID(ctx, site.ClientID.UUID)
				if err == nil {
					clientName = client.Name
					clientEmail = domain.NullStringValue(client.Email)
					clientPhone = domain.NullStringValue(client.Phone)
				} else {
					h.logger.Warn("Failed to fetch client for site",
						"site_id", inspection.SiteID.UUID,
						"client_id", site.ClientID.UUID,
						"error", err,
					)
				}
			}
		} else {
			h.logger.Warn("Failed to fetch site details",
				"site_id", inspection.SiteID.UUID,
				"error", err,
			)
		}
	}

	// Fetch confirmed violations
	violations, err := h.queries.ListConfirmedViolationsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("fetch confirmed violations: %w", err)
	}

	// Build report violations with regulations
	reportViolations := make([]domain.ReportViolation, 0, len(violations))
	for i, v := range violations {
		// Fetch regulations for this violation
		regs, err := h.queries.ListRegulationsByViolationID(ctx, v.ID)
		if err != nil {
			h.logger.Warn("Failed to fetch regulations for violation",
				"violation_id", v.ID,
				"error", err,
			)
			regs = nil // Continue without regulations
		}

		// Convert regulations to report format
		reportRegs := make([]domain.ReportRegulation, 0, len(regs))
		for _, r := range regs {
			relevanceScore := 0.0
			if r.RelevanceScore.Valid {
				if parsed, err := strconv.ParseFloat(r.RelevanceScore.String, 64); err == nil {
					relevanceScore = parsed
				}
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
			img, err := h.queries.GetImageByID(ctx, v.ImageID.UUID)
			if err == nil && img.ThumbnailKey.Valid {
				// Generate presigned URL (valid for 1 hour)
				url, err := h.storage.URL(ctx, img.ThumbnailKey.String, time.Hour)
				if err == nil {
					thumbnailURL = url
				} else {
					h.logger.Warn("Failed to generate thumbnail URL",
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

		// Site info
		SiteName:       inspection.SiteName,
		SiteAddress:    inspection.SiteAddress,
		SiteCity:       inspection.SiteCity,
		SiteState:      inspection.SiteState,
		SitePostalCode: sitePostalCode,

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
