// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements violation handlers for managing OSHA violations
// identified during construction site safety inspections.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/partials"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// ViolationCardData contains data for rendering a single violation card.
type ViolationCardData struct {
	Violation   *domain.Violation            // The violation
	Regulations []domain.ViolationRegulation // Linked regulations
	CanEdit     bool                         // Whether user can edit this violation
}

// =============================================================================
// Handler Configuration
// =============================================================================

// ViolationHandler handles violation-related HTTP requests.
type ViolationHandler struct {
	violationService  service.ViolationService
	inspectionService service.InspectionService
	imageService      service.ImageService
	logger            *slog.Logger
}

// NewViolationHandler creates a new ViolationHandler.
func NewViolationHandler(
	violationService service.ViolationService,
	inspectionService service.InspectionService,
	imageService service.ImageService,
	logger *slog.Logger,
) *ViolationHandler {
	return &ViolationHandler{
		violationService:  violationService,
		inspectionService: inspectionService,
		imageService:      imageService,
		logger:            logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all violation routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - POST   /inspections/{id}/violations -> Create
// - PUT    /violations/{id}             -> Update
// - PUT    /violations/{id}/status      -> UpdateStatus
// - DELETE /violations/{id}             -> Delete
// - GET    /violations/{id}/card        -> GetCard
// - PUT    /violations/batch/status     -> BatchUpdateStatus
func (h *ViolationHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("POST /inspections/{id}/violations", requireUser(http.HandlerFunc(h.Create)))
	mux.Handle("PUT /violations/{id}", requireUser(http.HandlerFunc(h.Update)))
	mux.Handle("PUT /violations/{id}/status", requireUser(http.HandlerFunc(h.UpdateStatus)))
	mux.Handle("PUT /violations/batch/status", requireUser(http.HandlerFunc(h.BatchUpdateStatus)))
	mux.Handle("DELETE /violations/{id}", requireUser(http.HandlerFunc(h.Delete)))
	mux.Handle("GET /violations/{id}/card", requireUser(http.HandlerFunc(h.GetCard)))
}

// =============================================================================
// POST /inspections/{id}/violations - Create Manual Violation
// =============================================================================

// Create handles creating a new manual violation.
func (h *ViolationHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("create violation handler called without authenticated user")
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

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	// Extract form values
	description := strings.TrimSpace(r.FormValue("description"))
	severityStr := r.FormValue("severity")
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Parse severity
	severity := domain.ViolationSeverity(severityStr)
	if !severity.IsValid() {
		http.Error(w, "Invalid severity", http.StatusBadRequest)
		return
	}

	// Create violation
	params := domain.CreateViolationParams{
		InspectionID:   inspectionID,
		UserID:         user.ID,
		ImageID:        nil, // Manual violations don't have images by default
		Description:    description,
		Severity:       severity,
		InspectorNotes: inspectorNotes,
	}

	violation, err := h.violationService.Create(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			http.Error(w, domain.ErrorMessage(err), http.StatusBadRequest)
		case domain.ENOTFOUND:
			http.Error(w, "Inspection not found", http.StatusNotFound)
		default:
			h.logger.Error("failed to create violation", "error", err, "inspection_id", inspectionID)
			http.Error(w, "Failed to create violation", http.StatusInternalServerError)
		}
		return
	}

	// Render the violation card partial
	data := ViolationCardData{
		Violation:   violation,
		Regulations: []domain.ViolationRegulation{}, // No regulations for manual violations
		CanEdit:     true,
	}

	templData := toTemplViolationCardData(data, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ViolationCard(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render violation card", "error", err)
	}
}

// =============================================================================
// PUT /violations/{id} - Update Violation
// =============================================================================

// Update handles updating a violation's details.
func (h *ViolationHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("update violation handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	// Extract form values
	description := strings.TrimSpace(r.FormValue("description"))
	severityStr := r.FormValue("severity")
	inspectorNotes := strings.TrimSpace(r.FormValue("inspector_notes"))

	// Parse severity
	severity := domain.ViolationSeverity(severityStr)
	if !severity.IsValid() {
		http.Error(w, "Invalid severity", http.StatusBadRequest)
		return
	}

	// Update violation
	params := domain.UpdateViolationParams{
		ID:             id,
		UserID:         user.ID,
		Description:    description,
		Severity:       severity,
		InspectorNotes: inspectorNotes,
	}

	err = h.violationService.Update(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			http.Error(w, domain.ErrorMessage(err), http.StatusBadRequest)
		case domain.ENOTFOUND:
			http.Error(w, "Violation not found", http.StatusNotFound)
		default:
			h.logger.Error("failed to update violation", "error", err, "violation_id", id)
			http.Error(w, "Failed to update violation", http.StatusInternalServerError)
		}
		return
	}

	// Get updated violation with regulations
	violation, regulations, err := h.violationService.GetByIDWithRegulations(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to get updated violation", "error", err, "violation_id", id)
		http.Error(w, "Failed to load updated violation", http.StatusInternalServerError)
		return
	}

	// Render the updated violation card with success toast (htmx response)
	data := ViolationCardData{
		Violation:   violation,
		Regulations: regulations,
		CanEdit:     true,
	}

	// Set toast header for htmx
	w.Header().Set("HX-Trigger", `{"showToast": {"type": "success", "message": "Violation updated successfully."}}`)
	templData := toTemplViolationCardData(data, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ViolationCard(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render violation card", "error", err)
	}
}

// =============================================================================
// PUT /violations/{id}/status - Update Violation Status
// =============================================================================

// UpdateStatus handles accepting or rejecting a violation.
func (h *ViolationHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("update status handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	// Extract status
	statusStr := r.FormValue("status")
	status := domain.ViolationStatus(statusStr)
	if !status.IsValid() {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Update status
	params := domain.UpdateViolationStatusParams{
		ID:     id,
		UserID: user.ID,
		Status: status,
	}

	err = h.violationService.UpdateStatus(r.Context(), params)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EINVALID:
			http.Error(w, domain.ErrorMessage(err), http.StatusBadRequest)
		case domain.ENOTFOUND:
			http.Error(w, "Violation not found", http.StatusNotFound)
		default:
			h.logger.Error("failed to update violation status", "error", err, "violation_id", id)
			http.Error(w, "Failed to update status", http.StatusInternalServerError)
		}
		return
	}

	// Get updated violation with regulations
	violation, regulations, err := h.violationService.GetByIDWithRegulations(r.Context(), id, user.ID)
	if err != nil {
		h.logger.Error("failed to get updated violation", "error", err, "violation_id", id)
		http.Error(w, "Failed to load updated violation", http.StatusInternalServerError)
		return
	}

	// Render the updated violation card (htmx response)
	data := ViolationCardData{
		Violation:   violation,
		Regulations: regulations,
		CanEdit:     true,
	}

	templData := toTemplViolationCardData(data, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ViolationCard(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render violation card", "error", err)
	}
}

// =============================================================================
// PUT /violations/batch/status - Batch Update Violation Status
// =============================================================================

// BatchUpdateStatus handles updating multiple violations' status at once.
// Accepts form data with violation_ids[] and status fields.
// Returns HX-Refresh header to reload the page with updated data.
func (h *ViolationHandler) BatchUpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("batch update status handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse batch update form", "error", err)
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	// Get violation IDs from form (supports both violation_ids and violation_ids[])
	violationIDs := r.Form["violation_ids"]
	if len(violationIDs) == 0 {
		violationIDs = r.Form["violation_ids[]"]
	}

	// Get status from form
	statusStr := r.FormValue("status")
	status := domain.ViolationStatus(statusStr)
	if !status.IsValid() {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Validate we have violations to update
	if len(violationIDs) == 0 {
		http.Error(w, "No violations specified", http.StatusBadRequest)
		return
	}

	// Update each violation
	var updated []uuid.UUID
	var updateErrors []string

	for _, idStr := range violationIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			updateErrors = append(updateErrors, "Invalid violation ID: "+idStr)
			continue
		}

		params := domain.UpdateViolationStatusParams{
			ID:     id,
			UserID: user.ID,
			Status: status,
		}

		err = h.violationService.UpdateStatus(r.Context(), params)
		if err != nil {
			code := domain.ErrorCode(err)
			if code == domain.ENOTFOUND {
				updateErrors = append(updateErrors, "Violation not found: "+idStr)
			} else {
				h.logger.Error("failed to update violation status", "error", err, "violation_id", id)
				updateErrors = append(updateErrors, "Failed to update: "+idStr)
			}
			continue
		}

		updated = append(updated, id)
	}

	h.logger.Info("batch violation status update completed",
		"user_id", user.ID,
		"status", status,
		"updated_count", len(updated),
		"error_count", len(updateErrors),
	)

	// Build toast message based on results
	var toastType, toastMessage string
	if len(updateErrors) == 0 {
		toastType = "success"
		toastMessage = fmt.Sprintf("Successfully updated %d violation(s) to %s", len(updated), status)
	} else if len(updated) > 0 {
		toastType = "warning"
		toastMessage = fmt.Sprintf("Updated %d violation(s), %d failed", len(updated), len(updateErrors))
	} else {
		toastType = "error"
		toastMessage = "Failed to update violations"
	}

	// Set htmx headers to show toast and refresh the page
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"type": "%s", "message": "%s"}}`, toastType, toastMessage))
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// =============================================================================
// DELETE /violations/{id} - Delete Violation
// =============================================================================

// Delete handles deleting a violation.
func (h *ViolationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("delete violation handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// Delete violation
	err = h.violationService.Delete(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to delete violation", "error", err, "violation_id", id)
			http.Error(w, "Failed to delete violation", http.StatusInternalServerError)
		}
		return
	}

	// For htmx delete, return 200 OK with empty body
	// htmx will remove the element using hx-swap="outerHTML"
	w.WriteHeader(http.StatusOK)
}

// =============================================================================
// GET /violations/{id}/card - Get Violation Card
// =============================================================================

// GetCard returns the violation card partial (for refreshing after edits).
func (h *ViolationHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("get card handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse violation ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// Get violation with regulations
	violation, regulations, err := h.violationService.GetByIDWithRegulations(r.Context(), id, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Violation not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get violation", "error", err, "violation_id", id)
			http.Error(w, "Failed to load violation", http.StatusInternalServerError)
		}
		return
	}

	// Render the violation card
	data := ViolationCardData{
		Violation:   violation,
		Regulations: regulations,
		CanEdit:     true,
	}

	templData := toTemplViolationCardData(data, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ViolationCard(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render violation card", "error", err)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// toTemplViolationCardData converts ViolationCardData to partials.ViolationCardData
func toTemplViolationCardData(data ViolationCardData, thumbnailURL string) partials.ViolationCardData {
	regulations := make([]partials.RegulationDisplay, len(data.Regulations))
	for i, reg := range data.Regulations {
		regulations[i] = partials.RegulationDisplay{
			StandardNumber: reg.StandardNumber,
			Title:          reg.Title,
			IsPrimary:      reg.IsPrimary,
		}
	}

	return partials.ViolationCardData{
		Violation: partials.ViolationDisplay{
			ID:             data.Violation.ID.String(),
			Description:    data.Violation.Description,
			AIDescription:  data.Violation.AIDescription,
			Severity:       string(data.Violation.Severity),
			Status:         string(data.Violation.Status),
			Confidence:     string(data.Violation.Confidence),
			InspectorNotes: data.Violation.InspectorNotes,
		},
		Regulations:  regulations,
		ThumbnailURL: thumbnailURL,
		CanEdit:      data.CanEdit,
	}
}
