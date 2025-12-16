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
	"strconv"
	"strings"
	"time"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/worker"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// InspectionListPageData contains data for the inspection list page.
type InspectionListPageData struct {
	CurrentPath string                  // Current URL path
	User        interface{}             // Authenticated user
	Inspections []domain.Inspection     // List of inspections
	Pagination  PaginationData          // Pagination information
	Flash       *Flash                  // Flash message (if any)
	CSRFToken   string                  // CSRF token for form protection
}

// InspectionFormPageData contains data for the inspection create/edit form.
type InspectionFormPageData struct {
	CurrentPath string            // Current URL path
	User        interface{}       // Authenticated user
	Inspection  *domain.Inspection // Inspection being edited (nil for create)
	Sites       []SiteOption       // Available sites for dropdown
	Form        map[string]string  // Form field values
	Errors      map[string]string  // Field-level validation errors
	Flash       *Flash             // Flash message (if any)
	IsEdit      bool               // true for edit, false for create
	CSRFToken   string             // CSRF token for form protection
}

// InspectionShowPageData contains data for the inspection detail page.
type InspectionShowPageData struct {
	CurrentPath    string              // Current URL path
	User           interface{}         // Authenticated user
	Inspection     *domain.Inspection  // Inspection details
	InspectionID   uuid.UUID           // Inspection ID for templates
	CanUpload      bool                // Whether user can upload images
	GalleryData    ImageGalleryData    // Image gallery data
	AnalysisStatus AnalysisStatusData  // Analysis status data
	Flash          *Flash              // Flash message (if any)
	CSRFToken      string              // CSRF token for form protection
}

// AnalysisStatusData contains data for the analysis status partial.
type AnalysisStatusData struct {
	InspectionID   uuid.UUID               // Inspection ID
	Status         domain.InspectionStatus // Current inspection status
	CanAnalyze     bool                    // Whether the analyze button should be enabled
	IsAnalyzing    bool                    // Whether analysis is currently running
	HasImages      bool                    // Whether inspection has any images
	PendingImages  int64                   // Number of images pending analysis
	ViolationCount int64                   // Number of violations found
	Message        string                  // Status message to display
	PollingEnabled bool                    // Whether to enable htmx polling
}

// InspectionReviewPageData contains data for the violation review page.
type InspectionReviewPageData struct {
	CurrentPath     string                   // Current URL path
	User            interface{}              // Authenticated user
	Inspection      *domain.Inspection       // Inspection details
	Violations      []ViolationWithDetails   // Violations with details
	ViolationCounts ViolationCounts          // Summary counts
	Flash           *Flash                   // Flash message (if any)
	CSRFToken       string                   // CSRF token for form protection
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
	CurrentPage int    // Current page number (1-indexed)
	TotalPages  int    // Total number of pages
	PerPage     int32  // Results per page
	Total       int64  // Total number of results
	HasPrevious bool   // True if previous page exists
	HasNext     bool   // True if next page exists
	PrevPage    int    // Previous page number
	NextPage    int    // Next page number
}

// SiteOption represents a site for the dropdown select.
type SiteOption struct {
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
	renderer          TemplateRenderer
	logger            *slog.Logger
}

// NewInspectionHandler creates a new InspectionHandler.
func NewInspectionHandler(
	inspectionService service.InspectionService,
	imageService service.ImageService,
	violationService service.ViolationService,
	queries *repository.Queries,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *InspectionHandler {
	return &InspectionHandler{
		inspectionService: inspectionService,
		imageService:      imageService,
		violationService:  violationService,
		queries:           queries,
		renderer:          renderer,
		logger:            logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all inspection routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - GET  /inspections          -> Index (list)
// - GET  /inspections/new      -> New (create form)
// - POST /inspections          -> Create
// - GET  /inspections/{id}     -> Show
// - GET  /inspections/{id}/edit -> Edit
// - PUT  /inspections/{id}     -> Update
// - DELETE /inspections/{id}   -> Delete
// - POST /inspections/{id}/analyze -> TriggerAnalysis
// - GET  /inspections/{id}/status  -> GetStatus
// - GET  /inspections/{id}/review  -> Review
func (h *InspectionHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /inspections", requireUser(http.HandlerFunc(h.Index)))
	mux.Handle("GET /inspections/new", requireUser(http.HandlerFunc(h.New)))
	mux.Handle("POST /inspections", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /inspections/{id}", requireUser(http.HandlerFunc(h.Show)))
	mux.Handle("GET /inspections/{id}/edit", requireUser(http.HandlerFunc(h.Edit)))
	mux.Handle("PUT /inspections/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /inspections/{id}", requireUser(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /inspections/{id}/analyze", requireUser(http.HandlerFunc(h.TriggerAnalysis)))
	mux.Handle("GET /inspections/{id}/status", requireUser(http.HandlerFunc(h.GetStatus)))
	mux.Handle("GET /inspections/{id}/review", requireUser(http.HandlerFunc(h.Review)))
}

// =============================================================================
// GET /inspections - List Inspections
// =============================================================================

// Index displays a paginated list of inspections.
func (h *InspectionHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("index handler called without authenticated user")
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
		h.renderError(w, r, "Failed to load inspections. Please try again.")
		return
	}

	// Build pagination data
	pagination := buildPaginationData(result)

	// Render template
	data := InspectionListPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Inspections: result.Inspections,
		Pagination:  pagination,
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "pages/inspections/index", data)
}

// =============================================================================
// GET /inspections/new - Show Create Form
// =============================================================================

// New displays the inspection creation form.
func (h *InspectionHandler) New(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("new handler called without authenticated user")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch sites for dropdown
	sites, err := h.fetchSiteOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch sites", "error", err, "user_id", user.ID)
		h.renderError(w, r, "Failed to load sites. Please try again.")
		return
	}

	// Render form with empty values
	data := InspectionFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Inspection:  nil,
		Sites:       sites,
		Form: map[string]string{
			"inspection_date": time.Now().Format("2006-01-02"),
		},
		Errors: make(map[string]string),
		Flash:  nil,
		IsEdit: false,
	}

	h.renderer.RenderHTTP(w, "pages/inspections/new", data)
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
	siteIDStr := r.FormValue("site_id")
	inspectionDateStr := r.FormValue("inspection_date")
	weatherConditions := strings.TrimSpace(r.FormValue("weather_conditions"))
	temperature := strings.TrimSpace(r.FormValue("temperature"))
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"title":              title,
		"site_id":            siteIDStr,
		"inspection_date":    inspectionDateStr,
		"weather_conditions": weatherConditions,
		"temperature":        temperature,
		"inspector_notes":    inspectorNotes,
	}

	// Validate and parse site_id
	var siteID *uuid.UUID
	if siteIDStr != "" {
		parsed, err := uuid.Parse(siteIDStr)
		if err != nil {
			h.renderFormError(w, r, user, formValues, map[string]string{
				"site_id": "Invalid site selected",
			}, nil, "", false)
			return
		}
		siteID = &parsed
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
		SiteID:            siteID,
		Title:             title,
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
				"site_id": "Selected site not found",
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
// GET /inspections/{id} - Show Inspection
// =============================================================================

// Show displays inspection details.
func (h *InspectionHandler) Show(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("show handler called without authenticated user")
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

	// Fetch inspection
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			h.renderError(w, r, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch images for this inspection
	images, err := h.imageService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch images", "error", err, "inspection_id", id)
		// Continue with empty image list rather than failing
		images = []domain.Image{}
	}

	// Populate thumbnail URLs for gallery
	imageDisplays := make([]ImageDisplay, 0, len(images))
	for _, img := range images {
		thumbnailURL, err := h.imageService.GetThumbnailURL(r.Context(), img.ID, user.ID)
		if err != nil {
			h.logger.Error("failed to generate thumbnail URL", "error", err, "image_id", img.ID)
			thumbnailURL = "" // Show broken image
		}

		imageDisplays = append(imageDisplays, ImageDisplay{
			ID:               img.ID,
			ThumbnailURL:     thumbnailURL,
			OriginalFilename: img.OriginalFilename,
			AnalysisStatus:   string(img.AnalysisStatus),
			SizeMB:           img.SizeMB(),
		})
	}

	// Build analysis status data
	analysisStatus, err := h.buildAnalysisStatusData(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to build analysis status", "error", err, "inspection_id", id)
		// Continue with empty status rather than failing
		analysisStatus = &AnalysisStatusData{
			InspectionID: id,
			Status:       inspection.Status,
			Message:      "Unable to load analysis status",
		}
	}

	// Render template
	data := InspectionShowPageData{
		CurrentPath:  r.URL.Path,
		User:         user,
		Inspection:   inspection,
		InspectionID: id,
		CanUpload:    inspection.CanAddPhotos(),
		GalleryData: ImageGalleryData{
			InspectionID: id,
			Images:       imageDisplays,
			Errors:       []string{},
			CanUpload:    inspection.CanAddPhotos(),
		},
		AnalysisStatus: *analysisStatus,
		Flash:          nil,
	}

	h.renderer.RenderHTTP(w, "inspections/show", data)
}

// =============================================================================
// GET /inspections/{id}/edit - Show Edit Form
// =============================================================================

// Edit displays the inspection edit form.
func (h *InspectionHandler) Edit(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("edit handler called without authenticated user")
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

	// Fetch inspection
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			h.renderError(w, r, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch sites for dropdown
	sites, err := h.fetchSiteOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch sites", "error", err, "user_id", user.ID)
		h.renderError(w, r, "Failed to load sites. Please try again.")
		return
	}

	// Populate form with inspection data
	siteIDStr := ""
	if inspection.SiteID != nil {
		siteIDStr = inspection.SiteID.String()
	}

	formValues := map[string]string{
		"title":              inspection.Title,
		"site_id":            siteIDStr,
		"inspection_date":    inspection.InspectionDate.Format("2006-01-02"),
		"weather_conditions": inspection.WeatherConditions,
		"temperature":        inspection.Temperature,
		"inspector_notes":    inspection.InspectorNotes,
	}

	// Render form
	data := InspectionFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Inspection:  inspection,
		Sites:       sites,
		Form:        formValues,
		Errors:      make(map[string]string),
		Flash:       nil,
		IsEdit:      true,
	}

	h.renderer.RenderHTTP(w, "pages/inspections/edit", data)
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
	siteIDStr := r.FormValue("site_id")
	inspectionDateStr := r.FormValue("inspection_date")
	weatherConditions := strings.TrimSpace(r.FormValue("weather_conditions"))
	temperature := strings.TrimSpace(r.FormValue("temperature"))
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Store form values for re-rendering
	formValues := map[string]string{
		"title":              title,
		"site_id":            siteIDStr,
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

	// Validate and parse site_id
	var siteID *uuid.UUID
	if siteIDStr != "" {
		parsed, err := uuid.Parse(siteIDStr)
		if err != nil {
			h.renderFormError(w, r, user, formValues, map[string]string{
				"site_id": "Invalid site selected",
			}, inspection, "", true)
			return
		}
		siteID = &parsed
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
		SiteID:            siteID,
		Title:             title,
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

	// Fetch inspection to verify ownership and status
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			http.Error(w, "Failed to load inspection", http.StatusInternalServerError)
		}
		return
	}

	// Check if inspection status allows analysis
	if inspection.Status != domain.InspectionStatusDraft && inspection.Status != domain.InspectionStatusReview {
		http.Error(w, "Inspection status does not allow analysis", http.StatusBadRequest)
		return
	}

	// Check if there are pending images
	pendingCount, err := h.queries.CountPendingImagesByInspectionID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to count pending images", "error", err, "inspection_id", id)
		http.Error(w, "Failed to check images", http.StatusInternalServerError)
		return
	}

	if pendingCount == 0 {
		http.Error(w, "No images to analyze", http.StatusBadRequest)
		return
	}

	// Check if there's already a pending or running job
	hasPending, err := h.queries.HasPendingAnalysisJob(r.Context(), id.String())
	if err != nil {
		h.logger.Error("failed to check pending jobs", "error", err, "inspection_id", id)
		http.Error(w, "Failed to check job status", http.StatusInternalServerError)
		return
	}

	if hasPending {
		http.Error(w, "Analysis is already in progress", http.StatusConflict)
		return
	}

	// Enqueue the analysis job
	_, err = worker.EnqueueAnalyzeInspection(r.Context(), h.queries, id, user.ID)
	if err != nil {
		h.logger.Error("failed to enqueue analysis job", "error", err, "inspection_id", id)
		http.Error(w, "Failed to start analysis", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Analysis job enqueued", "inspection_id", id, "user_id", user.ID, "pending_images", pendingCount)

	// Build and render the updated status
	statusData, err := h.buildAnalysisStatusData(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to build status data", "error", err, "inspection_id", id)
		http.Error(w, "Failed to load status", http.StatusInternalServerError)
		return
	}

	h.renderer.RenderHTTP(w, "partials/analysis_status", statusData)
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

	h.renderer.RenderHTTP(w, "partials/analysis_status", statusData)
}

// =============================================================================
// GET /inspections/{id}/review - Review Violations
// =============================================================================

// Review displays the violation review page where inspectors can accept/reject
// AI-detected violations and add manual violations.
func (h *InspectionHandler) Review(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("review handler called without authenticated user")
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

	// Fetch inspection
	inspection, err := h.inspectionService.GetByID(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			NotFoundResponse(w, r, h.logger)
		} else {
			h.logger.Error("failed to get inspection", "error", err, "inspection_id", id)
			h.renderError(w, r, "Failed to load inspection. Please try again.")
		}
		return
	}

	// Fetch violations
	violations, err := h.violationService.ListByInspection(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to list violations", "error", err, "inspection_id", id)
		h.renderError(w, r, "Failed to load violations. Please try again.")
		return
	}

	// Build violation details with regulations and thumbnail URLs
	violationDetails := make([]ViolationWithDetails, 0, len(violations))
	for _, v := range violations {
		// Get regulations for this violation
		_, regulations, err := h.violationService.GetByIDWithRegulations(r.Context(), v.ID, user.ID)
		if err != nil {
			h.logger.Warn("failed to get regulations for violation",
				"error", err,
				"violation_id", v.ID,
			)
			regulations = []domain.ViolationRegulation{} // Continue with empty list
		}

		// Get thumbnail URL if violation has an image
		thumbnailURL := ""
		if v.ImageID != nil {
			thumbnailURL, err = h.imageService.GetThumbnailURL(r.Context(), *v.ImageID, user.ID)
			if err != nil {
				h.logger.Warn("failed to generate thumbnail URL",
					"error", err,
					"image_id", *v.ImageID,
				)
				// Continue with empty thumbnail URL
			}
		}

		violationDetails = append(violationDetails, ViolationWithDetails{
			Violation:    &v,
			Regulations:  regulations,
			ThumbnailURL: thumbnailURL,
		})
	}

	// Calculate violation counts by status
	counts := ViolationCounts{
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

	// Render review page
	data := InspectionReviewPageData{
		CurrentPath:     r.URL.Path,
		User:            user,
		Inspection:      inspection,
		Violations:      violationDetails,
		ViolationCounts: counts,
		Flash:           nil,
	}

	h.renderer.RenderHTTP(w, "pages/inspections/review", data)
}

// =============================================================================
// Helper Functions
// =============================================================================

// fetchSiteOptions fetches all sites for a user and converts them to SiteOption.
func (h *InspectionHandler) fetchSiteOptions(ctx context.Context, userID uuid.UUID) ([]SiteOption, error) {
	sites, err := h.queries.ListSitesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	options := make([]SiteOption, len(sites))
	for i, site := range sites {
		options[i] = SiteOption{
			ID:   site.ID,
			Name: site.Name,
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
		PerPage:     result.Limit,
		Total:       result.Total,
		HasPrevious: result.HasPrevious(),
		HasNext:     result.HasMore(),
		PrevPage:    currentPage - 1,
		NextPage:    currentPage + 1,
	}
}

// renderError renders a generic error page.
func (h *InspectionHandler) renderError(w http.ResponseWriter, r *http.Request, message string) {
	user := auth.GetUserFromRequest(r)
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"User":        user,
		"Flash": &Flash{
			Type:    "error",
			Message: message,
		},
	}
	h.renderer.RenderHTTP(w, "pages/inspections/index", data)
}

// renderFormError re-renders the form with errors.
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
	// Fetch sites for dropdown
	sites, err := h.fetchSiteOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch sites", "error", err, "user_id", user.ID)
		sites = []SiteOption{} // Empty list on error
	}

	if formValues == nil {
		formValues = make(map[string]string)
	}
	if fieldErrors == nil {
		fieldErrors = make(map[string]string)
	}

	var flash *Flash
	if flashMessage != "" {
		flash = &Flash{
			Type:    "error",
			Message: flashMessage,
		}
	}

	template := "pages/inspections/new"
	if isEdit {
		template = "pages/inspections/edit"
	}

	data := InspectionFormPageData{
		CurrentPath: r.URL.Path,
		User:        user,
		Inspection:  inspection,
		Sites:       sites,
		Form:        formValues,
		Errors:      fieldErrors,
		Flash:       flash,
		IsEdit:      isEdit,
	}

	h.renderer.RenderHTTP(w, template, data)
}

// buildAnalysisStatusData builds the data needed for the analysis status partial.
func (h *InspectionHandler) buildAnalysisStatusData(ctx context.Context, inspectionID uuid.UUID, userID uuid.UUID) (*AnalysisStatusData, error) {
	// Fetch inspection
	inspection, err := h.inspectionService.GetByID(ctx, inspectionID, userID)
	if err != nil {
		return nil, err
	}

	// Get pending image count
	pendingCount, err := h.queries.CountPendingImagesByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("count pending images: %w", err)
	}

	// Get total image count
	totalCount, err := h.queries.CountImagesByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("count total images: %w", err)
	}

	// Get violation count
	violationCount, err := h.queries.CountViolationsByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("count violations: %w", err)
	}

	// Check for pending/running analysis job
	hasPendingJob, err := h.queries.HasPendingAnalysisJob(ctx, inspectionID.String())
	if err != nil {
		return nil, fmt.Errorf("check pending job: %w", err)
	}

	// Build status data based on inspection state
	data := &AnalysisStatusData{
		InspectionID:   inspectionID,
		Status:         inspection.Status,
		HasImages:      totalCount > 0,
		PendingImages:  pendingCount,
		ViolationCount: violationCount,
		IsAnalyzing:    hasPendingJob,
		PollingEnabled: hasPendingJob,
	}

	// Determine if analysis can be triggered
	canAnalyze := false
	message := ""

	switch inspection.Status {
	case domain.InspectionStatusDraft:
		if totalCount == 0 {
			message = "Upload photos to begin analysis"
		} else if pendingCount > 0 && !hasPendingJob {
			canAnalyze = true
			if pendingCount == 1 {
				message = "Ready to analyze 1 image"
			} else {
				message = fmt.Sprintf("Ready to analyze %d images", pendingCount)
			}
		} else if hasPendingJob {
			message = "Analyzing images..."
		} else {
			message = "All images have been analyzed"
		}

	case domain.InspectionStatusAnalyzing:
		message = "Analyzing images..."

	case domain.InspectionStatusReview:
		if pendingCount > 0 && !hasPendingJob {
			canAnalyze = true
			if pendingCount == 1 {
				message = "Ready to analyze 1 new image"
			} else {
				message = fmt.Sprintf("Ready to analyze %d new images", pendingCount)
			}
		} else if hasPendingJob {
			message = "Analyzing new images..."
		} else {
			message = "Analysis complete"
		}

	case domain.InspectionStatusCompleted:
		message = "Inspection finalized"
	}

	data.CanAnalyze = canAnalyze
	data.Message = message

	return data, nil
}
