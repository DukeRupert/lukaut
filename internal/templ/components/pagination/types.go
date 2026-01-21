// Package pagination provides shared pagination components for list pages.
package pagination

// Data contains pagination information for display.
type Data struct {
	CurrentPage int
	TotalPages  int
	PerPage     int
	Total       int
	HasPrevious bool
	HasNext     bool
	PrevPage    int
	NextPage    int
}

// Config allows customization of pagination behavior.
type Config struct {
	BaseURL   string // e.g., "/inspections"
	TargetID  string // htmx target, e.g., "content-area"
	UseHtmx   bool   // Enable htmx partial loading
	PushURL   bool   // Update browser URL with hx-push-url
}

// PageRange returns a slice of page numbers for pagination display.
// Returns -1 for ellipsis positions.
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
