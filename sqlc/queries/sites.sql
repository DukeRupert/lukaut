-- name: CreateSite :one
INSERT INTO sites (
    user_id,
    name,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    client_name,
    client_email,
    client_phone,
    notes,
    client_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetSiteByID :one
SELECT * FROM sites
WHERE id = $1;

-- name: GetSiteByIDAndUserID :one
SELECT * FROM sites
WHERE id = $1 AND user_id = $2;

-- name: ListSitesByUserID :many
SELECT * FROM sites
WHERE user_id = $1
ORDER BY name ASC;

-- name: ListSitesWithClientByUserID :many
SELECT
    s.id,
    s.user_id,
    s.name,
    s.address_line1,
    s.address_line2,
    s.city,
    s.state,
    s.postal_code,
    s.client_name,
    s.client_email,
    s.client_phone,
    s.notes,
    s.created_at,
    s.updated_at,
    s.client_id,
    c.name as linked_client_name
FROM sites s
LEFT JOIN clients c ON s.client_id = c.id
WHERE s.user_id = $1
ORDER BY s.name ASC;

-- name: UpdateSite :exec
UPDATE sites
SET name = $2,
    address_line1 = $3,
    address_line2 = $4,
    city = $5,
    state = $6,
    postal_code = $7,
    client_name = $8,
    client_email = $9,
    client_phone = $10,
    notes = $11,
    client_id = $12,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteSite :exec
DELETE FROM sites
WHERE id = $1;
