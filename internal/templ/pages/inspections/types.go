// Package inspections contains templ components for the inspections pages.
package inspections

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// =============================================================================
// Page Data Types
// =============================================================================

// ListPageData contains data for the inspections list page.
type ListPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Inspections []InspectionListItem
	Pagination  PaginationData
	Flash       *shared.Flash
}

// FormPageData contains data for the inspection create/edit form.
type FormPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Inspection  *InspectionDisplay
	Sites       []SiteOption
	Form        InspectionFormValues
	Errors      map[string]string
	Flash       *shared.Flash
	IsEdit      bool
}

// ShowPageData contains data for the inspection detail page.
type ShowPageData struct {
	CurrentPath     string
	CSRFToken       string
	User            *UserDisplay
	Inspection      *InspectionDisplay
	InspectionID    string
	CanUpload       bool
	IsAnalyzing     bool
	GalleryData     ImageGalleryData
	AnalysisStatus  AnalysisStatusData
	Violations      []ViolationDisplay
	ViolationCounts ViolationCountsData
	Flash           *shared.Flash
}

// ReviewPageData contains data for the violation review page.
type ReviewPageData struct {
	CurrentPath     string
	CSRFToken       string
	User            *UserDisplay
	Inspection      *InspectionDisplay
	Violations      []ViolationDisplay
	ViolationCounts ViolationCountsData
	Flash           *shared.Flash
	ClientEmail     string // Pre-populated client email for report delivery (if available)
}

// ReviewQueuePageData contains data for the queue-based review page.
type ReviewQueuePageData struct {
	CurrentPath     string
	CSRFToken       string
	User            *UserDisplay
	Inspection      *InspectionDisplay
	Violations      []ViolationDisplay
	ViolationCounts ViolationCountsData
	Flash           *shared.Flash
}

// =============================================================================
// Display Types
// =============================================================================

// UserDisplay represents user information for display in templates.
type UserDisplay struct {
	Name               string
	Email              string
	HasBusinessProfile bool
}

// InspectionListItem represents an inspection in the list view.
type InspectionListItem struct {
	ID             string
	Title          string
	SiteName       string
	InspectionDate string
	Status         string
	ViolationCount int
}

// InspectionDisplay represents full inspection details.
type InspectionDisplay struct {
	ID                string
	Title             string
	SiteID            string
	SiteName          string
	SiteAddress       string
	SiteCity          string
	SiteState         string
	InspectionDate    string
	Status            string
	WeatherConditions string
	Temperature       string
	InspectorNotes    string
	CreatedAt         string
	UpdatedAt         string
}

// InspectionFormValues contains form field values for create/edit.
type InspectionFormValues struct {
	Title             string
	SiteID            string
	InspectionDate    string
	WeatherConditions string
	Temperature       string
	InspectorNotes    string
}

// SiteOption represents a site for dropdown selection.
type SiteOption struct {
	ID   string
	Name string
}

// =============================================================================
// Image Gallery Types
// =============================================================================

// ImageGalleryData contains data for the image gallery component.
type ImageGalleryData struct {
	InspectionID string
	Images       []ImageDisplay
	Errors       []string
	CanUpload    bool
	IsAnalyzing  bool
}

// ImageDisplay represents an image for display.
type ImageDisplay struct {
	ID               string
	ThumbnailURL     string
	OriginalFilename string
	AnalysisStatus   string
	SizeMB           string
}

// =============================================================================
// Analysis Status Types
// =============================================================================

// AnalysisStatusData contains data for the analysis status component.
type AnalysisStatusData struct {
	InspectionID   string
	Status         string
	CanAnalyze     bool
	IsAnalyzing    bool
	HasImages      bool
	PendingImages  int64
	TotalImages    int64
	AnalyzedImages int64
	ViolationCount int64
	Message        string
	PollingEnabled bool
}

// =============================================================================
// Violation Types
// =============================================================================

// ViolationDisplay contains a violation with related data for display.
type ViolationDisplay struct {
	ID              string
	Description     string
	AIDescription   string
	Status          string
	Severity        string
	Confidence      string
	InspectorNotes  string
	ThumbnailURL    string
	OriginalURL     string
	ImageID         string
	Regulations     []ViolationRegulationDisplay
}

// ViolationRegulationDisplay represents a regulation linked to a violation.
type ViolationRegulationDisplay struct {
	RegulationID   string
	StandardNumber string
	Title          string
	IsPrimary      bool
}

// ViolationCountsData contains violation summary statistics.
type ViolationCountsData struct {
	Total     int
	Pending   int
	Confirmed int
	Rejected  int
}

// =============================================================================
// Pagination Types
// =============================================================================

// PaginationData contains pagination information.
type PaginationData struct {
	CurrentPage int
	TotalPages  int
	PerPage     int
	Total       int
	HasPrevious bool
	HasNext     bool
	PrevPage    int
	NextPage    int
}

// PageRange returns a slice of page numbers for pagination display.
func PageRange(currentPage, totalPages int) []int {
	if totalPages <= 7 {
		pages := make([]int, totalPages)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages
	}

	pages := []int{1}

	start := currentPage - 1
	end := currentPage + 1

	if start <= 2 {
		start = 2
	}
	if end >= totalPages {
		end = totalPages - 1
	}

	if start > 2 {
		pages = append(pages, -1) // ellipsis
	}

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	if end < totalPages-1 {
		pages = append(pages, -1) // ellipsis
	}

	if totalPages > 1 {
		pages = append(pages, totalPages)
	}

	return pages
}

// =============================================================================
// Helper Functions
// =============================================================================

// StatusColorClass returns the CSS class for a status badge.
func StatusColorClass(status string) string {
	switch status {
	case "draft":
		return "bg-gray-100 text-gray-800"
	case "analyzing":
		return "bg-blue-100 text-blue-800"
	case "review":
		return "bg-yellow-100 text-yellow-800"
	case "completed":
		return "bg-green-100 text-green-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// ViolationStatusColorClass returns the CSS class for a violation status badge.
func ViolationStatusColorClass(status string) string {
	switch status {
	case "pending":
		return "bg-yellow-50 text-yellow-800 ring-yellow-600/20"
	case "confirmed":
		return "bg-green-50 text-green-800 ring-green-600/20"
	case "rejected":
		return "bg-gray-50 text-gray-600 ring-gray-500/20"
	default:
		return "bg-gray-50 text-gray-600 ring-gray-500/20"
	}
}

// SeverityColorClass returns the CSS class for a severity badge.
func SeverityColorClass(severity string) string {
	switch severity {
	case "critical":
		return "bg-red-100 text-red-800"
	case "serious":
		return "bg-orange-100 text-orange-800"
	case "other":
		return "bg-yellow-100 text-yellow-800"
	case "recommendation":
		return "bg-blue-100 text-blue-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// ConfidenceColorClass returns the CSS class for a confidence badge.
func ConfidenceColorClass(confidence string) string {
	switch confidence {
	case "high":
		return "bg-green-50 text-green-700 ring-green-600/20"
	case "medium":
		return "bg-yellow-50 text-yellow-700 ring-yellow-600/20"
	case "low":
		return "bg-gray-50 text-gray-600 ring-gray-500/20"
	default:
		return "bg-gray-50 text-gray-600 ring-gray-500/20"
	}
}

// TitleCase converts a string to title case.
func TitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
