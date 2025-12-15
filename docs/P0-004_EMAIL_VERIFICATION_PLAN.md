# P0-004: Email Verification Tokens - Implementation Plan

## Overview

This document details the implementation plan for email verification tokens and password reset tokens. These features are prerequisites for the email service integration (P0-005) and password reset flow (P0-006).

## Files Created/Modified

### New Files

| File | Purpose |
|------|---------|
| `/workspaces/lukaut/internal/migrations/00002_email_tokens.sql` | Database migration for token tables |
| `/workspaces/lukaut/sqlc/queries/tokens.sql` | SQLC query definitions for token operations |
| `/workspaces/lukaut/internal/domain/token.go` | Domain types for tokens |
| `/workspaces/lukaut/docs/P0-004_EMAIL_VERIFICATION_PLAN.md` | This implementation plan |

### Modified Files

| File | Changes |
|------|---------|
| `/workspaces/lukaut/internal/service/user.go` | Added interface methods and implementation stubs |
| `/workspaces/lukaut/internal/handler/auth.go` | Added verification handler stubs and routes |

---

## 1. Database Schema

### Email Verification Tokens Table

```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_email_verification_tokens_user UNIQUE (user_id),
    CONSTRAINT uq_email_verification_tokens_hash UNIQUE (token_hash)
);
```

**Design Decisions:**

- `UNIQUE (user_id)` - Only one active verification token per user. Simplifies management and prevents confusion.
- `UNIQUE (token_hash)` - Prevents hash collisions (extremely unlikely but defense-in-depth).
- `ON DELETE CASCADE` - Tokens are automatically deleted when user is deleted.
- No `used_at` field - Verification tokens are deleted after use (vs password reset which keeps audit trail).

### Password Reset Tokens Table

```sql
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,  -- NULL = unused, set when password is changed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_password_reset_tokens_user UNIQUE (user_id),
    CONSTRAINT uq_password_reset_tokens_hash UNIQUE (token_hash)
);
```

**Design Decisions:**

- `used_at` field - Maintains audit trail. Token is marked used rather than deleted.
- Shorter expiration (1 hour vs 24 hours) - Higher risk operation requires tighter window.

### Indexes

```sql
-- Email verification tokens
CREATE INDEX idx_email_verification_tokens_hash ON email_verification_tokens(token_hash);
CREATE INDEX idx_email_verification_tokens_expires ON email_verification_tokens(expires_at);

-- Password reset tokens
CREATE INDEX idx_password_reset_tokens_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);
```

---

## 2. SQLC Queries

### Email Verification Token Queries

| Query | Purpose |
|-------|---------|
| `CreateEmailVerificationToken` | Create new token (returns full record) |
| `GetEmailVerificationTokenByHash` | Look up token by hash (filters expired) |
| `DeleteEmailVerificationToken` | Delete token by hash (after verification) |
| `DeleteUserEmailVerificationTokens` | Delete all tokens for user (before creating new) |
| `DeleteExpiredEmailVerificationTokens` | Cleanup job (periodic maintenance) |
| `GetEmailVerificationTokenByUserID` | Check if user has pending verification |

### Password Reset Token Queries

| Query | Purpose |
|-------|---------|
| `CreatePasswordResetToken` | Create new token |
| `GetPasswordResetTokenByHash` | Look up token (filters expired AND used) |
| `MarkPasswordResetTokenUsed` | Mark token as used (audit trail) |
| `DeleteUserPasswordResetTokens` | Delete all tokens for user |
| `DeleteExpiredPasswordResetTokens` | Cleanup job |
| `GetPasswordResetTokenByUserID` | Check if user has pending reset |
| `DeletePasswordResetToken` | Alternative to marking used |

---

## 3. Domain Types

### Token Duration Constants

```go
const (
    EmailVerificationTokenDuration = 24 * time.Hour
    PasswordResetTokenDuration     = 1 * time.Hour
    TokenBytes                     = 32  // 256 bits of entropy
)
```

### Result Types

```go
type EmailVerificationResult struct {
    Token     string    // Raw token to send in email
    ExpiresAt time.Time
    UserID    uuid.UUID
}

type PasswordResetResult struct {
    Token     string    // Raw token to send in email
    ExpiresAt time.Time
    UserID    uuid.UUID
}
```

---

## 4. Service Layer Interface

### New Methods Added to UserService

```go
// Email Verification
CreateEmailVerificationToken(ctx, userID) (*EmailVerificationResult, error)
VerifyEmail(ctx, token string) error
ResendVerificationEmail(ctx, email string) (*EmailVerificationResult, error)
DeleteExpiredEmailVerificationTokens(ctx) error

// Password Reset
CreatePasswordResetToken(ctx, email string) (*PasswordResetResult, error)
ValidatePasswordResetToken(ctx, token string) (uuid.UUID, error)
ResetPassword(ctx, params ResetPasswordParams) error
DeleteExpiredPasswordResetTokens(ctx) error
```

### Security Patterns

All methods follow the same security pattern as session tokens:

1. **Token Generation**: 32 random bytes from `crypto/rand`
2. **Token Encoding**: Hex encoding (64 characters)
3. **Token Storage**: SHA-256 hash stored in database
4. **Token Transmission**: Raw token sent via email, never stored in plaintext

---

## 5. Handler Routes

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/verify-email?token=xxx` | `ShowVerifyEmail` | Handle email verification link |
| POST | `/resend-verification` | `ResendVerification` | Request new verification email |

### Future Routes (P0-006)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/forgot-password` | `ShowForgotPassword` | Display forgot password form |
| POST | `/forgot-password` | `ForgotPassword` | Send reset email |
| GET | `/reset-password?token=xxx` | `ShowResetPassword` | Display password reset form |
| POST | `/reset-password` | `ResetPassword` | Process password change |

---

## 6. Implementation Steps

### Step 1: Run Migration

```bash
go run cmd/migrate/main.go up
```

### Step 2: Generate SQLC Code

```bash
sqlc generate
```

### Step 3: Implement Service Methods

Replace TODO comments in `/workspaces/lukaut/internal/service/user.go` with actual implementations. Each method has detailed implementation steps in comments.

### Step 4: Implement Handlers

Replace TODO comments in `/workspaces/lukaut/internal/handler/auth.go` with actual implementations.

### Step 5: Create Email Templates

Create templates in `/workspaces/lukaut/web/templates/`:
- `auth/verify_email.html` - Verification result page
- `auth/resend_verification.html` - Resend confirmation page
- `email/verification.html` - Email template (P0-005)

### Step 6: Integrate Email Service (P0-005)

After EmailService is implemented:
1. Inject EmailService into AuthHandler
2. Call `emailService.SendVerificationEmail()` after token creation
3. Call during registration flow to send initial verification

---

## 7. Security Considerations

### Token Generation

- 32 bytes (256 bits) of entropy from `crypto/rand`
- Hex-encoded for URL safety (64 characters)
- Same security level as session tokens

### Token Storage

- SHA-256 hashed before storage
- Even if database is compromised, tokens cannot be used
- Hash is deterministic, allowing lookup by hash

### Token Lifecycle

**Email Verification:**
- Created: After registration or on resend request
- Expires: 24 hours after creation
- Consumed: Deleted after successful verification
- Cleanup: Expired tokens deleted by periodic job

**Password Reset:**
- Created: After forgot password request
- Expires: 1 hour after creation
- Consumed: Marked as used (not deleted) for audit
- Cleanup: Expired unused tokens deleted by periodic job

### Rate Limiting (Future P2-007)

Consider adding rate limits to:
- POST `/resend-verification` - Prevent email spam
- POST `/forgot-password` - Prevent email spam
- GET `/verify-email` - Prevent token enumeration (less critical)

### Information Disclosure Prevention

- `ResendVerification` always shows success message regardless of email existence
- `ForgotPassword` always shows "if account exists" message
- Error messages do not reveal whether email is registered

---

## 8. Testing Checklist

### Unit Tests

- [ ] Token generation produces correct length
- [ ] Token hashing is deterministic
- [ ] CreateEmailVerificationToken deletes old tokens first
- [ ] VerifyEmail rejects expired tokens
- [ ] VerifyEmail rejects already-verified users
- [ ] ResetPassword invalidates all sessions

### Integration Tests

- [ ] Full registration -> verification flow
- [ ] Resend verification replaces old token
- [ ] Expired token shows appropriate error
- [ ] Password reset -> login flow works

### Security Tests

- [ ] Cannot enumerate emails via verification
- [ ] Cannot enumerate emails via password reset
- [ ] Cannot reuse verification token
- [ ] Cannot reuse password reset token

---

## 9. Dependencies

### This Feature Depends On

- [x] P0-002: Authentication Service Layer (complete)
- [x] Initial database schema with users table (complete)

### Features That Depend On This

- [ ] P0-005: Email Service Integration (uses token creation)
- [ ] P0-006: Password Reset Flow (uses password reset tokens)

---

## 10. File Locations Summary

```
/workspaces/lukaut/
├── internal/
│   ├── migrations/
│   │   └── 00002_email_tokens.sql          # NEW: Token tables migration
│   ├── domain/
│   │   ├── user.go                          # Existing domain types
│   │   └── token.go                         # NEW: Token domain types
│   ├── service/
│   │   └── user.go                          # MODIFIED: Added token methods
│   ├── handler/
│   │   └── auth.go                          # MODIFIED: Added verification handlers
│   └── repository/
│       └── tokens.sql.go                    # GENERATED: After sqlc generate
├── sqlc/
│   └── queries/
│       └── tokens.sql                       # NEW: Token query definitions
└── docs/
    └── P0-004_EMAIL_VERIFICATION_PLAN.md   # NEW: This document
```

---

## 11. Next Steps

1. **Run migration** to create tables
2. **Run sqlc generate** to create repository code
3. **Implement service methods** by uncommenting TODO code
4. **Implement handlers** by uncommenting TODO code
5. **Create templates** for verification pages
6. **Proceed to P0-005** for email service integration
