// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements image upload handlers for managing inspection photos.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/partials"
	"github.com/google/uuid"
)

// =============================================================================
// Template Data Types
// =============================================================================

// ImageGalleryData contains data for the image gallery partial.
type ImageGalleryData struct {
	InspectionID uuid.UUID      // Parent inspection ID
	Images       []ImageDisplay // Images to display
	Errors       []string       // Upload errors to display
	CanUpload    bool           // Whether user can upload more images
	IsAnalyzing  bool           // Whether analysis is currently running (for polling)
}

// ImageDisplay represents an image for display in the gallery.
type ImageDisplay struct {
	ID               uuid.UUID // Image ID
	ThumbnailURL     string    // URL for thumbnail
	OriginalFilename string    // Original filename
	AnalysisStatus   string    // Analysis status (pending, analyzing, completed, failed)
	SizeMB           float64   // File size in megabytes
}

// =============================================================================
// Handler Configuration
// =============================================================================

// ImageHandler handles image-related HTTP requests.
type ImageHandler struct {
	imageService      service.ImageService
	inspectionService service.InspectionService
	logger            *slog.Logger
}

// NewImageHandler creates a new ImageHandler.
func NewImageHandler(
	imageService service.ImageService,
	inspectionService service.InspectionService,
	logger *slog.Logger,
) *ImageHandler {
	return &ImageHandler{
		imageService:      imageService,
		inspectionService: inspectionService,
		logger:            logger,
	}
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all image routes with the provided mux.
//
// All routes require authentication via the requireUser middleware.
//
// Routes:
// - POST   /inspections/{id}/images         -> Upload
// - DELETE /inspections/{id}/images/{imageId} -> Delete
// - GET    /images/{id}/thumbnail           -> ServeThumbnail
// - GET    /images/{id}/original            -> ServeOriginal
// - GET    /inspections/{id}/images         -> ListImages
func (h *ImageHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("POST /inspections/{id}/images", requireUser(http.HandlerFunc(h.Upload)))
	mux.Handle("DELETE /inspections/{id}/images/{imageId}", requireUser(http.HandlerFunc(h.Delete)))
	mux.Handle("GET /images/{id}/thumbnail", requireUser(http.HandlerFunc(h.ServeThumbnail)))
	mux.Handle("GET /images/{id}/original", requireUser(http.HandlerFunc(h.ServeOriginal)))
	mux.Handle("GET /inspections/{id}/images", requireUser(http.HandlerFunc(h.ListImages)))
}

// =============================================================================
// POST /inspections/{id}/images - Upload Images
// =============================================================================

// Upload handles image upload for an inspection.
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("upload handler called without authenticated user")
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

	// Parse multipart form (32MB memory limit)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.logger.Error("failed to parse multipart form", "error", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get uploaded files
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No images uploaded", http.StatusBadRequest)
		return
	}

	// Track successes and errors
	var uploadErrors []string
	successCount := 0

	// Process each file
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			h.logger.Error("failed to open uploaded file", "error", err, "filename", fileHeader.Filename)
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: Failed to open file", fileHeader.Filename))
			continue
		}

		// Upload image
		_, err = h.imageService.Upload(r.Context(), file, fileHeader, inspectionID, user.ID)
		_ = file.Close()

		if err != nil {
			code := domain.ErrorCode(err)
			msg := domain.ErrorMessage(err)
			h.logger.Error("failed to upload image",
				"error", err,
				"filename", fileHeader.Filename,
				"code", code,
			)
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %s", fileHeader.Filename, msg))
			continue
		}

		successCount++
	}

	h.logger.Info("image upload completed",
		"inspection_id", inspectionID,
		"success_count", successCount,
		"error_count", len(uploadErrors),
	)

	// Auto-trigger analysis if images were uploaded successfully
	analysisEnqueued := false
	if successCount > 0 {
		// Check if there's already a pending or running analysis job
		hasPending, err := h.inspectionService.HasPendingAnalysisJob(r.Context(), inspectionID)
		if err != nil {
			h.logger.Warn("failed to check pending analysis jobs", "error", err, "inspection_id", inspectionID)
			// Continue anyway - don't fail the upload response
		} else if !hasPending {
			// Enqueue the analysis job via service
			err = h.inspectionService.TriggerAnalysis(r.Context(), inspectionID, user.ID)
			if err != nil {
				h.logger.Warn("failed to enqueue auto-analysis job", "error", err, "inspection_id", inspectionID)
				// Continue anyway - don't fail the upload response
			} else {
				h.logger.Info("Auto-analysis job enqueued", "inspection_id", inspectionID, "user_id", user.ID)
				analysisEnqueued = true
			}
		} else {
			// There's already a pending job, so analysis is in progress
			analysisEnqueued = true
		}
	}

	// Fetch updated image list
	images, err := h.imageService.ListByInspection(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch images after upload", "error", err)
		http.Error(w, "Upload completed but failed to refresh gallery", http.StatusInternalServerError)
		return
	}

	// Get inspection to check if user can upload
	inspection, err := h.inspectionService.GetByID(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch inspection", "error", err)
		http.Error(w, "Failed to fetch inspection", http.StatusInternalServerError)
		return
	}

	// Populate thumbnail URLs
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

	// Render image gallery partial
	data := ImageGalleryData{
		InspectionID: inspectionID,
		Images:       imageDisplays,
		Errors:       uploadErrors,
		CanUpload:    inspection.CanAddPhotos(),
		IsAnalyzing:  analysisEnqueued,
	}

	// Trigger analysis status refresh and violations summary refresh
	w.Header().Set("HX-Trigger", "galleryUpdated")

	// Render using templ
	templData := toTemplImageGalleryData(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ImageGallery(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render image gallery", "error", err)
	}
}

// =============================================================================
// DELETE /inspections/{id}/images/{imageId} - Delete Image
// =============================================================================

// Delete handles image deletion.
func (h *ImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("delete handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse inspection ID
	inspectionIDStr := r.PathValue("id")
	inspectionID, err := uuid.Parse(inspectionIDStr)
	if err != nil {
		http.Error(w, "Invalid inspection ID", http.StatusBadRequest)
		return
	}

	// Parse image ID
	imageIDStr := r.PathValue("imageId")
	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Delete image
	if err := h.imageService.Delete(r.Context(), imageID, user.ID); err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Image not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to delete image", "error", err, "image_id", imageID)
			http.Error(w, "Failed to delete image", http.StatusInternalServerError)
		}
		return
	}

	// Fetch updated image list
	images, err := h.imageService.ListByInspection(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch images after delete", "error", err)
		http.Error(w, "Delete completed but failed to refresh gallery", http.StatusInternalServerError)
		return
	}

	// Get inspection to check if user can upload
	inspection, err := h.inspectionService.GetByID(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch inspection", "error", err)
		http.Error(w, "Failed to fetch inspection", http.StatusInternalServerError)
		return
	}

	// Populate thumbnail URLs
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

	// Render updated image gallery partial
	data := ImageGalleryData{
		InspectionID: inspectionID,
		Images:       imageDisplays,
		Errors:       []string{},
		CanUpload:    inspection.CanAddPhotos(),
	}

	// Trigger analysis status refresh
	w.Header().Set("HX-Trigger", "galleryUpdated")

	// Render using templ
	templData := toTemplImageGalleryData(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ImageGallery(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render image gallery", "error", err)
	}
}

// =============================================================================
// GET /images/{id}/thumbnail - Serve Thumbnail
// =============================================================================

// ServeThumbnail redirects to the thumbnail URL.
func (h *ImageHandler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("serve thumbnail handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse image ID
	idStr := r.PathValue("id")
	imageID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Get thumbnail URL
	url, err := h.imageService.GetThumbnailURL(r.Context(), imageID, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Image not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get thumbnail URL", "error", err, "image_id", imageID)
			http.Error(w, "Failed to get thumbnail", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to URL
	http.Redirect(w, r, url, http.StatusFound)
}

// =============================================================================
// GET /images/{id}/original - Serve Original
// =============================================================================

// ServeOriginal redirects to the original image URL.
func (h *ImageHandler) ServeOriginal(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("serve original handler called without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse image ID
	idStr := r.PathValue("id")
	imageID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Get original URL
	url, err := h.imageService.GetOriginalURL(r.Context(), imageID, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Image not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to get original URL", "error", err, "image_id", imageID)
			http.Error(w, "Failed to get original", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to URL
	http.Redirect(w, r, url, http.StatusFound)
}

// =============================================================================
// GET /inspections/{id}/images - List Images (for htmx refresh)
// =============================================================================

// ListImages returns the image gallery partial for an inspection.
func (h *ImageHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		h.logger.Error("list images handler called without authenticated user")
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

	// Fetch images
	images, err := h.imageService.ListByInspection(r.Context(), inspectionID, user.ID)
	if err != nil {
		code := domain.ErrorCode(err)
		if code == domain.ENOTFOUND {
			http.Error(w, "Inspection not found", http.StatusNotFound)
		} else {
			h.logger.Error("failed to fetch images", "error", err, "inspection_id", inspectionID)
			http.Error(w, "Failed to fetch images", http.StatusInternalServerError)
		}
		return
	}

	// Get inspection to check if user can upload
	inspection, err := h.inspectionService.GetByID(r.Context(), inspectionID, user.ID)
	if err != nil {
		h.logger.Error("failed to fetch inspection", "error", err)
		http.Error(w, "Failed to fetch inspection", http.StatusInternalServerError)
		return
	}

	// Populate thumbnail URLs and check for analyzing images
	imageDisplays := make([]ImageDisplay, 0, len(images))
	isAnalyzing := false
	for _, img := range images {
		thumbnailURL, err := h.imageService.GetThumbnailURL(r.Context(), img.ID, user.ID)
		if err != nil {
			h.logger.Error("failed to generate thumbnail URL", "error", err, "image_id", img.ID)
			thumbnailURL = "" // Show broken image
		}

		// Check if any image is being analyzed
		if img.AnalysisStatus == domain.ImageAnalysisStatusAnalyzing {
			isAnalyzing = true
		}

		imageDisplays = append(imageDisplays, ImageDisplay{
			ID:               img.ID,
			ThumbnailURL:     thumbnailURL,
			OriginalFilename: img.OriginalFilename,
			AnalysisStatus:   string(img.AnalysisStatus),
			SizeMB:           img.SizeMB(),
		})
	}

	// Also check if inspection is in analyzing state
	if inspection.Status == domain.InspectionStatusAnalyzing {
		isAnalyzing = true
	}

	// Also check for pending/running analysis job in the queue
	// This handles the case where job is queued but hasn't started yet
	if !isAnalyzing {
		hasPendingJob, err := h.inspectionService.HasPendingAnalysisJob(r.Context(), inspectionID)
		if err != nil {
			h.logger.Warn("failed to check pending analysis job", "error", err)
		} else if hasPendingJob {
			isAnalyzing = true
		}
	}

	// Render image gallery partial
	data := ImageGalleryData{
		InspectionID: inspectionID,
		Images:       imageDisplays,
		Errors:       []string{},
		CanUpload:    inspection.CanAddPhotos(),
		IsAnalyzing:  isAnalyzing,
	}

	// Render using templ
	templData := toTemplImageGalleryData(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partials.ImageGallery(templData).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render image gallery", "error", err)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// toTemplImageGalleryData converts ImageGalleryData to partials.ImageGalleryData
func toTemplImageGalleryData(data ImageGalleryData) partials.ImageGalleryData {
	images := make([]partials.ImageDisplay, len(data.Images))
	for i, img := range data.Images {
		images[i] = partials.ImageDisplay{
			ID:               img.ID.String(),
			ThumbnailURL:     img.ThumbnailURL,
			OriginalFilename: img.OriginalFilename,
			AnalysisStatus:   img.AnalysisStatus,
			SizeMB:           img.SizeMB,
		}
	}
	return partials.ImageGalleryData{
		InspectionID: data.InspectionID.String(),
		Images:       images,
		Errors:       data.Errors,
		CanUpload:    data.CanUpload,
		IsAnalyzing:  data.IsAnalyzing,
	}
}
