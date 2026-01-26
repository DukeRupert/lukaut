package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/invite"
	"github.com/DukeRupert/lukaut/internal/session"
	"github.com/google/uuid"
)

// =============================================================================
// Mock UserService Implementation
// =============================================================================

// mockUserService implements the service.UserService interface for testing.
type mockUserService struct {
	RegisterFunc                             func(ctx context.Context, params domain.RegisterParams) (*domain.User, error)
	LoginFunc                                func(ctx context.Context, email, password string) (*domain.LoginResult, error)
	LogoutFunc                               func(ctx context.Context, token string) error
	GetByIDFunc                              func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetBySessionTokenFunc                    func(ctx context.Context, token string) (*domain.User, error)
	UpdateProfileFunc                        func(ctx context.Context, params domain.ProfileUpdateParams) error
	UpdateBusinessProfileFunc                func(ctx context.Context, params domain.BusinessProfileUpdateParams) error
	ChangePasswordFunc                       func(ctx context.Context, params domain.PasswordChangeParams) error
	DeleteExpiredSessionsFunc                func(ctx context.Context) error
	CreateEmailVerificationTokenFunc         func(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error)
	VerifyEmailFunc                          func(ctx context.Context, token string) error
	ResendVerificationEmailFunc              func(ctx context.Context, email string) (*domain.EmailVerificationResult, error)
	DeleteExpiredEmailVerificationTokensFunc func(ctx context.Context) error
	CreatePasswordResetTokenFunc             func(ctx context.Context, email string) (*domain.PasswordResetResult, error)
	ValidatePasswordResetTokenFunc           func(ctx context.Context, token string) (uuid.UUID, error)
	ResetPasswordFunc                        func(ctx context.Context, params domain.ResetPasswordParams) error
	DeleteExpiredPasswordResetTokensFunc     func(ctx context.Context) error
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

func (m *mockUserService) UpdateBusinessProfile(ctx context.Context, params domain.BusinessProfileUpdateParams) error {
	if m.UpdateBusinessProfileFunc != nil {
		return m.UpdateBusinessProfileFunc(ctx, params)
	}
	return errors.New("UpdateBusinessProfileFunc not implemented")
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
// Mock Email Service Implementation
// =============================================================================

// mockEmailService implements the email.EmailService interface for testing.
type mockEmailService struct {
	SendVerificationEmailFunc   func(ctx context.Context, to, name, token string) error
	SendPasswordResetEmailFunc  func(ctx context.Context, to, name, token string) error
	SendReportReadyEmailFunc    func(ctx context.Context, to, name, reportURL string) error
	SendReportToClientEmailFunc func(ctx context.Context, to, inspectorName, inspectorCompany, siteName, reportURL string) error
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, to, name, token string) error {
	if m.SendVerificationEmailFunc != nil {
		return m.SendVerificationEmailFunc(ctx, to, name, token)
	}
	return nil // Default: no-op for tests
}

func (m *mockEmailService) SendPasswordResetEmail(ctx context.Context, to, name, token string) error {
	if m.SendPasswordResetEmailFunc != nil {
		return m.SendPasswordResetEmailFunc(ctx, to, name, token)
	}
	return nil
}

func (m *mockEmailService) SendReportReadyEmail(ctx context.Context, to, name, reportURL string) error {
	if m.SendReportReadyEmailFunc != nil {
		return m.SendReportReadyEmailFunc(ctx, to, name, reportURL)
	}
	return nil
}

func (m *mockEmailService) SendReportToClientEmail(ctx context.Context, to, inspectorName, inspectorCompany, siteName, reportURL string) error {
	if m.SendReportToClientEmailFunc != nil {
		return m.SendReportToClientEmailFunc(ctx, to, inspectorName, inspectorCompany, siteName, reportURL)
	}
	return nil
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
func newTestAuthHandler(mock *mockUserService) *AuthHandler {
	// Create a disabled invite validator for tests (no invite code required)
	inviteValidator := invite.New(false, nil)
	return NewAuthHandler(mock, &mockEmailService{}, inviteValidator, newTestLogger(), false)
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

	handler := newTestAuthHandler(mock)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
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
		if cookie.Name == session.CookieName {
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

	handler := newTestAuthHandler(mock)

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
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
