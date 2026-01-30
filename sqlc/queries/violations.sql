-- name: CreateViolation :one
INSERT INTO violations (
    inspection_id,
    image_id,
    description,
    ai_description,
    confidence,
    bounding_box,
    status,
    severity,
    inspector_notes,
    sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetViolationByID :one
SELECT * FROM violations
WHERE id = $1;

-- name: GetViolationByIDAndInspectionID :one
SELECT * FROM violations
WHERE id = $1 AND inspection_id = $2;

-- name: ListViolationsByInspectionID :many
SELECT * FROM violations
WHERE inspection_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: ListConfirmedViolationsByInspectionID :many
SELECT * FROM violations
WHERE inspection_id = $1
AND status = 'confirmed'
ORDER BY sort_order ASC, created_at ASC;

-- name: UpdateViolationStatus :exec
UPDATE violations
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateViolationDescription :exec
UPDATE violations
SET description = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateViolationSeverity :exec
UPDATE violations
SET severity = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateViolationNotes :exec
UPDATE violations
SET inspector_notes = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteViolation :exec
DELETE FROM violations
WHERE id = $1;

-- name: CountViolationsByInspectionID :one
SELECT COUNT(*) FROM violations
WHERE inspection_id = $1;

-- name: CountViolationsByUserID :one
SELECT COUNT(*) FROM violations v
JOIN inspections i ON i.id = v.inspection_id
WHERE i.user_id = $1;

-- name: GetViolationWithImage :one
SELECT
    v.*,
    i.thumbnail_key,
    i.original_filename
FROM violations v
LEFT JOIN images i ON i.id = v.image_id
WHERE v.id = $1;

-- name: UpdateViolationDetails :exec
UPDATE violations
SET description = $2,
    severity = $3,
    inspector_notes = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: CountViolationsByStatus :one
SELECT COUNT(*) FROM violations
WHERE inspection_id = $1 AND status = $2;

-- name: GetViolationByIDAndUserID :one
SELECT v.* FROM violations v
JOIN inspections i ON i.id = v.inspection_id
WHERE v.id = $1 AND i.user_id = $2;

-- name: DeleteViolationByIDAndUserID :exec
DELETE FROM violations v
USING inspections i
WHERE v.id = $1
AND v.inspection_id = i.id
AND i.user_id = $2;

-- name: ListConfirmedViolationsByInspectionIDAndUserID :many
-- List confirmed violations with user authorization check (defense in depth)
SELECT v.* FROM violations v
JOIN inspections i ON i.id = v.inspection_id
WHERE v.inspection_id = $1
AND i.user_id = $2
AND v.status = 'confirmed'
ORDER BY v.sort_order ASC, v.created_at ASC;
