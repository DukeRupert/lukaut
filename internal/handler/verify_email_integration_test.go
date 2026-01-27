package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/middleware"
	"github.com/DukeRupert/lukaut/internal/session"
	"github.com/google/uuid"
)

// TestRouteEnforcement tests that middleware correctly enforces
// email verification and authentication requirements on routes.

// testUserService is a minimal mock implementing service.UserService for integration tests.
type testUserService struct {
	getBySessionTokenFunc func(ctx context.Context, token string) (*domain.User, error)
}

func (m *testUserService) Register(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) Login(ctx context.Context, email, password string) (*domain.LoginResult, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) Logout(ctx context.Context, token string) error { return nil }
func (m *testUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) GetBySessionToken(ctx context.Context, token string) (*domain.User, error) {
	if m.getBySessionTokenFunc != nil {
		return m.getBySessionTokenFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}
func (m *testUserService) UpdateProfile(ctx context.Context, params domain.ProfileUpdateParams) error {
	return nil
}
func (m *testUserService) UpdateBusinessProfile(ctx context.Context, params domain.BusinessProfileUpdateParams) error {
	return nil
}
func (m *testUserService) ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error {
	return nil
}
func (m *testUserService) DeleteExpiredSessions(ctx context.Context) error { return nil }
func (m *testUserService) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) VerifyEmail(ctx context.Context, token string) error {
	return errors.New("not implemented")
}
func (m *testUserService) ResendVerificationEmail(ctx context.Context, email string) (*domain.EmailVerificationResult, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) DeleteExpiredEmailVerificationTokens(ctx context.Context) error {
	return nil
}
func (m *testUserService) CreatePasswordResetToken(ctx context.Context, email string) (*domain.PasswordResetResult, error) {
	return nil, errors.New("not implemented")
}
func (m *testUserService) ValidatePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not implemented")
}
func (m *testUserService) ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error {
	return errors.New("not implemented")
}
func (m *testUserService) DeleteExpiredPasswordResetTokens(ctx context.Context) error { return nil }
func (m *testUserService) UpdateStripeCustomer(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error {
	return nil
}
func (m *testUserService) UpdateSubscription(ctx context.Context, userID uuid.UUID, status, tier, subscriptionID string) error {
	return nil
}
func (m *testUserService) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func setupMux(mock *testUserService) (*http.ServeMux, func(http.Handler) http.Handler, func(http.Handler) http.Handler) {
	logger := testLogger()
	authMw := middleware.NewAuthMiddleware(mock, logger, false)

	requireUser := middleware.Stack(authMw.WithUser, authMw.RequireUser)
	requireVerified := middleware.Stack(authMw.WithUser, authMw.RequireUser, authMw.RequireEmailVerified)

	mux := http.NewServeMux()
	return mux, requireUser, requireVerified
}

func dummyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func TestRouteEnforcement_UnverifiedUser_DashboardRedirects(t *testing.T) {
	mock := &testUserService{
		getBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:            uuid.New(),
				Email:         "test@example.com",
				Name:          "Test User",
				EmailVerified: false,
			}, nil
		},
	}

	mux, _, requireVerified := setupMux(mock)
	mux.Handle("GET /dashboard", requireVerified(dummyHandler()))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/verify-email-reminder" {
		t.Errorf("Location = %q, want /verify-email-reminder", loc)
	}
}

func TestRouteEnforcement_VerifiedUser_DashboardOK(t *testing.T) {
	mock := &testUserService{
		getBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:            uuid.New(),
				Email:         "test@example.com",
				Name:          "Test User",
				EmailVerified: true,
			}, nil
		},
	}

	mux, _, requireVerified := setupMux(mock)
	mux.Handle("GET /dashboard", requireVerified(dummyHandler()))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRouteEnforcement_UnverifiedUser_SettingsAllowed(t *testing.T) {
	mock := &testUserService{
		getBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:            uuid.New(),
				Email:         "test@example.com",
				Name:          "Test User",
				EmailVerified: false,
			}, nil
		},
	}

	mux, requireUser, _ := setupMux(mock)
	mux.Handle("GET /settings", requireUser(dummyHandler()))

	req := httptest.NewRequest("GET", "/settings", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRouteEnforcement_UnverifiedUser_VerifyEmailReminderNoLoop(t *testing.T) {
	mock := &testUserService{
		getBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:            uuid.New(),
				Email:         "test@example.com",
				Name:          "Test User",
				EmailVerified: false,
			}, nil
		},
	}

	mux, requireUser, _ := setupMux(mock)
	// verify-email-reminder uses requireUser (no email verification check)
	mux.Handle("GET /verify-email-reminder", requireUser(dummyHandler()))

	req := httptest.NewRequest("GET", "/verify-email-reminder", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d (should not redirect)", rec.Code, http.StatusOK)
	}
}

func TestRouteEnforcement_Unauthenticated_DashboardRedirectsToLogin(t *testing.T) {
	mock := &testUserService{
		getBySessionTokenFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return nil, domain.Unauthorized("test", "invalid session")
		},
	}

	mux, _, requireVerified := setupMux(mock)
	mux.Handle("GET /dashboard", requireVerified(dummyHandler()))

	// No session cookie — unauthenticated
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	loc := rec.Header().Get("Location")
	if len(loc) < 6 || loc[:6] != "/login" {
		t.Errorf("Location = %q, want prefix /login", loc)
	}
}
