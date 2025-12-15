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

	// ChangePassword changes a user's password.
	// Validates the current password before allowing the change.
	// Invalidates all existing sessions after password change.
	// Returns domain.EUNAUTHORIZED if current password is wrong.
	ChangePassword(ctx context.Context, params domain.PasswordChangeParams) error

	// DeleteExpiredSessions removes all expired sessions from the database.
	// This should be called periodically (e.g., daily) to clean up.
	DeleteExpiredSessions(ctx context.Context) error
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
	const op = "UserService.Logout"

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
	params.CompanyName = strings.TrimSpace(params.CompanyName)
	params.Phone = strings.TrimSpace(params.Phone)

	// Validate name
	if params.Name == "" {
		return domain.Invalid(op, "Name is required")
	}

	// Verify user exists
	_, err := s.queries.GetUserByID(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "user", params.UserID.String())
		}
		return domain.Internal(err, op, "Failed to retrieve user")
	}

	// Update user profile
	err = s.queries.UpdateUserProfile(ctx, repository.UpdateUserProfileParams{
		ID:          params.UserID,
		Name:        params.Name,
		CompanyName: domain.ToNullString(params.CompanyName),
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
// 1. If the database is compromised, attackers cannot use the hashes directly
// 2. SHA-256 is fast enough for per-request validation
// 3. Unlike passwords, session tokens are high-entropy random values,
//    so SHA-256 is sufficient (bcrypt would be overkill and slow)
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
// Compile-time interface check
// =============================================================================

// Ensure userService implements UserService
var _ UserService = (*userService)(nil)
