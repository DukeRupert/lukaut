package worker

import (
	"context"
	"errors"
)

// JobHandler defines the interface that all job handlers must implement.
// Each handler is responsible for executing a specific type of background job.
type JobHandler interface {
	// Type returns the job type identifier that this handler processes.
	// This must match the job_type column in the jobs table.
	Type() string

	// Handle executes the job with the given payload.
	// The payload is raw JSON from the database and must be unmarshaled by the handler.
	// Returns an error if the job fails. Use NewPermanentError to mark a job as
	// permanently failed (no retries).
	Handle(ctx context.Context, payload []byte) error
}

// PermanentError wraps an error to indicate it should not be retried.
// Jobs that fail with a PermanentError are immediately marked as 'failed'
// instead of being rescheduled for retry.
type PermanentError struct {
	Err error
}

// Error implements the error interface.
func (e *PermanentError) Error() string {
	return e.Err.Error()
}

// Unwrap allows errors.Is and errors.As to work with PermanentError.
func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewPermanentError creates a new PermanentError that wraps the given error.
// Use this to indicate that a job should not be retried.
func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

// IsPermanent checks if an error is a PermanentError.
// Returns true if the error (or any error it wraps) is a PermanentError.
func IsPermanent(err error) bool {
	var permErr *PermanentError
	return errors.As(err, &permErr)
}
