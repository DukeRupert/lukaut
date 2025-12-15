// Package domain contains core business types and interfaces.
//
// This file defines token-related domain types for email verification
// and password reset flows.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Token Configuration Constants
// =============================================================================

const (
	// EmailVerificationTokenDuration is how long email verification tokens remain valid.
	// 24 hours gives users reasonable time to verify while limiting exposure.
	EmailVerificationTokenDuration = 24 * time.Hour

	// PasswordResetTokenDuration is how long password reset tokens remain valid.
	// 1 hour is standard practice - short enough to limit exposure, long enough
	// for users to complete the flow.
	PasswordResetTokenDuration = 1 * time.Hour

	// TokenBytes is the number of random bytes for tokens.
	// 32 bytes = 256 bits of entropy, matching session token security.
	// The token is hex-encoded to 64 characters for URL safety.
	TokenBytes = 32
)

// =============================================================================
// Email Verification Token
// =============================================================================

// EmailVerificationToken represents a token sent to verify user email ownership.
//
// Security model:
// - Raw token (64 hex chars) is sent via email to user
// - Only SHA-256 hash of token is stored in database
// - Token expires after EmailVerificationTokenDuration
// - Only one active token per user at a time
//
// Flow:
// 1. User registers -> system creates token, sends email
// 2. User clicks link with raw token
// 3. System hashes token, looks up in DB
// 4. If valid and not expired -> mark email verified, delete token
type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string    // SHA-256 hash of raw token (64 char hex)
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired returns true if the token has expired.
func (t *EmailVerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid returns true if the token is not expired.
func (t *EmailVerificationToken) IsValid() bool {
	return !t.IsExpired()
}

// =============================================================================
// Password Reset Token
// =============================================================================

// PasswordResetToken represents a token sent to allow password reset.
//
// Security model:
// - Same hash-before-store approach as email verification
// - Shorter expiration (1 hour) due to higher risk
// - Marked as "used" rather than deleted for audit trail
// - All user sessions invalidated after successful reset
//
// Flow:
// 1. User requests reset -> system creates token, sends email
// 2. User clicks link with raw token
// 3. System hashes token, looks up in DB, shows password form
// 4. User submits new password
// 5. System validates token again, updates password, marks token used
// 6. All user sessions are invalidated
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string     // SHA-256 hash of raw token (64 char hex)
	ExpiresAt time.Time
	UsedAt    *time.Time // nil = unused, set when password is changed
	CreatedAt time.Time
}

// IsExpired returns true if the token has expired.
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been used.
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid returns true if the token is not expired and not used.
func (t *PasswordResetToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}

// =============================================================================
// Token Result Types
// =============================================================================

// EmailVerificationResult contains the result of creating an email verification token.
type EmailVerificationResult struct {
	Token     string    // Raw token to send in email (NOT the hash)
	ExpiresAt time.Time // When the token expires
	UserID    uuid.UUID // The user this token is for
}

// PasswordResetResult contains the result of creating a password reset token.
type PasswordResetResult struct {
	Token     string    // Raw token to send in email (NOT the hash)
	ExpiresAt time.Time // When the token expires
	UserID    uuid.UUID // The user this token is for
}

// =============================================================================
// Service Parameters
// =============================================================================

// VerifyEmailParams contains parameters for the email verification operation.
type VerifyEmailParams struct {
	Token string // Raw token from the verification link
}

// RequestPasswordResetParams contains parameters for requesting a password reset.
type RequestPasswordResetParams struct {
	Email string // User's email address
}

// ResetPasswordParams contains parameters for resetting a password.
type ResetPasswordParams struct {
	Token       string // Raw token from the reset link
	NewPassword string // The new password to set
}
