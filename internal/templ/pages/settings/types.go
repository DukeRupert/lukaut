package settings

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// Tab represents the active settings tab
type Tab string

const (
	TabProfile  Tab = "profile"
	TabBusiness Tab = "business"
	TabPassword Tab = "password"
)

// ProfilePageData contains data for the profile settings page
type ProfilePageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Form        ProfileFormData
	Errors      map[string]string
	Flash       *shared.Flash
	ActiveTab   Tab
}

// UserDisplay contains user info for display
type UserDisplay struct {
	Name               string
	Email              string
	HasBusinessProfile bool
}

// ProfileFormData contains the profile form field values
type ProfileFormData struct {
	Name  string
	Phone string
}

// PasswordPageData contains data for the password settings page
type PasswordPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Errors      map[string]string
	Flash       *shared.Flash
	ActiveTab   Tab
}

// BusinessPageData contains data for the business settings page
type BusinessPageData struct {
	CurrentPath string
	CSRFToken   string
	User        *UserDisplay
	Form        BusinessFormData
	Errors      map[string]string
	Flash       *shared.Flash
	ActiveTab   Tab
}

// BusinessFormData contains the business form field values
type BusinessFormData struct {
	BusinessName          string
	BusinessEmail         string
	BusinessPhone         string
	BusinessAddressLine1  string
	BusinessAddressLine2  string
	BusinessCity          string
	BusinessState         string
	BusinessPostalCode    string
	BusinessLicenseNumber string
	BusinessLogoURL       string
}
