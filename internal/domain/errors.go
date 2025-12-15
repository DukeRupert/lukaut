package domain

import (
	"errors"
	"fmt"
)

// Application error codes
const (
	EINVALID      = "invalid"       // Invalid input or validation failure
	EUNAUTHORIZED = "unauthorized"  // Authentication required
	EFORBIDDEN    = "forbidden"     // Permission denied
	ENOTFOUND     = "not_found"     // Resource not found
	ECONFLICT     = "conflict"      // Resource conflict (e.g., duplicate)
	EGONE         = "gone"          // Resource no longer available
	ETOOLARGE     = "too_large"     // Request entity too large
	ERATELIMIT    = "rate_limit"    // Rate limit exceeded
	EINTERNAL     = "internal"      // Internal server error
	ENOTIMPL      = "not_impl"      // Not implemented
	EPAYMENT      = "payment"       // Payment required
)

// Error represents an application error with structured information.
type Error struct {
	Code    string // Machine-readable error code
	Op      string // Operation that failed (e.g., "user.create")
	Message string // Human-readable message
	Err     error  // Underlying error
}

func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Errorf creates a new Error with the given code, operation, and formatted message.
func Errorf(code, op, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, code, op, message string) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// ErrorCode returns the code of the root error, or EINTERNAL if none.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

// ErrorMessage returns the human-readable message of the error.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		// For internal errors, return generic message
		if e.Code == EINTERNAL {
			return "An internal error occurred. Please try again later."
		}
		return e.Message
	}
	return "An internal error occurred. Please try again later."
}

// ErrorOp returns the operation of the root error, if any.
func ErrorOp(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Op
	}
	return ""
}

// Convenience constructors for common error types

// NotFound creates a not found error.
func NotFound(op, resource, id string) *Error {
	return &Error{
		Code:    ENOTFOUND,
		Op:      op,
		Message: fmt.Sprintf("%s with ID %q not found", resource, id),
	}
}

// Invalid creates a validation error.
func Invalid(op, message string) *Error {
	return &Error{
		Code:    EINVALID,
		Op:      op,
		Message: message,
	}
}

// Unauthorized creates an authentication error.
func Unauthorized(op, message string) *Error {
	return &Error{
		Code:    EUNAUTHORIZED,
		Op:      op,
		Message: message,
	}
}

// Forbidden creates a permission error.
func Forbidden(op, message string) *Error {
	return &Error{
		Code:    EFORBIDDEN,
		Op:      op,
		Message: message,
	}
}

// Conflict creates a conflict error.
func Conflict(op, message string) *Error {
	return &Error{
		Code:    ECONFLICT,
		Op:      op,
		Message: message,
	}
}

// Internal creates an internal error, wrapping the underlying error.
func Internal(err error, op, message string) *Error {
	return &Error{
		Code:    EINTERNAL,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// RateLimit creates a rate limit error.
func RateLimit(op string) *Error {
	return &Error{
		Code:    ERATELIMIT,
		Op:      op,
		Message: "Too many requests. Please try again later.",
	}
}

// ValidationError represents field-level validation errors.
type ValidationError struct {
	Op     string
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: validation failed", e.Op)
}

// NewValidationError creates a new validation error with the first field error.
func NewValidationError(op, field, message string) *ValidationError {
	return &ValidationError{
		Op: op,
		Fields: map[string]string{
			field: message,
		},
	}
}

// AddFieldError adds a field error to an existing validation error.
// If err is not a ValidationError, returns a new one.
func AddFieldError(err error, field, message string) *ValidationError {
	var ve *ValidationError
	if errors.As(err, &ve) {
		ve.Fields[field] = message
		return ve
	}
	return NewValidationError("", field, message)
}
