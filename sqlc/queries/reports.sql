-- name: CreateReport :one
INSERT INTO reports (
    inspection_id,
    user_id,
    pdf_storage_key,
    docx_storage_key,
    violation_count
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetReportByID :one
SELECT * FROM reports
WHERE id = $1;

-- name: ListReportsByInspectionID :many
SELECT * FROM reports
WHERE inspection_id = $1
ORDER BY generated_at DESC;
