package handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DukeRupert/lukaut/internal/domain"
)

// =============================================================================
// Error Response Tests - Security Focus
// =============================================================================

func TestValidationErrorResponse_DoesNotExposeOperationName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a validation error with an internal operation name
	ve := domain.NewValidationError("UserService.Register", "email", "Email is required")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ValidationErrorResponse(w, r, logger, ve)
	})

	// Test HTML response (non-JSON)
	req := httptest.NewRequest("POST", "/register", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain internal operation names
	if strings.Contains(body, "UserService") {
		t.Errorf("response exposes internal operation name: %s", body)
	}
	if strings.Contains(body, "Register") {
		t.Errorf("response exposes internal method name: %s", body)
	}

	// Should have a user-friendly message
	if !strings.Contains(body, "Validation failed") {
		t.Errorf("response should contain user-friendly message, got: %s", body)
	}
	if !strings.Contains(body, "check your input") {
		t.Errorf("response should have helpful guidance, got: %s", body)
	}
}

func TestValidationErrorResponse_JSON_DoesNotExposeOperationName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a validation error with an internal operation name
	ve := domain.NewValidationError("InspectionService.Create", "title", "Title is required")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ValidationErrorResponse(w, r, logger, ve)
	})

	// Test JSON response
	req := httptest.NewRequest("POST", "/api/inspections", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain internal operation names
	if strings.Contains(body, "InspectionService") {
		t.Errorf("JSON response exposes internal operation name: %s", body)
	}

	// Should contain the field error
	if !strings.Contains(body, "title") {
		t.Errorf("JSON response should contain field name: %s", body)
	}
	if !strings.Contains(body, "Title is required") {
		t.Errorf("JSON response should contain field message: %s", body)
	}
}

func TestErrorResponse_InternalErrorHidesDetails(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create an internal error wrapping a database error
	dbErr := &mockDatabaseError{message: "pq: relation \"users\" does not exist"}
	internalErr := domain.Internal(dbErr, "UserRepository.GetByEmail", "Database query failed")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ErrorResponse(w, r, logger, internalErr)
	})

	// Test HTML response
	req := httptest.NewRequest("GET", "/users/123", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain database error details
	if strings.Contains(body, "pq:") {
		t.Errorf("response exposes database error: %s", body)
	}
	if strings.Contains(body, "relation") {
		t.Errorf("response exposes database schema: %s", body)
	}
	if strings.Contains(body, "UserRepository") {
		t.Errorf("response exposes internal operation: %s", body)
	}

	// Should return generic message
	if !strings.Contains(body, "internal error") {
		t.Errorf("response should contain generic internal error message, got: %s", body)
	}
}

func TestErrorResponse_InternalErrorHidesDetails_JSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create an internal error wrapping a sensitive error
	sensitiveErr := &mockDatabaseError{message: "connection to 192.168.1.100:5432 refused"}
	internalErr := domain.Internal(sensitiveErr, "DB.Connect", "Failed to connect")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ErrorResponse(w, r, logger, internalErr)
	})

	// Test JSON response
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain sensitive details
	if strings.Contains(body, "192.168") {
		t.Errorf("JSON response exposes IP address: %s", body)
	}
	if strings.Contains(body, "5432") {
		t.Errorf("JSON response exposes port number: %s", body)
	}
	if strings.Contains(body, "DB.Connect") {
		t.Errorf("JSON response exposes internal operation: %s", body)
	}

	// Should contain generic message
	if !strings.Contains(body, "internal error") {
		t.Errorf("JSON response should contain generic error, got: %s", body)
	}
}

func TestErrorResponse_NotFoundDoesNotExposeInternals(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a not found error
	notFoundErr := domain.NotFound("InspectionRepository.GetByID", "inspection", "550e8400-e29b-41d4-a716-446655440000")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ErrorResponse(w, r, logger, notFoundErr)
	})

	req := httptest.NewRequest("GET", "/inspections/550e8400-e29b-41d4-a716-446655440000", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain internal operation name
	if strings.Contains(body, "Repository") {
		t.Errorf("response exposes repository name: %s", body)
	}

	// Should contain user-friendly not found message (resource type is OK)
	if !strings.Contains(body, "inspection") && !strings.Contains(body, "not found") {
		t.Errorf("response should indicate resource not found: %s", body)
	}
}

func TestErrorResponse_UnwrappedErrorReturnsGeneric(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a raw error (not a domain.Error)
	rawErr := &mockDatabaseError{message: "FATAL: password authentication failed for user \"postgres\""}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ErrorResponse(w, r, logger, rawErr)
	})

	req := httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Should NOT contain the raw error
	if strings.Contains(body, "FATAL") {
		t.Errorf("response exposes raw error: %s", body)
	}
	if strings.Contains(body, "password") {
		t.Errorf("response exposes password-related error: %s", body)
	}
	if strings.Contains(body, "postgres") {
		t.Errorf("response exposes database user: %s", body)
	}

	// Should return generic message
	if !strings.Contains(body, "internal error") {
		t.Errorf("response should contain generic message, got: %s", body)
	}
}

// mockDatabaseError simulates a database error for testing
type mockDatabaseError struct {
	message string
}

func (e *mockDatabaseError) Error() string {
	return e.message
}
