-- name: GetRegulationByID :one
SELECT * FROM regulations
WHERE id = $1;

-- name: GetRegulationByStandardNumber :one
SELECT * FROM regulations
WHERE standard_number = $1;

-- name: ListRegulationsByCategory :many
SELECT * FROM regulations
WHERE category = $1
AND (sqlc.narg('subcategory')::text IS NULL OR subcategory = sqlc.narg('subcategory'))
ORDER BY standard_number ASC;

-- name: SearchRegulations :many
SELECT *,
    ts_rank(search_vector, websearch_to_tsquery('english', $1)) as rank
FROM regulations
WHERE search_vector @@ websearch_to_tsquery('english', $1)
ORDER BY rank DESC, standard_number ASC
LIMIT $2;

-- name: CreateRegulation :one
INSERT INTO regulations (
    standard_number,
    title,
    category,
    subcategory,
    full_text,
    summary,
    severity_typical,
    parent_standard,
    effective_date,
    last_updated
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;
