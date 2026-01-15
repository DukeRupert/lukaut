package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Site represents a construction site location.
type Site struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Name             string
	AddressLine1     string
	AddressLine2     string
	City             string
	State            string
	PostalCode       string
	ClientName       string // Deprecated: use ClientID instead
	ClientEmail      string // Deprecated: use ClientID instead
	ClientPhone      string // Deprecated: use ClientID instead
	Notes            string
	ClientID         *uuid.UUID // FK to clients table
	LinkedClientName string     // Joined from clients table
	Client           *Client    // Joined client data (optional)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FullAddress returns the formatted multi-line address.
func (s *Site) FullAddress() string {
	lines := []string{s.AddressLine1}
	if s.AddressLine2 != "" {
		lines = append(lines, s.AddressLine2)
	}
	lines = append(lines, fmt.Sprintf("%s, %s %s", s.City, s.State, s.PostalCode))
	return strings.Join(lines, "\n")
}

// CityStateZip returns a single-line location summary.
func (s *Site) CityStateZip() string {
	return fmt.Sprintf("%s, %s %s", s.City, s.State, s.PostalCode)
}

// CreateSiteParams contains parameters for creating a site.
type CreateSiteParams struct {
	UserID       uuid.UUID
	Name         string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	ClientName   string // Deprecated: use ClientID instead
	ClientEmail  string // Deprecated: use ClientID instead
	ClientPhone  string // Deprecated: use ClientID instead
	Notes        string
	ClientID     *uuid.UUID
}

// UpdateSiteParams contains parameters for updating a site.
type UpdateSiteParams struct {
	ID           uuid.UUID
	UserID       uuid.UUID // For authorization
	Name         string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	ClientName   string // Deprecated: use ClientID instead
	ClientEmail  string // Deprecated: use ClientID instead
	ClientPhone  string // Deprecated: use ClientID instead
	Notes        string
	ClientID     *uuid.UUID
}
