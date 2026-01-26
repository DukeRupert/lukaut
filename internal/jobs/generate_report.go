package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/email"
	"github.com/DukeRupert/lukaut/internal/metrics"
	"github.com/DukeRupert/lukaut/internal/report"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/DukeRupert/lukaut/internal/worker"
)

// GenerateReportHandler processes jobs that generate PDF or DOCX reports.
// It creates a formatted report from inspection data and uploads it to storage.
type GenerateReportHandler struct {
	queries       *repository.Queries
	storage       storage.Storage
	emailService  email.EmailService
	reportService service.ReportService
	pdfGen        report.Generator
	docxGen       report.Generator
	logger        *slog.Logger
	baseURL       string
}

// NewGenerateReportHandler creates a new handler for report generation jobs.
func NewGenerateReportHandler(
	queries *repository.Queries,
	storage storage.Storage,
	emailService email.EmailService,
	reportService service.ReportService,
	logger *slog.Logger,
	baseURL string,
) *GenerateReportHandler {
	return &GenerateReportHandler{
		queries:       queries,
		storage:       storage,
		emailService:  emailService,
		reportService: reportService,
		pdfGen:        report.NewHTMLPDFGenerator(logger),
		docxGen:       report.NewHTMLDOCXGenerator(logger),
		logger:        logger,
		baseURL:       baseURL,
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
	reportData, err := h.reportService.PrepareReportData(ctx, p.InspectionID, p.UserID)
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
	metrics.ReportsGenerated.WithLabelValues(p.Format).Inc()

	// 10. Send email notification to inspector (optional - don't fail job if email fails)
	reportURL := fmt.Sprintf("%s/reports/%s/download?format=%s", h.baseURL, dbReport.ID, p.Format)
	if h.emailService != nil && reportData.InspectorEmail != "" {
		if err := h.emailService.SendReportReadyEmail(
			ctx,
			reportData.InspectorEmail,
			reportData.InspectorName,
			reportURL,
		); err != nil {
			// Log error but don't fail the job - report was generated successfully
			h.logger.Error("Failed to send report ready email to inspector",
				"error", err,
				"user_id", p.UserID,
				"report_id", dbReport.ID,
			)
		} else {
			h.logger.Info("Report ready email sent to inspector",
				"user_id", p.UserID,
				"email", reportData.InspectorEmail,
			)
		}
	}

	// 11. Send report to client/recipient if email was provided
	if h.emailService != nil && p.RecipientEmail != "" {
		if err := h.emailService.SendReportToClientEmail(
			ctx,
			p.RecipientEmail,
			reportData.InspectorName,
			reportData.InspectorCompany,
			reportData.SiteName,
			reportURL,
		); err != nil {
			// Log error but don't fail the job - report was generated successfully
			h.logger.Error("Failed to send report to client",
				"error", err,
				"recipient_email", p.RecipientEmail,
				"report_id", dbReport.ID,
			)
		} else {
			h.logger.Info("Report sent to client",
				"recipient_email", p.RecipientEmail,
				"report_id", dbReport.ID,
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

