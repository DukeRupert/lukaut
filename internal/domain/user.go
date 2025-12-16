// Package domain contains core business types and interfaces.
//
// This file defines the User domain type and related types for authentication.
// These types are separate from the repository models to allow for business logic
// enrichment and to decouple the domain layer from the database layer.
package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the possible states of a user's subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusInactive  SubscriptionStatus = "inactive"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled  SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid    SubscriptionStatus = "unpaid"
)

// SubscriptionTier represents the pricing tier of a subscription.
type SubscriptionTier string

const (
	SubscriptionTierStarter      SubscriptionTier = "starter"
	SubscriptionTierProfessional SubscriptionTier = "professional"
)

// User represents a registered user of the Lukaut platform.
//
// This is the domain representation of a user, designed for use in business logic.
// It differs from repository.User in that:
// - It uses proper Go types instead of sql.Null* types where appropriate
// - It provides helper methods for common checks
// - It can be extended with computed properties without affecting the database layer
type User struct {
	ID                 uuid.UUID
	Email              string
	PasswordHash       string // Never expose this in API responses
	Name               string
	CompanyName        string
	Phone              string
	StripeCustomerID   string
	SubscriptionStatus SubscriptionStatus
	SubscriptionTier   SubscriptionTier
	SubscriptionID     string
	EmailVerified      bool
	EmailVerifiedAt    *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsActive returns true if the user has an active subscription or is trialing.
func (u *User) IsActive() bool {
	return u.SubscriptionStatus == SubscriptionStatusActive ||
		u.SubscriptionStatus == SubscriptionStatusTrialing
}

// IsProfessional returns true if the user is on the professional tier.
func (u *User) IsProfessional() bool {
	return u.SubscriptionTier == SubscriptionTierProfessional
}

// CanGenerateReports returns true if the user can generate reports.
// This checks both subscription status and tier limits.
func (u *User) CanGenerateReports() bool {
	return u.IsActive()
}

// DisplayName returns the user's name or email if name is empty.
func (u *User) DisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Email
}

// Session represents an authenticated session.
//
// Sessions are stored in the database with a hashed token.
// The raw token is only given to the client once (at login).
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string    // SHA-256 hash of the session token
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// RegisterParams contains the validated parameters for user registration.
type RegisterParams struct {
	Email       string
	Password    string // Raw password, will be hashed by service
	Name        string
	CompanyName string // Optional
	Phone       string // Optional
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	User  *User
	Token string // Raw session token (not hashed) - only returned once
}

// PasswordChangeParams contains parameters for changing a user's password.
type PasswordChangeParams struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ProfileUpdateParams contains parameters for updating a user's profile.
type ProfileUpdateParams struct {
	UserID      uuid.UUID
	Name        string
	CompanyName string
	Phone       string
}

// =============================================================================
// Conversion helpers from repository types
// =============================================================================

// NullStringValue safely extracts a string from sql.NullString.
func NullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// NullTimeValue safely extracts a time pointer from sql.NullTime.
func NullTimeValue(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// NullBoolValue safely extracts a bool from sql.NullBool.
func NullBoolValue(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false
}

// ToNullString converts a string to sql.NullString.
func ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// ToNullUUID converts a uuid pointer to uuid.NullUUID.
func ToNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
