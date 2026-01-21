// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements templ-based inspection handlers.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/templ/components/pagination"
	"github.com/DukeRupert/lukaut/internal/templ/pages/inspections"
	"github.com/DukeRupert/lukaut/internal/templ/partials"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/DukeRupert/lukaut/internal/worker"
	"github.com/google/uuid"
)

// =============================================================================
// Templ-based Inspection Handlers
// =============================================================================

// IndexTempl displays a paginated list of inspections using templ.
func (h *InspectionHandler) IndexTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := int32(20)
	offset := int32((page - 1) * int(perPage))

	// Fetch inspections
	result, err := h.inspectionService.List(r.Context(), domain.ListInspectionsParams{
		UserID: user.ID,
		Limit:  perPage,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list inspections", "error", err, "user_id", user.ID)
		h.renderIndexErrorTempl(w, r, user, "Failed to load inspections. Please try again.")
		return
	}

	// Transform to display types
	displayInspections := make([]inspections.InspectionListItem, len(result.Inspections))
	for i, insp := range result.Inspections {
		displayInspections[i] = domainInspectionToListItem(&insp)
	}

	// Build pagination data
	paginationData := buildPaginationData(result)
	sharedPagination := pagination.Data{
		CurrentPage: paginationData.CurrentPage,
		TotalPages:  paginationData.TotalPages,
		PerPage:     paginationData.PerPage,
		Total:       paginationData.Total,
		HasPrevious: paginationData.HasPrevious,
		HasNext:     paginationData.HasNext,
		PrevPage:    paginationData.PrevPage,
		NextPage:    paginationData.NextPage,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if this is an htmx request - return just the table partial
	if r.Header.Get("HX-Request") == "true" {
		partialData := inspections.TablePartialData{
			Inspections: displayInspections,
			Pagination:  sharedPagination,
			BaseURL:     "/inspections",
		}
		if err := inspections.TablePartial(partialData).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render inspections table partial", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Full page render
	data := inspections.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspections: displayInspections,
		Pagination:  sharedPagination,
		Flash:       nil,
	}

	if err := inspections.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render inspections index", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewTempl displays the inspection creation form using templ.
func (h *InspectionHandler) NewTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch clients for dropdown
	clientOptions, err := h.fetchClientOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch clients", "error", err, "user_id", user.ID)
		h.renderIndexErrorTempl(w, r, user, "Failed to load clients. Please try again.")
		return
	}

	data := inspections.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspection:  nil,
		Clients:     domainClientsToOptions(clientOptions),
		Form: inspections.InspectionFormValues{
			InspectionDate: time.Now().Format("2006-01-02"),
		},
		Errors: make(map[string]string),
		Flash:  nil,
		IsEdit: false,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.NewPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render new inspection page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// EditTempl displays the inspection edit form using templ.
func (h *InspectionHandler) EditTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/inspections", http.StatusSeeOther)
		return
	}

	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Redirect(w, r, "/inspections", http.StatusSeeOther)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch clients for dropdown
	clientOptions, err := h.fetchClientOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch clients", "error", err, "user_id", user.ID)
		clientOptions = []ClientOption{}
	}

	clientIDStr := ""
	if inspection.ClientID != nil {
		clientIDStr = inspection.ClientID.String()
	}

	data := inspections.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspection:  domainInspectionToDisplay(inspection),
		Clients:     domainClientsToOptions(clientOptions),
		Form: inspections.InspectionFormValues{
			Title:             inspection.Title,
			ClientID:          clientIDStr,
			AddressLine1:      inspection.AddressLine1,
			AddressLine2:      inspection.AddressLine2,
			City:              inspection.City,
			State:             inspection.State,
			PostalCode:        inspection.PostalCode,
			InspectionDate:    inspection.InspectionDate.Format("2006-01-02"),
			WeatherConditions: inspection.WeatherConditions,
			Temperature:       inspection.Temperature,
			InspectorNotes:    inspection.InspectorNotes,
		},
		Errors: make(map[string]string),
		Flash:  nil,
		IsEdit: true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.EditPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render edit inspection page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ShowTempl displays inspection details using templ.
func (h *InspectionHandler) ShowTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch images for this inspection
	images, err := h.imageService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch images", "error", err, "inspection_id", id)
		images = []domain.Image{}
	}

	// Populate thumbnail URLs for gallery
	imageDisplays := make([]inspections.ImageDisplay, 0, len(images))
	for _, img := range images {
		thumbnailURL, err := h.imageService.GetThumbnailURL(r.Context(), img.ID, user.ID)
		if err != nil {
			h.logger.Error("failed to generate thumbnail URL", "error", err, "image_id", img.ID)
			thumbnailURL = ""
		}

		imageDisplays = append(imageDisplays, inspections.ImageDisplay{
			ID:               img.ID.String(),
			ThumbnailURL:     thumbnailURL,
			OriginalFilename: img.OriginalFilename,
			AnalysisStatus:   string(img.AnalysisStatus),
			SizeMB:           fmt.Sprintf("%.2f", img.SizeMB()),
		})
	}

	// Build analysis status data
	analysisStatus, err := h.buildAnalysisStatusData(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to build analysis status", "error", err, "inspection_id", id)
		analysisStatus = &AnalysisStatusData{
			InspectionID: id,
			Status:       inspection.Status,
			Message:      "Unable to load analysis status",
		}
	}

	// Fetch violations
	violations, err := h.violationService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to list violations", "error", err, "inspection_id", id)
		violations = []domain.Violation{}
	}

	// Calculate violation counts
	counts := inspections.ViolationCountsData{
		Total: len(violations),
	}
	for _, v := range violations {
		switch v.Status {
		case domain.ViolationStatusPending:
			counts.Pending++
		case domain.ViolationStatusConfirmed:
			counts.Confirmed++
		case domain.ViolationStatusRejected:
			counts.Rejected++
		}
	}

	// Fetch reports for this inspection
	repoReports, err := h.queries.ListReportsByInspectionID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to list reports", "error", err, "inspection_id", id)
		repoReports = []repository.Report{}
	}

	// Filter to user's reports and convert to display format
	var reportDisplays []inspections.ReportDisplay
	for _, report := range repoReports {
		if report.UserID == user.ID {
			reportDisplays = append(reportDisplays, inspections.ReportDisplay{
				ID:             report.ID.String(),
				GeneratedAt:    report.GeneratedAt.Time.Format("Jan 2, 2006 3:04 PM"),
				ViolationCount: int(report.ViolationCount),
				HasPDF:         report.PdfStorageKey.Valid,
				HasDOCX:        report.DocxStorageKey.Valid,
			})
		}
	}

	// Fetch client email if inspection has a client
	var clientEmail string
	if inspection.ClientID != nil {
		client, err := h.queries.GetClientByID(r.Context(), *inspection.ClientID)
		if err == nil && client.Email.Valid {
			clientEmail = client.Email.String
		}
	}

	// Can generate report if there are confirmed violations and inspection is in review or completed status
	canGenerateReport := counts.Confirmed > 0 && (inspection.Status == domain.InspectionStatusReview || inspection.Status == domain.InspectionStatusCompleted)

	data := inspections.ShowPageData{
		CurrentPath:  r.URL.Path,
		CSRFToken:    "",
		User:         domainUserToInspectionDisplay(user),
		Inspection:   domainInspectionToDisplay(inspection),
		InspectionID: id.String(),
		CanUpload:    inspection.CanAddPhotos(),
		IsAnalyzing:  analysisStatus.IsAnalyzing,
		GalleryData: inspections.ImageGalleryData{
			InspectionID: id.String(),
			Images:       imageDisplays,
			Errors:       []string{},
			CanUpload:    inspection.CanAddPhotos(),
			IsAnalyzing:  analysisStatus.IsAnalyzing,
		},
		AnalysisStatus:    domainAnalysisStatusToTempl(analysisStatus),
		Violations:        nil, // Not needed for show page
		ViolationCounts:   counts,
		Reports:           reportDisplays,
		ClientEmail:       clientEmail,
		CanGenerateReport: canGenerateReport,
		Flash:             nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.ShowPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render inspection show page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ReviewTempl redirects to the queue-based review interface.
func (h *InspectionHandler) ReviewTempl(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	http.Redirect(w, r, fmt.Sprintf("/inspections/%s/review/queue", idStr), http.StatusSeeOther)
}

// ReviewQueueTempl displays the keyboard-focused queue-based violation review page using templ.
// Supports htmx partial requests via ?pos=N query param.
func (h *InspectionHandler) ReviewQueueTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to fetch inspection", "error", err, "inspection_id", id)
			h.renderIndexErrorTempl(w, r, user, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch violations
	violations, err := h.violationService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to list violations", "error", err, "inspection_id", id)
		h.renderIndexErrorTempl(w, r, user, "Failed to load violations. Please try again.")
		return
	}

	// Calculate counts
	counts := inspections.ViolationCountsData{
		Total: len(violations),
	}
	for _, v := range violations {
		switch v.Status {
		case domain.ViolationStatusPending:
			counts.Pending++
		case domain.ViolationStatusConfirmed:
			counts.Confirmed++
		case domain.ViolationStatusRejected:
			counts.Rejected++
		}
	}

	// Parse position from query param (defaults to first pending or 0)
	position := 0
	if posStr := r.URL.Query().Get("pos"); posStr != "" {
		if p, err := strconv.Atoi(posStr); err == nil && p >= 0 && p < len(violations) {
			position = p
		}
	} else {
		// Default to first pending violation
		for i, v := range violations {
			if v.Status == domain.ViolationStatusPending {
				position = i
				break
			}
		}
	}

	// Check if all violations have been reviewed (complete state)
	isComplete := len(violations) > 0 && counts.Pending == 0

	// Determine if this is an htmx request
	isHTMX := r.Header.Get("HX-Request") == "true"

	// Build current violation data (if any)
	var currentViolation *inspections.ViolationDisplay
	if len(violations) > 0 && position < len(violations) {
		v := h.domainViolationToDisplay(r.Context(), violations[position], user.ID)
		currentViolation = &v
	}

	// For htmx requests, return just the partials
	if isHTMX {
		h.renderQueuePartials(w, r, id.String(), violations, position, counts, isComplete, currentViolation)
		return
	}

	// Full page render
	data := inspections.ReviewQueuePageData{
		CurrentPath:     r.URL.Path,
		CSRFToken:       "",
		User:            domainUserToInspectionDisplay(user),
		Inspection:      domainInspectionToDisplay(inspection),
		Violation:       currentViolation,
		Position:        position,
		TotalCount:      len(violations),
		ViolationCounts: counts,
		IsComplete:      isComplete,
		Flash:           nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.ReviewQueuePage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render review queue page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ReviewQueueUpdateStatus handles PUT requests to update a violation's status in queue context.
// PUT /inspections/{id}/review/queue/violations/{vid}/status?status=confirmed|rejected&pos=N
func (h *InspectionHandler) ReviewQueueUpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	inspectionIDStr := r.PathValue("id")
	inspectionID, err := uuid.Parse(inspectionIDStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	violationIDStr := r.PathValue("vid")
	violationID, err := uuid.Parse(violationIDStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// Get status from query param
	statusStr := r.URL.Query().Get("status")
	if statusStr != "confirmed" && statusStr != "rejected" {
		http.Error(w, "Invalid status: must be 'confirmed' or 'rejected'", http.StatusBadRequest)
		return
	}

	newStatus := domain.ViolationStatus(statusStr)

	// Get current position from query param
	currentPos := 0
	if posStr := r.URL.Query().Get("pos"); posStr != "" {
		if p, err := strconv.Atoi(posStr); err == nil && p >= 0 {
			currentPos = p
		}
	}

	// Update the violation status
	err = h.violationService.UpdateStatus(r.Context(), domain.UpdateViolationStatusParams{
		ID:     violationID,
		UserID: user.ID,
		Status: newStatus,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to update violation status", "error", err, "violation_id", violationID)
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
		}
		return
	}

	// Fetch all violations to recalculate counts and find next pending
	violations, err := h.violationService.ListByInspection(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to list violations after status update", "error", err, "inspection_id", inspectionID)
		http.Error(w, "Failed to refresh violations", http.StatusInternalServerError)
		return
	}

	// Calculate updated counts
	counts := inspections.ViolationCountsData{
		Total: len(violations),
	}
	for _, v := range violations {
		switch v.Status {
		case domain.ViolationStatusPending:
			counts.Pending++
		case domain.ViolationStatusConfirmed:
			counts.Confirmed++
		case domain.ViolationStatusRejected:
			counts.Rejected++
		}
	}

	// Find next pending violation
	nextPos := -1
	// First, look for pending violations after current position
	for i := currentPos + 1; i < len(violations); i++ {
		if violations[i].Status == domain.ViolationStatusPending {
			nextPos = i
			break
		}
	}
	// If none found, wrap around to beginning
	if nextPos == -1 {
		for i := 0; i < currentPos; i++ {
			if violations[i].Status == domain.ViolationStatusPending {
				nextPos = i
				break
			}
		}
	}

	// Check if all violations have been reviewed (complete state)
	isComplete := counts.Pending == 0

	// If complete, use current position; otherwise use next pending
	position := currentPos
	if !isComplete && nextPos >= 0 {
		position = nextPos
	}

	// Build current violation data (if any and not complete)
	var currentViolation *inspections.ViolationDisplay
	if len(violations) > 0 && position < len(violations) && !isComplete {
		v := h.domainViolationToDisplay(r.Context(), violations[position], user.ID)
		currentViolation = &v
	}

	// Render partials
	h.renderQueuePartials(w, r, inspectionID.String(), violations, position, counts, isComplete, currentViolation)
}

// =============================================================================
// Report Generation Handler
// =============================================================================

// GenerateReport enqueues a job to generate a report for an inspection.
// POST /inspections/{id}/reports
// Form data: format (pdf|docx)
func (h *InspectionHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	format := r.FormValue("format")
	if format != "pdf" && format != "docx" {
		http.Error(w, "Invalid format: must be 'pdf' or 'docx'", http.StatusBadRequest)
		return
	}

	// Verify inspection exists and user has access
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to fetch inspection", "error", err, "inspection_id", id)
			http.Error(w, "Failed to fetch inspection", http.StatusInternalServerError)
		}
		return
	}

	// Verify inspection status allows report generation (review or completed)
	if inspection.Status != domain.InspectionStatusReview && inspection.Status != domain.InspectionStatusCompleted {
		http.Error(w, "Inspection must be in 'review' or 'completed' status to generate a report", http.StatusBadRequest)
		return
	}

	// Get recipient email from form (optional)
	recipientEmail := r.FormValue("recipient_email")

	// Enqueue the report generation job
	_, err = worker.EnqueueGenerateReport(r.Context(), h.queries, id, user.ID, format, recipientEmail)
	if err != nil {
		h.logger.Error("failed to enqueue report generation job", "error", err, "inspection_id", id)
		http.Error(w, "Failed to start report generation", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Report generation job enqueued",
		"inspection_id", id,
		"user_id", user.ID,
		"format", format,
		"recipient_email", recipientEmail,
	)

	// Return success response (htmx partial or JSON)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Trigger", "reportQueued")

	// Customize message based on whether recipient email was provided
	if recipientEmail != "" {
		fmt.Fprintf(w, `<div class="rounded-md bg-green-50 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-green-800">
						Report generation started! The %s report will be emailed to %s when ready.
					</p>
				</div>
			</div>
		</div>`, format, recipientEmail)
	} else {
		fmt.Fprintf(w, `<div class="rounded-md bg-green-50 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-green-800">
						Report generation started! You'll receive an email when your %s report is ready.
					</p>
				</div>
			</div>
		</div>`, format)
	}
}

// UpdateStatusTempl handles updating an inspection's status via htmx.
// PUT /inspections/{id}/status
func (h *InspectionHandler) UpdateStatusTempl(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	statusStr := r.FormValue("status")
	if statusStr == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	newStatus := domain.InspectionStatus(statusStr)
	if !newStatus.IsValid() {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	// Update the status
	err = h.inspectionService.UpdateStatus(r.Context(), domain.UpdateInspectionStatusParams{
		ID:     id,
		UserID: user.ID,
		Status: newStatus,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			http.Error(w, "Inspection not found", http.StatusNotFound)
		case domain.EINVALID:
			http.Error(w, domain.ErrorMessage(err), http.StatusBadRequest)
		default:
			h.logger.Error("failed to update inspection status", "error", err, "inspection_id", id)
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Info("inspection status updated via htmx",
		"inspection_id", id,
		"user_id", user.ID,
		"new_status", newStatus,
	)

	// Return success message based on new status
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch newStatus {
	case domain.InspectionStatusCompleted:
		w.Header().Set("HX-Trigger", "inspectionCompleted")
		fmt.Fprint(w, `<div class="rounded-md bg-green-50 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-green-800">
						Inspection marked as completed. You can now generate the final report.
					</p>
				</div>
			</div>
		</div>`)
	case domain.InspectionStatusReview:
		w.Header().Set("HX-Trigger", "inspectionReopened")
		fmt.Fprint(w, `<div class="rounded-md bg-yellow-50 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-yellow-800">
						Inspection reopened for review.
					</p>
				</div>
			</div>
		</div>`)
	default:
		fmt.Fprintf(w, `<div class="rounded-md bg-blue-50 p-4">
			<div class="flex">
				<div class="ml-3">
					<p class="text-sm font-medium text-blue-800">
						Inspection status updated to %s.
					</p>
				</div>
			</div>
		</div>`, newStatus)
	}
}

// =============================================================================
// Templ Route Registration
// =============================================================================

// RegisterTemplRoutes registers templ-based inspection routes on the provided ServeMux.
func (h *InspectionHandler) RegisterTemplRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /inspections", requireUser(http.HandlerFunc(h.IndexTempl)))
	mux.Handle("GET /inspections/new", requireUser(http.HandlerFunc(h.NewTempl)))
	mux.Handle("POST /inspections", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /inspections/{id}", requireUser(http.HandlerFunc(h.ShowTempl)))
	mux.Handle("GET /inspections/{id}/edit", requireUser(http.HandlerFunc(h.EditTempl)))
	mux.Handle("PUT /inspections/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /inspections/{id}", requireUser(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /inspections/{id}/analyze", requireUser(http.HandlerFunc(h.TriggerAnalysis)))
	mux.Handle("GET /inspections/{id}/status", requireUser(http.HandlerFunc(h.GetStatus)))
	mux.Handle("GET /inspections/{id}/review", requireUser(http.HandlerFunc(h.ReviewTempl)))
	mux.Handle("GET /inspections/{id}/review/queue", requireUser(http.HandlerFunc(h.ReviewQueueTempl)))
	mux.Handle("PUT /inspections/{id}/review/queue/violations/{vid}/status", requireUser(http.HandlerFunc(h.ReviewQueueUpdateStatus)))
	mux.Handle("GET /inspections/{id}/violations-summary", requireUser(http.HandlerFunc(h.ViolationsSummary)))
	mux.Handle("POST /inspections/{id}/reports", requireUser(http.HandlerFunc(h.GenerateReport)))
	mux.Handle("PUT /inspections/{id}/status", requireUser(http.HandlerFunc(h.UpdateStatusTempl)))
}

// =============================================================================
// Helper Functions
// =============================================================================

// domainUserToInspectionDisplay converts domain.User to inspections.UserDisplay
func domainUserToInspectionDisplay(u *domain.User) *inspections.UserDisplay {
	if u == nil {
		return nil
	}
	return &inspections.UserDisplay{
		Name:               u.Name,
		Email:              u.Email,
		HasBusinessProfile: u.HasBusinessProfile(),
	}
}

// domainInspectionToListItem converts domain.Inspection to inspections.InspectionListItem
func domainInspectionToListItem(i *domain.Inspection) inspections.InspectionListItem {
	return inspections.InspectionListItem{
		ID:             i.ID.String(),
		Title:          i.Title,
		City:           i.City,
		State:          i.State,
		ClientName:     i.ClientName,
		InspectionDate: i.InspectionDate.Format("Jan 2, 2006"),
		Status:         string(i.Status),
		ViolationCount: i.ViolationCount,
	}
}

// domainInspectionToDisplay converts domain.Inspection to inspections.InspectionDisplay
func domainInspectionToDisplay(i *domain.Inspection) *inspections.InspectionDisplay {
	if i == nil {
		return nil
	}

	clientID := ""
	if i.ClientID != nil {
		clientID = i.ClientID.String()
	}

	return &inspections.InspectionDisplay{
		ID:                i.ID.String(),
		Title:             i.Title,
		ClientID:          clientID,
		ClientName:        i.ClientName,
		AddressLine1:      i.AddressLine1,
		AddressLine2:      i.AddressLine2,
		City:              i.City,
		State:             i.State,
		PostalCode:        i.PostalCode,
		InspectionDate:    i.InspectionDate.Format("Jan 2, 2006"),
		Status:            string(i.Status),
		WeatherConditions: i.WeatherConditions,
		Temperature:       i.Temperature,
		InspectorNotes:    i.InspectorNotes,
		CreatedAt:         i.CreatedAt.Format("Jan 2, 2006"),
		UpdatedAt:         i.UpdatedAt.Format("Jan 2, 2006"),
	}
}

// domainClientsToOptions converts []ClientOption to []inspections.ClientOption
func domainClientsToOptions(clients []ClientOption) []inspections.ClientOption {
	options := make([]inspections.ClientOption, len(clients))
	for i, c := range clients {
		options[i] = inspections.ClientOption{
			ID:   c.ID.String(),
			Name: c.Name,
		}
	}
	return options
}

// domainPaginationToShared converts PaginationData to shared pagination.Data
func domainPaginationToShared(p PaginationData) pagination.Data {
	return pagination.Data{
		CurrentPage: p.CurrentPage,
		TotalPages:  p.TotalPages,
		PerPage:     p.PerPage,
		Total:       p.Total,
		HasPrevious: p.HasPrevious,
		HasNext:     p.HasNext,
		PrevPage:    p.PrevPage,
		NextPage:    p.NextPage,
	}
}

// domainAnalysisStatusToTempl converts AnalysisStatusData to inspections.AnalysisStatusData
func domainAnalysisStatusToTempl(a *AnalysisStatusData) inspections.AnalysisStatusData {
	if a == nil {
		return inspections.AnalysisStatusData{}
	}
	return inspections.AnalysisStatusData{
		InspectionID:   a.InspectionID.String(),
		Status:         string(a.Status),
		CanAnalyze:     a.CanAnalyze,
		IsAnalyzing:    a.IsAnalyzing,
		HasImages:      a.HasImages,
		PendingImages:  a.PendingImages,
		TotalImages:    a.TotalImages,
		AnalyzedImages: a.AnalyzedImages,
		ViolationCount: a.ViolationCount,
		Message:        a.Message,
		PollingEnabled: a.PollingEnabled,
	}
}

// domainViolationToDisplay converts domain.Violation to inspections.ViolationDisplay with full details
func (h *InspectionHandler) domainViolationToDisplay(ctx context.Context, v domain.Violation, userID uuid.UUID) inspections.ViolationDisplay {
	// Get regulations for this violation
	_, regulations, err := h.violationService.GetByIDWithRegulations(ctx, v.ID, userID)
	if err != nil {
		h.logger.Warn("failed to get regulations for violation", "error", err, "violation_id", v.ID)
		regulations = []domain.ViolationRegulation{}
	}

	// Get thumbnail URL if violation has an image
	thumbnailURL := ""
	originalURL := ""
	imageID := ""
	if v.ImageID != nil {
		imageID = v.ImageID.String()
		thumbnailURL, err = h.imageService.GetThumbnailURL(ctx, *v.ImageID, userID)
		if err != nil {
			h.logger.Warn("failed to generate thumbnail URL", "error", err, "image_id", *v.ImageID)
		}
		// Build original URL path for linking to full image
		originalURL = fmt.Sprintf("/images/%s/original", imageID)
	}

	// Convert regulations
	regDisplays := make([]inspections.ViolationRegulationDisplay, len(regulations))
	for i, r := range regulations {
		regDisplays[i] = inspections.ViolationRegulationDisplay{
			RegulationID:   r.RegulationID.String(),
			StandardNumber: r.StandardNumber,
			Title:          r.Title,
			IsPrimary:      r.IsPrimary,
		}
	}

	return inspections.ViolationDisplay{
		ID:             v.ID.String(),
		Description:    v.Description,
		AIDescription:  v.AIDescription,
		Status:         string(v.Status),
		Severity:       string(v.Severity),
		Confidence:     string(v.Confidence),
		InspectorNotes: v.InspectorNotes,
		ThumbnailURL:   thumbnailURL,
		OriginalURL:    originalURL,
		ImageID:        imageID,
		Regulations:    regDisplays,
	}
}

// renderIndexErrorTempl renders the index page with an error flash using templ
func (h *InspectionHandler) renderIndexErrorTempl(w http.ResponseWriter, r *http.Request, user *domain.User, message string) {
	data := inspections.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspections: []inspections.InspectionListItem{},
		Pagination:  pagination.Data{},
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.IndexPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render inspections index error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderQueuePartials renders htmx partials for the review queue.
// It renders the header (OOB) and either the violation view, completion screen, or empty state.
func (h *InspectionHandler) renderQueuePartials(
	w http.ResponseWriter,
	r *http.Request,
	inspectionID string,
	violations []domain.Violation,
	position int,
	counts inspections.ViolationCountsData,
	isComplete bool,
	currentViolation *inspections.ViolationDisplay,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Empty state
	if len(violations) == 0 {
		if err := partials.QueueEmptyState(inspectionID).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render queue empty state", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Completion screen
	if isComplete {
		// Render OOB header update first
		headerData := partials.QueueHeaderData{
			InspectionID: inspectionID,
			Position:     position,
			TotalCount:   len(violations),
			Counts: partials.ViolationCounts{
				Total:     counts.Total,
				Pending:   counts.Pending,
				Confirmed: counts.Confirmed,
				Rejected:  counts.Rejected,
			},
		}
		if err := partials.QueueHeaderOOB(headerData).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render queue header OOB", "error", err)
		}

		completionData := partials.QueueCompletionData{
			InspectionID:   inspectionID,
			ConfirmedCount: counts.Confirmed,
			RejectedCount:  counts.Rejected,
		}
		if err := partials.QueueCompletion(completionData).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render queue completion", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Normal violation view
	if currentViolation != nil {
		// Render OOB header update first
		headerData := partials.QueueHeaderData{
			InspectionID: inspectionID,
			Position:     position,
			TotalCount:   len(violations),
			Counts: partials.ViolationCounts{
				Total:     counts.Total,
				Pending:   counts.Pending,
				Confirmed: counts.Confirmed,
				Rejected:  counts.Rejected,
			},
		}
		if err := partials.QueueHeaderOOB(headerData).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render queue header OOB", "error", err)
		}

		// Convert inspections.ViolationDisplay to partials.QueueViolationDisplay
		queueRegs := make([]partials.QueueRegulationDisplay, len(currentViolation.Regulations))
		for i, r := range currentViolation.Regulations {
			queueRegs[i] = partials.QueueRegulationDisplay{
				RegulationID:   r.RegulationID,
				StandardNumber: r.StandardNumber,
				Title:          r.Title,
				IsPrimary:      r.IsPrimary,
			}
		}

		violationData := partials.QueueViolationViewData{
			InspectionID: inspectionID,
			Violation: partials.QueueViolationDisplay{
				ID:             currentViolation.ID,
				Description:    currentViolation.Description,
				AIDescription:  currentViolation.AIDescription,
				Status:         currentViolation.Status,
				Severity:       currentViolation.Severity,
				Confidence:     currentViolation.Confidence,
				InspectorNotes: currentViolation.InspectorNotes,
				ThumbnailURL:   currentViolation.ThumbnailURL,
				OriginalURL:    currentViolation.OriginalURL,
				ImageID:        currentViolation.ImageID,
				Regulations:    queueRegs,
			},
			Position:   position,
			TotalCount: len(violations),
			HasPrev:    position > 0,
			HasNext:    position < len(violations)-1,
		}
		if err := partials.QueueViolationView(violationData).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render queue violation view", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
