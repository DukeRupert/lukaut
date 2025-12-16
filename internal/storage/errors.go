package storage

import (
	"errors"
	"fmt"
)

// =============================================================================
// Sentinel Errors
// =============================================================================

var (
	// ErrNotFound is returned when a requested object doesn't exist.
	ErrNotFound = errors.New("object not found")

	// ErrKeyExists is returned when attempting to create an object at a key
	// that already exists (when overwrite is disabled).
	ErrKeyExists = errors.New("object already exists at this key")

	// ErrInvalidKey is returned when a storage key is invalid or contains
	// forbidden characters (e.g., path traversal attempts like "../").
	ErrInvalidKey = errors.New("invalid storage key")

	// ErrTooLarge is returned when an object exceeds the maximum allowed size.
	ErrTooLarge = errors.New("object exceeds maximum size")

	// ErrAccessDenied is returned when the storage provider denies access
	// to an object (insufficient permissions, ACL restrictions, etc.).
	ErrAccessDenied = errors.New("access denied")
)

// =============================================================================
// Structured Error Type
// =============================================================================

// StorageError wraps storage operation errors with additional context.
// It implements the error interface and supports errors.Unwrap for sentinel
// error checking with errors.Is().
type StorageError struct {
	// Op is the operation that failed (e.g., "Put", "Get", "Delete").
	Op string

	// Key is the storage key involved in the operation.
	Key string

	// Err is the underlying error that occurred.
	Err error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("storage %s %q: %v", e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("storage %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is() and errors.As().
func (e *StorageError) Unwrap() error {
	return e.Err
}

// =============================================================================
// Helper Functions
// =============================================================================

// IsNotFound returns true if the error indicates an object was not found.
// It unwraps the error chain to check for ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsKeyExists returns true if the error indicates a key already exists.
// It unwraps the error chain to check for ErrKeyExists.
func IsKeyExists(err error) bool {
	return errors.Is(err, ErrKeyExists)
}

// IsAccessDenied returns true if the error indicates access was denied.
// It unwraps the error chain to check for ErrAccessDenied.
func IsAccessDenied(err error) bool {
	return errors.Is(err, ErrAccessDenied)
}

// IsInvalidKey returns true if the error indicates an invalid storage key.
// It unwraps the error chain to check for ErrInvalidKey.
func IsInvalidKey(err error) bool {
	return errors.Is(err, ErrInvalidKey)
}

// IsTooLarge returns true if the error indicates an object was too large.
// It unwraps the error chain to check for ErrTooLarge.
func IsTooLarge(err error) bool {
	return errors.Is(err, ErrTooLarge)
}
