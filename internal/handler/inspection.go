// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements inspection CRUD handlers for managing construction
// site safety inspections.
package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/components/pagination"
	"github.com/DukeRupert/lukaut/internal/templ/pages/inspections"
	"github.com/DukeRupert/lukaut/internal/templ/partials"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/DukeRupert/lukaut/internal/worker"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// InspectionListPageData contains data for the inspection list page.
type InspectionListPageData struct {
	CurrentPath string              // Current URL path
	User        interface{}         // Authenticated user
	Inspections []domain.Inspection // List of inspections
	Pagination  PaginationData      // Pagination information
	Flash       *Flash              // Flash message (if any)
	CSRFToken   string              // CSRF token for form protection
}

// InspectionFormPageData contains data for the inspection create/edit form.
type InspectionFormPageData struct {
	CurrentPath string             // Current URL path
	User        interface{}        // Authenticated user
	Inspection  *domain.Inspection // Inspection being edited (nil for create)
	Clients     []ClientOption     // Available clients for dropdown
	Form        map[string]string  // Form field values
	Errors      map[string]string  // Field-level validation errors
	Flash       *Flash             // Flash message (if any)
	IsEdit      bool               // true for edit, false for create
	CSRFToken   string             // CSRF token for form protection
}

// InspectionShowPageData contains data for the inspection detail page.
type InspectionShowPageData struct {
	CurrentPath     string                 // Current URL path
	User            interface{}            // Authenticated user
	Inspection      *domain.Inspection     // Inspection details
	InspectionID    uuid.UUID              // Inspection ID for templates
	CanUpload       bool                   // Whether user can upload images
	IsAnalyzing     bool                   // Whether analysis is currently running
	GalleryData     ImageGalleryData       // Image gallery data
	AnalysisStatus  AnalysisStatusData     // Analysis status data
	Violations      []ViolationWithDetails // Violations with details
	ViolationCounts ViolationCounts        // Summary counts
	Flash           *Flash                 // Flash message (if any)
	CSRFToken       string                 // CSRF token for form protection
}

// AnalysisStatusData contains data for the analysis status partial.
type AnalysisStatusData struct {
	InspectionID   uuid.UUID               // Inspection ID
	Status         domain.InspectionStatus // Current inspection status
	CanAnalyze     bool                    // Whether the analyze button should be enabled
	IsAnalyzing    bool                    // Whether analysis is currently running
	HasImages      bool                    // Whether inspection has any images
	PendingImages  int64                   // Number of images pending analysis
	TotalImages    int64                   // Total number of images in inspection
	AnalyzedImages int64                   // Number of images analyzed (completed/failed)
	ViolationCount int64                   // Number of violations found
	Message        string                  // Status message to display
	PollingEnabled bool                    // Whether to enable htmx polling
}

// InspectionReviewPageData contains data for the violation review page.
type InspectionReviewPageData struct {
	CurrentPath     string                 // Current URL path
	User            interface{}            // Authenticated user
	Inspection      *domain.Inspection     // Inspection details
	Violations      []ViolationWithDetails // Violations with details
	ViolationCounts ViolationCounts        // Summary counts
	Flash           *Flash                 // Flash message (if any)
	CSRFToken       string                 // CSRF token for form protection
}

// ViolationWithDetails contains a violation plus related data for display.
type ViolationWithDetails struct {
	Violation    *domain.Violation            // The violation
	Regulations  []domain.ViolationRegulation // Linked regulations
	ThumbnailURL string                       // Image thumbnail URL (if linked to image)
}

// ViolationCounts contains summary statistics for violations.
type ViolationCounts struct {
	Total     int // Total violations
	Pending   int // Pending review
	Confirmed int // Accepted by inspector
	Rejected  int // Rejected by inspector
}

// PaginationData contains pagination information.
type PaginationData struct {
	CurrentPage int  // Current page number (1-indexed)
	TotalPages  int  // Total number of pages
	PerPage     int  // Results per page
	Total       int  // Total number of results
	HasPrevious bool // True if previous page exists
	HasNext     bool // True if next page exists
	PrevPage    int  // Previous page number
	NextPage    int  // Next page number
}

// ClientOption represents a client for the dropdown select.
type ClientOption struct {
	ID   uuid.UUID
	Name string
}

// =============================================================================
// Handler Configuration
// =============================================================================

// InspectionHandler handles inspection-related HTTP requests.
type InspectionHandler struct {
	inspectionService service.InspectionService
	imageService      service.ImageService
	violationService  service.ViolationService
	queries           *repository.Queries
	logger            *slog.Logger
}

// NewInspectionHandler creates a new InspectionHandler.
func NewInspectionHandler(
	inspectionService service.InspectionService,
	imageService service.ImageService,
	violationService service.ViolationService,
	queries *repository.Queries,
	logger *slog.Logger,
) *InspectionHandler {
	return &InspectionHandler{
		inspectionService: inspectionService,
		imageService:      imageService,
		violationService:  violationService,
		queries:           queries,
		logger:            logger,
	}
}

// =============================================================================
// POST /inspections - Create Inspection
// =============================================================================

// Create processes the inspection creation form.
func (h *InspectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("create handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderFormError(w, r, user, nil, nil, nil, "Invalid form submission.", false)
		return
	}

	// Extract form values
	title := strings.TrimSpace(r.FormValue("title"))
	clientIDStr := r.FormValue("client_id")
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	addressLine2 := strings.TrimSpace(r.FormValue("address_line2"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	inspectionDateStr := r.FormValue("inspection_date")
	weatherConditions := strings.TrimSpace(r.FormValue("weather_conditions"))
	temperature := strings.TrimSpace(r.FormValue("temperature"))
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"title":              title,
		"client_id":          clientIDStr,
		"address_line1":      addressLine1,
		"address_line2":      addressLine2,
		"city":               city,
		"state":              state,
		"postal_code":        postalCode,
		"inspection_date":    inspectionDateStr,
		"weather_conditions": weatherConditions,
		"temperature":        temperature,
		"inspector_notes":    inspectorNotes,
	}

	// Validate and parse client_id
	var clientID *uuid.UUID
	if clientIDStr != "" {
		parsed, err := uuid.Parse(clientIDStr)
		if err != nil {
			h.renderFormError(w, r, user, formValues, map[string]string{
				"client_id": "Invalid client selected",
			}, nil, "", false)
			return
		}
		clientID = &parsed
	}

	// Parse inspection date
	inspectionDate, err := time.Parse("2006-01-02", inspectionDateStr)
	if err != nil {
		h.renderFormError(w, r, user, formValues, map[string]string{
			"inspection_date": "Invalid date format",
		}, nil, "", false)
		return
	}

	// Create inspection
	params := domain.CreateInspectionParams{
		UserID:            user.ID,
		ClientID:          clientID,
		Title:             title,
		AddressLine1:      addressLine1,
		AddressLine2:      addressLine2,
		City:              city,
		State:             state,
		PostalCode:        postalCode,
		InspectionDate:    inspectionDate,
		WeatherConditions: weatherConditions,
		Temperature:       temperature,
		InspectorNotes:    inspectorNotes,
	}

	inspection, err := h.inspectionService.Create(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderFormError(w, r, user, formValues, nil, nil, domain.ErrorMessage(err), false)
		case domain.ENOTFOUND:
			h.renderFormError(w, r, user, formValues, map[string]string{
				"client_id": "Selected client not found",
			}, nil, "", false)
		default:
			h.logger.Error("failed to create inspection", "error", err, "user_id", user.ID)
			h.renderFormError(w, r, user, formValues, nil, nil, "Failed to create inspection. Please try again.", false)
		}
		return
	}

	// Redirect to inspection detail page
	http.Redirect(w, r, fmt.Sprintf("/inspections/%s", inspection.ID), http.StatusSeeOther)
}

// =============================================================================
// PUT /inspections/{id} - Update Inspection
// =============================================================================

// Update processes the inspection update form.
func (h *InspectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("update handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse inspection ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderError(w, r, "Invalid form submission.")
		return
	}

	// Extract form values
	title := strings.TrimSpace(r.FormValue("title"))
	clientIDStr := r.FormValue("client_id")
	addressLine1 := strings.TrimSpace(r.FormValue("address_line1"))
	addressLine2 := strings.TrimSpace(r.FormValue("address_line2"))
	city := strings.TrimSpace(r.FormValue("city"))
	state := strings.TrimSpace(r.FormValue("state"))
	postalCode := strings.TrimSpace(r.FormValue("postal_code"))
	inspectionDateStr := r.FormValue("inspection_date")
	weatherConditions := strings.TrimSpace(r.FormValue("weather_conditions"))
	temperature := strings.TrimSpace(r.FormValue("temperature"))
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"title":              title,
		"client_id":          clientIDStr,
		"address_line1":      addressLine1,
		"address_line2":      addressLine2,
		"city":               city,
		"state":              state,
		"postal_code":        postalCode,
		"inspection_date":    inspectionDateStr,
		"weather_conditions": weatherConditions,
		"temperature":        temperature,
		"inspector_notes":    inspectorNotes,
	}

	// Fetch current inspection for re-rendering on error
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Validate and parse client_id
	var clientID *uuid.UUID
	if clientIDStr != "" {
		parsed, err := uuid.Parse(clientIDStr)
		if err != nil {
			h.renderFormError(w, r, user, formValues, map[string]string{
				"client_id": "Invalid client selected",
			}, inspection, "", true)
			return
		}
		clientID = &parsed
	}

	// Parse inspection date
	inspectionDate, err := time.Parse("2006-01-02", inspectionDateStr)
	if err != nil {
		h.renderFormError(w, r, user, formValues, map[string]string{
			"inspection_date": "Invalid date format",
		}, inspection, "", true)
		return
	}

	// Update inspection
	params := domain.UpdateInspectionParams{
		ID:                id,
		UserID:            user.ID,
		ClientID:          clientID,
		Title:             title,
		AddressLine1:      addressLine1,
		AddressLine2:      addressLine2,
		City:              city,
		State:             state,
		PostalCode:        postalCode,
		InspectionDate:    inspectionDate,
		WeatherConditions: weatherConditions,
		Temperature:       temperature,
		InspectorNotes:    inspectorNotes,
	}

	err = h.inspectionService.Update(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			h.renderFormError(w, r, user, formValues, nil, inspection, domain.ErrorMessage(err), true)
		case domain.ENOTFOUND:
			NotFoundResponse(w, r, h.logger)
		default:
			h.logger.Error("failed to update inspection", "error", err, "inspection_id", id)
			h.renderFormError(w, r, user, formValues, nil, inspection, "Failed to update inspection. Please try again.", true)
		}
		return
	}

	// Check if this is an htmx request
	if r.Header.Get("HX-Request") == "true" {
		// For htmx, redirect via HX-Redirect header
		w.Header().Set("HX-Redirect", fmt.Sprintf("/inspections/%s", id))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Regular form submission - redirect
	http.Redirect(w, r, fmt.Sprintf("/inspections/%s", id), http.StatusSeeOther)
}

// =============================================================================
// DELETE /inspections/{id} - Delete Inspection
// =============================================================================

// Delete deletes an inspection.
func (h *InspectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("delete handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse inspection ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		NotFoundResponse(w, r, h.logger)
		return
	}

	// Delete inspection
	err = h.inspectionService.Delete(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to delete inspection", "error", err, "inspection_id", id)
			h.renderError(w, r, "Failed to delete inspection. Please try again.")
		}
		return
	}

	// For htmx requests, redirect via HX-Redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/inspections")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Regular request - redirect
	http.Redirect(w, r, "/inspections", http.StatusSeeOther)
}

// =============================================================================
// POST /inspections/{id}/analyze - Trigger AI Analysis
// =============================================================================

// TriggerAnalysis enqueues a background job to analyze inspection images.
func (h *InspectionHandler) TriggerAnalysis(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("trigger analysis handler called without authenticated user")
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

	// Check analysis eligibility via service
	analysisStatus, err := h.inspectionService.GetAnalysisStatus(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get analysis status", "error", err, "inspection_id", id)
			http.Error(w, "Failed to check analysis eligibility", http.StatusInternalServerError)
		}
		return
	}

	if !analysisStatus.CanAnalyze {
		if analysisStatus.IsAnalyzing {
			http.Error(w, analysisStatus.Message, http.StatusConflict)
		} else {
			http.Error(w, analysisStatus.Message, http.StatusBadRequest)
		}
		return
	}

	// Enqueue the analysis job
	_, err = worker.EnqueueAnalyzeInspection(r.Context(), h.queries, id, user.ID)
	if err != nil {
		h.logger.Error("failed to enqueue analysis job", "error", err, "inspection_id", id)
		http.Error(w, "Failed to start analysis", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Analysis job enqueued", "inspection_id", id, "user_id", user.ID, "pending_images", analysisStatus.PendingImages)

	// Build and render the updated status
	statusData, err := h.buildAnalysisStatusData(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to build status data", "error", err, "inspection_id", id)
		http.Error(w, "Failed to load status", http.StatusInternalServerError)
		return
	}

	// Render templ partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templData := partials.AnalysisStatusData{
		InspectionID:   statusData.InspectionID.String(),
		Status:         string(statusData.Status),
		CanAnalyze:     statusData.CanAnalyze,
		IsAnalyzing:    statusData.IsAnalyzing,
		HasImages:      statusData.HasImages,
		PendingImages:  statusData.PendingImages,
		ViolationCount: statusData.ViolationCount,
		Message:        statusData.Message,
		PollingEnabled: statusData.PollingEnabled,
	}
	if err := partials.AnalysisStatus(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render analysis status", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /inspections/{id}/status - Get Analysis Status
// =============================================================================

// GetStatus returns the current analysis status as an htmx partial.
func (h *InspectionHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("get status handler called without authenticated user")
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

	// Build status data
	statusData, err := h.buildAnalysisStatusData(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to build status data", "error", err, "inspection_id", id)
			http.Error(w, "Failed to load status", http.StatusInternalServerError)
		}
		return
	}

	// Render templ partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Trigger violations summary refresh when analysis completes with violations
	if !statusData.IsAnalyzing && statusData.ViolationCount > 0 {
		w.Header().Set("HX-Trigger", "analysisComplete")
	}

	templData := partials.AnalysisStatusData{
		InspectionID:   statusData.InspectionID.String(),
		Status:         string(statusData.Status),
		CanAnalyze:     statusData.CanAnalyze,
		IsAnalyzing:    statusData.IsAnalyzing,
		HasImages:      statusData.HasImages,
		PendingImages:  statusData.PendingImages,
		ViolationCount: statusData.ViolationCount,
		Message:        statusData.Message,
		PollingEnabled: statusData.PollingEnabled,
	}
	if err := partials.AnalysisStatus(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render analysis status", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /inspections/{id}/violations-summary - Violations Summary Partial
// =============================================================================

// ViolationsSummary returns the violations summary partial for htmx polling.
func (h *InspectionHandler) ViolationsSummary(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("violations summary handler called without authenticated user")
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

	// Fetch inspection to get status
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to fetch inspection", "error", err)
			http.Error(w, "Failed to fetch inspection", http.StatusInternalServerError)
		}
		return
	}

	// Fetch violations
	violations, err := h.violationService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to list violations", "error", err, "inspection_id", id)
		violations = []domain.Violation{} // Continue with empty list
	}

	// Calculate violation counts by status
	counts := domain.CalculateViolationCounts(violations)

	// Check if analysis is running
	isAnalyzing := inspection.Status == domain.InspectionStatusAnalyzing

	// Render violations summary templ partial
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templData := partials.ViolationsSummaryData{
		InspectionID: id.String(),
		IsAnalyzing:  isAnalyzing,
		ViolationCounts: partials.ViolationCounts{
			Total:     counts.Total,
			Pending:   counts.Pending,
			Confirmed: counts.Confirmed,
			Rejected:  counts.Rejected,
		},
	}
	if err := partials.ViolationsSummary(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render violations summary", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// fetchClientOptions fetches all clients for a user and converts them to ClientOption.
func (h *InspectionHandler) fetchClientOptions(ctx context.Context, userID uuid.UUID) ([]ClientOption, error) {
	clients, err := h.queries.ListAllClientsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	options := make([]ClientOption, len(clients))
	for i, client := range clients {
		options[i] = ClientOption{
			ID:   client.ID,
			Name: client.Name,
		}
	}

	return options, nil
}

// buildPaginationData builds pagination data from a list result.
func buildPaginationData(result *domain.ListInspectionsResult) PaginationData {
	currentPage := result.CurrentPage()
	totalPages := result.TotalPages()

	return PaginationData{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		PerPage:     int(result.Limit),
		Total:       int(result.Total),
		HasPrevious: result.HasPrevious(),
		HasNext:     result.HasMore(),
		PrevPage:    currentPage - 1,
		NextPage:    currentPage + 1,
	}
}

// renderError renders a generic error page using templ.
func (h *InspectionHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
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

// renderFormError re-renders the form with errors using templ.
func (h *InspectionHandler) renderFormError(
	w http.ResponseWriter,
	r *http.Request,
	user *domain.User,
	formValues map[string]string,
	fieldErrors map[string]string,
	inspection *domain.Inspection,
	flashMessage string,
	isEdit bool,
) {
	// Fetch clients for dropdown
	clients, err := h.fetchClientOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch clients", "error", err, "user_id", user.ID)
		clients = []ClientOption{} // Empty list on error
	}

	if formValues == nil {
		formValues = make(map[string]string)
	}
	if fieldErrors == nil {
		fieldErrors = make(map[string]string)
	}

	var flash *shared.Flash
	if flashMessage != "" {
		flash = &shared.Flash{
			Type:    shared.FlashError,
			Message: flashMessage,
		}
	}

	data := inspections.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspection:  domainInspectionToDisplay(inspection),
		Clients:     domainClientsToOptions(clients),
		Form: inspections.InspectionFormValues{
			Title:             formValues["title"],
			ClientID:          formValues["client_id"],
			AddressLine1:      formValues["address_line1"],
			AddressLine2:      formValues["address_line2"],
			City:              formValues["city"],
			State:             formValues["state"],
			PostalCode:        formValues["postal_code"],
			InspectionDate:    formValues["inspection_date"],
			WeatherConditions: formValues["weather_conditions"],
			Temperature:       formValues["temperature"],
			InspectorNotes:    formValues["inspector_notes"],
		},
		Errors: fieldErrors,
		Flash:  flash,
		IsEdit: isEdit,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var renderErr error
	if isEdit {
		renderErr = inspections.EditPage(data).Render(r.Context(), w)
	} else {
		renderErr = inspections.NewPage(data).Render(r.Context(), w)
	}
	if renderErr != nil {
		h.logger.Error("failed to render form error", "error", renderErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildAnalysisStatusData builds the data needed for the analysis status partial.
func (h *InspectionHandler) buildAnalysisStatusData(ctx context.Context, inspectionID uuid.UUID, userID uuid.UUID) (*AnalysisStatusData, error) {
	status, err := h.inspectionService.GetAnalysisStatus(ctx, inspectionID, userID)
	if err != nil {
		return nil, err
	}

	return &AnalysisStatusData{
		InspectionID:   status.InspectionID,
		Status:         status.Status,
		CanAnalyze:     status.CanAnalyze,
		IsAnalyzing:    status.IsAnalyzing,
		HasImages:      status.HasImages,
		PendingImages:  status.PendingImages,
		TotalImages:    status.TotalImages,
		AnalyzedImages: status.AnalyzedImages,
		ViolationCount: status.ViolationCount,
		Message:        status.Message,
		PollingEnabled: status.PollingEnabled,
	}, nil
}
