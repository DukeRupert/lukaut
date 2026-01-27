// Package service contains the business logic layer.
//
// Services orchestrate interactions between repositories, external APIs,
// and domain logic. They are responsible for:
// - Input validation
// - Business rule enforcement
// - Transaction coordination
// - Error translation (database errors -> domain errors)
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// Configuration Constants
// =============================================================================

const (
	// BcryptCost is the cost factor for bcrypt password hashing.
	// Cost 12 provides good security (~250ms on modern hardware) while being
	// fast enough for login flows. NIST recommends cost 10+.
	//
	// SECURITY NOTE: This should NOT be configurable at runtime to prevent
	// accidental weakening. If you need to change it, do so here and redeploy.
	BcryptCost = 12

	// SessionTokenBytes is the number of random bytes for session tokens.
	// 32 bytes = 256 bits of entropy, sufficient for cryptographic security.
	// The token is then hex-encoded to 64 characters for storage/transmission.
	SessionTokenBytes = 32

	// SessionDuration is how long a session remains valid.
	// 7 days balances security with user convenience for a B2B application.
	// Consider shorter durations for higher security requirements.
	SessionDuration = 7 * 24 * time.Hour

	// MinPasswordLength is the minimum password length.
	// NIST SP 800-63B recommends 8+ characters minimum.
	MinPasswordLength = 8

	// MaxPasswordLength prevents DoS via bcrypt on very long passwords.
	// bcrypt has a 72-byte limit anyway, but we cap earlier for clarity.
	MaxPasswordLength = 72
)

// =============================================================================
// Interface Definition
// =============================================================================

// UserService defines the interface for user-related operations.
//
// This interface enables:
// - Mocking in unit tests
// - Potential future implementations (e.g., with caching)
// - Clear contract definition for handlers
type UserService interface {
	// Register creates a new user account.
	// Returns domain.ECONFLICT if email already exists.
	// Returns domain.EINVALID for validation errors.
	Register(ctx context.Context, params domain.RegisterParams) (*domain.User, error)

	// Login authenticates a user and creates a new session.
	// Returns the user and raw session token on success.
	// Returns domain.EUNAUTHORIZED for invalid credentials.
	Login(ctx context.Context, email, password string) (*domain.LoginResult, error)

	// Logout invalidates a session by its raw token.
	// This is idempotent - calling with an invalid token is not an error.
	Logout(ctx context.Context, token string) error

	// GetByID retrieves a user by their ID.
	// Returns domain.ENOTFOUND if user does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetBySessionToken retrieves a user by their session token.
	// This validates the session and returns the associated user.
	// Returns domain.EUNAUTHORIZED if token is invalid or expired.
	GetBySessionToken(ctx context.Context, token string) (*domain.User, error)

	// UpdateProfile updates a user's profile information.
	// Returns domain.ENOTFOUND if user does not exist.
	UpdateProfile(ctx context.Context, params domain.ProfileUpdateParams) error

	// UpdateBusinessProfile updates a user's business profile information.
	// Returns domain.ENOTFOUND if user does not exist.
	UpdateBusinessProfile(ctx context.Context, params domain.BusinessProfileUpdateParams) error

	// ChangePassword changes a user's password.
	// Validates the current password before allowing the change.
	// Invalidates all existing sessions after password change.
	// Returns domain.EUNAUTHORIZED if current password is wrong.
	ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error

	// DeleteExpiredSessions removes all expired sessions from the database.
	// This should be called periodically (e.g., daily) to clean up.
	DeleteExpiredSessions(ctx context.Context) error

	// =========================================================================
	// Email Verification Methods
	// =========================================================================

	// CreateEmailVerificationToken creates a new email verification token for a user.
	// Returns the raw token (to send in email) and expiration time.
	// Deletes any existing tokens for the user before creating a new one.
	// This should be called after user registration or when user requests resend.
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error)

	// VerifyEmail validates an email verification token and marks the user as verified.
	// Returns domain.ENOTFOUND if token is invalid or expired.
	// Returns domain.ECONFLICT if user is already verified.
	// On success: marks user email_verified=true, deletes the token.
	VerifyEmail(ctx context.Context, token string) error

	// ResendVerificationEmail creates a new verification token for an unverified user.
	// Returns domain.ENOTFOUND if user does not exist.
	// Returns domain.ECONFLICT if user is already verified.
	// This is a convenience method that combines user lookup + token creation.
	ResendVerificationEmail(ctx context.Context, email string) (*domain.EmailVerificationResult, error)

	// DeleteExpiredEmailVerificationTokens removes all expired email verification tokens.
	// This should be called periodically (e.g., daily) as a cleanup task.
	DeleteExpiredEmailVerificationTokens(ctx context.Context) error

	// =========================================================================
	// Password Reset Methods
	// =========================================================================

	// CreatePasswordResetToken creates a new password reset token for a user.
	// Returns the raw token (to send in email) and expiration time.
	// Returns domain.ENOTFOUND if email does not exist (for security, caller
	// should NOT expose this to end user - always show "if email exists..." message).
	// Deletes any existing tokens for the user before creating a new one.
	CreatePasswordResetToken(ctx context.Context, email string) (*domain.PasswordResetResult, error)

	// ValidatePasswordResetToken checks if a password reset token is valid.
	// Returns the associated user ID if valid.
	// Returns domain.ENOTFOUND if token is invalid, expired, or already used.
	// This is used to validate before showing the password reset form.
	ValidatePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)

	// ResetPassword validates the token and updates the user's password.
	// Returns domain.ENOTFOUND if token is invalid, expired, or already used.
	// On success: updates password, marks token as used, invalidates all sessions.
	ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error

	// DeleteExpiredPasswordResetTokens removes all expired password reset tokens.
	// This should be called periodically (e.g., daily) as a cleanup task.
	DeleteExpiredPasswordResetTokens(ctx context.Context) error

	// =========================================================================
	// Billing Methods
	// =========================================================================

	// UpdateStripeCustomer saves the Stripe customer ID for a user.
	UpdateStripeCustomer(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error

	// UpdateSubscription updates a user's subscription status, tier, and ID.
	UpdateSubscription(ctx context.Context, userID uuid.UUID, status, tier, subscriptionID string) error

	// GetByStripeCustomerID retrieves a user by their Stripe customer ID.
	// Returns domain.ENOTFOUND if no user has that customer ID.
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.User, error)
}

// =============================================================================
// Implementation
// =============================================================================

// userService is the concrete implementation of UserService.
type userService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewUserService creates a new UserService instance.
//
// Dependencies:
// - queries: sqlc-generated database queries
// - logger: structured logger for operation logging
func NewUserService(queries *repository.Queries, logger *slog.Logger) UserService {
	return &userService{
		queries: queries,
		logger:  logger,
	}
}

// =============================================================================
// Register Implementation
// =============================================================================

// Register creates a new user account with the provided parameters.
//
// Flow:
// 1. Validate input parameters (email format, password strength)
// 2. Check if email already exists
// 3. Hash the password with bcrypt
// 4. Create the user record
// 5. Return the created user (without password hash in response)
//
// Security Considerations:
// - Email uniqueness is checked before hashing to avoid unnecessary work
// - Password is hashed with bcrypt cost 12
// - Timing attacks are mitigated by always hashing even on duplicate email
// - The raw password is never logged or stored
func (s *userService) Register(ctx context.Context, params domain.RegisterParams) (*domain.User, error) {
	const op = "UserService.Register"

	// Validate and normalize input
	params.Email = strings.ToLower(strings.TrimSpace(params.Email))
	params.Name = strings.TrimSpace(params.Name)
	params.CompanyName = strings.TrimSpace(params.CompanyName)
	params.Phone = strings.TrimSpace(params.Phone)

	// Validate email format
	if err := validateEmail(params.Email); err != nil {
		return nil, domain.Wrap(err, domain.EINVALID, op, "Invalid email address")
	}

	// Validate name
	if params.Name == "" {
		return nil, domain.Invalid(op, "Name is required")
	}

	// Validate password
	if err := validatePassword(params.Password); err != nil {
		return nil, domain.Wrap(err, domain.EINVALID, op, "Invalid password")
	}

	// Check if email already exists
	_, err := s.queries.GetUserByEmail(ctx, params.Email)
	if err == nil {
		// User exists - to prevent timing attacks, we hash the password anyway
		_, _ = bcrypt.GenerateFromPassword([]byte(params.Password), BcryptCost)
		return nil, domain.Conflict(op, "Email already registered")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// Unexpected database error
		return nil, domain.Internal(err, op, "Failed to check email availability")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.Password), BcryptCost)
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to hash password")
	}

	// Create user in database
	repoUser, err := s.queries.CreateUser(ctx, repository.CreateUserParams{
		Email:        params.Email,
		PasswordHash: string(passwordHash),
		Name:         params.Name,
		CompanyName:  domain.ToNullString(params.CompanyName),
		Phone:        domain.ToNullString(params.Phone),
	})
	if err != nil {
		// Check for unique constraint violation (race condition)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, domain.Conflict(op, "Email already registered")
		}
		return nil, domain.Internal(err, op, "Failed to create user")
	}

	// Convert to domain user
	user := repoUserToDomain(repoUser)

	// Clear password hash before returning (security precaution)
	user.PasswordHash = ""

	// Log successful registration
	s.logger.Info("user registered", "user_id", user.ID, "email", user.Email)

	return user, nil
}

// =============================================================================
// Login Implementation
// =============================================================================

// Login authenticates a user and creates a new session.
//
// Flow:
// 1. Look up user by email
// 2. Compare password hash using bcrypt
// 3. Generate cryptographically secure session token
// 4. Hash the session token with SHA-256
// 5. Store the hashed token in database
// 6. Return user and raw token
//
// Security Considerations:
// - Constant-time password comparison via bcrypt
// - Generic error message prevents email enumeration
// - Session token is only returned once (not stored anywhere in plaintext)
// - Token is hashed before storage (if DB is compromised, tokens are useless)
func (s *userService) Login(ctx context.Context, email, password string) (*domain.LoginResult, error) {
	const op = "UserService.Login"

	// Normalize email to lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	// Get user by email
	repoUser, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		// If user not found, still do a bcrypt comparison to prevent timing attacks
		if errors.Is(err, sql.ErrNoRows) {
			// Use a dummy hash to maintain constant time
			dummyHash := "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW" // bcrypt hash of "dummy"
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
			return nil, domain.Unauthorized(op, "Invalid email or password")
		}
		// Other database error
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(repoUser.PasswordHash), []byte(password))
	if err != nil {
		// Password mismatch - use same error message as user not found
		return nil, domain.Unauthorized(op, "Invalid email or password")
	}

	// Generate session token
	token, err := generateSessionToken()
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to generate session token")
	}

	// Hash session token for storage
	tokenHash := hashSessionToken(token)

	// Calculate session expiration
	expiresAt := time.Now().Add(SessionDuration)

	// Create session in database
	_, err = s.queries.CreateSession(ctx, repository.CreateSessionParams{
		UserID:    repoUser.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to create session")
	}

	// Convert to domain user
	user := repoUserToDomain(repoUser)

	// Clear password hash before returning
	user.PasswordHash = ""

	// Log successful login
	s.logger.Info("user logged in", "user_id", user.ID, "email", user.Email)

	// Return result with user and RAW token (not hash)
	return &domain.LoginResult{
		User:  user,
		Token: token,
	}, nil
}

// =============================================================================
// Logout Implementation
// =============================================================================

// Logout invalidates a session.
//
// Flow:
// 1. Hash the provided raw token
// 2. Delete the session from database
//
// This operation is idempotent - calling with an invalid or already-deleted
// token simply does nothing and returns success.
func (s *userService) Logout(ctx context.Context, token string) error {
	// Validate token format
	if token == "" {
		return nil // Idempotent - empty token is fine
	}

	// Check token length (should be 64 hex characters)
	if len(token) != 64 {
		return nil // Invalid token, but logout is idempotent
	}

	// Hash the token
	tokenHash := hashSessionToken(token)

	// Delete session from database
	err := s.queries.DeleteSession(ctx, tokenHash)
	if err != nil {
		// Ignore not found errors - logout is idempotent
		// Log other errors but don't fail the operation
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Warn("failed to delete session", "error", err)
		}
	}

	// Log logout
	s.logger.Debug("session invalidated")

	return nil
}

// =============================================================================
// GetByID Implementation
// =============================================================================

// GetByID retrieves a user by their ID.
func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const op = "UserService.GetByID"

	// Get user from database
	repoUser, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "user", id.String())
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// Convert to domain user
	user := repoUserToDomain(repoUser)

	// Clear password hash for security
	user.PasswordHash = ""

	return user, nil
}

// =============================================================================
// GetBySessionToken Implementation
// =============================================================================

// GetBySessionToken retrieves a user by their session token.
//
// Flow:
// 1. Hash the provided raw token
// 2. Look up session by token hash
// 3. Verify session is not expired (database query handles this)
// 4. Look up associated user
// 5. Return user
//
// Security Considerations:
// - Token is hashed before database lookup
// - Expired sessions are rejected at database level
func (s *userService) GetBySessionToken(ctx context.Context, token string) (*domain.User, error) {
	const op = "UserService.GetBySessionToken"

	// Validate token format
	if token == "" {
		return nil, domain.Unauthorized(op, "Invalid or expired session")
	}

	// Check token length (should be 64 hex characters)
	if len(token) != 64 {
		return nil, domain.Unauthorized(op, "Invalid or expired session")
	}

	// Hash the token
	tokenHash := hashSessionToken(token)

	// Get session by token hash (query already filters expired sessions)
	session, err := s.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.Unauthorized(op, "Invalid or expired session")
		}
		return nil, domain.Internal(err, op, "Failed to retrieve session")
	}

	// Get user by session's user_id
	repoUser, err := s.queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Unlikely but possible if user was deleted
			return nil, domain.Unauthorized(op, "Invalid or expired session")
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// Convert to domain user
	user := repoUserToDomain(repoUser)

	// Clear password hash for security
	user.PasswordHash = ""

	return user, nil
}

// =============================================================================
// UpdateProfile Implementation
// =============================================================================

// UpdateProfile updates a user's profile information.
func (s *userService) UpdateProfile(ctx context.Context, params domain.ProfileUpdateParams) error {
	const op = "UserService.UpdateProfile"

	// Validate and normalize input
	params.Name = strings.TrimSpace(params.Name)
	params.Phone = strings.TrimSpace(params.Phone)

	// Validate name
	if params.Name == "" {
		return domain.Invalid(op, "Name is required")
	}

	// Verify user exists
	user, err := s.queries.GetUserByID(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "user", params.UserID.String())
		}
		return domain.Internal(err, op, "Failed to retrieve user")
	}

	// Update user profile (preserve existing company_name)
	err = s.queries.UpdateUserProfile(ctx, repository.UpdateUserProfileParams{
		ID:          params.UserID,
		Name:        params.Name,
		CompanyName: user.CompanyName,
		Phone:       domain.ToNullString(params.Phone),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update profile")
	}

	// Log update
	s.logger.Info("user profile updated", "user_id", params.UserID)

	return nil
}

// =============================================================================
// UpdateBusinessProfile Implementation
// =============================================================================

// UpdateBusinessProfile updates a user's business profile information.
func (s *userService) UpdateBusinessProfile(ctx context.Context, params domain.BusinessProfileUpdateParams) error {
	const op = "UserService.UpdateBusinessProfile"

	// Normalize input
	params.BusinessName = strings.TrimSpace(params.BusinessName)
	params.BusinessEmail = strings.TrimSpace(params.BusinessEmail)
	params.BusinessPhone = strings.TrimSpace(params.BusinessPhone)
	params.AddressLine1 = strings.TrimSpace(params.AddressLine1)
	params.AddressLine2 = strings.TrimSpace(params.AddressLine2)
	params.City = strings.TrimSpace(params.City)
	params.State = strings.TrimSpace(params.State)
	params.PostalCode = strings.TrimSpace(params.PostalCode)
	params.LicenseNumber = strings.TrimSpace(params.LicenseNumber)

	// Verify user exists
	_, err := s.queries.GetUserByID(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "user", params.UserID.String())
		}
		return domain.Internal(err, op, "Failed to retrieve user")
	}

	// Update user business profile
	err = s.queries.UpdateUserBusinessProfile(ctx, repository.UpdateUserBusinessProfileParams{
		ID:                    params.UserID,
		BusinessName:          domain.ToNullString(params.BusinessName),
		BusinessEmail:         domain.ToNullString(params.BusinessEmail),
		BusinessPhone:         domain.ToNullString(params.BusinessPhone),
		BusinessAddressLine1:  domain.ToNullString(params.AddressLine1),
		BusinessAddressLine2:  domain.ToNullString(params.AddressLine2),
		BusinessCity:          domain.ToNullString(params.City),
		BusinessState:         domain.ToNullString(params.State),
		BusinessPostalCode:    domain.ToNullString(params.PostalCode),
		BusinessLicenseNumber: domain.ToNullString(params.LicenseNumber),
		BusinessLogoUrl:       domain.ToNullString(params.LogoURL),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update business profile")
	}

	// Log update
	s.logger.Info("user business profile updated", "user_id", params.UserID)

	return nil
}

// =============================================================================
// ChangePassword Implementation
// =============================================================================

// ChangePassword changes a user's password.
//
// Flow:
// 1. Get current user to verify current password
// 2. Validate current password
// 3. Validate new password meets requirements
// 4. Hash new password
// 5. Update password in database
// 6. Invalidate all existing sessions for security
//
// Security Considerations:
// - Current password must be verified to prevent session hijacking attacks
// - All sessions are invalidated to force re-authentication
// - New password is validated for strength requirements
func (s *userService) ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error {
	const op = "UserService.ChangePassword"

	// Validate new password
	if err := validatePassword(params.NewPassword); err != nil {
		return domain.Wrap(err, domain.EINVALID, op, "Invalid new password")
	}

	// Get user to verify current password
	repoUser, err := s.queries.GetUserByID(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "user", params.UserID.String())
		}
		return domain.Internal(err, op, "Failed to retrieve user")
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(repoUser.PasswordHash), []byte(params.CurrentPassword))
	if err != nil {
		return domain.Unauthorized(op, "Current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(params.NewPassword), BcryptCost)
	if err != nil {
		return domain.Internal(err, op, "Failed to hash new password")
	}

	// Update password in database
	err = s.queries.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           params.UserID,
		PasswordHash: string(newPasswordHash),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update password")
	}

	// Invalidate all user sessions (force re-authentication)
	err = s.queries.DeleteUserSessions(ctx, params.UserID)
	if err != nil {
		// Log but don't fail - password was changed successfully
		s.logger.Warn("failed to delete user sessions after password change", "user_id", params.UserID, "error", err)
	}

	// Log password change
	s.logger.Info("user password changed", "user_id", params.UserID)

	return nil
}

// =============================================================================
// DeleteExpiredSessions Implementation
// =============================================================================

// DeleteExpiredSessions removes all expired sessions.
// This should be called periodically as a maintenance task.
func (s *userService) DeleteExpiredSessions(ctx context.Context) error {
	const op = "UserService.DeleteExpiredSessions"

	// Delete expired sessions
	err := s.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return domain.Internal(err, op, "Failed to delete expired sessions")
	}

	// Log cleanup
	s.logger.Info("expired sessions cleaned up")

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateSessionToken creates a cryptographically secure session token.
//
// The token is generated using crypto/rand and returned as a hex-encoded string.
// This provides 256 bits of entropy (32 bytes * 8 bits/byte).
//
// Returns:
// - 64-character hex string representing 32 random bytes
// - Error if crypto/rand fails (extremely rare, indicates system issue)
func generateSessionToken() (string, error) {
	bytes := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashSessionToken creates a SHA-256 hash of a session token.
//
// We hash session tokens before storing them because:
//  1. If the database is compromised, attackers cannot use the hashes directly
//  2. SHA-256 is fast enough for per-request validation
//  3. Unlike passwords, session tokens are high-entropy random values,
//     so SHA-256 is sufficient (bcrypt would be overkill and slow)
//
// Parameters:
// - token: The raw session token (64-char hex string)
//
// Returns:
// - 64-character hex string representing SHA-256 hash
func hashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// repoUserToDomain converts a repository.User to domain.User.
//
// This handles the conversion from database types (sql.Null*) to Go types,
// making the domain model easier to work with in business logic.
func repoUserToDomain(u repository.User) *domain.User {
	var emailVerifiedAt *time.Time
	if u.EmailVerifiedAt.Valid {
		emailVerifiedAt = &u.EmailVerifiedAt.Time
	}

	var createdAt time.Time
	if u.CreatedAt.Valid {
		createdAt = u.CreatedAt.Time
	}

	var updatedAt time.Time
	if u.UpdatedAt.Valid {
		updatedAt = u.UpdatedAt.Time
	}

	return &domain.User{
		ID:                 u.ID,
		Email:              u.Email,
		PasswordHash:       u.PasswordHash,
		Name:               u.Name,
		CompanyName:        domain.NullStringValue(u.CompanyName),
		Phone:              domain.NullStringValue(u.Phone),
		StripeCustomerID:   domain.NullStringValue(u.StripeCustomerID),
		SubscriptionStatus: domain.SubscriptionStatus(domain.NullStringValue(u.SubscriptionStatus)),
		SubscriptionTier:   domain.SubscriptionTier(domain.NullStringValue(u.SubscriptionTier)),
		SubscriptionID:     domain.NullStringValue(u.SubscriptionID),
		EmailVerified:      domain.NullBoolValue(u.EmailVerified),
		EmailVerifiedAt:    emailVerifiedAt,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,

		// Business profile fields
		BusinessName:          domain.NullStringValue(u.BusinessName),
		BusinessEmail:         domain.NullStringValue(u.BusinessEmail),
		BusinessPhone:         domain.NullStringValue(u.BusinessPhone),
		BusinessAddressLine1:  domain.NullStringValue(u.BusinessAddressLine1),
		BusinessAddressLine2:  domain.NullStringValue(u.BusinessAddressLine2),
		BusinessCity:          domain.NullStringValue(u.BusinessCity),
		BusinessState:         domain.NullStringValue(u.BusinessState),
		BusinessPostalCode:    domain.NullStringValue(u.BusinessPostalCode),
		BusinessLicenseNumber: domain.NullStringValue(u.BusinessLicenseNumber),
		BusinessLogoURL:       domain.NullStringValue(u.BusinessLogoUrl),
	}
}

// validateEmail validates an email address format.
//
// Checks:
// - Basic format validation (contains @, has domain)
// - Length limits (RFC 5321: 254 chars max)
// - Normalization (lowercase, trim whitespace)
func validateEmail(email string) error {
	if email == "" {
		return domain.Invalid("", "Email is required")
	}

	// Check length limit
	if len(email) > 254 {
		return domain.Invalid("", "Email must be 254 characters or less")
	}

	// Basic format validation
	// Must contain exactly one @, and domain part must have a dot
	atIndex := -1
	atCount := 0
	for i, c := range email {
		if c == '@' {
			atCount++
			atIndex = i
		}
	}

	if atCount != 1 {
		return domain.Invalid("", "Email must contain exactly one @ symbol")
	}

	if atIndex == 0 {
		return domain.Invalid("", "Email cannot start with @")
	}

	if atIndex == len(email)-1 {
		return domain.Invalid("", "Email cannot end with @")
	}

	// Check for domain part with at least one dot
	domainPart := email[atIndex+1:]
	hasDot := false
	for _, c := range domainPart {
		if c == '.' {
			hasDot = true
			break
		}
	}

	if !hasDot {
		return domain.Invalid("", "Email domain must contain a dot")
	}

	// Don't allow consecutive dots
	if contains(email, "..") {
		return domain.Invalid("", "Email cannot contain consecutive dots")
	}

	return nil
}

// validatePassword validates password strength requirements.
//
// Rules:
// - Minimum length: 8 characters (NIST SP 800-63B)
// - Maximum length: 72 characters (bcrypt limit)
func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return domain.Invalid("", "Password must be at least 8 characters")
	}

	if len(password) > MaxPasswordLength {
		return domain.Invalid("", "Password must be 72 characters or less")
	}

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Email Verification Token Implementation
// =============================================================================

// CreateEmailVerificationToken creates a new email verification token for a user.
//
// Flow:
// 1. Delete any existing verification tokens for user (one token per user)
// 2. Generate cryptographically secure random token
// 3. Hash token with SHA-256 for storage
// 4. Store hashed token with expiration
// 5. Return raw token (for email) and expiration
//
// Security Considerations:
// - Raw token is returned only once (not stored anywhere in plaintext)
// - Token is hashed before storage using same pattern as session tokens
// - Caller is responsible for sending the raw token via email
func (s *userService) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID) (*domain.EmailVerificationResult, error) {
	const op = "UserService.CreateEmailVerificationToken"

	// 1. Verify user exists
	_, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "user", userID.String())
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// 2. Delete existing tokens for user (enforce one-token-per-user)
	err = s.queries.DeleteUserEmailVerificationTokens(ctx, userID)
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to delete existing tokens")
	}

	// 3. Generate random token (reuse generateSessionToken helper)
	rawToken, err := generateSessionToken()
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to generate token")
	}

	// 4. Hash the token
	tokenHash := hashSessionToken(rawToken)

	// 5. Calculate expiration
	expiresAt := time.Now().Add(domain.EmailVerificationTokenDuration)

	// 6. Store in database
	_, err = s.queries.CreateEmailVerificationToken(ctx, repository.CreateEmailVerificationTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to create verification token")
	}

	// Log token creation
	s.logger.Info("email verification token created", "user_id", userID)

	// 7. Return result with raw token
	return &domain.EmailVerificationResult{
		Token:     rawToken,
		ExpiresAt: expiresAt,
		UserID:    userID,
	}, nil
}

// VerifyEmail validates an email verification token and marks the user as verified.
//
// Flow:
// 1. Hash the provided raw token
// 2. Look up token by hash (query filters expired tokens)
// 3. Verify user is not already verified (prevent double-verification)
// 4. Mark user as email verified with timestamp
// 5. Delete the used token
//
// Security Considerations:
// - Token lookup is by hash, not raw token
// - Expired tokens are filtered at query level
// - Token is deleted after use (one-time use)
func (s *userService) VerifyEmail(ctx context.Context, token string) error {
	const op = "UserService.VerifyEmail"

	// 1. Validate token format (should be 64 hex chars)
	if len(token) != 64 {
		return domain.Invalid(op, "Invalid verification token")
	}

	// 2. Hash the token
	tokenHash := hashSessionToken(token)

	// 3. Look up token (query filters expired)
	verificationToken, err := s.queries.GetEmailVerificationTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "verification token", "")
		}
		return domain.Internal(err, op, "Failed to retrieve verification token")
	}

	// 4. Get the user
	user, err := s.queries.GetUserByID(ctx, verificationToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "user", verificationToken.UserID.String())
		}
		return domain.Internal(err, op, "Failed to retrieve user")
	}

	// 5. Check if already verified
	if user.EmailVerified.Valid && user.EmailVerified.Bool {
		return domain.Conflict(op, "Email is already verified")
	}

	// 6. Mark user as verified
	err = s.queries.UpdateUserEmailVerification(ctx, repository.UpdateUserEmailVerificationParams{
		ID:              user.ID,
		EmailVerified:   sql.NullBool{Bool: true, Valid: true},
		EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to mark email as verified")
	}

	// 7. Delete the used token
	err = s.queries.DeleteEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		// Log but don't fail - verification already succeeded
		s.logger.Warn("failed to delete verification token after use", "error", err, "user_id", user.ID)
	}

	// 8. Log success
	s.logger.Info("email verified", "user_id", user.ID, "email", user.Email)

	return nil
}

// ResendVerificationEmail creates a new verification token for an unverified user.
//
// Flow:
// 1. Look up user by email
// 2. Check if already verified
// 3. Create new verification token
//
// Security Considerations:
// - Returns error if user not found (caller should use generic message to user)
// - Returns error if already verified (no need to spam verified users)
func (s *userService) ResendVerificationEmail(ctx context.Context, email string) (*domain.EmailVerificationResult, error) {
	const op = "UserService.ResendVerificationEmail"

	// 1. Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// 2. Look up user
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "user", email)
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// 3. Check if already verified
	if user.EmailVerified.Valid && user.EmailVerified.Bool {
		return nil, domain.Conflict(op, "Email is already verified")
	}

	// 4. Create new token (delegates to CreateEmailVerificationToken)
	return s.CreateEmailVerificationToken(ctx, user.ID)
}

// DeleteExpiredEmailVerificationTokens removes all expired email verification tokens.
func (s *userService) DeleteExpiredEmailVerificationTokens(ctx context.Context) error {
	const op = "UserService.DeleteExpiredEmailVerificationTokens"

	err := s.queries.DeleteExpiredEmailVerificationTokens(ctx)
	if err != nil {
		return domain.Internal(err, op, "Failed to delete expired tokens")
	}

	s.logger.Info("expired email verification tokens cleaned up")
	return nil
}

// =============================================================================
// Password Reset Token Implementation
// =============================================================================

// CreatePasswordResetToken creates a new password reset token for a user.
//
// Flow:
// 1. Look up user by email
// 2. Delete any existing reset tokens for user
// 3. Generate and hash new token
// 4. Store token with short expiration (1 hour)
// 5. Return raw token for email
//
// Security Considerations:
//   - Returns NotFound if email doesn't exist, but caller should NOT expose this
//     to end user (always show "if account exists, we sent an email" message)
//   - Shorter expiration than email verification (1 hour vs 24 hours)
//   - Old tokens are deleted before creating new one
func (s *userService) CreatePasswordResetToken(ctx context.Context, email string) (*domain.PasswordResetResult, error) {
	const op = "UserService.CreatePasswordResetToken"

	// 1. Normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// 2. Look up user
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "user", email)
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user")
	}

	// 3. Delete existing tokens
	err = s.queries.DeleteUserPasswordResetTokens(ctx, user.ID)
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to delete existing tokens")
	}

	// 4. Generate random token
	rawToken, err := generateSessionToken()
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to generate token")
	}

	// 5. Hash the token
	tokenHash := hashSessionToken(rawToken)

	// 6. Calculate expiration (shorter than email verification)
	expiresAt := time.Now().Add(domain.PasswordResetTokenDuration)

	// 7. Store in database
	_, err = s.queries.CreatePasswordResetToken(ctx, repository.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "Failed to create password reset token")
	}

	// Log token creation
	s.logger.Info("password reset token created", "user_id", user.ID, "email", user.Email)

	// 8. Return result
	return &domain.PasswordResetResult{
		Token:     rawToken,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
	}, nil
}

// ValidatePasswordResetToken checks if a password reset token is valid.
//
// Flow:
// 1. Hash the provided token
// 2. Look up by hash (query filters expired and used tokens)
// 3. Return user ID if valid
//
// Security Considerations:
// - Query filters both expired AND used tokens
// - Does not mark token as used (that happens in ResetPassword)
// - Used to validate before showing the password form
func (s *userService) ValidatePasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	const op = "UserService.ValidatePasswordResetToken"

	// 1. Validate token format
	if len(token) != 64 {
		return uuid.Nil, domain.Invalid(op, "Invalid reset token")
	}

	// 2. Hash the token
	tokenHash := hashSessionToken(token)

	// 3. Look up token (query filters expired and used)
	resetToken, err := s.queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, domain.NotFound(op, "reset token", "")
		}
		return uuid.Nil, domain.Internal(err, op, "Failed to retrieve reset token")
	}

	// 4. Return user ID
	return resetToken.UserID, nil
}

// ResetPassword validates the token and updates the user's password.
//
// Flow:
// 1. Validate token (same as ValidatePasswordResetToken)
// 2. Validate new password meets requirements
// 3. Hash new password with bcrypt
// 4. Update user's password
// 5. Mark token as used (not deleted, for audit trail)
// 6. Invalidate all user sessions
//
// Security Considerations:
// - Token is validated again (race condition protection)
// - Token is marked used, not deleted (audit trail)
// - All sessions are invalidated (force re-authentication)
// - New password is validated for strength
func (s *userService) ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error {
	const op = "UserService.ResetPassword"

	// 1. Validate token format
	if len(params.Token) != 64 {
		return domain.Invalid(op, "Invalid reset token")
	}

	// 2. Validate new password
	if err := validatePassword(params.NewPassword); err != nil {
		return domain.Wrap(err, domain.EINVALID, op, "Invalid new password")
	}

	// 3. Hash the token
	tokenHash := hashSessionToken(params.Token)

	// 4. Look up token
	resetToken, err := s.queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "reset token", "")
		}
		return domain.Internal(err, op, "Failed to retrieve reset token")
	}

	// 5. Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.NewPassword), BcryptCost)
	if err != nil {
		return domain.Internal(err, op, "Failed to hash new password")
	}

	// 6. Update password
	err = s.queries.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           resetToken.UserID,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update password")
	}

	// 7. Mark token as used
	err = s.queries.MarkPasswordResetTokenUsed(ctx, tokenHash)
	if err != nil {
		// Log but don't fail - password was already changed
		s.logger.Warn("failed to mark reset token as used", "error", err, "user_id", resetToken.UserID)
	}

	// 8. Invalidate all sessions
	err = s.queries.DeleteUserSessions(ctx, resetToken.UserID)
	if err != nil {
		// Log but don't fail - password was changed successfully
		s.logger.Warn("failed to delete user sessions after password reset", "error", err, "user_id", resetToken.UserID)
	}

	// 9. Log success
	s.logger.Info("password reset completed", "user_id", resetToken.UserID)

	return nil
}

// DeleteExpiredPasswordResetTokens removes all expired password reset tokens.
func (s *userService) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	const op = "UserService.DeleteExpiredPasswordResetTokens"

	err := s.queries.DeleteExpiredPasswordResetTokens(ctx)
	if err != nil {
		return domain.Internal(err, op, "Failed to delete expired tokens")
	}

	s.logger.Info("expired password reset tokens cleaned up")
	return nil
}

// =============================================================================
// Billing Methods Implementation
// =============================================================================

// UpdateStripeCustomer saves the Stripe customer ID for a user.
func (s *userService) UpdateStripeCustomer(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error {
	const op = "UserService.UpdateStripeCustomer"

	err := s.queries.UpdateUserStripeCustomer(ctx, repository.UpdateUserStripeCustomerParams{
		ID:               userID,
		StripeCustomerID: domain.ToNullString(stripeCustomerID),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update Stripe customer ID")
	}

	s.logger.Info("stripe customer ID updated", "user_id", userID, "stripe_customer_id", stripeCustomerID)
	return nil
}

// UpdateSubscription updates a user's subscription status, tier, and subscription ID.
func (s *userService) UpdateSubscription(ctx context.Context, userID uuid.UUID, status, tier, subscriptionID string) error {
	const op = "UserService.UpdateSubscription"

	err := s.queries.UpdateUserSubscription(ctx, repository.UpdateUserSubscriptionParams{
		ID:                 userID,
		SubscriptionStatus: domain.ToNullString(status),
		SubscriptionTier:   domain.ToNullString(tier),
		SubscriptionID:     domain.ToNullString(subscriptionID),
	})
	if err != nil {
		return domain.Internal(err, op, "Failed to update subscription")
	}

	s.logger.Info("subscription updated", "user_id", userID, "status", status, "tier", tier)
	return nil
}

// GetByStripeCustomerID retrieves a user by their Stripe customer ID.
func (s *userService) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*domain.User, error) {
	const op = "UserService.GetByStripeCustomerID"

	repoUser, err := s.queries.GetUserByStripeCustomerID(ctx, domain.ToNullString(stripeCustomerID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "user", stripeCustomerID)
		}
		return nil, domain.Internal(err, op, "Failed to retrieve user by Stripe customer ID")
	}

	user := repoUserToDomain(repoUser)
	user.PasswordHash = ""
	return user, nil
}

// =============================================================================
// Compile-time interface check
// =============================================================================

// Ensure userService implements UserService
var _ UserService = (*userService)(nil)
