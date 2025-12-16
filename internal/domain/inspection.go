// Package domain contains core business types and interfaces.
//
// This file defines the Inspection domain type and related types for
// managing construction site safety inspections.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Inspection Status
// =============================================================================

// InspectionStatus represents the lifecycle state of an inspection.
type InspectionStatus string

const (
	// InspectionStatusDraft indicates an inspection is being created/edited.
	// User can modify all fields, no photos have been uploaded yet.
	InspectionStatusDraft InspectionStatus = "draft"

	// InspectionStatusAnalyzing indicates photos have been uploaded and AI
	// analysis is in progress. User cannot modify certain fields.
	InspectionStatusAnalyzing InspectionStatus = "analyzing"

	// InspectionStatusReview indicates AI analysis is complete and inspector
	// is reviewing/accepting/rejecting findings.
	InspectionStatusReview InspectionStatus = "review"

	// InspectionStatusCompleted indicates inspection is finished and ready
	// for report generation. All findings have been reviewed.
	InspectionStatusCompleted InspectionStatus = "completed"
)

// String returns the string representation of the status.
func (s InspectionStatus) String() string {
	return string(s)
}

// IsValid returns true if the status is a recognized value.
func (s InspectionStatus) IsValid() bool {
	switch s {
	case InspectionStatusDraft, InspectionStatusAnalyzing,
		InspectionStatusReview, InspectionStatusCompleted:
		return true
	}
	return false
}

// CanTransitionTo checks if the inspection can transition to the target status.
//
// Valid transitions:
// - draft -> analyzing (when photos uploaded)
// - analyzing -> review (when AI analysis complete)
// - review -> completed (when all findings reviewed)
// - Any status -> draft (to allow re-editing, though this may invalidate findings)
func (s InspectionStatus) CanTransitionTo(target InspectionStatus) bool {
	// Can always transition to draft (re-opening for edits)
	if target == InspectionStatusDraft {
		return true
	}

	switch s {
	case InspectionStatusDraft:
		return target == InspectionStatusAnalyzing
	case InspectionStatusAnalyzing:
		return target == InspectionStatusReview
	case InspectionStatusReview:
		return target == InspectionStatusCompleted
	case InspectionStatusCompleted:
		// Completed inspections can go back to review if needed
		return target == InspectionStatusReview
	}

	return false
}

// =============================================================================
// Inspection Domain Type
// =============================================================================

// Inspection represents a construction site safety inspection.
//
// This is the domain representation designed for use in business logic.
// It includes computed fields that are not stored directly in the database.
type Inspection struct {
	ID                uuid.UUID        // Unique identifier
	UserID            uuid.UUID        // Owner of the inspection
	SiteID            *uuid.UUID       // Optional: Associated site
	Title             string           // Inspection title/name
	Status            InspectionStatus // Current status
	InspectionDate    time.Time        // Date when inspection was/will be conducted
	WeatherConditions string           // Optional: Weather conditions during inspection
	Temperature       string           // Optional: Temperature during inspection
	InspectorNotes    string           // Optional: General notes from inspector
	CreatedAt         time.Time        // When inspection was created
	UpdatedAt         time.Time        // When inspection was last modified

	// Computed fields (not stored in database, populated by queries/services)
	ViolationCount int    // Number of violations found in this inspection
	SiteName       string // Name of associated site (if any)
	SiteAddress    string // Full address of site (for display)
	SiteCity       string // City of site
	SiteState      string // State of site
}

// HasSite returns true if the inspection is associated with a site.
func (i *Inspection) HasSite() bool {
	return i.SiteID != nil
}

// IsEditable returns true if the inspection can be edited.
// Inspections in analyzing status should not be edited as it may conflict
// with ongoing AI analysis.
func (i *Inspection) IsEditable() bool {
	return i.Status == InspectionStatusDraft || i.Status == InspectionStatusReview
}

// CanAddPhotos returns true if photos can be added to the inspection.
func (i *Inspection) CanAddPhotos() bool {
	// Can add photos in draft or review status
	// Cannot add in analyzing (analysis in progress) or completed (finalized)
	return i.Status == InspectionStatusDraft || i.Status == InspectionStatusReview
}

// CanGenerateReport returns true if the inspection is ready for report generation.
func (i *Inspection) CanGenerateReport() bool {
	return i.Status == InspectionStatusCompleted
}

// =============================================================================
// Inspection Service Parameters
// =============================================================================

// CreateInspectionParams contains validated parameters for creating an inspection.
type CreateInspectionParams struct {
	UserID            uuid.UUID  // Owner of the inspection (from auth context)
	SiteID            *uuid.UUID // Optional: Associated site
	Title             string     // Required: Inspection title
	InspectionDate    time.Time  // Required: Date of inspection
	WeatherConditions string     // Optional
	Temperature       string     // Optional
	InspectorNotes    string     // Optional
}

// UpdateInspectionParams contains validated parameters for updating an inspection.
type UpdateInspectionParams struct {
	ID                uuid.UUID  // Inspection to update
	UserID            uuid.UUID  // Owner (for authorization)
	SiteID            *uuid.UUID // Optional: Associated site
	Title             string     // Required: Inspection title
	InspectionDate    time.Time  // Required: Date of inspection
	WeatherConditions string     // Optional
	Temperature       string     // Optional
	InspectorNotes    string     // Optional
}

// ListInspectionsParams contains parameters for listing inspections.
type ListInspectionsParams struct {
	UserID uuid.UUID // Filter by user
	Limit  int32     // Max results to return
	Offset int32     // Number of results to skip
}

// UpdateInspectionStatusParams contains parameters for updating inspection status.
type UpdateInspectionStatusParams struct {
	ID     uuid.UUID        // Inspection to update
	UserID uuid.UUID        // Owner (for authorization)
	Status InspectionStatus // New status
}

// =============================================================================
// List Result with Pagination
// =============================================================================

// ListInspectionsResult contains the result of a paginated inspection list query.
type ListInspectionsResult struct {
	Inspections []Inspection // The inspection results
	Total       int64        // Total number of inspections (for pagination)
	Limit       int32        // Number of results requested
	Offset      int32        // Number of results skipped
}

// HasMore returns true if there are more results available.
func (r *ListInspectionsResult) HasMore() bool {
	return int64(r.Offset+r.Limit) < r.Total
}

// HasPrevious returns true if there are previous results available.
func (r *ListInspectionsResult) HasPrevious() bool {
	return r.Offset > 0
}

// CurrentPage returns the current page number (1-indexed).
func (r *ListInspectionsResult) CurrentPage() int {
	if r.Limit == 0 {
		return 1
	}
	return int(r.Offset/r.Limit) + 1
}

// TotalPages returns the total number of pages.
func (r *ListInspectionsResult) TotalPages() int {
	if r.Limit == 0 {
		return 1
	}
	pages := r.Total / int64(r.Limit)
	if r.Total%int64(r.Limit) > 0 {
		pages++
	}
	return int(pages)
}
