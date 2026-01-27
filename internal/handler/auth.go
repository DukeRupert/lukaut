// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements authentication handlers for user registration, login,
// and logout functionality.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	authpkg "github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/csrf"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/email"
	"github.com/DukeRupert/lukaut/internal/invite"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/session"
	"github.com/DukeRupert/lukaut/internal/templ/pages/auth"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
	"github.com/google/uuid"
)

// =============================================================================
// Handler Configuration
// =============================================================================

// TemplateRenderer is the interface for rendering HTML templates.
// This interface allows for mocking in tests.
type TemplateRenderer interface {
	RenderHTTP(w http.ResponseWriter, name string, data interface{})
	RenderHTTPWithToast(w http.ResponseWriter, name string, data interface{}, toast ToastData)
	RenderPartial(w http.ResponseWriter, name string, data interface{})
}

// AuthHandler handles authentication-related HTTP requests.
//
// Dependencies:
// - userService: Business logic for user operations (registration, login, logout)
// - emailService: Service for sending transactional emails
// - inviteValidator: Validates invite codes for MVP testing
// - logger: Structured logging for request handling
// - isSecure: Whether to set Secure flag on cookies (true in production)
//
// Routes handled:
// - GET  /register -> ShowRegisterTempl
// - POST /register -> RegisterTempl
// - GET  /login    -> ShowLoginTempl
// - POST /login    -> LoginTempl
// - POST /logout   -> Logout
type AuthHandler struct {
	userService     service.UserService
	emailService    email.EmailService
	inviteValidator *invite.Validator
	logger          *slog.Logger
	isSecure        bool
}

// NewAuthHandler creates a new AuthHandler with the required dependencies.
//
// Parameters:
// - userService: Service for user-related operations
// - emailService: Service for sending transactional emails
// - inviteValidator: Validator for invite codes (MVP testing)
// - logger: Structured logger for request logging
// - isSecure: Set to true in production (enables Secure cookie flag)
//
// Example usage in main.go:
//
//	authHandler := handler.NewAuthHandler(userService, emailService, inviteValidator, logger, cfg.Env != "development")
func NewAuthHandler(
	userService service.UserService,
	emailService email.EmailService,
	inviteValidator *invite.Validator,
	logger *slog.Logger,
	isSecure bool,
) *AuthHandler {
	return &AuthHandler{
		userService:     userService,
		emailService:    emailService,
		inviteValidator: inviteValidator,
		logger:          logger,
		isSecure:        isSecure,
	}
}

// =============================================================================
// Template Data Types
// =============================================================================

// Flash represents a flash message to display to the user.
// Used for success messages, error notifications, and info alerts.
//
// The Type field determines styling in templates:
// - "success" -> green background
// - "error"   -> red background
// - "info"    -> blue background
type Flash struct {
	Type    string // "success", "error", or "info"
	Message string
}

// AuthPageData contains common data for authentication pages.
// This struct is passed to login.html and register.html templates.
type AuthPageData struct {
	CurrentPath        string            // Current URL path for navigation highlighting
	CSRFToken          string            // CSRF token for form protection
	Form               map[string]string // Form field values for re-populating on error
	Errors             map[string]string // Field-level validation errors
	Flash              *Flash            // Flash message to display
	ReturnTo           string            // URL to redirect to after successful login
	InviteCodesEnabled bool              // Whether invite codes are required
}

// =============================================================================
// POST /logout - Process Logout
// =============================================================================

// Logout invalidates the user's session and clears the session cookie.
//
// Flow:
// 1. Get session token from cookie
// 2. Call userService.Logout() to invalidate session in database
// 3. Clear session cookie
// 4. Redirect to login page with success message
//
// Notes:
// - This operation is idempotent - calling without a session is fine
// - Always clear the cookie even if database logout fails
// - Always redirect to login (don't show error pages)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		// Invalidate session in database
		if err := h.userService.Logout(r.Context(), cookie.Value); err != nil {
			// Log error but continue - cookie will be cleared anyway
			h.logger.Warn("failed to invalidate session in database", "error", err)
		}
	}

	// Clear session cookie (always, regardless of database result)
	clearSessionCookie(w, h.isSecure)

	// Log logout
	h.logger.Debug("user logged out")

	// Redirect to login with success message
	http.Redirect(w, r, "/login?logout=1", http.StatusSeeOther)
}

// =============================================================================
// Email Helpers
// =============================================================================

// sendVerificationEmail creates a verification token and sends the verification email.
//
// This is run asynchronously (via goroutine) to not block the HTTP response.
// The context passed should be from the original request; we create a new
// background context with timeout for the async operation.
//
// Parameters:
// - ctx: Original request context (used only to capture any relevant values)
// - userID: ID of the user to send verification to
// - email: User's email address
// - name: User's name for personalization
func (h *AuthHandler) sendVerificationEmail(ctx context.Context, userID uuid.UUID, emailAddr, name string) {
	// Create a new context with timeout for the async operation
	asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create verification token
	result, err := h.userService.CreateEmailVerificationToken(asyncCtx, userID)
	if err != nil {
		h.logger.Error("failed to create verification token",
			"error", err,
			"user_id", userID,
			"email", emailAddr,
		)
		return
	}

	// Send the verification email
	if err := h.emailService.SendVerificationEmail(asyncCtx, emailAddr, name, result.Token); err != nil {
		h.logger.Error("failed to send verification email",
			"error", err,
			"user_id", userID,
			"email", emailAddr,
		)
		return
	}

	h.logger.Info("verification email sent",
		"user_id", userID,
		"email", emailAddr,
	)
}

// =============================================================================
// Session Cookie Helpers
// =============================================================================

// setSessionCookie sets the session cookie on the response.
//
// Cookie Settings:
// - HttpOnly: true - Prevents JavaScript access (XSS protection)
// - Secure: configurable - Set true in production (HTTPS only)
// - SameSite: Lax - Prevents CSRF while allowing normal navigation
// - Path: / - Cookie sent with all requests
// - MaxAge: 7 days - Matches session duration
//
// Parameters:
// - w: Response writer to set cookie on
// - token: Raw session token (64-char hex string)
// - isSecure: Whether to set Secure flag (true in production)
func setSessionCookie(w http.ResponseWriter, token string, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    token,
		Path:     session.CookiePath,
		MaxAge:   session.CookieMaxAge,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie removes the session cookie from the client.
//
// This is done by setting MaxAge to -1, which tells the browser to delete
// the cookie immediately.
func clearSessionCookie(w http.ResponseWriter, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    "",
		Path:     session.CookiePath,
		MaxAge:   -1, // Delete immediately
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// isValidEmail performs basic email format validation.
//
// This is a simple check - the UserService performs more thorough validation.
// We do this basic check to provide immediate feedback to users.
func isValidEmail(email string) bool {
	// Basic check: contains @ and has characters before and after
	atIndex := strings.Index(email, "@")
	if atIndex < 1 {
		return false
	}
	if atIndex >= len(email)-1 {
		return false
	}

	// Check for a dot in the domain part
	domainPart := email[atIndex+1:]
	return strings.Contains(domainPart, ".")
}

// isSafeRedirectURL checks if a URL is safe to redirect to.
//
// This prevents open redirect vulnerabilities by ensuring:
// - URL is relative (starts with /)
// - URL is not a protocol-relative URL (not //)
// - URL does not redirect to external domain
//
// Examples:
// - "/dashboard"              -> true (relative URL)
// - "/settings?tab=profile"   -> true (relative URL with query)
// - "//evil.com"              -> false (protocol-relative, could be external)
// - "https://evil.com"        -> false (absolute URL to external domain)
// - "javascript:alert(1)"     -> false (javascript URL)
func isSafeRedirectURL(rawURL string) bool {
	// Must start with /
	if !strings.HasPrefix(rawURL, "/") {
		return false
	}

	// Must not start with // (protocol-relative URL)
	if strings.HasPrefix(rawURL, "//") {
		return false
	}

	// Parse and validate
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Must not have a scheme (http, https, javascript, etc.)
	if parsed.Scheme != "" {
		return false
	}

	// Must not have a host (would indicate external URL)
	if parsed.Host != "" {
		return false
	}

	return true
}

// =============================================================================
// Templ-based Auth Routes with CSRF Protection
// =============================================================================

// RegisterTemplRoutes registers all auth routes using templ components with CSRF protection.
//
// Routes registered:
// - GET  /register            -> ShowRegisterTempl
// - POST /register            -> RegisterTempl
// - GET  /login               -> ShowLoginTempl
// - POST /login               -> LoginTempl
// - POST /logout              -> Logout (same as before)
// - GET  /verify-email        -> ShowVerifyEmailTempl
// - GET  /resend-verification -> ShowResendVerificationTempl
// - POST /resend-verification -> ResendVerificationTempl
// - GET  /forgot-password     -> ShowForgotPasswordTempl
// - POST /forgot-password     -> ForgotPasswordTempl
// - GET  /reset-password      -> ShowResetPasswordTempl
// - POST /reset-password      -> ResetPasswordTempl
func (h *AuthHandler) RegisterTemplRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /register", h.ShowRegisterTempl)
	mux.HandleFunc("POST /register", h.RegisterTempl)
	mux.HandleFunc("GET /login", h.ShowLoginTempl)
	mux.HandleFunc("POST /login", h.LoginTempl)
	mux.HandleFunc("POST /logout", h.Logout)

	// Email verification routes (public — user may not be logged in)
	mux.HandleFunc("GET /verify-email", h.ShowVerifyEmailTempl)
	mux.HandleFunc("GET /resend-verification", h.ShowResendVerificationTempl)
	mux.HandleFunc("POST /resend-verification", h.ResendVerificationTempl)

	// Password reset routes
	mux.HandleFunc("GET /forgot-password", h.ShowForgotPasswordTempl)
	mux.HandleFunc("POST /forgot-password", h.ForgotPasswordTempl)
	mux.HandleFunc("GET /reset-password", h.ShowResetPasswordTempl)
	mux.HandleFunc("POST /reset-password", h.ResetPasswordTempl)
}

// RegisterVerifyEmailReminderRoutes registers the email verification reminder routes.
//
// These routes require authentication (requireUser) but must NOT use RequireEmailVerified
// middleware, otherwise unverified users would be stuck in a redirect loop.
// Call this separately from RegisterTemplRoutes because it needs the requireUser middleware.
func (h *AuthHandler) RegisterVerifyEmailReminderRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /verify-email-reminder", requireUser(http.HandlerFunc(h.ShowVerifyEmailReminderTempl)))
	mux.Handle("POST /verify-email-reminder/resend", requireUser(http.HandlerFunc(h.ResendVerificationForCurrentUserTempl)))
}

// =============================================================================
// GET /login (Templ) - Show Login Form
// =============================================================================

// ShowLoginTempl renders the login form using templ components with CSRF protection.
func (h *AuthHandler) ShowLoginTempl(w http.ResponseWriter, r *http.Request) {
	// Generate CSRF token
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	// Check for success query params
	var flash *shared.Flash
	if r.URL.Query().Get("registered") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Account created successfully! Please sign in.",
		}
	} else if r.URL.Query().Get("reset") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Password reset successfully! Please sign in with your new password.",
		}
	} else if r.URL.Query().Get("logout") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "You have been signed out.",
		}
	}

	// Get return_to from query params
	returnTo := r.URL.Query().Get("return_to")

	data := auth.LoginPageData{
		CSRFToken: csrfToken,
		Form:      auth.FormData{},
		Errors:    make(map[string]string),
		Flash:     flash,
		ReturnTo:  returnTo,
	}

	if err := auth.LoginPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render login page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /login (Templ) - Process Login
// =============================================================================

// LoginTempl processes the login form submission with CSRF validation.
func (h *AuthHandler) LoginTempl(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("login attempt", "method", r.Method, "path", r.URL.Path)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderLoginTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Validate CSRF token
	if !csrf.ValidateRequest(r) {
		h.logger.Warn("CSRF validation failed")
		h.renderLoginTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid security token. Please try again.",
		})
		return
	}

	// Extract form values
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	returnTo := r.FormValue("return_to")

	// Store form values for re-rendering (except password)
	formValues := auth.FormData{
		Email: email,
	}

	// Basic validation
	errors := make(map[string]string)

	if email == "" {
		errors["email"] = "Email is required"
	}

	if password == "" {
		errors["password"] = "Password is required"
	}

	if len(errors) > 0 {
		h.renderLoginTemplError(w, r, formValues, errors, nil)
		return
	}

	// Call UserService.Login
	loginResult, err := h.userService.Login(r.Context(), email, password)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.EUNAUTHORIZED:
			h.logger.Info("login failed: invalid credentials", "email", email)
			h.renderLoginTemplError(w, r, formValues, nil, &shared.Flash{
				Type:    shared.FlashError,
				Message: "Invalid email or password",
			})
		default:
			h.logger.Error("login failed", "error", err, "email", email)
			h.renderLoginTemplError(w, r, formValues, nil, &shared.Flash{
				Type:    shared.FlashError,
				Message: "Login failed. Please try again later.",
			})
		}
		return
	}

	// Set session cookie
	setSessionCookie(w, loginResult.Token, h.isSecure)

	// Refresh CSRF token after successful login
	csrf.RefreshToken(w, h.isSecure)

	// Log successful login
	h.logger.Info("user logged in",
		"user_id", loginResult.User.ID,
		"email", loginResult.User.Email,
	)

	// Redirect to return_to URL or dashboard
	redirectURL := "/dashboard"
	if returnTo != "" && isSafeRedirectURL(returnTo) {
		redirectURL = returnTo
	}

	// For htmx requests, use HX-Redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// renderLoginTemplError re-renders the login form with errors using templ.
func (h *AuthHandler) renderLoginTemplError(
	w http.ResponseWriter,
	r *http.Request,
	formValues auth.FormData,
	errors map[string]string,
	flash *shared.Flash,
) {
	if errors == nil {
		errors = make(map[string]string)
	}

	// Generate new CSRF token for re-render
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.LoginPageData{
		CSRFToken: csrfToken,
		Form:      formValues,
		Errors:    errors,
		Flash:     flash,
		ReturnTo:  r.FormValue("return_to"),
	}

	// For htmx requests, return just the form partial
	if r.Header.Get("HX-Request") == "true" {
		if err := auth.LoginForm(data).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render login form", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if err := auth.LoginPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render login page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /register (Templ) - Show Registration Form
// =============================================================================

// ShowRegisterTempl renders the registration form using templ components with CSRF protection.
func (h *AuthHandler) ShowRegisterTempl(w http.ResponseWriter, r *http.Request) {
	// Generate CSRF token
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	// Get return_to from query params
	returnTo := r.URL.Query().Get("return_to")

	data := auth.RegisterPageData{
		CSRFToken:          csrfToken,
		Form:               auth.FormData{},
		Errors:             make(map[string]string),
		Flash:              nil,
		ReturnTo:           returnTo,
		InviteCodesEnabled: h.inviteValidator.IsEnabled(),
	}

	if err := auth.RegisterPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render register page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /register (Templ) - Process Registration
// =============================================================================

// RegisterTempl processes the registration form submission with CSRF validation.
func (h *AuthHandler) RegisterTempl(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderRegisterTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Validate CSRF token
	if !csrf.ValidateRequest(r) {
		h.logger.Warn("CSRF validation failed")
		h.renderRegisterTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid security token. Please try again.",
		})
		return
	}

	// Extract and normalize form values
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	passwordConfirmation := r.FormValue("password_confirmation")
	terms := r.FormValue("terms")
	inviteCode := strings.TrimSpace(r.FormValue("invite_code"))
	returnTo := r.FormValue("return_to")

	// Store form values for re-rendering (except passwords)
	formValues := auth.FormData{
		Name:       name,
		Email:      email,
		InviteCode: inviteCode,
	}

	// Validate form fields
	errors := make(map[string]string)

	if name == "" {
		errors["name"] = "Name is required"
	}

	if email == "" {
		errors["email"] = "Email is required"
	} else if !isValidEmail(email) {
		errors["email"] = "Please enter a valid email address"
	}

	if password == "" {
		errors["password"] = "Password is required"
	} else if len(password) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	}

	if passwordConfirmation == "" {
		errors["password_confirmation"] = "Please confirm your password"
	} else if password != passwordConfirmation {
		errors["password_confirmation"] = "Passwords do not match"
	}

	if terms != "on" {
		errors["terms"] = "You must accept the Terms of Service"
	}

	// Validate invite code if enabled
	if h.inviteValidator.IsEnabled() {
		if inviteCode == "" {
			errors["invite_code"] = "Invite code is required"
		} else if !h.inviteValidator.ValidateCode(inviteCode) {
			errors["invite_code"] = "Invalid invite code"
		}
	}

	// If validation errors, re-render form
	if len(errors) > 0 {
		h.renderRegisterTemplError(w, r, formValues, errors, nil)
		return
	}

	// Call UserService.Register
	user, err := h.userService.Register(r.Context(), domain.RegisterParams{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ECONFLICT:
			errors["email"] = "An account with this email already exists"
			h.renderRegisterTemplError(w, r, formValues, errors, nil)
		case domain.EINVALID:
			h.renderRegisterTemplError(w, r, formValues, nil, &shared.Flash{
				Type:    shared.FlashError,
				Message: domain.ErrorMessage(err),
			})
		default:
			h.logger.Error("registration failed", "error", err, "email", email)
			h.renderRegisterTemplError(w, r, formValues, nil, &shared.Flash{
				Type:    shared.FlashError,
				Message: "Registration failed. Please try again later.",
			})
		}
		return
	}

	// Create verification token and send email
	go h.sendVerificationEmail(r.Context(), user.ID, user.Email, user.Name)

	// Registration successful - log the user in automatically
	loginResult, err := h.userService.Login(r.Context(), email, password)
	if err != nil {
		h.logger.Error("auto-login after registration failed", "error", err, "email", email)
		http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
		return
	}

	// Set session cookie
	setSessionCookie(w, loginResult.Token, h.isSecure)

	// Refresh CSRF token after successful registration/login
	csrf.RefreshToken(w, h.isSecure)

	// Log successful registration
	h.logger.Info("user registered and logged in",
		"user_id", loginResult.User.ID,
		"email", loginResult.User.Email,
	)

	// Redirect to return_to URL or dashboard
	redirectURL := "/dashboard"
	if returnTo != "" && isSafeRedirectURL(returnTo) {
		redirectURL = returnTo
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// renderRegisterTemplError re-renders the registration form with errors using templ.
func (h *AuthHandler) renderRegisterTemplError(
	w http.ResponseWriter,
	r *http.Request,
	formValues auth.FormData,
	errors map[string]string,
	flash *shared.Flash,
) {
	if errors == nil {
		errors = make(map[string]string)
	}

	// Generate new CSRF token for re-render
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.RegisterPageData{
		CSRFToken:          csrfToken,
		Form:               formValues,
		Errors:             errors,
		Flash:              flash,
		ReturnTo:           r.FormValue("return_to"),
		InviteCodesEnabled: h.inviteValidator.IsEnabled(),
	}

	if err := auth.RegisterPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render register page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /verify-email (Templ) - Verify Email Token
// =============================================================================

// ShowVerifyEmailTempl handles the email verification link click using templ.
func (h *AuthHandler) ShowVerifyEmailTempl(w http.ResponseWriter, r *http.Request) {
	// Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderVerifyEmailTemplError(w, r, "Invalid verification link. Please check your email for the correct link.")
		return
	}

	// Call UserService.VerifyEmail
	err := h.userService.VerifyEmail(r.Context(), token)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			h.renderVerifyEmailTemplError(w, r, "This verification link has expired or is invalid. Please request a new verification email.")
		case domain.ECONFLICT:
			h.renderVerifyEmailTemplSuccess(w, r, "Your email is already verified. You can sign in to your account.")
		default:
			h.logger.Error("email verification failed", "error", err)
			h.renderVerifyEmailTemplError(w, r, "Verification failed. Please try again later.")
		}
		return
	}

	// Success
	h.renderVerifyEmailTemplSuccess(w, r, "Your email has been verified! You can now sign in to your account.")
}

func (h *AuthHandler) renderVerifyEmailTemplSuccess(w http.ResponseWriter, r *http.Request, message string) {
	data := auth.VerifyEmailPageData{
		Success: true,
		Message: message,
	}
	if err := auth.VerifyEmailPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render verify email page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderVerifyEmailTemplError(w http.ResponseWriter, r *http.Request, message string) {
	data := auth.VerifyEmailPageData{
		Success: false,
		Message: message,
	}
	if err := auth.VerifyEmailPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render verify email page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /resend-verification (Templ) - Show Resend Verification Form
// =============================================================================

// ShowResendVerificationTempl renders the resend verification form using templ.
func (h *AuthHandler) ShowResendVerificationTempl(w http.ResponseWriter, r *http.Request) {
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.ResendVerificationPageData{
		CSRFToken: csrfToken,
		Form:      auth.FormData{},
		Errors:    make(map[string]string),
		Flash:     nil,
	}

	if err := auth.ResendVerificationPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render resend verification page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /resend-verification (Templ) - Request New Verification Email
// =============================================================================

// ResendVerificationTempl handles requests to resend the verification email with CSRF validation.
func (h *AuthHandler) ResendVerificationTempl(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.renderResendVerificationTemplError(w, r, "Invalid form submission")
		return
	}

	// Validate CSRF token
	if !csrf.ValidateRequest(r) {
		h.logger.Warn("CSRF validation failed")
		h.renderResendVerificationTemplError(w, r, "Invalid security token. Please try again.")
		return
	}

	// Get email
	emailAddr := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if emailAddr == "" {
		h.renderResendVerificationTemplError(w, r, "Email is required")
		return
	}

	// Call UserService.ResendVerificationEmail
	result, err := h.userService.ResendVerificationEmail(r.Context(), emailAddr)
	if err != nil {
		// Log error but show generic success (prevents enumeration)
		h.logger.Debug("resend verification failed", "error", err, "email", emailAddr)
	} else {
		// Get user name for the email
		user, err := h.userService.GetByID(r.Context(), result.UserID)
		if err != nil {
			h.logger.Error("failed to get user for resend verification", "error", err, "user_id", result.UserID)
		} else {
			// Send the email asynchronously
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := h.emailService.SendVerificationEmail(ctx, emailAddr, user.Name, result.Token); err != nil {
					h.logger.Error("failed to send verification email", "error", err, "email", emailAddr)
				} else {
					h.logger.Info("verification email sent", "email", emailAddr)
				}
			}()
		}
	}

	// Always show success (don't reveal if email exists)
	if err := auth.ResendVerificationSentPage().Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render resend verification sent page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderResendVerificationTemplError(w http.ResponseWriter, r *http.Request, message string) {
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.ResendVerificationPageData{
		CSRFToken: csrfToken,
		Form:      auth.FormData{},
		Errors:    make(map[string]string),
		Flash: &shared.Flash{
			Type:    shared.FlashError,
			Message: message,
		},
	}

	if err := auth.ResendVerificationPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render resend verification page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /verify-email-reminder - Show Email Verification Reminder
// =============================================================================
//
// Purpose: Display a page telling the logged-in user they need to verify their
//          email before they can access the application. This is the redirect
//          target of the RequireEmailVerified middleware.
//
// Inputs:
//   - Authenticated user from context (via auth.GetUser)
//
// Outputs:
//   - Renders the verify-email-reminder template with the user's email address
//   - Page shows: message, email address, resend button, sign out link
//
// Note: This route needs requireUser middleware (user must be logged in)
//       but NOT RequireEmailVerified (that would cause a redirect loop).

func (h *AuthHandler) ShowVerifyEmailReminderTempl(w http.ResponseWriter, r *http.Request) {
	user := authpkg.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Already verified — redirect to dashboard
	if user.EmailVerified {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	data := auth.VerifyEmailReminderPageData{
		Email: user.Email,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := auth.VerifyEmailReminderPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render verify email reminder page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /verify-email-reminder/resend - Resend Verification (htmx, for logged-in users)
// =============================================================================
//
// Purpose: Resend the verification email for the currently logged-in user.
//          Called via htmx from the reminder page. Unlike POST /resend-verification,
//          this doesn't need an email form field — it uses the authenticated user.
//
// Inputs:
//   - Authenticated user from context
//
// Outputs (htmx):
//   - Returns the VerifyEmailReminderResent partial (replaces the resend button)
//
// Side effects:
//   - Creates a new verification token
//   - Sends verification email asynchronously

func (h *AuthHandler) ResendVerificationForCurrentUserTempl(w http.ResponseWriter, r *http.Request) {
	user := authpkg.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.EmailVerified {
		// Already verified — redirect to dashboard
		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Send verification email asynchronously
	go h.sendVerificationEmail(r.Context(), user.ID, user.Email, user.Name)

	h.logger.Info("verification email resend requested from reminder page",
		"user_id", user.ID,
		"email", user.Email,
	)

	// Return the "sent" confirmation partial for htmx swap
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := auth.VerifyEmailReminderResent().Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render verification resent partial", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /forgot-password (Templ) - Show Forgot Password Form
// =============================================================================

// ShowForgotPasswordTempl renders the forgot password form using templ.
func (h *AuthHandler) ShowForgotPasswordTempl(w http.ResponseWriter, r *http.Request) {
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.ForgotPasswordPageData{
		CSRFToken: csrfToken,
		Form:      auth.FormData{},
		Errors:    make(map[string]string),
		Flash:     nil,
	}

	if err := auth.ForgotPasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render forgot password page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /forgot-password (Templ) - Process Forgot Password Request
// =============================================================================

// ForgotPasswordTempl processes the forgot password form submission with CSRF validation.
func (h *AuthHandler) ForgotPasswordTempl(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderForgotPasswordTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Validate CSRF token
	if !csrf.ValidateRequest(r) {
		h.logger.Warn("CSRF validation failed")
		h.renderForgotPasswordTemplError(w, r, auth.FormData{}, nil, &shared.Flash{
			Type:    shared.FlashError,
			Message: "Invalid security token. Please try again.",
		})
		return
	}

	// Get email
	emailAddr := strings.ToLower(strings.TrimSpace(r.FormValue("email")))

	// Store form values for re-rendering
	formValues := auth.FormData{
		Email: emailAddr,
	}

	// Basic validation
	if emailAddr == "" {
		h.renderForgotPasswordTemplError(w, r, formValues, map[string]string{
			"email": "Email is required",
		}, nil)
		return
	}

	if !isValidEmail(emailAddr) {
		h.renderForgotPasswordTemplError(w, r, formValues, map[string]string{
			"email": "Please enter a valid email address",
		}, nil)
		return
	}

	// Create password reset token (if user exists)
	result, err := h.userService.CreatePasswordResetToken(r.Context(), emailAddr)
	if err != nil {
		// Log error but show generic success (prevents enumeration)
		h.logger.Debug("password reset token creation failed", "error", err, "email", emailAddr)
	} else {
		// Get user name for the email
		user, err := h.userService.GetByID(r.Context(), result.UserID)
		if err != nil {
			h.logger.Error("failed to get user for password reset", "error", err, "user_id", result.UserID)
		} else {
			// Send the email asynchronously
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := h.emailService.SendPasswordResetEmail(ctx, emailAddr, user.Name, result.Token); err != nil {
					h.logger.Error("failed to send password reset email", "error", err, "email", emailAddr)
				} else {
					h.logger.Info("password reset email sent", "email", emailAddr)
				}
			}()
		}
	}

	// Always show success (don't reveal if email exists)
	if err := auth.ForgotPasswordSentPage().Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render forgot password sent page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderForgotPasswordTemplError(
	w http.ResponseWriter,
	r *http.Request,
	formValues auth.FormData,
	errors map[string]string,
	flash *shared.Flash,
) {
	if errors == nil {
		errors = make(map[string]string)
	}

	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.ForgotPasswordPageData{
		CSRFToken: csrfToken,
		Form:      formValues,
		Errors:    errors,
		Flash:     flash,
	}

	if err := auth.ForgotPasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render forgot password page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// GET /reset-password (Templ) - Show Reset Password Form
// =============================================================================

// ShowResetPasswordTempl renders the reset password form using templ.
func (h *AuthHandler) ShowResetPasswordTempl(w http.ResponseWriter, r *http.Request) {
	// Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderResetPasswordTemplInvalid(w, r, "Invalid reset link. Please check your email for the correct link.")
		return
	}

	// Validate the token
	_, err := h.userService.ValidatePasswordResetToken(r.Context(), token)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			h.renderResetPasswordTemplInvalid(w, r, "This reset link has expired or is invalid. Please request a new password reset.")
		case domain.EINVALID:
			h.renderResetPasswordTemplInvalid(w, r, "This reset link is invalid. Please request a new password reset.")
		default:
			h.logger.Error("password reset token validation failed", "error", err)
			h.renderResetPasswordTemplInvalid(w, r, "Something went wrong. Please request a new password reset.")
		}
		return
	}

	// Token is valid - show reset form
	csrfToken := csrf.EnsureToken(w, r, h.isSecure)

	data := auth.ResetPasswordPageData{
		CSRFToken: csrfToken,
		Token:     token,
		Form:      auth.FormData{},
		Errors:    make(map[string]string),
		Flash:     nil,
	}

	if err := auth.ResetPasswordPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render reset password page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderResetPasswordTemplInvalid(w http.ResponseWriter, r *http.Request, message string) {
	if err := auth.ResetPasswordInvalidPage(message).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render reset password invalid page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// =============================================================================
// POST /reset-password (Templ) - Process Password Reset
// =============================================================================

// ResetPasswordTempl processes the password reset form submission with CSRF validation.
func (h *AuthHandler) ResetPasswordTempl(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderResetPasswordTemplInvalid(w, r, "Invalid form submission. Please try again.")
		return
	}

	// Validate CSRF token
	if !csrf.ValidateRequest(r) {
		h.logger.Warn("CSRF validation failed")
		h.renderResetPasswordTemplInvalid(w, r, "Invalid security token. Please try again.")
		return
	}

	// Get form values
	token := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirmation := r.FormValue("password_confirmation")

	// Validate token is present
	if token == "" {
		h.renderResetPasswordTemplInvalid(w, r, "Invalid reset link. Please request a new password reset.")
		return
	}

	// Validate passwords
	errors := make(map[string]string)

	if password == "" {
		errors["password"] = "Password is required"
	} else if len(password) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	}

	if passwordConfirmation == "" {
		errors["password_confirmation"] = "Please confirm your password"
	} else if password != passwordConfirmation {
		errors["password_confirmation"] = "Passwords do not match"
	}

	// If validation errors, re-render form
	if len(errors) > 0 {
		csrfToken := csrf.EnsureToken(w, r, h.isSecure)
		data := auth.ResetPasswordPageData{
			CSRFToken: csrfToken,
			Token:     token,
			Form:      auth.FormData{},
			Errors:    errors,
			Flash:     nil,
		}
		if err := auth.ResetPasswordPage(data).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render reset password page", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Call UserService.ResetPassword
	err := h.userService.ResetPassword(r.Context(), domain.ResetPasswordParams{
		Token:       token,
		NewPassword: password,
	})
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			h.renderResetPasswordTemplInvalid(w, r, "This reset link has expired or is invalid. Please request a new password reset.")
		case domain.EINVALID:
			csrfToken := csrf.EnsureToken(w, r, h.isSecure)
			data := auth.ResetPasswordPageData{
				CSRFToken: csrfToken,
				Token:     token,
				Form:      auth.FormData{},
				Errors:    make(map[string]string),
				Flash: &shared.Flash{
					Type:    shared.FlashError,
					Message: domain.ErrorMessage(err),
				},
			}
			if err := auth.ResetPasswordPage(data).Render(r.Context(), w); err != nil {
				h.logger.Error("failed to render reset password page", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		default:
			h.logger.Error("password reset failed", "error", err)
			h.renderResetPasswordTemplInvalid(w, r, "Something went wrong. Please try again.")
		}
		return
	}

	// Success - redirect to login with success message
	h.logger.Info("password reset completed")
	http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
}

// =============================================================================
// Integration Notes for main.go
// =============================================================================

/*
To integrate this handler into main.go:

1. Initialize the UserService (already done in service layer):

   userService := service.NewUserService(repo, logger)

2. Create the AuthHandler:

   isSecure := cfg.Env != "development"
   authHandler := handler.NewAuthHandler(userService, renderer, logger, isSecure)

3. Create the AuthMiddleware:

   authMw := middleware.NewAuthMiddleware(userService, logger, isSecure)

4. Register routes - Option A (using RegisterRoutes helper):

   authHandler.RegisterRoutes(mux)

4. Register routes - Option B (manually for more control):

   // Public auth routes (no auth middleware)
   mux.HandleFunc("GET /register", authHandler.ShowRegister)
   mux.HandleFunc("POST /register", authHandler.Register)
   mux.HandleFunc("GET /login", authHandler.ShowLogin)
   mux.HandleFunc("POST /login", authHandler.Login)

   // Logout requires being logged in (or is idempotent if not)
   mux.HandleFunc("POST /logout", authHandler.Logout)

5. Update authenticated routes to use middleware:

   // Create middleware stack for authenticated routes
   authStack := middleware.Stack(authMw.WithUser, authMw.RequireUser)

   // Protected routes
   mux.Handle("GET /dashboard", authStack(http.HandlerFunc(dashboardHandler)))

6. Update the existing inline handlers in main.go:

   Replace:
   mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
       renderer.RenderHTTP(w, "auth/login", map[string]interface{}{...})
   })

   With:
   mux.HandleFunc("GET /login", authHandler.ShowLogin)
   mux.HandleFunc("POST /login", authHandler.Login)

Example complete main.go changes:

   // In run() function, after initializing repository...

   // Initialize services
   userService := service.NewUserService(repo, logger)

   // Initialize middleware
   isSecure := cfg.Env != "development"
   authMw := middleware.NewAuthMiddleware(userService, logger, isSecure)

   // Initialize handlers
   authHandler := handler.NewAuthHandler(userService, renderer, logger, isSecure)

   // Create middleware stacks
   // withUser := authMw.WithUser  // Loads user if logged in, continues either way
   // requireUser := middleware.Stack(authMw.WithUser, authMw.RequireUser)

   // Register routes
   mux := http.NewServeMux()

   // Static files
   mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

   // Health check
   mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
       w.WriteHeader(http.StatusOK)
       w.Write([]byte("OK"))
   })

   // Public pages
   mux.HandleFunc("GET /", homeHandler)

   // Auth routes
   authHandler.RegisterRoutes(mux)

   // Protected routes (uncomment when ready)
   // mux.Handle("GET /dashboard", requireUser(http.HandlerFunc(dashboardHandler)))
*/
