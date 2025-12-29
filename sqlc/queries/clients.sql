-- name: CreateClient :one
INSERT INTO clients (
    user_id,
    name,
    email,
    phone,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM clients
WHERE id = $1;

-- name: GetClientByIDAndUserID :one
SELECT * FROM clients
WHERE id = $1 AND user_id = $2;

-- name: ListClientsByUserID :many
SELECT * FROM clients
WHERE user_id = $1
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: CountClientsByUserID :one
SELECT COUNT(*) FROM clients
WHERE user_id = $1;

-- name: ListClientsWithSiteCountByUserID :many
SELECT
    c.id,
    c.user_id,
    c.name,
    c.email,
    c.phone,
    c.address_line1,
    c.address_line2,
    c.city,
    c.state,
    c.postal_code,
    c.notes,
    c.created_at,
    c.updated_at,
    COALESCE(COUNT(s.id), 0)::int AS site_count
FROM clients c
LEFT JOIN sites s ON s.client_id = c.id
WHERE c.user_id = $1
GROUP BY c.id, c.user_id, c.name, c.email, c.phone, c.address_line1, c.address_line2,
         c.city, c.state, c.postal_code, c.notes, c.created_at, c.updated_at
ORDER BY c.name ASC
LIMIT $2 OFFSET $3;

-- name: GetClientWithSiteCount :one
SELECT
    c.id,
    c.user_id,
    c.name,
    c.email,
    c.phone,
    c.address_line1,
    c.address_line2,
    c.city,
    c.state,
    c.postal_code,
    c.notes,
    c.created_at,
    c.updated_at,
    COALESCE(COUNT(s.id), 0)::int AS site_count
FROM clients c
LEFT JOIN sites s ON s.client_id = c.id
WHERE c.id = $1 AND c.user_id = $2
GROUP BY c.id, c.user_id, c.name, c.email, c.phone, c.address_line1, c.address_line2,
         c.city, c.state, c.postal_code, c.notes, c.created_at, c.updated_at;

-- name: UpdateClient :exec
UPDATE clients
SET name = $2,
    email = $3,
    phone = $4,
    address_line1 = $5,
    address_line2 = $6,
    city = $7,
    state = $8,
    postal_code = $9,
    notes = $10,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateClientByIDAndUserID :exec
UPDATE clients
SET name = $3,
    email = $4,
    phone = $5,
    address_line1 = $6,
    address_line2 = $7,
    city = $8,
    state = $9,
    postal_code = $10,
    notes = $11,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: DeleteClient :exec
DELETE FROM clients
WHERE id = $1;

-- name: DeleteClientByIDAndUserID :exec
DELETE FROM clients
WHERE id = $1 AND user_id = $2;

-- name: CountSitesByClientID :one
SELECT COUNT(*) FROM sites
WHERE client_id = $1;

-- name: ListAllClientsByUserID :many
SELECT * FROM clients
WHERE user_id = $1
ORDER BY name ASC;
