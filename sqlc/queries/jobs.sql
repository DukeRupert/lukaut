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
UPDATE jobs
SET status = CASE
    WHEN attempts >= max_attempts THEN 'failed'
    ELSE 'pending'
END,
error_message = $2,
scheduled_at = CASE
    WHEN attempts < max_attempts THEN NOW() + INTERVAL '5 minutes'
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
