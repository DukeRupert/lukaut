package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/google/uuid"
)

// =============================================================================
// Mock UserService Implementation
// =============================================================================

// mockUserService implements the service.UserService interface for testing.
type mockUserService struct {
	RegisterFunc                           func(ctx context.Context, params domain.RegisterParams) (*domain.User, error)
	LoginFunc                              func(ctx context.Context, email, password string) (*domain.LoginResult, error)
	LogoutFunc                             func(ctx context.Context, token string) error
	GetByIDFunc                            func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetBySessionTokenFunc                  func(ctx context.Context, token string) (*domain.User, error)
	UpdateProfileFunc                      func(ctx context.Context, params domain.ProfileUpdateParams) error
	ChangePasswordFunc                     func(ctx context.Context, params domain.PasswordChangeParams) error
	DeleteExpiredSessionsFunc              func(ctx context.Context) error
	CreateEmailVerificationTokenFunc       func(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error)
	VerifyEmailFunc                        func(ctx context.Context, token string) error
	ResendVerificationEmailFunc            func(ctx context.Context, email string) (*domain.EmailVerificationResult, error)
	DeleteExpiredEmailVerificationTokensFunc func(ctx context.Context) error
	CreatePasswordResetTokenFunc           func(ctx context.Context, email string) (*domain.PasswordResetResult, error)
	ValidatePasswordResetTokenFunc         func(ctx context.Context, token string) (uuid.UUID, error)
	ResetPasswordFunc                      func(ctx context.Context, params domain.ResetPasswordParams) error
	DeleteExpiredPasswordResetTokensFunc   func(ctx context.Context) error
}

func (m *mockUserService) Register(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, params)
	}
	return nil, errors.New("RegisterFunc not implemented")
}

func (m *mockUserService) Login(ctx context.Context, email, password string) (*domain.LoginResult, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password)
	}
	return nil, errors.New("LoginFunc not implemented")
}

func (m *mockUserService) Logout(ctx context.Context, token string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, token)
	}
	return nil
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("GetByIDFunc not implemented")
}

func (m *mockUserService) GetBySessionToken(ctx context.Context, token string) (*domain.User, error) {
	if m.GetBySessionTokenFunc != nil {
		return m.GetBySessionTokenFunc(ctx, token)
	}
	return nil, errors.New("GetBySessionTokenFunc not implemented")
}

func (m *mockUserService) UpdateProfile(ctx context.Context, params domain.ProfileUpdateParams) error {
	if m.UpdateProfileFunc != nil {
		return m.UpdateProfileFunc(ctx, params)
	}
	return errors.New("UpdateProfileFunc not implemented")
}

func (m *mockUserService) ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error {
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(ctx, params)
	}
	return errors.New("ChangePasswordFunc not implemented")
}

func (m *mockUserService) DeleteExpiredSessions(ctx context.Context) error {
	if m.DeleteExpiredSessionsFunc != nil {
		return m.DeleteExpiredSessionsFunc(ctx)
	}
	return errors.New("DeleteExpiredSessionsFunc not implemented")
}

func (m *mockUserService) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error) {
	if m.CreateEmailVerificationTokenFunc != nil {
		return m.CreateEmailVerificationTokenFunc(ctx, userID)
	}
	return nil, errors.New("CreateEmailVerificationTokenFunc not implemented")
}

func (m *mockUserService) VerifyEmail(ctx context.Context, token string) error {
	if m.VerifyEmailFunc != nil {
		return m.VerifyEmailFunc(ctx, token)
	}
	return errors.New("VerifyEmailFunc not implemented")
}

func (m *mockUserService) ResendVerificationEmail(ctx context.Context, email string) (*domain.EmailVerificationResult, error) {
	if m.ResendVerificationEmailFunc != nil {
		return m.ResendVerificationEmailFunc(ctx, email)
	}
	return nil, errors.New("ResendVerificationEmailFunc not implemented")
}

func (m *mockUserService) DeleteExpiredEmailVerificationTokens(ctx context.Context) error {
	if m.DeleteExpiredEmailVerificationTokensFunc != nil {
		return m.DeleteExpiredEmailVerificationTokensFunc(ctx)
	}
	return nil
}

func (m *mockUserService) CreatePasswordResetToken(ctx context.Context, email string) (*domain.PasswordResetResult, error) {
	if m.CreatePasswordResetTokenFunc != nil {
		return m.CreatePasswordResetTokenFunc(ctx, email)
	}
	return nil, errors.New("CreatePasswordResetTokenFunc not implemented")
}

func (m *mockUserService) ValidatePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	if m.ValidatePasswordResetTokenFunc != nil {
		return m.ValidatePasswordResetTokenFunc(ctx, token)
	}
	return uuid.Nil, errors.New("ValidatePasswordResetTokenFunc not implemented")
}

func (m *mockUserService) ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error {
	if m.ResetPasswordFunc != nil {
		return m.ResetPasswordFunc(ctx, params)
	}
	return errors.New("ResetPasswordFunc not implemented")
}

func (m *mockUserService) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	if m.DeleteExpiredPasswordResetTokensFunc != nil {
		return m.DeleteExpiredPasswordResetTokensFunc(ctx)
	}
	return nil
}

// =============================================================================
// Mock Renderer Implementation
// =============================================================================

// mockRenderer implements the Renderer interface for testing.
type mockRenderer struct {
	RenderHTTPFunc func(w http.ResponseWriter, templateName string, data interface{})
}

func (m *mockRenderer) RenderHTTP(w http.ResponseWriter, templateName string, data interface{}) {
	if m.RenderHTTPFunc != nil {
		m.RenderHTTPFunc(w, templateName, data)
	} else {
		// Default: write template name and status 200
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(templateName))
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

// newTestLogger creates a logger that discards output for testing.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors in tests
	}))
}

// newTestAuthHandler creates an AuthHandler with mock dependencies for testing.
func newTestAuthHandler(mock *mockUserService, renderer *mockRenderer) *AuthHandler {
	return NewAuthHandler(mock, renderer, newTestLogger(), false)
}

// =============================================================================
// ShowRegister Tests (P0)
// =============================================================================

func TestShowRegister_GET_RendersForm(t *testing.T) {
	mock := &mockUserService{}

	templateRendered := false
	var renderedTemplate string

	renderer := &mockRenderer{
		RenderHTTPFunc: func(w http.ResponseWriter, templateName string, data interface{}) {
			templateRendered = true
			renderedTemplate = templateName
			w.WriteHeader(http.StatusOK)
		},
	}

	handler := newTestAuthHandler(mock, renderer)

	req := httptest.NewRequest("GET", "/register", nil)
	rec := httptest.NewRecorder()

	handler.ShowRegister(rec, req)

	// Verify template was rendered
	if !templateRendered {
		t.Error("template was not rendered")
	}

	// Verify correct template
	if renderedTemplate != "auth/register" {
		t.Errorf("template = %q, want %q", renderedTemplate, "auth/register")
	}

	// Verify 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

// =============================================================================
// Register Tests (P0)
// =============================================================================

func TestRegister_POST_Success_RedirectsToDashboard(t *testing.T) {
	createdUser := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	loginResult := &domain.LoginResult{
		User:  createdUser,
		Token: "session-token-123",
	}

	mock := &mockUserService{
		RegisterFunc: func(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
			return createdUser, nil
		},
		LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
			return loginResult, nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	// Create form data
	form := url.Values{}
	form.Set("name", "Test User")
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("password_confirmation", "password123")
	form.Set("terms", "on")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Verify redirect to dashboard
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("Location = %q, want %q", location, "/dashboard")
	}
}

func TestRegister_POST_Success_SetsSessionCookie(t *testing.T) {
	createdUser := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	expectedToken := "session-token-abc123def456"

	loginResult := &domain.LoginResult{
		User:  createdUser,
		Token: expectedToken,
	}

	mock := &mockUserService{
		RegisterFunc: func(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
			return createdUser, nil
		},
		LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
			return loginResult, nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	// Create form data
	form := url.Values{}
	form.Set("name", "Test User")
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("password_confirmation", "password123")
	form.Set("terms", "on")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Verify session cookie is set
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}

	if sessionCookie.Value != expectedToken {
		t.Errorf("cookie value = %q, want %q", sessionCookie.Value, expectedToken)
	}

	// Verify HttpOnly flag
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should have HttpOnly flag")
	}
}

func TestRegister_POST_DuplicateEmail_RerendersForm(t *testing.T) {
	mock := &mockUserService{
		RegisterFunc: func(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
			return nil, domain.Conflict("test", "Email already registered")
		},
	}

	templateRendered := false
	var renderedData AuthPageData

	renderer := &mockRenderer{
		RenderHTTPFunc: func(w http.ResponseWriter, templateName string, data interface{}) {
			templateRendered = true
			if pageData, ok := data.(AuthPageData); ok {
				renderedData = pageData
			}
			w.WriteHeader(http.StatusOK)
		},
	}

	handler := newTestAuthHandler(mock, renderer)

	// Create form data with duplicate email
	form := url.Values{}
	form.Set("name", "Test User")
	form.Set("email", "existing@example.com")
	form.Set("password", "password123")
	form.Set("password_confirmation", "password123")
	form.Set("terms", "on")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Verify form was re-rendered (not redirected)
	if !templateRendered {
		t.Error("template was not rendered")
	}

	// Verify error is set for email field
	if renderedData.Errors == nil {
		t.Fatal("expected errors, got nil")
	}

	emailError, ok := renderedData.Errors["email"]
	if !ok {
		t.Error("expected error for email field")
	}

	if !strings.Contains(strings.ToLower(emailError), "already") {
		t.Errorf("email error = %q, should mention 'already'", emailError)
	}

	// Verify form values are preserved (except password)
	if renderedData.Form["Email"] != "existing@example.com" {
		t.Errorf("form email = %q, want %q", renderedData.Form["Email"], "existing@example.com")
	}

	if renderedData.Form["Name"] != "Test User" {
		t.Errorf("form name = %q, want %q", renderedData.Form["Name"], "Test User")
	}
}

// =============================================================================
// ShowLogin Tests (P0)
// =============================================================================

func TestShowLogin_GET_RendersForm(t *testing.T) {
	mock := &mockUserService{}

	templateRendered := false
	var renderedTemplate string

	renderer := &mockRenderer{
		RenderHTTPFunc: func(w http.ResponseWriter, templateName string, data interface{}) {
			templateRendered = true
			renderedTemplate = templateName
			w.WriteHeader(http.StatusOK)
		},
	}

	handler := newTestAuthHandler(mock, renderer)

	req := httptest.NewRequest("GET", "/login", nil)
	rec := httptest.NewRecorder()

	handler.ShowLogin(rec, req)

	// Verify template was rendered
	if !templateRendered {
		t.Error("template was not rendered")
	}

	// Verify correct template
	if renderedTemplate != "auth/login" {
		t.Errorf("template = %q, want %q", renderedTemplate, "auth/login")
	}

	// Verify 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

// =============================================================================
// Login Tests (P0)
// =============================================================================

func TestLogin_POST_Success_RedirectsToDashboard(t *testing.T) {
	loginResult := &domain.LoginResult{
		User: &domain.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test User",
		},
		Token: "session-token-123",
	}

	mock := &mockUserService{
		LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
			return loginResult, nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	// Create form data
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	// Verify redirect to dashboard
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("Location = %q, want %q", location, "/dashboard")
	}
}

func TestLogin_POST_Success_SetsSessionCookie(t *testing.T) {
	expectedToken := "session-token-abc123def456"

	loginResult := &domain.LoginResult{
		User: &domain.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test User",
		},
		Token: expectedToken,
	}

	mock := &mockUserService{
		LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
			return loginResult, nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	// Create form data
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	// Verify session cookie is set
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}

	if sessionCookie.Value != expectedToken {
		t.Errorf("cookie value = %q, want %q", sessionCookie.Value, expectedToken)
	}

	// Verify HttpOnly flag
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should have HttpOnly flag")
	}

	// Verify SameSite
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie SameSite = %v, want %v", sessionCookie.SameSite, http.SameSiteLaxMode)
	}
}

func TestLogin_POST_InvalidCredentials_RerendersForm(t *testing.T) {
	mock := &mockUserService{
		LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
			return nil, domain.Unauthorized("test", "Invalid email or password")
		},
	}

	templateRendered := false
	var renderedData AuthPageData

	renderer := &mockRenderer{
		RenderHTTPFunc: func(w http.ResponseWriter, templateName string, data interface{}) {
			templateRendered = true
			if pageData, ok := data.(AuthPageData); ok {
				renderedData = pageData
			}
			w.WriteHeader(http.StatusOK)
		},
	}

	handler := newTestAuthHandler(mock, renderer)

	// Create form data with invalid credentials
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	// Verify form was re-rendered (not redirected)
	if !templateRendered {
		t.Error("template was not rendered")
	}

	// Verify flash message is set
	if renderedData.Flash == nil {
		t.Fatal("expected flash message, got nil")
	}

	if renderedData.Flash.Type != "error" {
		t.Errorf("flash type = %q, want %q", renderedData.Flash.Type, "error")
	}

	// Verify email is preserved in form
	if renderedData.Form["Email"] != "test@example.com" {
		t.Errorf("form email = %q, want %q", renderedData.Form["Email"], "test@example.com")
	}
}

func TestLogin_POST_GenericErrorMessage(t *testing.T) {
	tests := []struct {
		name  string
		email string
		error error
	}{
		{
			name:  "invalid password",
			email: "test@example.com",
			error: domain.Unauthorized("test", "Invalid email or password"),
		},
		{
			name:  "nonexistent email",
			email: "nonexistent@example.com",
			error: domain.Unauthorized("test", "Invalid email or password"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserService{
				LoginFunc: func(ctx context.Context, email, password string) (*domain.LoginResult, error) {
					return nil, tt.error
				},
			}

			var renderedData AuthPageData

			renderer := &mockRenderer{
				RenderHTTPFunc: func(w http.ResponseWriter, templateName string, data interface{}) {
					if pageData, ok := data.(AuthPageData); ok {
						renderedData = pageData
					}
					w.WriteHeader(http.StatusOK)
				},
			}

			handler := newTestAuthHandler(mock, renderer)

			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", "password123")

			req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			// Verify generic error message
			if renderedData.Flash == nil {
				t.Fatal("expected flash message, got nil")
			}

			expectedMsg := "Invalid email or password"
			if renderedData.Flash.Message != expectedMsg {
				t.Errorf("flash message = %q, want %q", renderedData.Flash.Message, expectedMsg)
			}
		})
	}
}

// =============================================================================
// Logout Tests (P0)
// =============================================================================

func TestLogout_POST_ClearsCookie(t *testing.T) {
	logoutCalled := false

	mock := &mockUserService{
		LogoutFunc: func(ctx context.Context, token string) error {
			logoutCalled = true
			return nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "session-token-123",
	})
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	// Verify logout was called
	if !logoutCalled {
		t.Error("logout service method was not called")
	}

	// Verify cookie is cleared (MaxAge=-1)
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("session cookie not found in response")
	}

	if sessionCookie.MaxAge != -1 {
		t.Errorf("cookie MaxAge = %d, want -1 (deleted)", sessionCookie.MaxAge)
	}
}

func TestLogout_POST_RedirectsToLogin(t *testing.T) {
	mock := &mockUserService{
		LogoutFunc: func(ctx context.Context, token string) error {
			return nil
		},
	}

	renderer := &mockRenderer{}
	handler := newTestAuthHandler(mock, renderer)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "session-token-123",
	})
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	// Verify redirect to login
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/login") {
		t.Errorf("Location = %q, want prefix /login", location)
	}
}

// =============================================================================
// isSafeRedirectURL Tests (P0)
// =============================================================================

func TestIsSafeRedirectURL_RelativeURLs_Safe(t *testing.T) {
	tests := []struct {
		name string
		url  string
		safe bool
	}{
		{"simple path", "/dashboard", true},
		{"path with query", "/settings?tab=profile", true},
		{"nested path", "/inspections/123/edit", true},
		{"root path", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeRedirectURL(tt.url)
			if result != tt.safe {
				t.Errorf("isSafeRedirectURL(%q) = %v, want %v", tt.url, result, tt.safe)
			}
		})
	}
}

func TestIsSafeRedirectURL_ProtocolRelativeURLs_Unsafe(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"protocol-relative", "//evil.com"},
		{"protocol-relative with path", "//evil.com/phishing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeRedirectURL(tt.url)
			if result {
				t.Errorf("isSafeRedirectURL(%q) = true, want false (unsafe)", tt.url)
			}
		})
	}
}

func TestIsSafeRedirectURL_AbsoluteURLs_Unsafe(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"http URL", "http://evil.com"},
		{"https URL", "https://evil.com"},
		{"ftp URL", "ftp://evil.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeRedirectURL(tt.url)
			if result {
				t.Errorf("isSafeRedirectURL(%q) = true, want false (unsafe)", tt.url)
			}
		})
	}
}

func TestIsSafeRedirectURL_JavaScriptURL_Unsafe(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<script>alert(1)</script>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeRedirectURL(tt.url)
			if result {
				t.Errorf("isSafeRedirectURL(%q) = true, want false (unsafe)", tt.url)
			}
		})
	}
}

func TestIsSafeRedirectURL_EmptyURL_Unsafe(t *testing.T) {
	result := isSafeRedirectURL("")
	if result {
		t.Error("isSafeRedirectURL(\"\") = true, want false")
	}
}

func TestIsSafeRedirectURL_NoLeadingSlash_Unsafe(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"relative without slash", "dashboard"},
		{"domain-like", "evil.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeRedirectURL(tt.url)
			if result {
				t.Errorf("isSafeRedirectURL(%q) = true, want false (unsafe)", tt.url)
			}
		})
	}
}
