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

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/email"
	"github.com/DukeRupert/lukaut/internal/invite"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/google/uuid"
)

// =============================================================================
// Session Cookie Configuration
// =============================================================================

// Session cookie constants - these match the values in middleware/auth.go
// to ensure consistency. If these change, update both locations.
//
// NOTE: These are duplicated from middleware/auth.go to avoid import cycle.
// The middleware package imports handler for error responses, so handler
// cannot import middleware.
//
// ARCHITECTURE NOTE: A future refactor could move these constants to a shared
// package (e.g., internal/session) that both handler and middleware import.
const (
	// sessionCookieName is the name of the cookie that stores the session token.
	sessionCookieName = "lukaut_session"

	// sessionCookiePath ensures the cookie is sent with all requests.
	sessionCookiePath = "/"

	// sessionCookieMaxAge sets the cookie expiration (7 days = 604800 seconds).
	sessionCookieMaxAge = 7 * 24 * 60 * 60
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
// - renderer: Template rendering for HTML responses
// - logger: Structured logging for request handling
// - isSecure: Whether to set Secure flag on cookies (true in production)
//
// Routes handled:
// - GET  /register -> ShowRegister
// - POST /register -> Register
// - GET  /login    -> ShowLogin
// - POST /login    -> Login
// - POST /logout   -> Logout
type AuthHandler struct {
	userService     service.UserService
	emailService    email.EmailService
	inviteValidator *invite.Validator
	renderer        TemplateRenderer
	logger          *slog.Logger
	isSecure        bool
}

// NewAuthHandler creates a new AuthHandler with the required dependencies.
//
// Parameters:
// - userService: Service for user-related operations
// - emailService: Service for sending transactional emails
// - inviteValidator: Validator for invite codes (MVP testing)
// - renderer: Template renderer for HTML pages
// - logger: Structured logger for request logging
// - isSecure: Set to true in production (enables Secure cookie flag)
//
// Example usage in main.go:
//
//	authHandler := handler.NewAuthHandler(userService, emailService, inviteValidator, renderer, logger, cfg.Env != "development")
func NewAuthHandler(
	userService service.UserService,
	emailService email.EmailService,
	inviteValidator *invite.Validator,
	renderer TemplateRenderer,
	logger *slog.Logger,
	isSecure bool,
) *AuthHandler {
	return &AuthHandler{
		userService:     userService,
		emailService:    emailService,
		inviteValidator: inviteValidator,
		renderer:        renderer,
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
// GET /register - Show Registration Form
// =============================================================================

// ShowRegister renders the registration form.
//
// Template: auth/register
//
// Query Parameters:
// - return_to (optional): URL to redirect to after successful registration and login
//
// Template Data:
// - CurrentPath: "/register"
// - CSRFToken: Token for form protection
// - Form: Empty map (no pre-filled values)
// - Errors: Empty map (no validation errors)
// - Flash: nil (no flash message)
// - ReturnTo: Value from query param if present
//
// Implementation Notes:
// - If user is already logged in, redirect to dashboard
// - Generate CSRF token and store in session/cookie
// - Pass empty form values for initial render
func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	// TODO: Check if user is already authenticated
	// user := middleware.GetUser(r.Context())
	// if user != nil {
	//     http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	//     return
	// }

	// TODO: Generate CSRF token
	// For MVP, we can start without CSRF and add it later, or use the
	// double-submit cookie pattern where the token is stored in a cookie
	// and also submitted in the form. The server compares both values.
	//
	// Option A: Session-based CSRF (requires session storage)
	// csrfToken := generateCSRFToken()
	// storeCSRFInSession(r.Context(), csrfToken)
	//
	// Option B: Double-submit cookie pattern (simpler, no session storage needed)
	// csrfToken := generateCSRFToken()
	// http.SetCookie(w, &http.Cookie{
	//     Name:     "csrf_token",
	//     Value:    csrfToken,
	//     Path:     "/",
	//     HttpOnly: false, // Must be readable by JavaScript/form
	//     Secure:   h.isSecure,
	//     SameSite: http.SameSiteStrictMode,
	// })

	// Get return_to from query params for post-registration redirect
	returnTo := r.URL.Query().Get("return_to")

	data := AuthPageData{
		CurrentPath:        r.URL.Path,
		CSRFToken:          "", // TODO: Set CSRF token
		Form:               make(map[string]string),
		Errors:             make(map[string]string),
		Flash:              nil,
		ReturnTo:           returnTo,
		InviteCodesEnabled: h.inviteValidator.IsEnabled(),
	}

	h.renderer.RenderHTTP(w, "auth/register", data)
}

// =============================================================================
// POST /register - Process Registration
// =============================================================================

// Register processes the registration form submission.
//
// Form Fields:
// - name (required): User's full name
// - email (required): User's email address
// - password (required): User's password (min 8 chars)
// - password_confirmation (required): Must match password
// - terms (required): Checkbox for terms acceptance
//
// Validation:
// 1. Parse form data
// 2. Validate CSRF token
// 3. Validate all required fields are present
// 4. Validate email format
// 5. Validate password length (8+ chars)
// 6. Validate password confirmation matches
// 7. Validate terms checkbox is checked
//
// Success Flow:
// 1. Call userService.Register() to create user
// 2. Call userService.Login() to create session
// 3. Set session cookie
// 4. Redirect to return_to URL or /dashboard
//
// Error Flow:
// 1. Re-render form with:
//    - Original form values (except password)
//    - Validation error messages
//    - Flash message for service errors (e.g., email already exists)
//
// Implementation Notes:
// - Never log passwords, even on error
// - Clear password fields on error (don't re-populate)
// - Normalize email to lowercase
// - Trim whitespace from all fields
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderRegisterError(w, r, nil, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
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
	formValues := map[string]string{
		"Name":       name,
		"Email":      email,
		"InviteCode": inviteCode,
	}

	// TODO: Validate CSRF token
	// csrfToken := r.FormValue("csrf_token")
	// if !validateCSRFToken(r, csrfToken) {
	//     h.renderRegisterError(w, r, formValues, nil, &Flash{
	//         Type:    "error",
	//         Message: "Invalid security token. Please try again.",
	//     })
	//     return
	// }

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
		h.renderRegisterError(w, r, formValues, errors, nil)
		return
	}

	// Call UserService.Register
	user, err := h.userService.Register(r.Context(), domain.RegisterParams{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		// Handle specific error types
		code := domain.ErrorCode(err)
		switch code {
		case domain.ECONFLICT:
			// Email already exists
			errors["email"] = "An account with this email already exists"
			h.renderRegisterError(w, r, formValues, errors, nil)
		case domain.EINVALID:
			// Validation error from service
			h.renderRegisterError(w, r, formValues, nil, &Flash{
				Type:    "error",
				Message: domain.ErrorMessage(err),
			})
		default:
			// Internal error - log and show generic message
			h.logger.Error("registration failed", "error", err, "email", email)
			h.renderRegisterError(w, r, formValues, nil, &Flash{
				Type:    "error",
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
		// Registration succeeded but login failed - redirect to login page
		h.logger.Error("auto-login after registration failed", "error", err, "email", email)
		http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
		return
	}

	// Set session cookie
	setSessionCookie(w, loginResult.Token, h.isSecure)

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

// renderRegisterError re-renders the registration form with errors.
func (h *AuthHandler) renderRegisterError(
	w http.ResponseWriter,
	r *http.Request,
	formValues map[string]string,
	errors map[string]string,
	flash *Flash,
) {
	if formValues == nil {
		formValues = make(map[string]string)
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := AuthPageData{
		CurrentPath:        "/register",
		CSRFToken:          "", // TODO: Regenerate CSRF token
		Form:               formValues,
		Errors:             errors,
		Flash:              flash,
		ReturnTo:           r.FormValue("return_to"),
		InviteCodesEnabled: h.inviteValidator.IsEnabled(),
	}

	h.renderer.RenderHTTP(w, "auth/register", data)
}

// =============================================================================
// GET /login - Show Login Form
// =============================================================================

// ShowLogin renders the login form.
//
// Template: auth/login
//
// Query Parameters:
// - return_to (optional): URL to redirect to after successful login
// - registered (optional): If "1", show success message for new registration
// - reset (optional): If "1", show success message for password reset
//
// Template Data:
// - CurrentPath: "/login"
// - CSRFToken: Token for form protection
// - Form: Empty map (no pre-filled values)
// - Errors: Empty map (no validation errors)
// - Flash: Success message if registered=1 or reset=1
// - ReturnTo: Value from query param if present
//
// Implementation Notes:
// - If user is already logged in, redirect to dashboard (or return_to)
// - Generate CSRF token
// - Check for success query params to show appropriate flash message
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: Check if user is already authenticated
	// user := middleware.GetUser(r.Context())
	// if user != nil {
	//     returnTo := r.URL.Query().Get("return_to")
	//     if returnTo != "" && isSafeRedirectURL(returnTo) {
	//         http.Redirect(w, r, returnTo, http.StatusSeeOther)
	//         return
	//     }
	//     http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	//     return
	// }

	// Check for success query params
	var flash *Flash
	if r.URL.Query().Get("registered") == "1" {
		flash = &Flash{
			Type:    "success",
			Message: "Account created successfully! Please sign in.",
		}
	} else if r.URL.Query().Get("reset") == "1" {
		flash = &Flash{
			Type:    "success",
			Message: "Password reset successfully! Please sign in with your new password.",
		}
	} else if r.URL.Query().Get("logout") == "1" {
		flash = &Flash{
			Type:    "success",
			Message: "You have been signed out.",
		}
	}

	// Get return_to from query params
	returnTo := r.URL.Query().Get("return_to")

	data := AuthPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "", // TODO: Set CSRF token
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       flash,
		ReturnTo:    returnTo,
	}

	h.renderer.RenderHTTP(w, "auth/login", data)
}

// =============================================================================
// POST /login - Process Login
// =============================================================================

// Login processes the login form submission.
//
// Form Fields:
// - email (required): User's email address
// - password (required): User's password
// - remember-me (optional): Checkbox for extended session (not implemented yet)
// - return_to (optional): URL to redirect to after successful login
//
// Validation:
// 1. Parse form data
// 2. Validate CSRF token
// 3. Validate email is present
// 4. Validate password is present
//
// Success Flow:
// 1. Call userService.Login() to authenticate and create session
// 2. Set session cookie
// 3. Redirect to return_to URL or /dashboard
//
// Error Flow:
// 1. Re-render form with:
//    - Email value preserved
//    - Generic error message (don't reveal if email exists)
//
// Security Notes:
// - Always use generic error message: "Invalid email or password"
// - Do NOT reveal whether email exists in database
// - Clear password field on error
// - Consider rate limiting failed login attempts (future enhancement)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderLoginError(w, r, nil, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Extract form values
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	returnTo := r.FormValue("return_to")

	// Store form values for re-rendering (except password)
	formValues := map[string]string{
		"Email": email,
	}

	// TODO: Validate CSRF token
	// csrfToken := r.FormValue("csrf_token")
	// if !validateCSRFToken(r, csrfToken) {
	//     h.renderLoginError(w, r, formValues, nil, &Flash{
	//         Type:    "error",
	//         Message: "Invalid security token. Please try again.",
	//     })
	//     return
	// }

	// Basic validation
	errors := make(map[string]string)

	if email == "" {
		errors["email"] = "Email is required"
	}

	if password == "" {
		errors["password"] = "Password is required"
	}

	if len(errors) > 0 {
		h.renderLoginError(w, r, formValues, errors, nil)
		return
	}

	// Call UserService.Login
	loginResult, err := h.userService.Login(r.Context(), email, password)
	if err != nil {
		// Handle specific error types
		code := domain.ErrorCode(err)
		switch code {
		case domain.EUNAUTHORIZED:
			// Invalid credentials - use generic message
			h.renderLoginError(w, r, formValues, nil, &Flash{
				Type:    "error",
				Message: "Invalid email or password",
			})
		default:
			// Internal error - log and show generic message
			h.logger.Error("login failed", "error", err, "email", email)
			h.renderLoginError(w, r, formValues, nil, &Flash{
				Type:    "error",
				Message: "Login failed. Please try again later.",
			})
		}
		return
	}

	// Set session cookie
	setSessionCookie(w, loginResult.Token, h.isSecure)

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
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// renderLoginError re-renders the login form with errors.
func (h *AuthHandler) renderLoginError(
	w http.ResponseWriter,
	r *http.Request,
	formValues map[string]string,
	errors map[string]string,
	flash *Flash,
) {
	if formValues == nil {
		formValues = make(map[string]string)
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := AuthPageData{
		CurrentPath: "/login",
		CSRFToken:   "", // TODO: Regenerate CSRF token
		Form:        formValues,
		Errors:      errors,
		Flash:       flash,
		ReturnTo:    r.FormValue("return_to"),
	}

	h.renderer.RenderHTTP(w, "auth/login", data)
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
	cookie, err := r.Cookie(sessionCookieName)
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
		Name:     sessionCookieName,
		Value:    token,
		Path:     sessionCookiePath,
		MaxAge:   sessionCookieMaxAge,
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
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
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
	if !strings.Contains(domainPart, ".") {
		return false
	}

	return true
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
// Route Registration Helper
// =============================================================================

// RegisterRoutes registers all auth routes on the provided ServeMux.
//
// Routes registered:
// - GET  /register            -> ShowRegister
// - POST /register            -> Register
// - GET  /login               -> ShowLogin
// - POST /login               -> Login
// - POST /logout              -> Logout
// - GET  /verify-email        -> ShowVerifyEmail (email verification link handler)
// - POST /resend-verification -> ResendVerification (request new verification email)
//
// Usage in main.go:
//
//	authHandler := handler.NewAuthHandler(userService, renderer, logger, isSecure)
//	authHandler.RegisterRoutes(mux)
//
// Or manually:
//
//	mux.HandleFunc("GET /register", authHandler.ShowRegister)
//	mux.HandleFunc("POST /register", authHandler.Register)
//	mux.HandleFunc("GET /login", authHandler.ShowLogin)
//	mux.HandleFunc("POST /login", authHandler.Login)
//	mux.HandleFunc("POST /logout", authHandler.Logout)
//	mux.HandleFunc("GET /verify-email", authHandler.ShowVerifyEmail)
//	mux.HandleFunc("POST /resend-verification", authHandler.ResendVerification)
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /register", h.ShowRegister)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("GET /login", h.ShowLogin)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /logout", h.Logout)

	// Email verification routes (P0-004, P0-005)
	mux.HandleFunc("GET /verify-email", h.ShowVerifyEmail)
	mux.HandleFunc("GET /resend-verification", h.ShowResendVerification)
	mux.HandleFunc("POST /resend-verification", h.ResendVerification)

	// Password reset routes (P0-006)
	mux.HandleFunc("GET /forgot-password", h.ShowForgotPassword)
	mux.HandleFunc("POST /forgot-password", h.ForgotPassword)
	mux.HandleFunc("GET /reset-password", h.ShowResetPassword)
	mux.HandleFunc("POST /reset-password", h.ResetPassword)
}

// =============================================================================
// GET /verify-email - Verify Email Token
// =============================================================================

// ShowVerifyEmail handles the email verification link click.
//
// Query Parameters:
// - token (required): The raw verification token from the email link
//
// Template: auth/verify_email (shows success or error message)
//
// Flow:
// 1. Extract token from query string
// 2. Call userService.VerifyEmail(token)
// 3. On success: Show success message with link to login
// 4. On error: Show error message with option to resend
//
// Error Scenarios:
// - Missing token -> "Invalid verification link"
// - Invalid/expired token -> "Verification link expired"
// - Already verified -> "Email already verified"
//
// Implementation Notes:
// - This is a GET handler because email links should be clickable
// - The token is validated server-side, so GET is safe
// - Consider rate limiting to prevent token enumeration
func (h *AuthHandler) ShowVerifyEmail(w http.ResponseWriter, r *http.Request) {
	// 1. Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderVerifyEmailError(w, r, "Invalid verification link. Please check your email for the correct link.")
		return
	}

	// 2. Call UserService.VerifyEmail
	err := h.userService.VerifyEmail(r.Context(), token)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			h.renderVerifyEmailError(w, r, "This verification link has expired or is invalid. Please request a new verification email.")
		case domain.ECONFLICT:
			h.renderVerifyEmailSuccess(w, r, "Your email is already verified. You can sign in to your account.")
		default:
			h.logger.Error("email verification failed", "error", err)
			h.renderVerifyEmailError(w, r, "Verification failed. Please try again later.")
		}
		return
	}

	// 3. Success
	h.renderVerifyEmailSuccess(w, r, "Your email has been verified! You can now sign in to your account.")
}

// VerifyEmailPageData contains data for the verify email template.
type VerifyEmailPageData struct {
	CurrentPath string
	Success     bool   // true = verification succeeded, false = error
	Message     string // Success or error message
	CanResend   bool   // Show resend verification option
	Flash       *Flash // Required by auth layout
}

// renderVerifyEmailSuccess renders the verify email page with a success message.
func (h *AuthHandler) renderVerifyEmailSuccess(w http.ResponseWriter, r *http.Request, message string) {
	data := VerifyEmailPageData{
		CurrentPath: r.URL.Path,
		Success:     true,
		Message:     message,
		CanResend:   false,
	}
	h.renderer.RenderHTTP(w, "auth/verify_email", data)
}

// renderVerifyEmailError renders the verify email page with an error message.
func (h *AuthHandler) renderVerifyEmailError(w http.ResponseWriter, r *http.Request, message string) {
	data := VerifyEmailPageData{
		CurrentPath: r.URL.Path,
		Success:     false,
		Message:     message,
		CanResend:   true, // Allow user to request new verification email
	}
	h.renderer.RenderHTTP(w, "auth/verify_email", data)
}

// =============================================================================
// GET /resend-verification - Show Resend Verification Form
// =============================================================================

// ResendVerificationPageData contains data for the resend verification template.
type ResendVerificationPageData struct {
	CurrentPath string
	CSRFToken   string
	Form        map[string]string
	Errors      map[string]string
	Flash       *Flash
	Success     bool
	Message     string
}

// ShowResendVerification renders the resend verification form.
func (h *AuthHandler) ShowResendVerification(w http.ResponseWriter, r *http.Request) {
	data := ResendVerificationPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "",
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       nil,
		Success:     false,
	}
	h.renderer.RenderHTTP(w, "auth/resend_verification", data)
}

// =============================================================================
// POST /resend-verification - Request New Verification Email
// =============================================================================

// ResendVerification handles requests to resend the verification email.
//
// Form Fields:
// - email (required): The email address to send verification to
//
// Flow:
// 1. Parse and validate email
// 2. Call userService.ResendVerificationEmail(email)
// 3. Send email if token created successfully
// 4. Always show success message (don't reveal if email exists)
//
// Security Notes:
// - Always show success message regardless of whether email exists
// - This prevents email enumeration attacks
// - Consider rate limiting to prevent abuse
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	// 1. Parse form
	if err := r.ParseForm(); err != nil {
		h.renderResendVerificationError(w, r, "Invalid form submission")
		return
	}

	// 2. Get email
	emailAddr := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if emailAddr == "" {
		h.renderResendVerificationError(w, r, "Email is required")
		return
	}

	// 3. Call UserService.ResendVerificationEmail
	result, err := h.userService.ResendVerificationEmail(r.Context(), emailAddr)
	if err != nil {
		// Log error but show generic success (prevents enumeration)
		h.logger.Debug("resend verification failed", "error", err, "email", emailAddr)
	} else {
		// Get user name for the email (we need to look it up)
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

	// 4. Always show success (don't reveal if email exists)
	h.renderResendVerificationSuccess(w, r)
}

// renderResendVerificationSuccess renders success message (always shown to prevent enumeration).
func (h *AuthHandler) renderResendVerificationSuccess(w http.ResponseWriter, r *http.Request) {
	data := ResendVerificationPageData{
		CurrentPath: r.URL.Path,
		Success:     true,
		Message:     "If an account exists with that email, a verification link has been sent.",
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
	}
	h.renderer.RenderHTTP(w, "auth/resend_verification", data)
}

// renderResendVerificationError renders error message for resend failures.
func (h *AuthHandler) renderResendVerificationError(w http.ResponseWriter, r *http.Request, message string) {
	data := ResendVerificationPageData{
		CurrentPath: r.URL.Path,
		Success:     false,
		Message:     message,
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash: &Flash{
			Type:    "error",
			Message: message,
		},
	}
	h.renderer.RenderHTTP(w, "auth/resend_verification", data)
}

// =============================================================================
// GET /forgot-password - Show Forgot Password Form
// =============================================================================

// ForgotPasswordPageData contains data for the forgot password template.
type ForgotPasswordPageData struct {
	CurrentPath string
	CSRFToken   string
	Form        map[string]string
	Errors      map[string]string
	Flash       *Flash
}

// ShowForgotPassword renders the forgot password form.
//
// Template: auth/forgot_password
//
// Template Data:
// - CurrentPath: "/forgot-password"
// - CSRFToken: Token for form protection
// - Form: Empty map (no pre-filled values)
// - Errors: Empty map (no validation errors)
// - Flash: nil (no flash message)
func (h *AuthHandler) ShowForgotPassword(w http.ResponseWriter, r *http.Request) {
	data := ForgotPasswordPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "", // TODO: Set CSRF token
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "auth/forgot_password", data)
}

// =============================================================================
// POST /forgot-password - Process Forgot Password Request
// =============================================================================

// ForgotPassword processes the forgot password form submission.
//
// Form Fields:
// - email (required): User's email address
//
// Flow:
// 1. Parse and validate email
// 2. Create password reset token (if user exists)
// 3. Send password reset email (if user exists)
// 4. Always show success message (don't reveal if email exists)
//
// Security Notes:
// - Always show success message regardless of whether email exists
// - This prevents email enumeration attacks
// - Consider rate limiting to prevent abuse
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderForgotPasswordError(w, r, nil, nil, &Flash{
			Type:    "error",
			Message: "Invalid form submission. Please try again.",
		})
		return
	}

	// Get email
	emailAddr := strings.ToLower(strings.TrimSpace(r.FormValue("email")))

	// Store form values for re-rendering
	formValues := map[string]string{
		"Email": emailAddr,
	}

	// Basic validation
	if emailAddr == "" {
		h.renderForgotPasswordError(w, r, formValues, map[string]string{
			"email": "Email is required",
		}, nil)
		return
	}

	if !isValidEmail(emailAddr) {
		h.renderForgotPasswordError(w, r, formValues, map[string]string{
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
	h.renderer.RenderHTTP(w, "auth/forgot_password_sent", map[string]interface{}{
		"CurrentPath": r.URL.Path,
	})
}

// renderForgotPasswordError re-renders the forgot password form with errors.
func (h *AuthHandler) renderForgotPasswordError(
	w http.ResponseWriter,
	r *http.Request,
	formValues map[string]string,
	errors map[string]string,
	flash *Flash,
) {
	if formValues == nil {
		formValues = make(map[string]string)
	}
	if errors == nil {
		errors = make(map[string]string)
	}

	data := ForgotPasswordPageData{
		CurrentPath: "/forgot-password",
		CSRFToken:   "", // TODO: Regenerate CSRF token
		Form:        formValues,
		Errors:      errors,
		Flash:       flash,
	}

	h.renderer.RenderHTTP(w, "auth/forgot_password", data)
}

// =============================================================================
// GET /reset-password - Show Reset Password Form
// =============================================================================

// ResetPasswordPageData contains data for the reset password template.
type ResetPasswordPageData struct {
	CurrentPath string
	CSRFToken   string
	Token       string // The reset token from URL
	Form        map[string]string
	Errors      map[string]string
	Flash       *Flash
}

// ShowResetPassword renders the reset password form.
//
// Query Parameters:
// - token (required): The password reset token from the email link
//
// Template: auth/reset_password (if token is valid)
// Template: auth/reset_password_invalid (if token is invalid/expired)
//
// Flow:
// 1. Extract token from query string
// 2. Validate the token
// 3. If valid: show reset password form
// 4. If invalid: show error message with link to request new reset
func (h *AuthHandler) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	// Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderResetPasswordInvalid(w, r, "Invalid reset link. Please check your email for the correct link.")
		return
	}

	// Validate the token
	_, err := h.userService.ValidatePasswordResetToken(r.Context(), token)
	if err != nil {
		code := domain.ErrorCode(err)
		switch code {
		case domain.ENOTFOUND:
			h.renderResetPasswordInvalid(w, r, "This reset link has expired or is invalid. Please request a new password reset.")
		case domain.EINVALID:
			h.renderResetPasswordInvalid(w, r, "This reset link is invalid. Please request a new password reset.")
		default:
			h.logger.Error("password reset token validation failed", "error", err)
			h.renderResetPasswordInvalid(w, r, "Something went wrong. Please request a new password reset.")
		}
		return
	}

	// Token is valid - show reset form
	data := ResetPasswordPageData{
		CurrentPath: r.URL.Path,
		CSRFToken:   "", // TODO: Set CSRF token
		Token:       token,
		Form:        make(map[string]string),
		Errors:      make(map[string]string),
		Flash:       nil,
	}

	h.renderer.RenderHTTP(w, "auth/reset_password", data)
}

// renderResetPasswordInvalid renders the invalid token page.
func (h *AuthHandler) renderResetPasswordInvalid(w http.ResponseWriter, r *http.Request, message string) {
	data := map[string]interface{}{
		"CurrentPath": r.URL.Path,
		"Message":     message,
	}
	h.renderer.RenderHTTP(w, "auth/reset_password_invalid", data)
}

// =============================================================================
// POST /reset-password - Process Password Reset
// =============================================================================

// ResetPassword processes the password reset form submission.
//
// Form Fields:
// - token (required): The password reset token (hidden field)
// - password (required): New password (min 8 chars)
// - password_confirmation (required): Must match password
//
// Flow:
// 1. Parse and validate form data
// 2. Validate passwords match and meet requirements
// 3. Call userService.ResetPassword to update password
// 4. Redirect to login with success message
//
// Security Notes:
// - Token is re-validated during the reset operation
// - All existing sessions are invalidated after password change
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		h.renderResetPasswordInvalid(w, r, "Invalid form submission. Please try again.")
		return
	}

	// Get form values
	token := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirmation := r.FormValue("password_confirmation")

	// Validate token is present
	if token == "" {
		h.renderResetPasswordInvalid(w, r, "Invalid reset link. Please request a new password reset.")
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
		data := ResetPasswordPageData{
			CurrentPath: "/reset-password",
			CSRFToken:   "", // TODO: Regenerate CSRF token
			Token:       token,
			Form:        make(map[string]string),
			Errors:      errors,
			Flash:       nil,
		}
		h.renderer.RenderHTTP(w, "auth/reset_password", data)
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
			h.renderResetPasswordInvalid(w, r, "This reset link has expired or is invalid. Please request a new password reset.")
		case domain.EINVALID:
			// Password validation error from service
			data := ResetPasswordPageData{
				CurrentPath: "/reset-password",
				CSRFToken:   "",
				Token:       token,
				Form:        make(map[string]string),
				Errors:      make(map[string]string),
				Flash: &Flash{
					Type:    "error",
					Message: domain.ErrorMessage(err),
				},
			}
			h.renderer.RenderHTTP(w, "auth/reset_password", data)
		default:
			h.logger.Error("password reset failed", "error", err)
			h.renderResetPasswordInvalid(w, r, "Something went wrong. Please try again.")
		}
		return
	}

	// Success - redirect to login with success message
	h.logger.Info("password reset completed")
	http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
}

// =============================================================================
// CSRF Token Generation (To Be Implemented)
// =============================================================================

/*
CSRF Protection Approach - Double-Submit Cookie Pattern

The double-submit cookie pattern is recommended for this application because:
1. It doesn't require server-side session storage for the token
2. It works well with the existing cookie-based session management
3. It's simpler to implement than synchronizer token pattern

Implementation:

1. On GET requests (ShowLogin, ShowRegister):
   - Generate a random 32-byte token
   - Set it in a cookie (csrf_token) that is NOT HttpOnly (JS needs to read it)
   - Also pass it to the template to embed in forms

2. On POST requests (Login, Register):
   - Read the csrf_token from the cookie
   - Read the csrf_token from the form body
   - Compare them - they must match

3. Token generation:

   func generateCSRFToken() string {
       b := make([]byte, 32)
       if _, err := rand.Read(b); err != nil {
           panic(err) // crypto/rand failure is catastrophic
       }
       return base64.URLEncoding.EncodeToString(b)
   }

4. Token validation:

   func validateCSRFToken(r *http.Request, formToken string) bool {
       cookie, err := r.Cookie("csrf_token")
       if err != nil {
           return false
       }
       return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) == 1
   }

5. Cookie settings:

   http.SetCookie(w, &http.Cookie{
       Name:     "csrf_token",
       Value:    csrfToken,
       Path:     "/",
       MaxAge:   3600, // 1 hour
       HttpOnly: false, // Must be readable for form submission
       Secure:   isSecure,
       SameSite: http.SameSiteStrictMode, // Strict for CSRF tokens
   })

Note: SameSite=Lax on the session cookie already provides significant CSRF
protection for modern browsers. The explicit CSRF token is defense-in-depth
and supports older browsers.
*/

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
