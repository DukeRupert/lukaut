package settings

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// Tab represents the active settings tab
type Tab string

const (
	TabProfile  Tab = "profile"
	TabBusiness Tab = "business"
	TabPassword Tab = "password"
	TabBilling  Tab = "billing"
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

// BillingPageData contains data for the billing settings page.
//
// This page shows the user's current subscription plan, status, and actions
// (upgrade, manage via Stripe portal, cancel, reactivate).
type BillingPageData struct {
	CurrentPath string
	User        *UserDisplay
	Plan        PlanInfo
	Flash       *shared.Flash
	ActiveTab   Tab
}

// PlanInfo contains the user's current subscription plan details.
//
// Fields populated from domain.User and Stripe subscription data:
//   - Tier: "starter" or "professional" (from domain.SubscriptionTier)
//   - Status: "active", "trialing", "canceled", "past_due", "inactive" (from domain.SubscriptionStatus)
//   - PeriodEnd: end of current billing period (from Stripe subscription.current_period_end)
//   - CancelAtEnd: whether the subscription is set to cancel at period end
type PlanInfo struct {
	Tier        string // "starter", "professional", or "" for no plan
	Status      string // "active", "trialing", "canceled", "past_due", "inactive"
	PeriodEnd   string // formatted date string, e.g. "January 15, 2026"
	CancelAtEnd bool   // true if subscription will cancel at period end
}
