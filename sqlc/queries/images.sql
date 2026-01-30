-- name: CreateImage :one
INSERT INTO images (
    inspection_id,
    storage_key,
    thumbnail_key,
    original_filename,
    content_type,
    size_bytes,
    width,
    height,
    analysis_status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetImageByID :one
SELECT * FROM images
WHERE id = $1;

-- name: GetImageByIDWithInspection :one
SELECT i.*, ins.user_id
FROM images i
JOIN inspections ins ON ins.id = i.inspection_id
WHERE i.id = $1;

-- name: GetImageByIDAndInspectionID :one
SELECT * FROM images
WHERE id = $1 AND inspection_id = $2;

-- name: ListImagesByInspectionID :many
SELECT * FROM images
WHERE inspection_id = $1
ORDER BY created_at DESC;

-- name: ListPendingImagesByInspectionID :many
SELECT * FROM images
WHERE inspection_id = $1
AND analysis_status = 'pending'
ORDER BY created_at ASC;

-- name: CountImagesByInspectionID :one
SELECT COUNT(*) FROM images
WHERE inspection_id = $1;

-- name: UpdateImageAnalysisStatus :exec
UPDATE images
SET analysis_status = $2,
    analysis_completed_at = $3
WHERE id = $1;

-- name: DeleteImageByID :exec
DELETE FROM images
WHERE id = $1;

-- name: CountPendingImagesByInspectionID :one
-- Count images that haven't been analyzed yet
SELECT COUNT(*) FROM images
WHERE inspection_id = $1
AND (analysis_status IS NULL OR analysis_status = 'pending');

-- name: ListPendingImagesByInspectionIDAndUserID :many
-- List pending images with user authorization check (defense in depth)
SELECT img.* FROM images img
JOIN inspections ins ON ins.id = img.inspection_id
WHERE ins.id = $1
AND ins.user_id = $2
AND img.analysis_status = 'pending'
ORDER BY img.created_at ASC;

-- name: UpdateImageAnalysisStatusWithAuth :exec
-- Update image analysis status with user authorization check
UPDATE images
SET analysis_status = $3,
    analysis_completed_at = $4
FROM inspections
WHERE images.id = $1
AND images.inspection_id = inspections.id
AND inspections.user_id = $2;
