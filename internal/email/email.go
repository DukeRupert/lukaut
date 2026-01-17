// Package email provides email sending functionality for the Lukaut application.
//
// This package defines an EmailService interface with implementations for:
// - SMTP (for development with Mailhog and production with services like Postmark SMTP)
// - Future: Postmark API implementation for advanced features
package email

import (
	"context"
)

// =============================================================================
// Interface Definition
// =============================================================================

// EmailService defines the interface for sending transactional emails.
//
// Implementations:
// - SMTPEmailService: Uses SMTP protocol (Mailhog for dev, Postmark SMTP for prod)
// - Future: PostmarkAPIService for API-based sending with delivery tracking
//
// All methods are context-aware for timeout and cancellation support.
type EmailService interface {
	// SendVerificationEmail sends an email verification link to a new user.
	// Parameters:
	// - to: Recipient email address
	// - name: Recipient's name for personalization
	// - token: Raw verification token to include in the link
	SendVerificationEmail(ctx context.Context, to, name, token string) error

	// SendPasswordResetEmail sends a password reset link to a user.
	// Parameters:
	// - to: Recipient email address
	// - name: Recipient's name for personalization
	// - token: Raw reset token to include in the link
	SendPasswordResetEmail(ctx context.Context, to, name, token string) error

	// SendReportReadyEmail notifies a user that their inspection report is ready.
	// Parameters:
	// - to: Recipient email address
	// - name: Recipient's name for personalization
	// - reportURL: URL where the report can be downloaded
	SendReportReadyEmail(ctx context.Context, to, name, reportURL string) error

	// SendReportToClientEmail sends an inspection report to a client.
	// Parameters:
	// - to: Recipient email address (client)
	// - inspectorName: Name of the inspector who conducted the inspection
	// - inspectorCompany: Company name of the inspector
	// - siteName: Name of the inspection site
	// - reportURL: URL where the report can be downloaded
	SendReportToClientEmail(ctx context.Context, to, inspectorName, inspectorCompany, siteName, reportURL string) error
}

// =============================================================================
// Email Data Types
// =============================================================================

// Email represents a single email message.
type Email struct {
	To       string // Recipient email address
	Subject  string // Email subject line
	HTMLBody string // HTML content of the email
	TextBody string // Plain text fallback content
}

// =============================================================================
// Configuration Types
// =============================================================================

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Host     string // SMTP server hostname (e.g., "localhost" for Mailhog)
	Port     int    // SMTP server port (e.g., 1025 for Mailhog)
	Username string // SMTP authentication username (empty for Mailhog)
	Password string // SMTP authentication password (empty for Mailhog)
	From     string // Default sender email address
	FromName string // Default sender display name
}

// BaseURL is used for constructing links in emails.
// This should be set to the application's public URL.
type BaseURL string

// =============================================================================
// Common Constants
// =============================================================================

const (
	// DefaultFromEmail is the default sender email for transactional emails.
	DefaultFromEmail = "noreply@lukaut.com"

	// DefaultFromName is the default sender display name.
	DefaultFromName = "Lukaut"
)
