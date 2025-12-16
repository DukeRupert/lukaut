// Package domain contains core business types and interfaces.
//
// This file defines the Image domain type and related types for
// managing construction site inspection photos and their AI analysis.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Image Analysis Status
// =============================================================================

// ImageAnalysisStatus represents the state of AI analysis for an image.
type ImageAnalysisStatus string

const (
	// ImageAnalysisStatusPending indicates the image is queued for analysis.
	ImageAnalysisStatusPending ImageAnalysisStatus = "pending"

	// ImageAnalysisStatusAnalyzing indicates AI analysis is in progress.
	ImageAnalysisStatusAnalyzing ImageAnalysisStatus = "analyzing"

	// ImageAnalysisStatusCompleted indicates AI analysis finished successfully.
	ImageAnalysisStatusCompleted ImageAnalysisStatus = "completed"

	// ImageAnalysisStatusFailed indicates AI analysis failed.
	ImageAnalysisStatusFailed ImageAnalysisStatus = "failed"
)

// String returns the string representation of the status.
func (s ImageAnalysisStatus) String() string {
	return string(s)
}

// IsValid returns true if the status is a recognized value.
func (s ImageAnalysisStatus) IsValid() bool {
	switch s {
	case ImageAnalysisStatusPending, ImageAnalysisStatusAnalyzing,
		ImageAnalysisStatusCompleted, ImageAnalysisStatusFailed:
		return true
	}
	return false
}

// =============================================================================
// Image Constants
// =============================================================================

// SupportedImageTypes maps MIME types to their human-readable names.
// For MVP, only JPEG and PNG are supported (HEIC requires CGO).
var SupportedImageTypes = map[string]string{
	"image/jpeg": "JPEG",
	"image/png":  "PNG",
}

const (
	// MaxImageSize is the maximum allowed size for uploaded images (20MB).
	MaxImageSize = 20 * 1024 * 1024 // 20MB in bytes

	// ThumbnailMaxWidth is the maximum width for generated thumbnails.
	ThumbnailMaxWidth = 200

	// ThumbnailMaxHeight is the maximum height for generated thumbnails.
	ThumbnailMaxHeight = 200

	// ThumbnailJPEGQuality is the JPEG quality for thumbnail generation (0-100).
	ThumbnailJPEGQuality = 85
)

// =============================================================================
// Image Domain Type
// =============================================================================

// Image represents an uploaded construction site inspection photo.
//
// This is the domain representation designed for use in business logic.
// It includes computed fields that are not stored directly in the database.
type Image struct {
	ID               uuid.UUID           // Unique identifier
	InspectionID     uuid.UUID           // Parent inspection
	StorageKey       string              // Key/path in storage service for original image
	ThumbnailKey     string              // Key/path in storage service for thumbnail
	OriginalFilename string              // Original filename from upload
	ContentType      string              // MIME type (e.g., "image/jpeg")
	SizeBytes        int64               // File size in bytes
	Width            int32               // Image width in pixels
	Height           int32               // Image height in pixels
	AnalysisStatus   ImageAnalysisStatus // Current AI analysis status
	CreatedAt        time.Time           // When image was uploaded
	UpdatedAt        time.Time           // When image was last modified

	// Computed fields (not stored in database, populated by services)
	ThumbnailURL string // Presigned/public URL for thumbnail
	OriginalURL  string // Presigned/public URL for original image
}

// IsAnalyzed returns true if the image has been analyzed by AI.
func (i *Image) IsAnalyzed() bool {
	return i.AnalysisStatus == ImageAnalysisStatusCompleted
}

// IsPending returns true if the image is waiting for AI analysis.
func (i *Image) IsPending() bool {
	return i.AnalysisStatus == ImageAnalysisStatusPending
}

// HasFailed returns true if AI analysis failed for this image.
func (i *Image) HasFailed() bool {
	return i.AnalysisStatus == ImageAnalysisStatusFailed
}

// AspectRatio returns the aspect ratio of the image (width/height).
func (i *Image) AspectRatio() float64 {
	if i.Height == 0 {
		return 0
	}
	return float64(i.Width) / float64(i.Height)
}

// SizeMB returns the file size in megabytes.
func (i *Image) SizeMB() float64 {
	return float64(i.SizeBytes) / (1024 * 1024)
}

// =============================================================================
// Image Service Parameters
// =============================================================================

// UploadImageParams contains validated parameters for uploading an image.
type UploadImageParams struct {
	InspectionID     uuid.UUID // Parent inspection
	UserID           uuid.UUID // Owner (for authorization)
	OriginalFilename string    // Original filename
	ContentType      string    // MIME type
	Width            int32     // Image dimensions
	Height           int32     // Image dimensions
	StorageKey       string    // Storage key for original
	ThumbnailKey     string    // Storage key for thumbnail
	SizeBytes        int64     // File size
}

// =============================================================================
// Validation Helpers
// =============================================================================

// IsValidImageContentType checks if the content type is supported.
func IsValidImageContentType(contentType string) bool {
	_, ok := SupportedImageTypes[contentType]
	return ok
}

// ValidateImageSize checks if the file size is within limits.
func ValidateImageSize(size int64) error {
	if size > MaxImageSize {
		return Errorf(ETOOLARGE, "image.validate", "Image size %d bytes exceeds maximum of %d bytes (%.1fMB)", size, MaxImageSize, float64(MaxImageSize)/(1024*1024))
	}
	if size == 0 {
		return Invalid("image.validate", "Image file is empty")
	}
	return nil
}
