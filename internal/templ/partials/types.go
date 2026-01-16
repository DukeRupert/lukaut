package partials

import "github.com/google/uuid"

// ImageGalleryData contains data for the image gallery partial.
type ImageGalleryData struct {
	InspectionID string         // Parent inspection ID (as string for templates)
	Images       []ImageDisplay // Images to display
	Errors       []string       // Upload errors to display
	CanUpload    bool           // Whether user can upload more images
	IsAnalyzing  bool           // Whether analysis is currently running (for polling)
}

// ImageDisplay represents an image for display in the gallery.
type ImageDisplay struct {
	ID               string  // Image ID (as string for templates)
	ThumbnailURL     string  // URL for thumbnail
	OriginalFilename string  // Original filename
	AnalysisStatus   string  // Analysis status (pending, analyzing, completed, failed)
	SizeMB           float64 // File size in megabytes
}

// ViolationCardData contains data for rendering a single violation card.
type ViolationCardData struct {
	Violation    ViolationDisplay       // The violation
	Regulations  []RegulationDisplay    // Linked regulations
	ThumbnailURL string                 // Image thumbnail URL (if linked to image)
	CanEdit      bool                   // Whether user can edit this violation
}

// ViolationDisplay represents a violation for display.
type ViolationDisplay struct {
	ID             string // Violation ID
	Description    string // Description
	AIDescription  string // AI-generated description (if any)
	Severity       string // Severity (critical, serious, other, recommendation)
	Status         string // Status (pending, confirmed, rejected)
	Confidence     string // AI confidence (high, medium, low)
	InspectorNotes string // Inspector notes
}

// RegulationDisplay represents a linked regulation for display.
type RegulationDisplay struct {
	StandardNumber string // OSHA standard number
	Title          string // Regulation title
	IsPrimary      bool   // Whether this is the primary regulation
}

// AnalysisStatusData contains data for the analysis status partial.
type AnalysisStatusData struct {
	InspectionID   string // Inspection ID
	Status         string // Current inspection status
	CanAnalyze     bool   // Whether the analyze button should be enabled
	IsAnalyzing    bool   // Whether analysis is currently running
	HasImages      bool   // Whether inspection has any images
	PendingImages  int64  // Number of images pending analysis
	ViolationCount int64  // Number of violations found
	Message        string // Status message to display
	PollingEnabled bool   // Whether to enable htmx polling
}

// ViolationsSummaryData contains data for the violations summary partial.
type ViolationsSummaryData struct {
	InspectionID    string          // Inspection ID
	IsAnalyzing     bool            // Whether analysis is running
	ViolationCounts ViolationCounts // Summary counts
}

// ViolationCounts contains summary statistics for violations.
type ViolationCounts struct {
	Total     int // Total violations
	Pending   int // Pending review
	Confirmed int // Accepted by inspector
	Rejected  int // Rejected by inspector
}

// ToImageGalleryData converts handler data to template data.
func ToImageGalleryData(inspectionID uuid.UUID, images []ImageDisplay, errors []string, canUpload, isAnalyzing bool) ImageGalleryData {
	return ImageGalleryData{
		InspectionID: inspectionID.String(),
		Images:       images,
		Errors:       errors,
		CanUpload:    canUpload,
		IsAnalyzing:  isAnalyzing,
	}
}
