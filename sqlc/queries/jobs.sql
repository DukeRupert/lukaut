-- name: EnqueueJob :one
INSERT INTO jobs (
    job_type,
    payload,
    priority,
    max_attempts,
    scheduled_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: DequeueJob :one
SELECT * FROM jobs
WHERE status = 'pending'
AND scheduled_at <= NOW()
ORDER BY priority DESC, scheduled_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: UpdateJobStarted :exec
UPDATE jobs
SET status = 'running',
    started_at = NOW(),
    attempts = attempts + 1
WHERE id = $1;

-- name: UpdateJobCompleted :exec
UPDATE jobs
SET status = 'completed',
    completed_at = NOW()
WHERE id = $1;

-- name: UpdateJobFailed :exec
-- Updates a failed job with exponential backoff (30s * 2^attempts, max 1 hour)
UPDATE jobs
SET status = CASE
    WHEN attempts >= max_attempts THEN 'failed'
    ELSE 'pending'
END,
error_message = $2,
scheduled_at = CASE
    WHEN attempts < max_attempts THEN NOW() + (LEAST(POWER(2, attempts - 1) * 30, 3600) * INTERVAL '1 second')
    ELSE scheduled_at
END
WHERE id = $1;

-- name: GetJobByID :one
SELECT * FROM jobs
WHERE id = $1;

-- name: DeleteCompletedJobsOlderThan :exec
DELETE FROM jobs
WHERE status = 'completed'
AND completed_at < $1;

-- name: RecoverStaleJobs :execrows
-- Recovers jobs that have been running too long (worker may have crashed)
-- $1 is the threshold in seconds (e.g., 600 for 10 minutes)
UPDATE jobs
SET status = 'pending',
    error_message = 'Job timed out - worker may have crashed'
WHERE status = 'running'
AND started_at < NOW() - make_interval(secs => $1);

-- name: HasPendingAnalysisJob :one
-- Check if there's a pending or running analysis job for this inspection
SELECT EXISTS (
    SELECT 1 FROM jobs
    WHERE job_type = 'analyze_inspection'
    AND status IN ('pending', 'running')
    AND payload->>'inspection_id' = $1::text
) AS has_pending;

-- name: CountCompletedJobsByUserAndType :one
-- Count completed jobs for a user within a date range (for quota checking)
SELECT COUNT(*) as count
FROM jobs
WHERE job_type = $1
AND status = 'completed'
AND payload->>'user_id' = $2::text
AND completed_at >= $3
AND completed_at < $4;
