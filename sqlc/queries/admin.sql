-- Admin queries for platform management

-- name: AdminListUsers :many
-- List all users with their AI usage summary for admin dashboard
SELECT
    u.id,
    u.email,
    u.name,
    u.subscription_status,
    u.subscription_tier,
    u.email_verified,
    u.created_at,
    COALESCE(SUM(a.input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(a.output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(a.cost_cents), 0)::bigint as total_cost_cents,
    COUNT(DISTINCT a.id) as ai_request_count,
    COUNT(DISTINCT i.id) as inspection_count
FROM users u
LEFT JOIN ai_usage a ON u.id = a.user_id
LEFT JOIN inspections i ON u.id = i.user_id
GROUP BY u.id
ORDER BY u.created_at DESC;

-- name: AdminGetUserByID :one
-- Get full user details for admin view
SELECT
    u.*,
    COALESCE(SUM(a.input_tokens), 0)::bigint as total_input_tokens,
    COALESCE(SUM(a.output_tokens), 0)::bigint as total_output_tokens,
    COALESCE(SUM(a.cost_cents), 0)::bigint as total_cost_cents,
    COUNT(DISTINCT a.id) as ai_request_count,
    COUNT(DISTINCT i.id) as inspection_count,
    COUNT(DISTINCT r.id) as report_count
FROM users u
LEFT JOIN ai_usage a ON u.id = a.user_id
LEFT JOIN inspections i ON u.id = i.user_id
LEFT JOIN reports r ON u.id = r.user_id
WHERE u.id = $1
GROUP BY u.id;

-- name: AdminGetPlatformStats :one
-- Get platform-wide statistics for admin dashboard
SELECT
    (SELECT COUNT(*) FROM users) as total_users,
    (SELECT COUNT(*) FROM users WHERE created_at >= date_trunc('month', NOW())) as new_users_this_month,
    (SELECT COUNT(*) FROM inspections) as total_inspections,
    (SELECT COUNT(*) FROM inspections WHERE created_at >= date_trunc('month', NOW())) as inspections_this_month,
    (SELECT COUNT(*) FROM reports) as total_reports,
    (SELECT COUNT(*) FROM reports WHERE created_at >= date_trunc('month', NOW())) as reports_this_month,
    (SELECT COALESCE(SUM(cost_cents), 0) FROM ai_usage) as total_ai_cost_cents,
    (SELECT COALESCE(SUM(cost_cents), 0) FROM ai_usage WHERE created_at >= date_trunc('month', NOW())) as ai_cost_this_month_cents;

-- name: AdminGetRecentSignups :many
-- Get recent user signups for admin dashboard
SELECT id, email, name, subscription_status, email_verified, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1;

-- name: AdminGetUserInspections :many
-- Get inspections for a specific user (admin view)
SELECT id, title, status, inspection_date, created_at
FROM inspections
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: AdminGetUserAIUsageHistory :many
-- Get AI usage history for a specific user
SELECT
    id,
    model,
    input_tokens,
    output_tokens,
    cost_cents,
    request_type,
    created_at
FROM ai_usage
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: AdminUpdateUserDisabled :exec
-- Enable or disable a user account
UPDATE users
SET updated_at = NOW()
WHERE id = $1;
