package worker

import (
	"fmt"
	"time"
)

// Config holds the configuration for the background job worker.
type Config struct {
	// Concurrency is the number of worker goroutines to run in parallel.
	// Each goroutine polls for and processes jobs independently.
	// Default: 2
	Concurrency int

	// PollInterval is how often each worker checks for new jobs when idle.
	// Default: 5 seconds
	PollInterval time.Duration

	// JobTimeout is the maximum time a single job is allowed to run.
	// If a job exceeds this timeout, its context is canceled and it's marked as failed.
	// Default: 5 minutes
	JobTimeout time.Duration

	// ShutdownTimeout is how long to wait for running jobs to complete during graceful shutdown.
	// After this timeout, the worker stops even if jobs are still running.
	// Default: 30 seconds
	ShutdownTimeout time.Duration

	// StaleJobThreshold defines how old a 'running' job must be before it's considered stale.
	// Stale jobs are recovered on worker startup (likely from crashed workers).
	// Default: 10 minutes
	StaleJobThreshold time.Duration
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		Concurrency:       2,
		PollInterval:      5 * time.Second,
		JobTimeout:        5 * time.Minute,
		ShutdownTimeout:   30 * time.Second,
		StaleJobThreshold: 10 * time.Minute,
	}
}

// Validate checks if the configuration is valid.
// Returns an error if any values are invalid.
func (c Config) Validate() error {
	if c.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1, got %d", c.Concurrency)
	}
	if c.Concurrency > 100 {
		return fmt.Errorf("concurrency too high (max 100), got %d", c.Concurrency)
	}
	if c.PollInterval < 1*time.Second {
		return fmt.Errorf("poll interval must be at least 1 second, got %v", c.PollInterval)
	}
	if c.JobTimeout < 1*time.Second {
		return fmt.Errorf("job timeout must be at least 1 second, got %v", c.JobTimeout)
	}
	if c.ShutdownTimeout < 1*time.Second {
		return fmt.Errorf("shutdown timeout must be at least 1 second, got %v", c.ShutdownTimeout)
	}
	if c.StaleJobThreshold < 1*time.Minute {
		return fmt.Errorf("stale job threshold must be at least 1 minute, got %v", c.StaleJobThreshold)
	}
	return nil
}
