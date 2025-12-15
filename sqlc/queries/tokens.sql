-- =============================================================================
-- Email Verification Token Queries
-- =============================================================================

-- name: CreateEmailVerificationToken :one
-- Creates a new email verification token for a user.
-- The token_hash should be SHA-256 hash of the raw token (64 char hex).
-- The raw token is sent to user via email; only the hash is stored.
--
-- Note: Caller should delete existing tokens for user before calling this
-- to enforce the one-token-per-user constraint.
INSERT INTO email_verification_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetEmailVerificationTokenByHash :one
-- Retrieves a valid (non-expired) email verification token by its hash.
-- Returns sql.ErrNoRows if token doesn't exist or is expired.
-- Used during the email verification flow when user clicks the link.
SELECT * FROM email_verification_tokens
WHERE token_hash = $1
AND expires_at > NOW();

-- name: DeleteEmailVerificationToken :exec
-- Deletes a specific email verification token by its hash.
-- Called after successful email verification.
-- Idempotent - no error if token doesn't exist.
DELETE FROM email_verification_tokens
WHERE token_hash = $1;

-- name: DeleteUserEmailVerificationTokens :exec
-- Deletes all email verification tokens for a specific user.
-- Called before creating a new token to ensure one-token-per-user.
-- Also useful when user requests a new verification email.
DELETE FROM email_verification_tokens
WHERE user_id = $1;

-- name: DeleteExpiredEmailVerificationTokens :exec
-- Removes all expired email verification tokens.
-- Should be called periodically (e.g., daily) as a cleanup task.
DELETE FROM email_verification_tokens
WHERE expires_at <= NOW();

-- name: GetEmailVerificationTokenByUserID :one
-- Retrieves the verification token for a specific user.
-- Useful to check if user already has a pending verification.
-- Note: Only returns non-expired tokens.
SELECT * FROM email_verification_tokens
WHERE user_id = $1
AND expires_at > NOW();

-- =============================================================================
-- Password Reset Token Queries
-- =============================================================================

-- name: CreatePasswordResetToken :one
-- Creates a new password reset token for a user.
-- The token_hash should be SHA-256 hash of the raw token (64 char hex).
-- The raw token is sent to user via email; only the hash is stored.
--
-- Note: Caller should delete existing tokens for user before calling this
-- to enforce the one-token-per-user constraint.
INSERT INTO password_reset_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetPasswordResetTokenByHash :one
-- Retrieves a valid (non-expired, unused) password reset token by its hash.
-- Returns sql.ErrNoRows if token doesn't exist, is expired, or was used.
-- Used during the password reset flow when user clicks the link.
SELECT * FROM password_reset_tokens
WHERE token_hash = $1
AND expires_at > NOW()
AND used_at IS NULL;

-- name: MarkPasswordResetTokenUsed :exec
-- Marks a password reset token as used after successful password change.
-- Sets used_at to current timestamp for audit trail.
-- This prevents token reuse while maintaining history.
UPDATE password_reset_tokens
SET used_at = NOW()
WHERE token_hash = $1;

-- name: DeleteUserPasswordResetTokens :exec
-- Deletes all password reset tokens for a specific user.
-- Called before creating a new token to ensure one-token-per-user.
-- Also useful when user successfully logs in (no longer needs reset).
DELETE FROM password_reset_tokens
WHERE user_id = $1;

-- name: DeleteExpiredPasswordResetTokens :exec
-- Removes all expired password reset tokens.
-- Should be called periodically (e.g., daily) as a cleanup task.
-- Note: This keeps used tokens for audit; remove the AND clause if not needed.
DELETE FROM password_reset_tokens
WHERE expires_at <= NOW()
AND used_at IS NULL;

-- name: GetPasswordResetTokenByUserID :one
-- Retrieves the password reset token for a specific user.
-- Useful to check if user already has a pending reset request.
-- Note: Only returns non-expired, unused tokens.
SELECT * FROM password_reset_tokens
WHERE user_id = $1
AND expires_at > NOW()
AND used_at IS NULL;

-- name: DeletePasswordResetToken :exec
-- Deletes a specific password reset token by its hash.
-- Alternative to marking as used - use when no audit trail needed.
DELETE FROM password_reset_tokens
WHERE token_hash = $1;
