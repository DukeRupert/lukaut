// Package domain contains the core business types and logic.
//
// This file defines types for OSHA regulations - reference data used
// to link violations to specific safety standards.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Regulation Types
// =============================================================================

// Regulation represents an OSHA safety regulation.
// Regulations are reference data - they are not owned by users.
type Regulation struct {
	ID              uuid.UUID
	StandardNumber  string // OSHA standard number (e.g., "1926.501(b)(1)")
	Title           string
	Category        string
	Subcategory     string
	FullText        string
	Summary         string
	SeverityTypical string
	ParentStandard  string
	EffectiveDate   *time.Time
	LastUpdated     *time.Time
}

// RegulationSummary represents a regulation in list/search views.
// Contains only the fields needed for display in lists.
type RegulationSummary struct {
	ID              uuid.UUID
	StandardNumber  string
	Title           string
	Category        string
	Subcategory     string
	Summary         string
	SeverityTypical string
	Rank            float32 // Search relevance rank (only populated for search results)
}

// =============================================================================
// Search Results
// =============================================================================

// RegulationSearchResult contains paginated regulation search/browse results.
type RegulationSearchResult struct {
	Regulations []RegulationSummary
	Total       int64
}

// =============================================================================
// Service Parameters
// =============================================================================

// LinkRegulationParams contains parameters for linking a regulation to a violation.
type LinkRegulationParams struct {
	ViolationID    uuid.UUID
	RegulationID   uuid.UUID
	UserID         uuid.UUID
	RelevanceScore float64 // Optional, defaults to 1.0
	Explanation    string  // Optional, defaults to "Manually added by inspector"
	IsPrimary      bool    // Optional, defaults to false
}

// UnlinkRegulationParams contains parameters for unlinking a regulation from a violation.
type UnlinkRegulationParams struct {
	ViolationID  uuid.UUID
	RegulationID uuid.UUID
	UserID       uuid.UUID
}
