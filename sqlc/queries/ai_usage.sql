-- name: CreateAIUsage :one
INSERT INTO ai_usage (
    user_id,
    inspection_id,
    model,
    input_tokens,
    output_tokens,
    cost_cents,
    request_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetUserAIUsageThisMonth :one
SELECT
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens,
    COALESCE(SUM(cost_cents), 0) as total_cost_cents,
    COUNT(*) as request_count
FROM ai_usage
WHERE user_id = $1
AND created_at >= date_trunc('month', NOW())
AND created_at < date_trunc('month', NOW()) + INTERVAL '1 month';
