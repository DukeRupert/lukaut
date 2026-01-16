// Package regulations contains templ components for the regulations pages.
package regulations

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// =============================================================================
// Page Data Types
// =============================================================================

// ListPageData contains all data needed to render the regulations index page.
type ListPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Regulations []RegulationDisplay
	Categories  []string
	Filter      FilterData
	Pagination  PaginationData
	Flash       *shared.Flash
}

// SearchResultsData contains data for htmx search results partial.
type SearchResultsData struct {
	Regulations  []RegulationDisplay
	Filter       FilterData
	Pagination   PaginationData
	ViolationID  string
	EmptyMessage string
}

// DetailData contains data for the regulation detail modal partial.
type DetailData struct {
	Regulation    RegulationDetailDisplay
	ViolationID   string
	AlreadyLinked bool
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

// RegulationDisplay represents a regulation in list view.
type RegulationDisplay struct {
	ID              string
	StandardNumber  string
	Title           string
	Category        string
	Subcategory     string
	Summary         string
	SeverityTypical string
	Rank            float32
}

// RegulationDetailDisplay represents full regulation details for the modal.
type RegulationDetailDisplay struct {
	ID              string
	StandardNumber  string
	Title           string
	Category        string
	Subcategory     string
	FullText        string
	Summary         string
	SeverityTypical string
	ParentStandard  string
	EffectiveDate   string
	LastUpdated     string
}

// FilterData represents the current filter/search criteria.
type FilterData struct {
	Query    string
	Category string
}

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

	// Show first, last, and pages around current
	pages := []int{}

	// Always include page 1
	pages = append(pages, 1)

	// Calculate range around current page
	start := currentPage - 1
	end := currentPage + 1

	if start <= 2 {
		start = 2
	}
	if end >= totalPages {
		end = totalPages - 1
	}

	// Add ellipsis indicator if needed (using -1)
	if start > 2 {
		pages = append(pages, -1) // ellipsis
	}

	// Add pages around current
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	// Add ellipsis before last page if needed
	if end < totalPages-1 {
		pages = append(pages, -1) // ellipsis
	}

	// Always include last page
	if totalPages > 1 {
		pages = append(pages, totalPages)
	}

	return pages
}
