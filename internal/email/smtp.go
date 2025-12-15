package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// SMTP Email Service Implementation
// =============================================================================

// SMTPEmailService sends emails via SMTP.
//
// This implementation works with:
// - Mailhog (development): No authentication required
// - Postmark SMTP (production): Uses username/password authentication
// - Any standard SMTP server
//
// Email templates are loaded from the templates directory and rendered
// with Go's html/template package.
type SMTPEmailService struct {
	config    SMTPConfig
	baseURL   string
	templates *template.Template
	logger    *slog.Logger
}

// NewSMTPEmailService creates a new SMTP-based email service.
//
// Parameters:
// - config: SMTP server configuration
// - baseURL: Application base URL for constructing links (e.g., "http://localhost:8080")
// - templatesDir: Path to email templates directory (e.g., "web/templates/email")
// - logger: Structured logger for error reporting
//
// Example usage:
//
//	emailService, err := email.NewSMTPEmailService(
//	    email.SMTPConfig{
//	        Host: "localhost",
//	        Port: 1025,
//	        From: "noreply@lukaut.com",
//	        FromName: "Lukaut",
//	    },
//	    "http://localhost:8080",
//	    "web/templates/email",
//	    logger,
//	)
func NewSMTPEmailService(
	config SMTPConfig,
	baseURL string,
	templatesDir string,
	logger *slog.Logger,
) (*SMTPEmailService, error) {
	// Set defaults
	if config.From == "" {
		config.From = DefaultFromEmail
	}
	if config.FromName == "" {
		config.FromName = DefaultFromName
	}

	// Load email templates
	pattern := filepath.Join(templatesDir, "*.html")
	templates, err := template.New("email").Funcs(emailTemplateFuncs()).ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email templates: %w", err)
	}

	return &SMTPEmailService{
		config:    config,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		templates: templates,
		logger:    logger,
	}, nil
}

// =============================================================================
// EmailService Interface Implementation
// =============================================================================

// SendVerificationEmail sends an email verification link to a new user.
func (s *SMTPEmailService) SendVerificationEmail(ctx context.Context, to, name, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)

	data := map[string]interface{}{
		"Name":      name,
		"VerifyURL": verifyURL,
		"Year":      time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("verification.html", data)
	if err != nil {
		return fmt.Errorf("failed to render verification email template: %w", err)
	}

	textBody := fmt.Sprintf(`Hi %s,

Welcome to Lukaut! Please verify your email address by clicking the link below:

%s

This link will expire in 24 hours.

If you didn't create an account with Lukaut, you can safely ignore this email.

Thanks,
The Lukaut Team
`, name, verifyURL)

	email := Email{
		To:       to,
		Subject:  "Verify your Lukaut account",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.send(ctx, email)
}

// SendPasswordResetEmail sends a password reset link to a user.
func (s *SMTPEmailService) SendPasswordResetEmail(ctx context.Context, to, name, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)

	data := map[string]interface{}{
		"Name":     name,
		"ResetURL": resetURL,
		"Year":     time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("password_reset.html", data)
	if err != nil {
		return fmt.Errorf("failed to render password reset email template: %w", err)
	}

	textBody := fmt.Sprintf(`Hi %s,

We received a request to reset your password. Click the link below to choose a new password:

%s

This link will expire in 1 hour.

If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.

Thanks,
The Lukaut Team
`, name, resetURL)

	email := Email{
		To:       to,
		Subject:  "Reset your Lukaut password",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.send(ctx, email)
}

// SendReportReadyEmail notifies a user that their inspection report is ready.
func (s *SMTPEmailService) SendReportReadyEmail(ctx context.Context, to, name, reportURL string) error {
	data := map[string]interface{}{
		"Name":      name,
		"ReportURL": reportURL,
		"Year":      time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("report_ready.html", data)
	if err != nil {
		return fmt.Errorf("failed to render report ready email template: %w", err)
	}

	textBody := fmt.Sprintf(`Hi %s,

Your inspection report is ready! You can download it here:

%s

Thanks,
The Lukaut Team
`, name, reportURL)

	email := Email{
		To:       to,
		Subject:  "Your inspection report is ready",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	return s.send(ctx, email)
}

// =============================================================================
// Internal Methods
// =============================================================================

// send sends an email via SMTP.
func (s *SMTPEmailService) send(ctx context.Context, email Email) error {
	// Build the email message
	msg := s.buildMessage(email)

	// Create SMTP address
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create auth if credentials are provided (not needed for Mailhog)
	var auth smtp.Auth
	if s.config.Username != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	// Send the email
	err := smtp.SendMail(addr, auth, s.config.From, []string{email.To}, msg)
	if err != nil {
		s.logger.Error("failed to send email",
			"to", email.To,
			"subject", email.Subject,
			"error", err,
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("email sent",
		"to", email.To,
		"subject", email.Subject,
	)

	return nil
}

// buildMessage constructs the raw email message with headers.
func (s *SMTPEmailService) buildMessage(email Email) []byte {
	var buf bytes.Buffer

	// From header with display name
	fromHeader := fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)

	// Write headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", email.To))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Create multipart message for HTML + text
	boundary := "===============LUKAUT_BOUNDARY==============="
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	buf.WriteString("\r\n")

	// Plain text part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(email.TextBody)
	buf.WriteString("\r\n")

	// HTML part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(email.HTMLBody)
	buf.WriteString("\r\n")

	// End boundary
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.Bytes()
}

// renderTemplate renders an email template with the given data.
func (s *SMTPEmailService) renderTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// =============================================================================
// Template Functions
// =============================================================================

// emailTemplateFuncs returns template functions available in email templates.
func emailTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"currentYear": func() int {
			return time.Now().Year()
		},
	}
}

// =============================================================================
// Compile-time interface check
// =============================================================================

var _ EmailService = (*SMTPEmailService)(nil)
