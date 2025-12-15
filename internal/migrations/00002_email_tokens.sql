-- +goose Up

-- =============================================================================
-- Email Verification Tokens
-- =============================================================================
--
-- Purpose: Store hashed email verification tokens for new user registration.
--
-- Security Design:
-- - Tokens are stored as SHA-256 hashes (64 char hex string)
-- - Raw token is sent to user via email, never stored
-- - ON DELETE CASCADE ensures tokens are cleaned up when user is deleted
-- - expires_at enforces time-limited validity (24 hours recommended)
-- - UNIQUE constraint on user_id ensures only one active token per user
--   (new token creation should delete old tokens first)
--
-- Usage Flow:
-- 1. User registers -> Create token, send email with raw token
-- 2. User clicks link -> Look up by hash, verify not expired
-- 3. Valid token -> Mark user email_verified=true, delete token
-- 4. Invalid/expired -> Show error, offer to resend
--
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Only one active verification token per user
    -- This simplifies token management and prevents confusion
    CONSTRAINT uq_email_verification_tokens_user UNIQUE (user_id),

    -- Token hash must be unique across all tokens
    -- Prevents extremely unlikely hash collision issues
    CONSTRAINT uq_email_verification_tokens_hash UNIQUE (token_hash)
);

-- Index for efficient token lookup by hash (primary lookup pattern)
-- The query will be: WHERE token_hash = $1 AND expires_at > NOW()
CREATE INDEX idx_email_verification_tokens_hash
    ON email_verification_tokens(token_hash);

-- Index for cleanup queries that delete expired tokens
CREATE INDEX idx_email_verification_tokens_expires
    ON email_verification_tokens(expires_at);

-- =============================================================================
-- Password Reset Tokens
-- =============================================================================
--
-- Purpose: Store hashed password reset tokens for forgot password flow.
--
-- Security Design:
-- - Same hashing strategy as email verification tokens
-- - used_at field tracks if token was consumed (for audit trail)
-- - Shorter expiration than email verification (1 hour recommended)
-- - Only one active token per user (prevents token farming)
--
-- Usage Flow:
-- 1. User requests reset -> Delete old tokens, create new, send email
-- 2. User clicks link -> Look up by hash, verify not expired and not used
-- 3. Valid token -> Show password form
-- 4. User submits new password -> Update password, mark token used_at
-- 5. Invalidate all user sessions (security best practice)
--
-- Note: We mark tokens as "used" rather than deleting them immediately
-- to maintain an audit trail and prevent replay attacks during the
-- brief window between validation and password update.
--
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,  -- NULL = unused, set when password is changed
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Only one active reset token per user
    -- Prevents token farming and simplifies management
    CONSTRAINT uq_password_reset_tokens_user UNIQUE (user_id),

    -- Token hash must be unique
    CONSTRAINT uq_password_reset_tokens_hash UNIQUE (token_hash)
);

-- Index for efficient token lookup by hash
CREATE INDEX idx_password_reset_tokens_hash
    ON password_reset_tokens(token_hash);

-- Index for cleanup queries
CREATE INDEX idx_password_reset_tokens_expires
    ON password_reset_tokens(expires_at);

-- =============================================================================
-- Comments for future maintainers
-- =============================================================================
--
-- Token Lifecycle:
--
-- Email Verification:
-- - Created: During user registration
-- - Deleted: When email is verified OR when new token is requested
-- - Expiry: 24 hours (configurable in service layer)
--
-- Password Reset:
-- - Created: When user requests password reset
-- - Marked used: When password is successfully changed
-- - Deleted: By cleanup job after expiration OR when new token requested
-- - Expiry: 1 hour (configurable in service layer)
--
-- Cleanup Strategy:
-- Run periodic cleanup (e.g., daily) to remove:
-- - Expired email_verification_tokens
-- - Expired password_reset_tokens (keep used ones for audit if needed)
--
-- Why separate tables instead of one generic tokens table?
-- 1. Different expiration policies
-- 2. Different constraints (email: delete on use, password: mark used)
-- 3. Clearer queries without type discrimination
-- 4. Easier to add token-type-specific fields later
-- 5. No risk of accidentally mixing up token types

-- +goose Down

DROP INDEX IF EXISTS idx_password_reset_tokens_expires;
DROP INDEX IF EXISTS idx_password_reset_tokens_hash;
DROP TABLE IF EXISTS password_reset_tokens;

DROP INDEX IF EXISTS idx_email_verification_tokens_expires;
DROP INDEX IF EXISTS idx_email_verification_tokens_hash;
DROP TABLE IF EXISTS email_verification_tokens;
