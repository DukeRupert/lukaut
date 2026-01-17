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

-- name: ListAllCategories :many
SELECT DISTINCT category
FROM regulations
ORDER BY category ASC;

-- name: ListRegulations :many
SELECT id, standard_number, title, category, subcategory, summary, severity_typical
FROM regulations
WHERE (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
ORDER BY category ASC, standard_number ASC
LIMIT $1 OFFSET $2;

-- name: CountRegulations :one
SELECT COUNT(*) FROM regulations
WHERE (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'));

-- name: SearchRegulationsWithOffset :many
SELECT id, standard_number, title, category, subcategory, summary, severity_typical,
    ts_rank(search_vector, websearch_to_tsquery('english', $1)) as rank
FROM regulations
WHERE search_vector @@ websearch_to_tsquery('english', $1)
ORDER BY rank DESC, standard_number ASC
LIMIT $2 OFFSET $3;

-- name: CountSearchResults :one
SELECT COUNT(*) FROM regulations
WHERE search_vector @@ websearch_to_tsquery('english', $1);

-- name: GetRegulationDetail :one
SELECT id, standard_number, title, category, subcategory, full_text, summary,
       severity_typical, parent_standard, effective_date, last_updated
FROM regulations
WHERE id = $1;

-- name: GetRegulationsByStandardNumbers :many
-- Look up regulations by their standard numbers (e.g., ["1926.501(b)(1)", "1926.502(d)"])
-- Results are returned in the order they appear in the input array
SELECT r.*, array_position($1::text[], r.standard_number) as sort_order
FROM regulations r
WHERE r.standard_number = ANY($1::text[])
ORDER BY array_position($1::text[], r.standard_number);
