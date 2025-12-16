package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// Job type constants - these must match the JobHandler.Type() values
const (
	JobTypeAnalyzeInspection = "analyze_inspection"
	JobTypeGenerateReport    = "generate_report"
)

// Priority constants for job scheduling
const (
	PriorityLow    = 0
	PriorityNormal = 10
	PriorityHigh   = 20
)

// AnalyzeInspectionPayload is the payload for inspection analysis jobs.
type AnalyzeInspectionPayload struct {
	InspectionID uuid.UUID `json:"inspection_id"`
	UserID       uuid.UUID `json:"user_id"`
}

// GenerateReportPayload is the payload for report generation jobs.
type GenerateReportPayload struct {
	InspectionID uuid.UUID `json:"inspection_id"`
	UserID       uuid.UUID `json:"user_id"`
	Format       string    `json:"format"` // "pdf" or "docx"
}

// EnqueueOption is a functional option for customizing job enqueue parameters.
type EnqueueOption func(*repository.EnqueueJobParams)

// WithPriority sets the job priority.
func WithPriority(priority int32) EnqueueOption {
	return func(p *repository.EnqueueJobParams) {
		p.Priority = priority
	}
}

// WithMaxAttempts sets the maximum number of retry attempts.
func WithMaxAttempts(attempts int32) EnqueueOption {
	return func(p *repository.EnqueueJobParams) {
		p.MaxAttempts = attempts
	}
}

// WithDelay schedules the job to run after a delay.
func WithDelay(delay time.Duration) EnqueueOption {
	return func(p *repository.EnqueueJobParams) {
		p.ScheduledAt = time.Now().Add(delay)
	}
}

// EnqueueJob is a generic helper for enqueuing jobs with custom options.
func EnqueueJob(
	ctx context.Context,
	queries *repository.Queries,
	jobType string,
	payload interface{},
	opts ...EnqueueOption,
) (repository.Job, error) {
	// Marshal the payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return repository.Job{}, fmt.Errorf("marshal payload: %w", err)
	}

	// Default parameters
	params := repository.EnqueueJobParams{
		JobType:     jobType,
		Payload:     payloadJSON,
		Priority:    PriorityNormal,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(&params)
	}

	// Enqueue the job
	job, err := queries.EnqueueJob(ctx, params)
	if err != nil {
		return repository.Job{}, fmt.Errorf("enqueue job: %w", err)
	}

	return job, nil
}

// EnqueueAnalyzeInspection enqueues a job to analyze an inspection's images.
// This is typically called after images are uploaded to an inspection.
func EnqueueAnalyzeInspection(
	ctx context.Context,
	queries *repository.Queries,
	inspectionID uuid.UUID,
	userID uuid.UUID,
	opts ...EnqueueOption,
) (repository.Job, error) {
	payload := AnalyzeInspectionPayload{
		InspectionID: inspectionID,
		UserID:       userID,
	}

	return EnqueueJob(ctx, queries, JobTypeAnalyzeInspection, payload, opts...)
}

// EnqueueGenerateReport enqueues a job to generate a report for an inspection.
// The format should be "pdf" or "docx".
func EnqueueGenerateReport(
	ctx context.Context,
	queries *repository.Queries,
	inspectionID uuid.UUID,
	userID uuid.UUID,
	format string,
	opts ...EnqueueOption,
) (repository.Job, error) {
	payload := GenerateReportPayload{
		InspectionID: inspectionID,
		UserID:       userID,
		Format:       format,
	}

	return EnqueueJob(ctx, queries, JobTypeGenerateReport, payload, opts...)
}
