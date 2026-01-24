// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements report handlers for downloading generated reports.
package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	reporttempl "github.com/DukeRupert/lukaut/internal/templ/report"
	"github.com/google/uuid"
)

// ReportHandler handles HTTP requests related to reports.
type ReportHandler struct {
	queries *repository.Queries
	storage storage.Storage
	logger  *slog.Logger
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(
	queries *repository.Queries,
	storage storage.Storage,
	logger *slog.Logger,
) *ReportHandler {
	return &ReportHandler{
		queries: queries,
		storage: storage,
		logger:  logger,
	}
}

// Download handles downloading a report file.
// GET /reports/{id}/download?format=pdf|docx
func (h *ReportHandler) Download(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse report ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	// Get format from query string
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf" // default to PDF
	}
	if format != "pdf" && format != "docx" {
		http.Error(w, "Invalid format: must be 'pdf' or 'docx'", http.StatusBadRequest)
		return
	}

	// Fetch report with user authorization
	report, err := h.queries.GetReportByIDAndUserID(r.Context(), repository.GetReportByIDAndUserIDParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		h.logger.Error("failed to fetch report", "error", err, "report_id", id)
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Get the appropriate storage key
	var storageKey string
	reportFormat := domain.ReportFormat(format)
	if reportFormat == domain.ReportFormatPDF && report.PdfStorageKey.Valid {
		storageKey = report.PdfStorageKey.String
	} else if reportFormat == domain.ReportFormatDOCX && report.DocxStorageKey.Valid {
		storageKey = report.DocxStorageKey.String
	} else {
		http.Error(w, fmt.Sprintf("%s version not available for this report", format), http.StatusNotFound)
		return
	}

	// Fetch from storage
	reader, info, err := h.storage.Get(r.Context(), storageKey)
	if err != nil {
		h.logger.Error("failed to fetch report from storage", "error", err, "storage_key", storageKey)
		http.Error(w, "Failed to retrieve report", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set response headers
	w.Header().Set("Content-Type", reportFormat.ContentType())
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))

	// Generate filename
	filename := fmt.Sprintf("report-%s.%s", report.ID.String()[:8], format)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Stream the file
	_, err = io.Copy(w, reader)
	if err != nil {
		h.logger.Error("failed to stream report", "error", err, "report_id", id)
		return
	}

	h.logger.Info("Report downloaded",
		"report_id", id,
		"user_id", user.ID,
		"format", format,
	)
}

// GetDownloadURL returns a presigned URL for downloading a report.
// GET /reports/{id}/url?format=pdf|docx
func (h *ReportHandler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse report ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	// Get format from query string
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	if format != "pdf" && format != "docx" {
		http.Error(w, "Invalid format: must be 'pdf' or 'docx'", http.StatusBadRequest)
		return
	}

	// Fetch report with user authorization
	report, err := h.queries.GetReportByIDAndUserID(r.Context(), repository.GetReportByIDAndUserIDParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		h.logger.Error("failed to fetch report", "error", err, "report_id", id)
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Get the appropriate storage key
	var storageKey string
	reportFormat := domain.ReportFormat(format)
	if reportFormat == domain.ReportFormatPDF && report.PdfStorageKey.Valid {
		storageKey = report.PdfStorageKey.String
	} else if reportFormat == domain.ReportFormatDOCX && report.DocxStorageKey.Valid {
		storageKey = report.DocxStorageKey.String
	} else {
		http.Error(w, fmt.Sprintf("%s version not available for this report", format), http.StatusNotFound)
		return
	}

	// Generate presigned URL (valid for 1 hour)
	url, err := h.storage.URL(r.Context(), storageKey, time.Hour)
	if err != nil {
		h.logger.Error("failed to generate presigned URL", "error", err, "storage_key", storageKey)
		http.Error(w, "Failed to generate download URL", http.StatusInternalServerError)
		return
	}

	// Redirect to the presigned URL
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// ListByInspection returns all reports for an inspection.
// GET /inspections/{id}/reports
func (h *ReportHandler) ListByInspection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse inspection ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	// Fetch reports for this inspection
	reports, err := h.queries.ListReportsByInspectionID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to list reports", "error", err, "inspection_id", id)
		http.Error(w, "Failed to list reports", http.StatusInternalServerError)
		return
	}

	// Filter to only reports owned by this user (for security)
	var userReports []repository.Report
	for _, report := range reports {
		if report.UserID == user.ID {
			userReports = append(userReports, report)
		}
	}

	// Return HTML partial showing available reports
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(userReports) == 0 {
		fmt.Fprint(w, `<p class="text-sm text-gray-500">No reports generated yet.</p>`)
		return
	}

	// Add report-ready class to signal polling should stop
	fmt.Fprint(w, `<div class="space-y-2 report-ready">`)
	for _, report := range userReports {
		fmt.Fprintf(w, `<div class="flex items-center justify-between bg-gray-50 p-3 rounded-md">`)
		fmt.Fprintf(w, `<div>`)
		fmt.Fprintf(w, `<span class="text-sm font-medium text-gray-900">Report generated %s</span>`, report.GeneratedAt.Time.Format("Jan 2, 2006 3:04 PM"))
		fmt.Fprintf(w, `<span class="ml-2 text-xs text-gray-500">%d violations</span>`, report.ViolationCount)
		fmt.Fprintf(w, `</div>`)
		fmt.Fprintf(w, `<div class="flex gap-2">`)
		if report.PdfStorageKey.Valid {
			fmt.Fprintf(w, `<a href="/reports/%s/download?format=pdf" class="inline-flex items-center rounded-md bg-red-50 px-2 py-1 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-600/10 hover:bg-red-100">PDF</a>`, report.ID)
		}
		if report.DocxStorageKey.Valid {
			fmt.Fprintf(w, `<a href="/reports/%s/download?format=docx" class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 ring-1 ring-inset ring-blue-600/10 hover:bg-blue-100">Word</a>`, report.ID)
		}
		fmt.Fprintf(w, `</div>`)
		fmt.Fprintf(w, `</div>`)
	}
	fmt.Fprint(w, `</div>`)
}

// Preview renders an HTML preview of the report without generating PDF/DOCX.
// GET /inspections/{id}/reports/preview
func (h *ReportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse inspection ID
	idStr := r.PathValue("id")
	inspectionID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	// Aggregate report data
	reportData, err := h.aggregateReportData(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to aggregate report data for preview", "error", err, "inspection_id", inspectionID)
		http.Error(w, "Failed to load report data", http.StatusInternalServerError)
		return
	}

	// Render HTML template
	templateData := &reporttempl.ReportTemplateData{
		ReportData: reportData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := reporttempl.Report(templateData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render report preview", "error", err, "inspection_id", inspectionID)
		http.Error(w, "Failed to render preview", http.StatusInternalServerError)
		return
	}
}

// aggregateReportData fetches all data needed for report preview/generation.
func (h *ReportHandler) aggregateReportData(
	ctx context.Context,
	inspectionID uuid.UUID,
	userID uuid.UUID,
) (*domain.ReportData, error) {
	// Fetch user with business profile
	user, err := h.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	// Fetch inspection with client info
	inspection, err := h.queries.GetInspectionWithClientByIDAndUserID(ctx, repository.GetInspectionWithClientByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch inspection: %w", err)
	}

	// Fetch client details if inspection has client_id
	var clientName, clientEmail, clientPhone string
	if inspection.ClientID.Valid {
		client, err := h.queries.GetClientByID(ctx, inspection.ClientID.UUID)
		if err == nil {
			clientName = client.Name
			clientEmail = domain.NullStringValue(client.Email)
			clientPhone = domain.NullStringValue(client.Phone)
		}
	}

	// Fetch confirmed violations
	violations, err := h.queries.ListConfirmedViolationsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("fetch violations: %w", err)
	}

	// Build report violations with regulations
	reportViolations := make([]domain.ReportViolation, 0, len(violations))
	for i, v := range violations {
		// Fetch regulations for this violation
		regs, err := h.queries.ListRegulationsByViolationID(ctx, v.ID)
		if err != nil {
			regs = nil
		}

		// Convert regulations to report format
		reportRegs := make([]domain.ReportRegulation, 0, len(regs))
		for _, reg := range regs {
			relevanceScore := 0.0
			if reg.RelevanceScore.Valid {
				if parsed, parseErr := parseFloat(reg.RelevanceScore.String); parseErr == nil {
					relevanceScore = parsed
				}
			}

			reportRegs = append(reportRegs, domain.ReportRegulation{
				StandardNumber: reg.StandardNumber,
				Title:          reg.Title,
				Category:       reg.Category,
				FullText:       reg.FullText,
				IsPrimary:      reg.IsPrimary.Valid && reg.IsPrimary.Bool,
				RelevanceScore: relevanceScore,
			})
		}

		// Get thumbnail URL if violation has image
		thumbnailURL := ""
		if v.ImageID.Valid {
			img, err := h.queries.GetImageByID(ctx, v.ImageID.UUID)
			if err == nil && img.ThumbnailKey.Valid {
				url, err := h.storage.URL(ctx, img.ThumbnailKey.String, time.Hour)
				if err == nil {
					thumbnailURL = url
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

	// Build inspector name and email with fallbacks
	inspectorName := domain.NullStringValue(user.BusinessName)
	if inspectorName == "" {
		inspectorName = user.Name
	}

	inspectorEmail := domain.NullStringValue(user.BusinessEmail)
	if inspectorEmail == "" {
		inspectorEmail = user.Email
	}

	return &domain.ReportData{
		InspectorName:    inspectorName,
		InspectorCompany: domain.NullStringValue(user.BusinessName),
		InspectorLicense: domain.NullStringValue(user.BusinessLicenseNumber),
		InspectorEmail:   inspectorEmail,
		InspectorPhone:   domain.NullStringValue(user.BusinessPhone),

		InspectionID:      inspection.ID,
		InspectionTitle:   inspection.Title,
		InspectionDate:    inspection.InspectionDate,
		WeatherConditions: domain.NullStringValue(inspection.WeatherConditions),
		Temperature:       domain.NullStringValue(inspection.Temperature),
		InspectorNotes:    domain.NullStringValue(inspection.InspectorNotes),

		SiteName:       inspection.Title,
		SiteAddress:    inspection.AddressLine1,
		SiteCity:       inspection.City,
		SiteState:      inspection.State,
		SitePostalCode: inspection.PostalCode,

		ClientName:  clientName,
		ClientEmail: clientEmail,
		ClientPhone: clientPhone,

		Violations:  reportViolations,
		GeneratedAt: time.Now(),
	}, nil
}

// parseFloat is a helper to parse float strings.
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// RegisterRoutes registers report routes on the provided ServeMux.
func (h *ReportHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /reports/{id}/download", requireUser(http.HandlerFunc(h.Download)))
	mux.Handle("GET /reports/{id}/url", requireUser(http.HandlerFunc(h.GetDownloadURL)))
	mux.Handle("GET /inspections/{id}/reports", requireUser(http.HandlerFunc(h.ListByInspection)))
	mux.Handle("GET /inspections/{id}/reports/preview", requireUser(http.HandlerFunc(h.Preview)))
}
