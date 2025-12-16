package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// Worker manages background job processing with concurrent workers.
type Worker struct {
	db       *sql.DB
	queries  *repository.Queries
	handlers map[string]JobHandler
	config   Config
	logger   *slog.Logger

	// Synchronization
	wg     sync.WaitGroup
	stopCh chan struct{}
}

// New creates a new Worker with the given configuration.
// The worker must be started with Start() and stopped with Stop().
func New(db *sql.DB, queries *repository.Queries, config Config, logger *slog.Logger) (*Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Worker{
		db:       db,
		queries:  queries,
		handlers: make(map[string]JobHandler),
		config:   config,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}, nil
}

// Register adds a job handler to the worker.
// The handler's Type() must be unique. Call this before Start().
func (w *Worker) Register(handler JobHandler) {
	jobType := handler.Type()
	if _, exists := w.handlers[jobType]; exists {
		w.logger.Warn("Overwriting existing handler", "job_type", jobType)
	}
	w.handlers[jobType] = handler
	w.logger.Debug("Registered job handler", "job_type", jobType)
}

// Start begins processing jobs with the configured number of concurrent workers.
// It also recovers any stale jobs from previous worker crashes.
func (w *Worker) Start(ctx context.Context) {
	// Recover stale jobs from crashed workers
	if err := w.recoverStaleJobs(ctx); err != nil {
		w.logger.Error("Failed to recover stale jobs", "error", err)
	}

	// Start worker goroutines
	for i := 0; i < w.config.Concurrency; i++ {
		w.wg.Add(1)
		go w.runWorker(ctx, i+1)
	}

	w.logger.Info("Worker started", "concurrency", w.config.Concurrency)
}

// Stop signals all workers to stop and waits for them to finish.
// It respects the configured ShutdownTimeout.
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker...")
	close(w.stopCh)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("Worker stopped gracefully")
	case <-time.After(w.config.ShutdownTimeout):
		w.logger.Warn("Worker shutdown timeout exceeded, some jobs may still be running")
	}
}

// recoverStaleJobs finds jobs that have been running too long and resets them to pending.
// This handles the case where a worker crashed while processing a job.
func (w *Worker) recoverStaleJobs(ctx context.Context) error {
	thresholdSeconds := w.config.StaleJobThreshold.Seconds()
	count, err := w.queries.RecoverStaleJobs(ctx, thresholdSeconds)
	if err != nil {
		return fmt.Errorf("recover stale jobs: %w", err)
	}

	if count > 0 {
		w.logger.Warn("Recovered stale jobs", "count", count, "threshold", w.config.StaleJobThreshold)
	}

	return nil
}

// runWorker is the main loop for a worker goroutine.
// It continuously polls for jobs until stopCh is closed.
func (w *Worker) runWorker(ctx context.Context, workerID int) {
	defer w.wg.Done()

	logger := w.logger.With("worker_id", workerID)
	logger.Debug("Worker started")

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			logger.Debug("Worker stopping")
			return
		case <-ticker.C:
			if err := w.processNextJob(ctx, logger); err != nil {
				if err == sql.ErrNoRows {
					// No jobs available, this is normal
					continue
				}
				logger.Error("Failed to process job", "error", err)
			}
		}
	}
}

// processNextJob attempts to dequeue and execute a single job.
// Returns sql.ErrNoRows if no jobs are available.
func (w *Worker) processNextJob(ctx context.Context, logger *slog.Logger) error {
	// Start a transaction for dequeuing
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	// Dequeue the next job
	job, err := qtx.DequeueJob(ctx)
	if err != nil {
		return err // Will be sql.ErrNoRows if no jobs available
	}

	// Mark the job as running
	if err := qtx.UpdateJobStarted(ctx, job.ID); err != nil {
		return fmt.Errorf("mark job started: %w", err)
	}

	// Commit the dequeue transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit dequeue: %w", err)
	}

	// Execute the job (outside the transaction)
	logger = logger.With("job_id", job.ID, "job_type", job.JobType, "attempt", job.Attempts+1)
	logger.Info("Processing job")

	if err := w.executeJob(ctx, job, logger); err != nil {
		logger.Error("Job failed", "error", err)
		w.markJobFailed(ctx, job.ID, err)
		return fmt.Errorf("execute job: %w", err)
	}

	logger.Info("Job completed")
	if err := w.markJobCompleted(ctx, job.ID); err != nil {
		logger.Error("Failed to mark job as completed", "error", err)
		return err
	}

	return nil
}

// executeJob runs the appropriate handler for the job with a timeout context.
func (w *Worker) executeJob(ctx context.Context, job repository.Job, logger *slog.Logger) error {
	// Find the handler for this job type
	handler, ok := w.handlers[job.JobType]
	if !ok {
		// No handler registered - this is a permanent error
		return NewPermanentError(fmt.Errorf("no handler registered for job type: %s", job.JobType))
	}

	// Create a context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, w.config.JobTimeout)
	defer cancel()

	// Execute the handler
	if err := handler.Handle(jobCtx, job.Payload); err != nil {
		return err
	}

	return nil
}

// markJobCompleted marks a job as successfully completed.
func (w *Worker) markJobCompleted(ctx context.Context, jobID uuid.UUID) error {
	if err := w.queries.UpdateJobCompleted(ctx, jobID); err != nil {
		return fmt.Errorf("update job completed: %w", err)
	}
	return nil
}

// markJobFailed marks a job as failed.
// If the error is permanent or max attempts reached, the job is marked as 'failed'.
// Otherwise, it's rescheduled with exponential backoff.
func (w *Worker) markJobFailed(ctx context.Context, jobID uuid.UUID, jobErr error) {
	errorMessage := jobErr.Error()

	// Check if this is a permanent error
	if IsPermanent(jobErr) {
		w.logger.Warn("Job failed with permanent error, will not retry", "job_id", jobID, "error", errorMessage)
	}

	params := repository.UpdateJobFailedParams{
		ID: jobID,
		ErrorMessage: sql.NullString{
			String: errorMessage,
			Valid:  true,
		},
	}

	if err := w.queries.UpdateJobFailed(ctx, params); err != nil {
		w.logger.Error("Failed to mark job as failed", "job_id", jobID, "error", err)
	}
}
