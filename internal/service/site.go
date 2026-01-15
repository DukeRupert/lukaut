package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
)

// SiteService defines the interface for site-related operations.
type SiteService interface {
	// Create creates a new site for the user.
	Create(ctx context.Context, params domain.CreateSiteParams) (*domain.Site, error)

	// GetByID retrieves a site by ID, verifying user ownership.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Site, error)

	// List retrieves all sites for a user, ordered by name.
	List(ctx context.Context, userID uuid.UUID) ([]domain.Site, error)

	// Update updates an existing site.
	Update(ctx context.Context, params domain.UpdateSiteParams) error

	// Delete deletes a site.
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// siteService implements SiteService.
type siteService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewSiteService creates a new SiteService.
func NewSiteService(queries *repository.Queries, logger *slog.Logger) SiteService {
	return &siteService{
		queries: queries,
		logger:  logger,
	}
}

// Create creates a new site for the user.
func (s *siteService) Create(ctx context.Context, params domain.CreateSiteParams) (*domain.Site, error) {
	const op = "SiteService.Create"

	// Validate required fields
	if err := s.validateSiteParams(params.Name, params.AddressLine1, params.City, params.State, params.PostalCode); err != nil {
		return nil, err
	}

	// Create site in database
	repoSite, err := s.queries.CreateSite(ctx, repository.CreateSiteParams{
		UserID:       params.UserID,
		Name:         strings.TrimSpace(params.Name),
		AddressLine1: strings.TrimSpace(params.AddressLine1),
		AddressLine2: toNullString(params.AddressLine2),
		City:         strings.TrimSpace(params.City),
		State:        strings.TrimSpace(params.State),
		PostalCode:   strings.TrimSpace(params.PostalCode),
		ClientName:   toNullString(params.ClientName),
		ClientEmail:  toNullString(params.ClientEmail),
		ClientPhone:  toNullString(params.ClientPhone),
		Notes:        toNullString(params.Notes),
		ClientID:     toNullUUID(params.ClientID),
	})
	if err != nil {
		s.logger.Error("failed to create site", "error", err, "op", op)
		return nil, domain.Internal(err, op, "Failed to create site")
	}

	site := repoSiteToDomain(repoSite)
	s.logger.Info("site created", "site_id", site.ID, "user_id", site.UserID, "name", site.Name)

	return &site, nil
}

// GetByID retrieves a site by ID, verifying user ownership.
func (s *siteService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Site, error) {
	const op = "SiteService.GetByID"

	repoSite, err := s.queries.GetSiteByIDAndUserID(ctx, repository.GetSiteByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "site", id.String())
		}
		s.logger.Error("failed to get site", "error", err, "op", op, "site_id", id)
		return nil, domain.Internal(err, op, "Failed to retrieve site")
	}

	site := repoSiteToDomain(repoSite)
	return &site, nil
}

// List retrieves all sites for a user, ordered by name.
func (s *siteService) List(ctx context.Context, userID uuid.UUID) ([]domain.Site, error) {
	const op = "SiteService.List"

	repoSites, err := s.queries.ListSitesWithClientByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list sites", "error", err, "op", op, "user_id", userID)
		return nil, domain.Internal(err, op, "Failed to list sites")
	}

	sites := make([]domain.Site, len(repoSites))
	for i, rs := range repoSites {
		sites[i] = repoSiteWithClientToDomain(rs)
	}

	return sites, nil
}

// Update updates an existing site.
func (s *siteService) Update(ctx context.Context, params domain.UpdateSiteParams) error {
	const op = "SiteService.Update"

	// Verify site exists and belongs to user
	_, err := s.queries.GetSiteByIDAndUserID(ctx, repository.GetSiteByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "site", params.ID.String())
		}
		s.logger.Error("failed to get site for update", "error", err, "op", op, "site_id", params.ID)
		return domain.Internal(err, op, "Failed to retrieve site")
	}

	// Validate required fields
	if err := s.validateSiteParams(params.Name, params.AddressLine1, params.City, params.State, params.PostalCode); err != nil {
		return err
	}

	// Update site
	err = s.queries.UpdateSite(ctx, repository.UpdateSiteParams{
		ID:           params.ID,
		Name:         strings.TrimSpace(params.Name),
		AddressLine1: strings.TrimSpace(params.AddressLine1),
		AddressLine2: toNullString(params.AddressLine2),
		City:         strings.TrimSpace(params.City),
		State:        strings.TrimSpace(params.State),
		PostalCode:   strings.TrimSpace(params.PostalCode),
		ClientName:   toNullString(params.ClientName),
		ClientEmail:  toNullString(params.ClientEmail),
		ClientPhone:  toNullString(params.ClientPhone),
		Notes:        toNullString(params.Notes),
		ClientID:     toNullUUID(params.ClientID),
	})
	if err != nil {
		s.logger.Error("failed to update site", "error", err, "op", op, "site_id", params.ID)
		return domain.Internal(err, op, "Failed to update site")
	}

	s.logger.Info("site updated", "site_id", params.ID)
	return nil
}

// Delete deletes a site.
func (s *siteService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const op = "SiteService.Delete"

	// Verify site exists and belongs to user
	_, err := s.queries.GetSiteByIDAndUserID(ctx, repository.GetSiteByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "site", id.String())
		}
		s.logger.Error("failed to get site for delete", "error", err, "op", op, "site_id", id)
		return domain.Internal(err, op, "Failed to retrieve site")
	}

	// Delete site
	err = s.queries.DeleteSite(ctx, id)
	if err != nil {
		s.logger.Error("failed to delete site", "error", err, "op", op, "site_id", id)
		return domain.Internal(err, op, "Failed to delete site")
	}

	s.logger.Info("site deleted", "site_id", id)
	return nil
}

// validateSiteParams validates required site fields.
func (s *siteService) validateSiteParams(name, addressLine1, city, state, postalCode string) error {
	const op = "SiteService.validateSiteParams"

	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Invalid(op, "Site name is required")
	}
	if len(name) > 200 {
		return domain.Invalid(op, "Site name must be 200 characters or less")
	}

	if strings.TrimSpace(addressLine1) == "" {
		return domain.Invalid(op, "Address is required")
	}
	if strings.TrimSpace(city) == "" {
		return domain.Invalid(op, "City is required")
	}
	if strings.TrimSpace(state) == "" {
		return domain.Invalid(op, "State is required")
	}
	if strings.TrimSpace(postalCode) == "" {
		return domain.Invalid(op, "Postal code is required")
	}

	return nil
}

// repoSiteToDomain converts a repository Site to a domain Site.
func repoSiteToDomain(rs repository.Site) domain.Site {
	return domain.Site{
		ID:           rs.ID,
		UserID:       rs.UserID,
		Name:         rs.Name,
		AddressLine1: rs.AddressLine1,
		AddressLine2: fromNullString(rs.AddressLine2),
		City:         rs.City,
		State:        rs.State,
		PostalCode:   rs.PostalCode,
		ClientName:   fromNullString(rs.ClientName),
		ClientEmail:  fromNullString(rs.ClientEmail),
		ClientPhone:  fromNullString(rs.ClientPhone),
		Notes:        fromNullString(rs.Notes),
		ClientID:     fromNullUUID(rs.ClientID),
		CreatedAt:    rs.CreatedAt.Time,
		UpdatedAt:    rs.UpdatedAt.Time,
	}
}

// repoSiteWithClientToDomain converts a repository ListSitesWithClientByUserIDRow to a domain Site.
func repoSiteWithClientToDomain(rs repository.ListSitesWithClientByUserIDRow) domain.Site {
	return domain.Site{
		ID:               rs.ID,
		UserID:           rs.UserID,
		Name:             rs.Name,
		AddressLine1:     rs.AddressLine1,
		AddressLine2:     fromNullString(rs.AddressLine2),
		City:             rs.City,
		State:            rs.State,
		PostalCode:       rs.PostalCode,
		ClientName:       fromNullString(rs.ClientName),
		ClientEmail:      fromNullString(rs.ClientEmail),
		ClientPhone:      fromNullString(rs.ClientPhone),
		Notes:            fromNullString(rs.Notes),
		ClientID:         fromNullUUID(rs.ClientID),
		LinkedClientName: fromNullString(rs.LinkedClientName),
		CreatedAt:        rs.CreatedAt.Time,
		UpdatedAt:        rs.UpdatedAt.Time,
	}
}

// toNullString converts a string to sql.NullString.
func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// fromNullString converts sql.NullString to string.
func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// toNullUUID converts a *uuid.UUID to uuid.NullUUID.
func toNullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}

// fromNullUUID converts uuid.NullUUID to *uuid.UUID.
func fromNullUUID(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		return &nu.UUID
	}
	return nil
}
