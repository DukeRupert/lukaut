-- name: CreateInspection :one
INSERT INTO inspections (
    user_id,
    site_id,
    title,
    status,
    inspection_date,
    weather_conditions,
    temperature,
    inspector_notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetInspectionByID :one
SELECT * FROM inspections
WHERE id = $1;

-- name: GetInspectionByIDAndUserID :one
SELECT * FROM inspections
WHERE id = $1 AND user_id = $2;

-- name: UpdateInspection :exec
UPDATE inspections
SET title = $2,
    site_id = $3,
    inspection_date = $4,
    weather_conditions = $5,
    temperature = $6,
    inspector_notes = $7,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateInspectionStatus :exec
UPDATE inspections
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ListInspectionsByUserID :many
SELECT * FROM inspections
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInspectionsByUserID :one
SELECT COUNT(*) FROM inspections
WHERE user_id = $1;

-- name: DeleteInspection :exec
DELETE FROM inspections
WHERE id = $1;

-- name: ListRecentInspectionsWithViolationCount :many
SELECT
    i.id,
    i.user_id,
    i.site_id,
    i.title,
    i.status,
    i.inspection_date,
    i.weather_conditions,
    i.temperature,
    i.inspector_notes,
    i.created_at,
    i.updated_at,
    COALESCE(COUNT(v.id), 0)::int AS violation_count
FROM inspections i
LEFT JOIN violations v ON v.inspection_id = i.id
WHERE i.user_id = $1
GROUP BY i.id
ORDER BY i.created_at DESC
LIMIT $2;
