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
	"github.com/DukeRupert/lukaut/internal/templ/pages/inspections"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
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
	pagination := buildPaginationData(result)

	data := inspections.ListPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspections: displayInspections,
		Pagination:  domainPaginationToTempl(pagination),
		Flash:       nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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

	// Fetch sites for dropdown
	siteOptions, err := h.fetchSiteOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch sites", "error", err, "user_id", user.ID)
		h.renderIndexErrorTempl(w, r, user, "Failed to load sites. Please try again.")
		return
	}

	data := inspections.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspection:  nil,
		Sites:       domainSitesToOptions(siteOptions),
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

	// Fetch sites for dropdown
	siteOptions, err := h.fetchSiteOptions(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to fetch sites", "error", err, "user_id", user.ID)
		siteOptions = []SiteOption{}
	}

	siteIDStr := ""
	if inspection.SiteID != nil {
		siteIDStr = inspection.SiteID.String()
	}

	data := inspections.FormPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		User:        domainUserToInspectionDisplay(user),
		Inspection:  domainInspectionToDisplay(inspection),
		Sites:       domainSitesToOptions(siteOptions),
		Form: inspections.InspectionFormValues{
			Title:             inspection.Title,
			SiteID:            siteIDStr,
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
		AnalysisStatus:  domainAnalysisStatusToTempl(analysisStatus),
		Violations:      nil, // Not needed for show page
		ViolationCounts: counts,
		Flash:           nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.ShowPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render inspection show page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ReviewTempl displays the list-based violation review page using templ.
func (h *InspectionHandler) ReviewTempl(w http.ResponseWriter, r *http.Request) {
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

	// Build violation details with regulations and thumbnail URLs
	violationDetails := make([]inspections.ViolationDisplay, 0, len(violations))
	for _, v := range violations {
		violationDetails = append(violationDetails, h.domainViolationToDisplay(r.Context(), v, user.ID))
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

	data := inspections.ReviewPageData{
		CurrentPath:     r.URL.Path,
		CSRFToken:       "",
		User:            domainUserToInspectionDisplay(user),
		Inspection:      domainInspectionToDisplay(inspection),
		Violations:      violationDetails,
		ViolationCounts: counts,
		Flash:           nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.ReviewPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render review page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ReviewQueueTempl displays the keyboard-focused queue-based violation review page using templ.
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

	// Build violation details with regulations and thumbnail URLs
	violationDetails := make([]inspections.ViolationDisplay, 0, len(violations))
	for _, v := range violations {
		violationDetails = append(violationDetails, h.domainViolationToDisplay(r.Context(), v, user.ID))
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

	data := inspections.ReviewQueuePageData{
		CurrentPath:     r.URL.Path,
		CSRFToken:       "",
		User:            domainUserToInspectionDisplay(user),
		Inspection:      domainInspectionToDisplay(inspection),
		Violations:      violationDetails,
		ViolationCounts: counts,
		Flash:           nil,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := inspections.ReviewQueuePage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render review queue page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	mux.Handle("GET /inspections/{id}/violations-summary", requireUser(http.HandlerFunc(h.ViolationsSummary)))
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
		SiteName:       i.SiteName,
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

	siteID := ""
	if i.SiteID != nil {
		siteID = i.SiteID.String()
	}

	return &inspections.InspectionDisplay{
		ID:                i.ID.String(),
		Title:             i.Title,
		SiteID:            siteID,
		SiteName:          i.SiteName,
		SiteAddress:       i.SiteAddress,
		SiteCity:          i.SiteCity,
		SiteState:         i.SiteState,
		InspectionDate:    i.InspectionDate.Format("Jan 2, 2006"),
		Status:            string(i.Status),
		WeatherConditions: i.WeatherConditions,
		Temperature:       i.Temperature,
		InspectorNotes:    i.InspectorNotes,
		CreatedAt:         i.CreatedAt.Format("Jan 2, 2006"),
		UpdatedAt:         i.UpdatedAt.Format("Jan 2, 2006"),
	}
}

// domainSitesToOptions converts []SiteOption to []inspections.SiteOption
func domainSitesToOptions(sites []SiteOption) []inspections.SiteOption {
	options := make([]inspections.SiteOption, len(sites))
	for i, s := range sites {
		options[i] = inspections.SiteOption{
			ID:   s.ID.String(),
			Name: s.Name,
		}
	}
	return options
}

// domainPaginationToTempl converts PaginationData to inspections.PaginationData
func domainPaginationToTempl(p PaginationData) inspections.PaginationData {
	return inspections.PaginationData{
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
		Pagination:  inspections.PaginationData{},
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
