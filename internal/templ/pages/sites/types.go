package sites

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// DisplaySite contains site data formatted for display in list
type DisplaySite struct {
	ID               string
	Name             string
	CityStateZip     string
	LinkedClientID   string
	LinkedClientName string
}

// SiteDetail contains full site data for forms
type SiteDetail struct {
	ID           string
	Name         string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Notes        string
	ClientID     string
}

// SiteFormValues contains form field values
type SiteFormValues struct {
	Name         string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	ClientID     string
	Notes        string
}

// ClientOption represents a client option in the dropdown
type ClientOption struct {
	ID   string
	Name string
}

// UserDisplay contains user info for display
type UserDisplay struct {
	Name               string
	Email              string
	HasBusinessProfile bool
}

// ListPageData contains data for the sites list page
type ListPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Sites       []DisplaySite
	Flash       *shared.Flash
}

// FormPageData contains data for the site form page (new/edit)
type FormPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Site        *SiteDetail // nil for create, populated for edit
	Form        SiteFormValues
	Errors      map[string]string
	Flash       *shared.Flash
	IsEdit      bool
	Clients     []ClientOption
}
