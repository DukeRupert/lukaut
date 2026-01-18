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

-- name: GetUserAIUsageByDateRange :one
-- Get AI usage for a specific user within a date range
SELECT
    COALESCE(SUM(input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint as total_cost_cents,
    COUNT(*) as request_count
FROM ai_usage
WHERE user_id = $1
AND created_at >= $2
AND created_at < $3;

-- name: GetAllUsersAIUsageSummary :many
-- Get AI usage summary for all users (for admin/billing reports)
SELECT
    u.id as user_id,
    u.email,
    u.name,
    COALESCE(SUM(a.input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(a.output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(a.cost_cents), 0)::bigint as total_cost_cents,
    COUNT(a.id) as request_count
FROM users u
LEFT JOIN ai_usage a ON u.id = a.user_id
    AND a.created_at >= $1
    AND a.created_at < $2
GROUP BY u.id, u.email, u.name
ORDER BY total_cost_cents DESC;

-- name: GetPlatformAIUsageTotal :one
-- Get total platform AI usage (all users combined)
SELECT
    COALESCE(SUM(input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint as total_cost_cents,
    COUNT(*) as request_count
FROM ai_usage
WHERE created_at >= $1
AND created_at < $2;

-- name: GetUserAIUsageByDay :many
-- Get daily AI usage breakdown for a user (for usage charts)
SELECT
    date_trunc('day', created_at)::date as date,
    COALESCE(SUM(input_tokens), 0)::bigint as input_tokens,
    COALESCE(SUM(output_tokens), 0)::bigint as output_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint as cost_cents,
    COUNT(*) as request_count
FROM ai_usage
WHERE user_id = $1
AND created_at >= $2
AND created_at < $3
GROUP BY date_trunc('day', created_at)
ORDER BY date;
