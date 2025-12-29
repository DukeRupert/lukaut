// Package service contains the business logic layer.
//
// This file implements the client service for managing construction
// company clients.
package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// =============================================================================
// Interface Definition
// =============================================================================

// ClientService defines the interface for client-related operations.
//
// This interface enables:
// - Mocking in unit tests
// - Clear contract definition for handlers
// - Potential future implementations with different backends
type ClientService interface {
	// Create creates a new client.
	// Returns domain.EINVALID for validation errors.
	Create(ctx context.Context, params domain.CreateClientParams) (*domain.Client, error)

	// GetByID retrieves a client by ID and user ID (for authorization).
	// Returns domain.ENOTFOUND if client does not exist or doesn't belong to user.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error)

	// List retrieves a paginated list of clients for a user.
	// Returns empty result if user has no clients.
	List(ctx context.Context, params domain.ListClientsParams) (*domain.ListClientsResult, error)

	// ListAll retrieves all clients for a user (for dropdowns).
	// Returns empty slice if user has no clients.
	ListAll(ctx context.Context, userID uuid.UUID) ([]domain.Client, error)

	// Update updates an existing client.
	// Returns domain.ENOTFOUND if client does not exist or doesn't belong to user.
	// Returns domain.EINVALID for validation errors.
	Update(ctx context.Context, params domain.UpdateClientParams) error

	// Delete deletes a client by ID.
	// Returns domain.ENOTFOUND if client does not exist or doesn't belong to user.
	// Returns domain.EINVALID if client has associated sites.
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// =============================================================================
// Implementation
// =============================================================================

// clientService implements the ClientService interface.
type clientService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewClientService creates a new ClientService.
//
// Parameters:
// - queries: Repository queries for database access
// - logger: Structured logger for operation logging
func NewClientService(
	queries *repository.Queries,
	logger *slog.Logger,
) ClientService {
	return &clientService{
		queries: queries,
		logger:  logger,
	}
}

// =============================================================================
// Create
// =============================================================================

// Create creates a new client.
func (s *clientService) Create(ctx context.Context, params domain.CreateClientParams) (*domain.Client, error) {
	const op = "client.create"

	// Validate parameters
	if err := s.validateCreateParams(params); err != nil {
		return nil, err
	}

	// Create the client
	row, err := s.queries.CreateClient(ctx, repository.CreateClientParams{
		UserID:       params.UserID,
		Name:         params.Name,
		Email:        domain.ToNullString(params.Email),
		Phone:        domain.ToNullString(params.Phone),
		AddressLine1: domain.ToNullString(params.AddressLine1),
		AddressLine2: domain.ToNullString(params.AddressLine2),
		City:         domain.ToNullString(params.City),
		State:        domain.ToNullString(params.State),
		PostalCode:   domain.ToNullString(params.PostalCode),
		Notes:        domain.ToNullString(params.Notes),
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to create client")
	}

	// Convert to domain type
	client := s.rowToClient(row)

	s.logger.Info("client created",
		"client_id", client.ID,
		"user_id", params.UserID,
		"name", params.Name,
	)

	return client, nil
}

// validateCreateParams validates client creation parameters.
func (s *clientService) validateCreateParams(params domain.CreateClientParams) error {
	const op = "client.validate"

	// Name is required and must be 1-255 characters
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return domain.Invalid(op, "name is required")
	}
	if len(name) > 255 {
		return domain.Invalid(op, "name must be 255 characters or less")
	}

	return nil
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves a client by ID.
func (s *clientService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error) {
	const op = "client.get"

	// Get client with site count
	row, err := s.queries.GetClientWithSiteCount(ctx, repository.GetClientWithSiteCountParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "client", id.String())
		}
		return nil, domain.Internal(err, op, "failed to get client")
	}

	// Convert to domain type
	return s.rowWithSiteCountToClient(row), nil
}

// =============================================================================
// List
// =============================================================================

// List retrieves a paginated list of clients.
func (s *clientService) List(ctx context.Context, params domain.ListClientsParams) (*domain.ListClientsResult, error) {
	const op = "client.list"

	// Get total count
	total, err := s.queries.CountClientsByUserID(ctx, params.UserID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count clients")
	}

	// Get paginated results
	rows, err := s.queries.ListClientsWithSiteCountByUserID(ctx, repository.ListClientsWithSiteCountByUserIDParams{
		UserID: params.UserID,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list clients")
	}

	// Convert to domain types
	clients := make([]domain.Client, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, *s.rowWithSiteCountListToClient(row))
	}

	return &domain.ListClientsResult{
		Clients: clients,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
	}, nil
}

// ListAll retrieves all clients for a user (for dropdowns).
func (s *clientService) ListAll(ctx context.Context, userID uuid.UUID) ([]domain.Client, error) {
	const op = "client.list_all"

	rows, err := s.queries.ListAllClientsByUserID(ctx, userID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to list clients")
	}

	clients := make([]domain.Client, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, *s.rowToClient(row))
	}

	return clients, nil
}

// =============================================================================
// Update
// =============================================================================

// Update updates an existing client.
func (s *clientService) Update(ctx context.Context, params domain.UpdateClientParams) error {
	const op = "client.update"

	// Validate parameters
	if err := s.validateUpdateParams(params); err != nil {
		return err
	}

	// Verify client exists and belongs to user
	_, err := s.queries.GetClientByIDAndUserID(ctx, repository.GetClientByIDAndUserIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "client", params.ID.String())
		}
		return domain.Internal(err, op, "failed to get client")
	}

	// Update the client
	err = s.queries.UpdateClientByIDAndUserID(ctx, repository.UpdateClientByIDAndUserIDParams{
		ID:           params.ID,
		UserID:       params.UserID,
		Name:         params.Name,
		Email:        domain.ToNullString(params.Email),
		Phone:        domain.ToNullString(params.Phone),
		AddressLine1: domain.ToNullString(params.AddressLine1),
		AddressLine2: domain.ToNullString(params.AddressLine2),
		City:         domain.ToNullString(params.City),
		State:        domain.ToNullString(params.State),
		PostalCode:   domain.ToNullString(params.PostalCode),
		Notes:        domain.ToNullString(params.Notes),
	})
	if err != nil {
		return domain.Internal(err, op, "failed to update client")
	}

	s.logger.Info("client updated",
		"client_id", params.ID,
		"user_id", params.UserID,
	)

	return nil
}

// validateUpdateParams validates client update parameters.
func (s *clientService) validateUpdateParams(params domain.UpdateClientParams) error {
	const op = "client.validate"

	// Name is required and must be 1-255 characters
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return domain.Invalid(op, "name is required")
	}
	if len(name) > 255 {
		return domain.Invalid(op, "name must be 255 characters or less")
	}

	return nil
}

// =============================================================================
// Delete
// =============================================================================

// Delete deletes a client.
func (s *clientService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const op = "client.delete"

	// Verify client exists and belongs to user
	_, err := s.queries.GetClientByIDAndUserID(ctx, repository.GetClientByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NotFound(op, "client", id.String())
		}
		return domain.Internal(err, op, "failed to get client")
	}

	// Check if client is associated with any sites
	siteCount, err := s.queries.CountSitesByClientID(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return domain.Internal(err, op, "failed to count sites")
	}
	if siteCount > 0 {
		return domain.Invalid(op, "cannot delete client that is associated with sites")
	}

	// Delete the client
	err = s.queries.DeleteClientByIDAndUserID(ctx, repository.DeleteClientByIDAndUserIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return domain.Internal(err, op, "failed to delete client")
	}

	s.logger.Info("client deleted",
		"client_id", id,
		"user_id", userID,
	)

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToClient converts a repository client row to a domain Client.
func (s *clientService) rowToClient(row repository.Client) *domain.Client {
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	return &domain.Client{
		ID:           row.ID,
		UserID:       row.UserID,
		Name:         row.Name,
		Email:        domain.NullStringValue(row.Email),
		Phone:        domain.NullStringValue(row.Phone),
		AddressLine1: domain.NullStringValue(row.AddressLine1),
		AddressLine2: domain.NullStringValue(row.AddressLine2),
		City:         domain.NullStringValue(row.City),
		State:        domain.NullStringValue(row.State),
		PostalCode:   domain.NullStringValue(row.PostalCode),
		Notes:        domain.NullStringValue(row.Notes),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

// rowWithSiteCountToClient converts a GetClientWithSiteCount row to a domain Client.
func (s *clientService) rowWithSiteCountToClient(row repository.GetClientWithSiteCountRow) *domain.Client {
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	return &domain.Client{
		ID:           row.ID,
		UserID:       row.UserID,
		Name:         row.Name,
		Email:        domain.NullStringValue(row.Email),
		Phone:        domain.NullStringValue(row.Phone),
		AddressLine1: domain.NullStringValue(row.AddressLine1),
		AddressLine2: domain.NullStringValue(row.AddressLine2),
		City:         domain.NullStringValue(row.City),
		State:        domain.NullStringValue(row.State),
		PostalCode:   domain.NullStringValue(row.PostalCode),
		Notes:        domain.NullStringValue(row.Notes),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		SiteCount:    int(row.SiteCount),
	}
}

// rowWithSiteCountListToClient converts a ListClientsWithSiteCountByUserID row to a domain Client.
func (s *clientService) rowWithSiteCountListToClient(row repository.ListClientsWithSiteCountByUserIDRow) *domain.Client {
	createdAt := time.Time{}
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}
	updatedAt := time.Time{}
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	return &domain.Client{
		ID:           row.ID,
		UserID:       row.UserID,
		Name:         row.Name,
		Email:        domain.NullStringValue(row.Email),
		Phone:        domain.NullStringValue(row.Phone),
		AddressLine1: domain.NullStringValue(row.AddressLine1),
		AddressLine2: domain.NullStringValue(row.AddressLine2),
		City:         domain.NullStringValue(row.City),
		State:        domain.NullStringValue(row.State),
		PostalCode:   domain.NullStringValue(row.PostalCode),
		Notes:        domain.NullStringValue(row.Notes),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		SiteCount:    int(row.SiteCount),
	}
}
