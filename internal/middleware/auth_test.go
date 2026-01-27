package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/session"
	"github.com/google/uuid"
)

// =============================================================================
// Mock UserService Implementation
// =============================================================================

// mockUserService implements the service.UserService interface for testing.
type mockUserService struct {
	GetBySessionTokenFunc func(ctx context.Context, token string) (*domain.User, error)
	LogoutFunc            func(ctx context.Context, token string) error
}

func (m *mockUserService) Register(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) Login(ctx context.Context, email, password string) (*domain.LoginResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) Logout(ctx context.Context, token string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, token)
	}
	return nil
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) GetBySessionToken(ctx context.Context, token string) (*domain.User, error) {
	if m.GetBySessionTokenFunc != nil {
		return m.GetBySessionTokenFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserService) UpdateProfile(ctx context.Context, params domain.ProfileUpdateParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) UpdateBusinessProfile(ctx context.Context, params domain.BusinessProfileUpdateParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) DeleteExpiredSessions(ctx context.Context) error {
	return errors.New("not implemented")
}

func (m *mockUserService) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) VerifyEmail(ctx context.Context, token string) error {
	return errors.New("not implemented")
}

func (m *mockUserService) ResendVerificationEmail(ctx context.Context, email string) (*domain.EmailVerificationResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) DeleteExpiredEmailVerificationTokens(ctx context.Context) error {
	return nil
}

func (m *mockUserService) CreatePasswordResetToken(ctx context.Context, email string) (*domain.PasswordResetResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserService) ValidatePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not implemented")
}

func (m *mockUserService) ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error {
	return errors.New("not implemented")
}

func (m *mockUserService) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	return nil
}

func (m *mockUserService) UpdateStripeCustomer(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error {
	return nil
}

func (m *mockUserService) UpdateSubscription(ctx context.Context, userID uuid.UUID, status, tier, subscriptionID string) error {
	return nil
}

func (m *mockUserService) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.User, error) {
	return nil, errors.New("not implemented")
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

// newTestAuthMiddleware creates an AuthMiddleware with mock service for testing.
func newTestAuthMiddleware(mock *mockUserService) *AuthMiddleware {
	return NewAuthMiddleware(mock, newTestLogger(), false)
}

// =============================================================================
// WithUser Middleware Tests (P0)
// =============================================================================

func TestWithUser_NoCookie_ContinuesWithoutUser(t *testing.T) {
	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify user is nil
		user := GetUser(r.Context())
		if user != nil {
			t.Errorf("expected nil user, got %+v", user)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create request without session cookie
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Wrap handler with middleware
	wrappedHandler := mw.WithUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("handler was not called")
	}

	// Verify response is successful
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWithUser_ValidCookie_SetsUserInContext(t *testing.T) {
	expectedUser := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	mock := &mockUserService{
		GetBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			if token != "valid-token-123" {
				t.Errorf("GetBySessionToken called with token = %q, want %q", token, "valid-token-123")
			}
			return expectedUser, nil
		},
	}

	mw := newTestAuthMiddleware(mock)

	var capturedUser *domain.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture user from context
		capturedUser = GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Create request with valid session cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "valid-token-123",
	})
	rec := httptest.NewRecorder()

	// Wrap handler with middleware
	wrappedHandler := mw.WithUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify user was set in context
	if capturedUser == nil {
		t.Fatal("user not set in context")
	}

	if capturedUser.ID != expectedUser.ID {
		t.Errorf("user.ID = %v, want %v", capturedUser.ID, expectedUser.ID)
	}

	if capturedUser.Email != expectedUser.Email {
		t.Errorf("user.Email = %q, want %q", capturedUser.Email, expectedUser.Email)
	}
}

func TestWithUser_InvalidCookie_ClearsAndContinues(t *testing.T) {
	mock := &mockUserService{
		GetBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			// Return unauthorized error for invalid token
			return nil, domain.Unauthorized("test", "invalid session")
		},
	}

	mw := newTestAuthMiddleware(mock)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify user is nil
		user := GetUser(r.Context())
		if user != nil {
			t.Errorf("expected nil user, got %+v", user)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create request with invalid session cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "invalid-token",
	})
	rec := httptest.NewRecorder()

	// Wrap handler with middleware
	wrappedHandler := mw.WithUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("handler was not called")
	}

	// Verify cookie was cleared (MaxAge=-1)
	cookies := rec.Result().Cookies()
	cookieCleared := false
	for _, cookie := range cookies {
		if cookie.Name == session.CookieName {
			if cookie.MaxAge == -1 {
				cookieCleared = true
			}
		}
	}

	if !cookieCleared {
		t.Error("invalid session cookie was not cleared")
	}
}

func TestWithUser_ContextPropagation(t *testing.T) {
	expectedUser := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	mock := &mockUserService{
		GetBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return expectedUser, nil
		},
	}

	mw := newTestAuthMiddleware(mock)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use GetUser helper to retrieve user
		user := GetUser(r.Context())

		if user == nil {
			t.Error("GetUser returned nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if user.ID != expectedUser.ID {
			t.Errorf("user.ID = %v, want %v", user.ID, expectedUser.ID)
		}

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "valid-token",
	})
	rec := httptest.NewRecorder()

	wrappedHandler := mw.WithUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

// =============================================================================
// RequireUser Middleware Tests (P0)
// =============================================================================

func TestRequireUser_WithUser_ContinuesToHandler(t *testing.T) {
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request with user in context
	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := auth.SetUser(req.Context(), user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Wrap handler with RequireUser middleware
	wrappedHandler := mw.RequireUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("handler was not called")
	}

	// Verify response is successful
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireUser_NoUser_HTMLRequest_Redirects(t *testing.T) {
	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
		w.WriteHeader(http.StatusOK)
	})

	// Create HTML request without user in context
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	// Wrap handler with RequireUser middleware
	wrappedHandler := mw.RequireUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify redirect to login
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/login") {
		t.Errorf("Location header = %q, want prefix /login", location)
	}

	// Verify return_to parameter is set
	if !strings.Contains(location, "return_to=") {
		t.Error("Location should include return_to parameter")
	}
}

func TestRequireUser_NoUser_APIRequest_Returns401(t *testing.T) {
	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
		w.WriteHeader(http.StatusOK)
	})

	// Create API request (Accept: application/json) without user
	req := httptest.NewRequest("GET", "/api/inspections", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	// Wrap handler with RequireUser middleware
	wrappedHandler := mw.RequireUser(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify 401 Unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	// Verify JSON response
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

func TestRequireUser_HandlerNotCalled_WhenNoUser(t *testing.T) {
	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name   string
		accept string
	}{
		{"HTML request", "text/html"},
		{"API request", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept", tt.accept)
			rec := httptest.NewRecorder()

			wrappedHandler := mw.RequireUser(handler)
			wrappedHandler.ServeHTTP(rec, req)

			if handlerCalled {
				t.Error("handler should not be called when user is not authenticated")
			}
		})
	}
}

// =============================================================================
// RequireEmailVerified Middleware Tests (P0)
// =============================================================================

func TestRequireEmailVerified_Verified_Continues(t *testing.T) {
	user := &domain.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Name:          "Test User",
		EmailVerified: true,
	}

	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request with verified user in context
	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := auth.SetUser(req.Context(), user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Wrap handler with RequireEmailVerified middleware
	wrappedHandler := mw.RequireEmailVerified(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("handler was not called")
	}

	// Verify response is successful
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireEmailVerified_NotVerified_Redirects(t *testing.T) {
	user := &domain.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Name:          "Test User",
		EmailVerified: false, // Not verified
	}

	mock := &mockUserService{}
	mw := newTestAuthMiddleware(mock)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
		w.WriteHeader(http.StatusOK)
	})

	// Create HTML request with unverified user
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("Accept", "text/html")
	ctx := auth.SetUser(req.Context(), user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Wrap handler with RequireEmailVerified middleware
	wrappedHandler := mw.RequireEmailVerified(handler)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify redirect to verification reminder
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/verify-email-reminder" {
		t.Errorf("Location header = %q, want /verify-email-reminder", location)
	}
}

// =============================================================================
// RequireActiveSubscription Middleware Tests (P0)
// =============================================================================

func TestRequireActiveSubscription_Active_Continues(t *testing.T) {
	tests := []struct {
		name   string
		status domain.SubscriptionStatus
	}{
		{"active subscription", domain.SubscriptionStatusActive},
		{"trialing subscription", domain.SubscriptionStatusTrialing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &domain.User{
				ID:                 uuid.New(),
				Email:              "test@example.com",
				Name:               "Test User",
				SubscriptionStatus: tt.status,
			}

			mock := &mockUserService{}
			mw := newTestAuthMiddleware(mock)

			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/dashboard", nil)
			ctx := auth.SetUser(req.Context(), user)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			wrappedHandler := mw.RequireActiveSubscription(handler)
			wrappedHandler.ServeHTTP(rec, req)

			if !handlerCalled {
				t.Error("handler was not called")
			}

			if rec.Code != http.StatusOK {
				t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

// =============================================================================
// Cookie Tests (P0)
// =============================================================================

func TestSetSessionCookie_HttpOnlyFlag(t *testing.T) {
	rec := httptest.NewRecorder()

	SetSessionCookie(rec, "test-token", false)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}

	sessionCookie := cookies[0]
	if !sessionCookie.HttpOnly {
		t.Error("HttpOnly flag should be true")
	}
}

func TestSetSessionCookie_SameSite(t *testing.T) {
	rec := httptest.NewRecorder()

	SetSessionCookie(rec, "test-token", false)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}

	sessionCookie := cookies[0]
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want %v", sessionCookie.SameSite, http.SameSiteLaxMode)
	}
}

// =============================================================================
// RequireAdmin Tests
// =============================================================================

func TestRequireAdmin_AdminUser_Continues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := &mockUserService{}

	// Create middleware with admin emails
	authMw := NewAuthMiddleware(mockService, logger, false).WithAdminEmails([]string{"admin@example.com", "boss@example.com"})

	// Create a test user with admin email
	adminUser := &domain.User{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Name:  "Admin User",
	}

	// Create a test handler that sets a flag when called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request with admin user in context
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctx := auth.SetUser(req.Context(), adminUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Execute middleware
	authMw.RequireAdmin(testHandler).ServeHTTP(rec, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("expected handler to be called for admin user")
	}

	// Verify response status
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireAdmin_NonAdminUser_HTMLRequest_Returns403(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := &mockUserService{}

	// Create middleware with admin emails
	authMw := NewAuthMiddleware(mockService, logger, false).WithAdminEmails([]string{"admin@example.com"})

	// Create a test user with non-admin email
	regularUser := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Name:  "Regular User",
	}

	// Create a test handler that should NOT be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create HTML request with regular user in context
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Accept", "text/html")
	ctx := auth.SetUser(req.Context(), regularUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Execute middleware
	authMw.RequireAdmin(testHandler).ServeHTTP(rec, req)

	// Verify handler was NOT called
	if handlerCalled {
		t.Error("expected handler NOT to be called for non-admin user")
	}

	// Verify response status is 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
	}

	// Verify response body contains forbidden message
	body := rec.Body.String()
	if !strings.Contains(body, "403") && !strings.Contains(body, "Forbidden") {
		t.Errorf("response body should contain 403 or Forbidden, got: %s", body)
	}
}

func TestRequireAdmin_NonAdminUser_APIRequest_Returns403JSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := &mockUserService{}

	// Create middleware with admin emails
	authMw := NewAuthMiddleware(mockService, logger, false).WithAdminEmails([]string{"admin@example.com"})

	// Create a test user with non-admin email
	regularUser := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Name:  "Regular User",
	}

	// Create a test handler that should NOT be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create API request with regular user in context
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Accept", "application/json")
	ctx := auth.SetUser(req.Context(), regularUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Execute middleware
	authMw.RequireAdmin(testHandler).ServeHTTP(rec, req)

	// Verify handler was NOT called
	if handlerCalled {
		t.Error("expected handler NOT to be called for non-admin user")
	}

	// Verify response status is 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireAdmin_NoUser_RedirectsToLogin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := &mockUserService{}

	// Create middleware with admin emails
	authMw := NewAuthMiddleware(mockService, logger, false).WithAdminEmails([]string{"admin@example.com"})

	// Create a test handler that should NOT be called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request without user in context
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	// Execute middleware
	authMw.RequireAdmin(testHandler).ServeHTTP(rec, req)

	// Verify handler was NOT called
	if handlerCalled {
		t.Error("expected handler NOT to be called when user is not in context")
	}

	// Verify redirect to login
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("location header = %q, want %q", location, "/login")
	}
}

func TestRequireAdmin_CaseInsensitiveEmail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockService := &mockUserService{}

	// Create middleware with admin emails in mixed case
	authMw := NewAuthMiddleware(mockService, logger, false).WithAdminEmails([]string{"Admin@Example.Com"})

	// Create a test user with lowercase email
	adminUser := &domain.User{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Name:  "Admin User",
	}

	// Create a test handler that sets a flag when called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request with admin user in context
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctx := auth.SetUser(req.Context(), adminUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Execute middleware
	authMw.RequireAdmin(testHandler).ServeHTTP(rec, req)

	// Verify handler was called (case-insensitive match should work)
	if !handlerCalled {
		t.Error("expected handler to be called for admin user (case-insensitive)")
	}

	// Verify response status
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}
