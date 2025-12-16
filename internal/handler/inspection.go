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

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/service"
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
}

// InspectionShowPageData contains data for the inspection detail page.
type InspectionShowPageData struct {
	CurrentPath string              // Current URL path
	User        interface{}         // Authenticated user
	Inspection  *domain.Inspection  // Inspection details
	InspectionID uuid.UUID          // Inspection ID for templates
	CanUpload   bool                // Whether user can upload images
	GalleryData ImageGalleryData    // Image gallery data
	Flash       *Flash              // Flash message (if any)
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
	queries           *repository.Queries
	renderer          TemplateRenderer
	logger            *slog.Logger
}

// NewInspectionHandler creates a new InspectionHandler.
func NewInspectionHandler(
	inspectionService service.InspectionService,
	imageService service.ImageService,
	queries *repository.Queries,
	renderer TemplateRenderer,
	logger *slog.Logger,
) *InspectionHandler {
	return &InspectionHandler{
		inspectionService: inspectionService,
		imageService:      imageService,
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
func (h *InspectionHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /inspections", requireUser(http.HandlerFunc(h.Index)))
	mux.Handle("GET /inspections/new", requireUser(http.HandlerFunc(h.New)))
	mux.Handle("POST /inspections", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /inspections/{id}", requireUser(http.HandlerFunc(h.Show)))
	mux.Handle("GET /inspections/{id}/edit", requireUser(http.HandlerFunc(h.Edit)))
	mux.Handle("PUT /inspections/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /inspections/{id}", requireUser(http.HandlerFunc(h.Delete)))
}

// =============================================================================
// GET /inspections - List Inspections
// =============================================================================

// Index displays a paginated list of inspections.
func (h *InspectionHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
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
	user := getUser(r)
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
	user := getUser(r)
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
	user := getUser(r)
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
		Flash: nil,
	}

	h.renderer.RenderHTTP(w, "inspections/show", data)
}

// =============================================================================
// GET /inspections/{id}/edit - Show Edit Form
// =============================================================================

// Edit displays the inspection edit form.
func (h *InspectionHandler) Edit(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
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
	user := getUser(r)
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
	user := getUser(r)
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
	user := getUser(r)
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
