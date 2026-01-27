package auth

import "github.com/DukeRupert/lukaut/internal/templ/shared"

// LoginPageData contains data for the login page
type LoginPageData struct {
	Form      FormData
	Errors    map[string]string
	Flash     *shared.Flash
	CSRFToken string
	ReturnTo  string
}

// RegisterPageData contains data for the registration page
type RegisterPageData struct {
	Form               FormData
	Errors             map[string]string
	Flash              *shared.Flash
	CSRFToken          string
	ReturnTo           string
	InviteCodesEnabled bool
}

// ForgotPasswordPageData contains data for the forgot password page
type ForgotPasswordPageData struct {
	Form      FormData
	Errors    map[string]string
	Flash     *shared.Flash
	CSRFToken string
}

// ResetPasswordPageData contains data for the reset password page
type ResetPasswordPageData struct {
	Form      FormData
	Errors    map[string]string
	Flash     *shared.Flash
	CSRFToken string
	Token     string
}

// VerifyEmailPageData contains data for the verify email page
type VerifyEmailPageData struct {
	Flash     *shared.Flash
	CSRFToken string
	Success   bool
	Message   string
}

// ResendVerificationPageData contains data for the resend verification page
type ResendVerificationPageData struct {
	Form      FormData
	Errors    map[string]string
	Flash     *shared.Flash
	CSRFToken string
}

// VerifyEmailReminderPageData contains data for the email verification reminder page.
//
// This page is shown to logged-in users whose email is not yet verified.
// The RequireEmailVerified middleware redirects unverified users here.
type VerifyEmailReminderPageData struct {
	Email string // The user's email address (shown so they know where to check)
}

// FormData holds form field values for repopulation after validation errors
type FormData struct {
	Email       string
	Name        string
	CompanyName string
	Password    string
	InviteCode  string
}
