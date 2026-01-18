package metrics

import "time"

// JobStarted should be called when a job begins processing
func JobStarted(jobType string) {
	// Currently just a placeholder for future in-flight tracking
	// Could add a gauge for jobs_in_flight if needed
}

// JobCompleted records a successful job completion
func JobCompleted(jobType string, duration time.Duration) {
	JobsTotal.WithLabelValues(jobType, "completed").Inc()
	JobDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

// JobFailed records a job failure
func JobFailed(jobType string) {
	JobsTotal.WithLabelValues(jobType, "failed").Inc()
}

// JobRetried records a job retry attempt
func JobRetried(jobType string) {
	JobRetriesTotal.WithLabelValues(jobType).Inc()
}
