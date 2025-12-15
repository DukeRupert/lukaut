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
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteSite :exec
DELETE FROM sites
WHERE id = $1;
