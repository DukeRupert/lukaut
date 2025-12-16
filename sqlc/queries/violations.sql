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
