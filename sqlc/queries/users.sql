-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash,
    name,
    company_name,
    phone
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserEmailVerification :exec
UPDATE users
SET email_verified = $2,
    email_verified_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserProfile :exec
UPDATE users
SET name = $2,
    company_name = $3,
    phone = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserStripeCustomer :exec
UPDATE users
SET stripe_customer_id = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserSubscription :exec
UPDATE users
SET subscription_status = $2,
    subscription_tier = $3,
    subscription_id = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: GetUserByStripeCustomerID :one
SELECT * FROM users
WHERE stripe_customer_id = $1;

-- name: UpdateUserBusinessProfile :exec
UPDATE users
SET business_name = $2,
    business_email = $3,
    business_phone = $4,
    business_address_line1 = $5,
    business_address_line2 = $6,
    business_city = $7,
    business_state = $8,
    business_postal_code = $9,
    business_license_number = $10,
    business_logo_url = $11,
    updated_at = NOW()
WHERE id = $1;
