package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/DukeRupert/lukaut/internal/worker"
)

// GenerateReportHandler processes jobs that generate PDF or DOCX reports.
// It creates a formatted report from inspection data and uploads it to storage.
type GenerateReportHandler struct {
	queries *repository.Queries
	storage storage.Storage
	logger  *slog.Logger
}

// NewGenerateReportHandler creates a new handler for report generation jobs.
func NewGenerateReportHandler(queries *repository.Queries, storage storage.Storage, logger *slog.Logger) *GenerateReportHandler {
	return &GenerateReportHandler{
		queries: queries,
		storage: storage,
		logger:  logger,
	}
}

// Type returns the job type identifier.
func (h *GenerateReportHandler) Type() string {
	return worker.JobTypeGenerateReport
}

// Handle executes the report generation job.
// This is a skeleton implementation - actual report generation will be implemented in P1-013.
func (h *GenerateReportHandler) Handle(ctx context.Context, payload []byte) error {
	// Unmarshal the payload
	var p worker.GenerateReportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return worker.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	// Validate format
	if p.Format != "pdf" && p.Format != "docx" {
		return worker.NewPermanentError(fmt.Errorf("invalid format: %s (must be 'pdf' or 'docx')", p.Format))
	}

	h.logger.Info("Generating report",
		"inspection_id", p.InspectionID,
		"user_id", p.UserID,
		"format", p.Format,
	)

	// TODO: P1-013 will implement the actual report generation logic:
	// 1. Fetch inspection, violations, and regulations from database
	// 2. Generate PDF or DOCX report using report package
	// 3. Upload generated report to storage
	// 4. Create report record in database with storage key
	// 5. Update inspection status if needed

	h.logger.Info("Report generation completed (placeholder)",
		"inspection_id", p.InspectionID,
		"format", p.Format,
	)

	return nil
}
