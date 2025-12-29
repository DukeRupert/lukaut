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

-- name: ListSitesByClientID :many
SELECT * FROM sites
WHERE client_id = $1 AND user_id = $2
ORDER BY name ASC;

-- name: GetSiteWithClientByIDAndUserID :one
SELECT
    s.id,
    s.user_id,
    s.name,
    s.address_line1,
    s.address_line2,
    s.city,
    s.state,
    s.postal_code,
    s.client_id,
    s.client_name,
    s.client_email,
    s.client_phone,
    s.notes,
    s.created_at,
    s.updated_at,
    COALESCE(c.name, '') AS client_name_resolved,
    COALESCE(c.email, '') AS client_email_resolved,
    COALESCE(c.phone, '') AS client_phone_resolved
FROM sites s
LEFT JOIN clients c ON c.id = s.client_id
WHERE s.id = $1 AND s.user_id = $2;

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
    s.client_id,
    s.client_name,
    s.client_email,
    s.client_phone,
    s.notes,
    s.created_at,
    s.updated_at,
    COALESCE(c.name, '') AS client_name_resolved
FROM sites s
LEFT JOIN clients c ON c.id = s.client_id
WHERE s.user_id = $1
ORDER BY s.name ASC;

-- name: CreateSiteWithClient :one
INSERT INTO sites (
    user_id,
    name,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    client_id,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpdateSiteWithClient :exec
UPDATE sites
SET name = $2,
    address_line1 = $3,
    address_line2 = $4,
    city = $5,
    state = $6,
    postal_code = $7,
    client_id = $8,
    notes = $9,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSiteWithClientByIDAndUserID :exec
UPDATE sites
SET name = $3,
    address_line1 = $4,
    address_line2 = $5,
    city = $6,
    state = $7,
    postal_code = $8,
    client_id = $9,
    notes = $10,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;
