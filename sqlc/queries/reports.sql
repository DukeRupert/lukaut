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

-- name: GetReportByIDAndUserID :one
SELECT * FROM reports
WHERE id = $1 AND user_id = $2;

-- name: ListReportsByInspectionID :many
SELECT * FROM reports
WHERE inspection_id = $1
ORDER BY generated_at DESC;

-- name: CountReportsByUserID :one
SELECT COUNT(*) FROM reports
WHERE user_id = $1;

-- name: CountReportsThisMonthByUserID :one
SELECT COUNT(*) FROM reports
WHERE user_id = $1
  AND generated_at >= DATE_TRUNC('month', CURRENT_TIMESTAMP);
