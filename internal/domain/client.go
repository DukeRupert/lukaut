// Package domain contains core business types and interfaces.
//
// This file defines the Client domain type and related types for
// managing construction company clients.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Client Domain Type
// =============================================================================

// Client represents a construction company's client (e.g., general contractor, property owner).
//
// This is the domain representation designed for use in business logic.
// Clients can be associated with multiple Sites.
type Client struct {
	ID           uuid.UUID // Unique identifier
	UserID       uuid.UUID // Owner of the client record
	Name         string    // Client company name
	Email        string    // Primary contact email
	Phone        string    // Primary contact phone
	AddressLine1 string    // Corporate HQ address line 1
	AddressLine2 string    // Corporate HQ address line 2 (optional)
	City         string    // Corporate HQ city
	State        string    // Corporate HQ state
	PostalCode   string    // Corporate HQ postal code
	Notes        string    // Optional notes about this client
	CreatedAt    time.Time // When client was created
	UpdatedAt    time.Time // When client was last modified

	// Computed fields (not stored in database, populated by queries/services)
	SiteCount int // Number of sites associated with this client
}

// HasAddress returns true if the client has any address information.
func (c *Client) HasAddress() bool {
	return c.AddressLine1 != "" || c.City != "" || c.State != ""
}

// FullAddress returns a formatted single-line address.
func (c *Client) FullAddress() string {
	if c.AddressLine1 == "" {
		return ""
	}
	addr := c.AddressLine1
	if c.AddressLine2 != "" {
		addr += ", " + c.AddressLine2
	}
	if c.City != "" {
		addr += ", " + c.City
	}
	if c.State != "" {
		addr += ", " + c.State
	}
	if c.PostalCode != "" {
		addr += " " + c.PostalCode
	}
	return addr
}

// =============================================================================
// Client Service Parameters
// =============================================================================

// CreateClientParams contains validated parameters for creating a client.
type CreateClientParams struct {
	UserID       uuid.UUID // Owner of the client (from auth context)
	Name         string    // Required: Client company name
	Email        string    // Optional: Primary contact email
	Phone        string    // Optional: Primary contact phone
	AddressLine1 string    // Optional: Corporate HQ address
	AddressLine2 string    // Optional
	City         string    // Optional
	State        string    // Optional
	PostalCode   string    // Optional
	Notes        string    // Optional
}

// UpdateClientParams contains validated parameters for updating a client.
type UpdateClientParams struct {
	ID           uuid.UUID // Client to update
	UserID       uuid.UUID // Owner (for authorization)
	Name         string    // Required: Client company name
	Email        string    // Optional
	Phone        string    // Optional
	AddressLine1 string    // Optional
	AddressLine2 string    // Optional
	City         string    // Optional
	State        string    // Optional
	PostalCode   string    // Optional
	Notes        string    // Optional
}

// ListClientsParams contains parameters for listing clients.
type ListClientsParams struct {
	UserID uuid.UUID // Filter by user
	Limit  int32     // Max results to return
	Offset int32     // Number of results to skip
}

// =============================================================================
// List Result with Pagination
// =============================================================================

// ListClientsResult contains the result of a paginated client list query.
type ListClientsResult struct {
	Clients []Client // The client results
	Total   int64    // Total number of clients (for pagination)
	Limit   int32    // Number of results requested
	Offset  int32    // Number of results skipped
}

// HasMore returns true if there are more results available.
func (r *ListClientsResult) HasMore() bool {
	return int64(r.Offset+r.Limit) < r.Total
}

// HasPrevious returns true if there are previous results available.
func (r *ListClientsResult) HasPrevious() bool {
	return r.Offset > 0
}

// CurrentPage returns the current page number (1-indexed).
func (r *ListClientsResult) CurrentPage() int {
	if r.Limit == 0 {
		return 1
	}
	return int(r.Offset/r.Limit) + 1
}

// TotalPages returns the total number of pages.
func (r *ListClientsResult) TotalPages() int {
	if r.Limit == 0 {
		return 1
	}
	pages := r.Total / int64(r.Limit)
	if r.Total%int64(r.Limit) > 0 {
		pages++
	}
	return int(pages)
}
