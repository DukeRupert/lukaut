// Package domain contains core business types and interfaces.
//
// This file defines the Violation domain type and related types for
// managing OSHA violations identified during construction site inspections.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Violation Status
// =============================================================================

// ViolationStatus represents the review state of a violation.
type ViolationStatus string

const (
	// ViolationStatusPending indicates a violation is awaiting inspector review.
	// This is the default status for AI-detected violations.
	ViolationStatusPending ViolationStatus = "pending"

	// ViolationStatusConfirmed indicates the inspector has accepted the violation.
	ViolationStatusConfirmed ViolationStatus = "confirmed"

	// ViolationStatusRejected indicates the inspector has rejected the violation.
	ViolationStatusRejected ViolationStatus = "rejected"
)

// String returns the string representation of the status.
func (s ViolationStatus) String() string {
	return string(s)
}

// IsValid returns true if the status is a recognized value.
func (s ViolationStatus) IsValid() bool {
	switch s {
	case ViolationStatusPending, ViolationStatusConfirmed, ViolationStatusRejected:
		return true
	}
	return false
}

// =============================================================================
// Violation Severity
// =============================================================================

// ViolationSeverity represents the severity level of a violation.
type ViolationSeverity string

const (
	// ViolationSeverityCritical indicates an imminent danger situation.
	ViolationSeverityCritical ViolationSeverity = "critical"

	// ViolationSeveritySerious indicates a serious violation with potential
	// for severe injury or death.
	ViolationSeveritySerious ViolationSeverity = "serious"

	// ViolationSeverityOther indicates a violation that doesn't fit the
	// serious or willful categories.
	ViolationSeverityOther ViolationSeverity = "other"

	// ViolationSeverityRecommendation indicates a best practice recommendation
	// that may not be a regulatory violation.
	ViolationSeverityRecommendation ViolationSeverity = "recommendation"
)

// String returns the string representation of the severity.
func (s ViolationSeverity) String() string {
	return string(s)
}

// IsValid returns true if the severity is a recognized value.
func (s ViolationSeverity) IsValid() bool {
	switch s {
	case ViolationSeverityCritical, ViolationSeveritySerious,
		ViolationSeverityOther, ViolationSeverityRecommendation:
		return true
	}
	return false
}

// =============================================================================
// Violation Confidence
// =============================================================================

// ViolationConfidence represents the AI's confidence level in the detection.
type ViolationConfidence string

const (
	// ViolationConfidenceHigh indicates high confidence (>80%).
	ViolationConfidenceHigh ViolationConfidence = "high"

	// ViolationConfidenceMedium indicates medium confidence (50-80%).
	ViolationConfidenceMedium ViolationConfidence = "medium"

	// ViolationConfidenceLow indicates low confidence (<50%).
	ViolationConfidenceLow ViolationConfidence = "low"
)

// String returns the string representation of the confidence.
func (c ViolationConfidence) String() string {
	return string(c)
}

// IsValid returns true if the confidence is a recognized value.
func (c ViolationConfidence) IsValid() bool {
	switch c {
	case ViolationConfidenceHigh, ViolationConfidenceMedium, ViolationConfidenceLow:
		return true
	}
	return false
}

// =============================================================================
// Violation Domain Type
// =============================================================================

// Violation represents an OSHA violation found during an inspection.
//
// Violations can be:
// - AI-detected: Created by analyzing inspection photos
// - Manual: Created directly by the inspector
type Violation struct {
	ID             uuid.UUID           // Unique identifier
	InspectionID   uuid.UUID           // Inspection this violation belongs to
	ImageID        *uuid.UUID          // Optional: Image where violation was detected
	Description    string              // Inspector-editable description
	AIDescription  string              // Original AI-generated description
	Confidence     ViolationConfidence // AI confidence level
	BoundingBox    string              // Optional: JSON coordinates for image annotation
	Status         ViolationStatus     // Review status (pending, confirmed, rejected)
	Severity       ViolationSeverity   // Severity level
	InspectorNotes string              // Optional: Additional notes from inspector
	SortOrder      int                 // Display order in reports
	CreatedAt      time.Time           // When violation was created
	UpdatedAt      time.Time           // When violation was last modified

	// Computed fields (not stored in database)
	ThumbnailKey     string // Image thumbnail key (if linked to image)
	OriginalFilename string // Image filename (if linked to image)
}

// IsAIDetected returns true if this violation was detected by AI.
func (v *Violation) IsAIDetected() bool {
	return v.AIDescription != ""
}

// IsManual returns true if this violation was created manually.
func (v *Violation) IsManual() bool {
	return v.AIDescription == ""
}

// HasImage returns true if this violation is linked to an image.
func (v *Violation) HasImage() bool {
	return v.ImageID != nil
}

// =============================================================================
// Violation Regulation Link
// =============================================================================

// ViolationRegulation represents a link between a violation and an OSHA regulation.
type ViolationRegulation struct {
	ID             uuid.UUID // Unique identifier
	ViolationID    uuid.UUID // Violation this regulation applies to
	RegulationID   uuid.UUID // OSHA regulation
	RelevanceScore float64   // AI-assigned relevance score (0.0-1.0)
	AIExplanation  string    // AI's explanation for why this regulation applies
	IsPrimary      bool      // Whether this is the primary regulation for this violation
	CreatedAt      time.Time // When link was created

	// Regulation details (joined from regulations table)
	StandardNumber string // OSHA standard number (e.g., "1926.501(b)(1)")
	Title          string // Regulation title
	Category       string // Category (e.g., "Fall Protection")
}

// =============================================================================
// Violation Service Parameters
// =============================================================================

// CreateViolationParams contains validated parameters for creating a violation.
type CreateViolationParams struct {
	InspectionID   uuid.UUID         // Inspection to add violation to
	UserID         uuid.UUID         // User creating the violation (for authorization)
	ImageID        *uuid.UUID        // Optional: Associated image
	Description    string            // Required: Violation description
	Severity       ViolationSeverity // Required: Severity level
	InspectorNotes string            // Optional: Additional notes
}

// UpdateViolationParams contains validated parameters for updating a violation.
type UpdateViolationParams struct {
	ID             uuid.UUID         // Violation to update
	UserID         uuid.UUID         // User updating (for authorization)
	Description    string            // Updated description
	Severity       ViolationSeverity // Updated severity
	InspectorNotes string            // Updated notes
}

// UpdateViolationStatusParams contains parameters for updating violation status.
type UpdateViolationStatusParams struct {
	ID     uuid.UUID       // Violation to update
	UserID uuid.UUID       // User updating (for authorization)
	Status ViolationStatus // New status
}
