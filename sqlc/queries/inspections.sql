-- name: CreateInspection :one
INSERT INTO inspections (
    user_id,
    client_id,
    title,
    status,
    inspection_date,
    weather_conditions,
    temperature,
    inspector_notes,
    address_line1,
    address_line2,
    city,
    state,
    postal_code
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
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
    client_id = $3,
    inspection_date = $4,
    weather_conditions = $5,
    temperature = $6,
    inspector_notes = $7,
    address_line1 = $8,
    address_line2 = $9,
    city = $10,
    state = $11,
    postal_code = $12,
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
    i.client_id,
    i.title,
    i.status,
    i.inspection_date,
    i.weather_conditions,
    i.temperature,
    i.inspector_notes,
    i.address_line1,
    i.address_line2,
    i.city,
    i.state,
    i.postal_code,
    i.created_at,
    i.updated_at,
    COALESCE(COUNT(v.id), 0)::int AS violation_count
FROM inspections i
LEFT JOIN violations v ON v.inspection_id = i.id
WHERE i.user_id = $1
GROUP BY i.id
ORDER BY i.created_at DESC
LIMIT $2;

-- name: ListInspectionsWithClientByUserID :many
SELECT
    i.id,
    i.user_id,
    i.client_id,
    i.title,
    i.status,
    i.inspection_date,
    i.weather_conditions,
    i.temperature,
    i.inspector_notes,
    i.address_line1,
    i.address_line2,
    i.city,
    i.state,
    i.postal_code,
    i.created_at,
    i.updated_at,
    COALESCE(c.name, '') AS client_name,
    COALESCE(COUNT(v.id), 0)::int AS violation_count
FROM inspections i
LEFT JOIN clients c ON c.id = i.client_id
LEFT JOIN violations v ON v.inspection_id = i.id
WHERE i.user_id = $1
GROUP BY i.id, i.user_id, i.client_id, i.title, i.status, i.inspection_date,
         i.weather_conditions, i.temperature, i.inspector_notes,
         i.address_line1, i.address_line2, i.city, i.state, i.postal_code,
         i.created_at, i.updated_at, c.name
ORDER BY i.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetInspectionWithClientByIDAndUserID :one
SELECT
    i.id,
    i.user_id,
    i.client_id,
    i.title,
    i.status,
    i.inspection_date,
    i.weather_conditions,
    i.temperature,
    i.inspector_notes,
    i.address_line1,
    i.address_line2,
    i.city,
    i.state,
    i.postal_code,
    i.created_at,
    i.updated_at,
    COALESCE(c.name, '') AS client_name
FROM inspections i
LEFT JOIN clients c ON c.id = i.client_id
WHERE i.id = $1 AND i.user_id = $2;

-- name: DeleteInspectionByIDAndUserID :exec
DELETE FROM inspections
WHERE id = $1 AND user_id = $2;

-- name: UpdateInspectionByIDAndUserID :exec
UPDATE inspections
SET title = $3,
    client_id = $4,
    inspection_date = $5,
    weather_conditions = $6,
    temperature = $7,
    inspector_notes = $8,
    address_line1 = $9,
    address_line2 = $10,
    city = $11,
    state = $12,
    postal_code = $13,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateInspectionStatusByIDAndUserID :exec
UPDATE inspections
SET status = $3,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;
