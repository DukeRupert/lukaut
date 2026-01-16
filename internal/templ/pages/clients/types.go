package clients

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// DisplayClient contains client data formatted for display
type DisplayClient struct {
	ID        string
	Name      string
	Email     string
	Phone     string
	SiteCount int
}

// ClientDetail contains full client data for detail view
type ClientDetail struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Notes        string
	SiteCount    int
	HasAddress   bool
}

// ClientFormValues contains form field values
type ClientFormValues struct {
	Name         string
	Email        string
	Phone        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Notes        string
}

// PaginationData contains pagination information
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

// UserDisplay contains user info for display
type UserDisplay struct {
	Name  string
	Email string
}

// ListPageData contains data for the clients list page
type ListPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Clients     []DisplayClient
	Pagination  PaginationData
	Flash       *shared.Flash
}

// FormPageData contains data for the client form page (new/edit)
type FormPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Client      *ClientDetail // nil for create, populated for edit
	Form        ClientFormValues
	Errors      map[string]string
	Flash       *shared.Flash
	IsEdit      bool
}

// ShowPageData contains data for the client detail page
type ShowPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Client      *ClientDetail
	Flash       *shared.Flash
}
