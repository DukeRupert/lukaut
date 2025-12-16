// Package service contains business logic for the Lukaut application.
//
// This file implements the image service for managing inspection photos.
package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/google/uuid"
)

// =============================================================================
// Interface Definition
// =============================================================================

// ImageService defines the interface for image-related operations.
type ImageService interface {
	// Upload uploads an image file to storage and creates a database record.
	// This includes generating a thumbnail and storing both in the storage service.
	// Returns domain.EINVALID for validation errors.
	// Returns domain.ENOTFOUND if inspection doesn't exist or doesn't belong to user.
	// Returns domain.EFORBIDDEN if inspection status doesn't allow uploads.
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, inspectionID, userID uuid.UUID) (*domain.Image, error)

	// Delete removes an image from storage and database.
	// Returns domain.ENOTFOUND if image doesn't exist or doesn't belong to user.
	Delete(ctx context.Context, imageID, userID uuid.UUID) error

	// GetByID retrieves an image by ID with authorization check.
	// Returns domain.ENOTFOUND if image doesn't exist or doesn't belong to user.
	GetByID(ctx context.Context, imageID, userID uuid.UUID) (*domain.Image, error)

	// ListByInspection retrieves all images for an inspection.
	// Returns domain.ENOTFOUND if inspection doesn't exist or doesn't belong to user.
	ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Image, error)

	// GetThumbnailURL returns a presigned/public URL for the image thumbnail.
	GetThumbnailURL(ctx context.Context, imageID, userID uuid.UUID) (string, error)

	// GetOriginalURL returns a presigned/public URL for the original image.
	GetOriginalURL(ctx context.Context, imageID, userID uuid.UUID) (string, error)
}

// =============================================================================
// Implementation
// =============================================================================

// imageService implements the ImageService interface.
type imageService struct {
	queries            *repository.Queries
	storage            storage.Storage
	thumbnailProcessor ThumbnailProcessor
	logger             *slog.Logger
}

// NewImageService creates a new ImageService.
func NewImageService(
	queries *repository.Queries,
	storage storage.Storage,
	thumbnailProcessor ThumbnailProcessor,
	logger *slog.Logger,
) ImageService {
	return &imageService{
		queries:            queries,
		storage:            storage,
		thumbnailProcessor: thumbnailProcessor,
		logger:             logger,
	}
}

// =============================================================================
// Upload
// =============================================================================

// Upload uploads an image file to storage and creates a database record.
func (s *imageService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, inspectionID, userID uuid.UUID) (*domain.Image, error) {
	const op = "image.upload"

	// Verify inspection exists and user owns it
	inspection, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "inspection", inspectionID.String())
		}
		return nil, domain.Internal(err, op, "failed to fetch inspection")
	}

	// Check if inspection status allows uploads
	inspStatus := domain.InspectionStatus(inspection.Status)
	if !inspStatus.CanTransitionTo(domain.InspectionStatusAnalyzing) && inspStatus != domain.InspectionStatusReview {
		return nil, domain.Forbidden(op, "Cannot upload images to inspection in current status")
	}

	// Validate file size
	if err := domain.ValidateImageSize(header.Size); err != nil {
		return nil, err
	}

	// Detect content type from file header (read first 512 bytes)
	headerBytes := make([]byte, 512)
	n, err := file.Read(headerBytes)
	if err != nil && err != io.EOF {
		return nil, domain.Internal(err, op, "failed to read file header")
	}
	contentType := http.DetectContentType(headerBytes[:n])

	// Validate content type
	if !domain.IsValidImageContentType(contentType) {
		return nil, domain.Invalid(op, fmt.Sprintf("Unsupported image type: %s. Only JPEG and PNG are supported.", contentType))
	}

	// Reset file pointer to beginning after reading header
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, 0); err != nil {
			return nil, domain.Internal(err, op, "failed to reset file pointer")
		}
	}

	// Read entire file into memory for processing
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to read file data")
	}

	// Generate thumbnail
	thumbnailBytes, width, height, err := s.thumbnailProcessor.GenerateThumbnail(
		bytes.NewReader(fileData),
		domain.ThumbnailMaxWidth,
		domain.ThumbnailMaxHeight,
	)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to generate thumbnail")
	}

	// Generate storage keys
	ext := filepath.Ext(header.Filename)
	imageID := uuid.New()
	storageKey := fmt.Sprintf("inspections/%s/images/%s%s", inspectionID, imageID, ext)
	thumbnailKey := fmt.Sprintf("inspections/%s/thumbnails/%s.jpg", inspectionID, imageID)

	// Upload original to storage
	if err := s.storage.Put(ctx, storageKey, bytes.NewReader(fileData), storage.PutOptions{
		ContentType: contentType,
		MaxSize:     domain.MaxImageSize,
		Overwrite:   false,
		Public:      false,
	}); err != nil {
		return nil, domain.Internal(err, op, "failed to upload original image")
	}

	// Upload thumbnail to storage
	if err := s.storage.Put(ctx, thumbnailKey, bytes.NewReader(thumbnailBytes), storage.PutOptions{
		ContentType: "image/jpeg",
		MaxSize:     0, // No limit for thumbnails
		Overwrite:   false,
		Public:      false,
	}); err != nil {
		// Clean up original image on thumbnail upload failure
		_ = s.storage.Delete(ctx, storageKey)
		return nil, domain.Internal(err, op, "failed to upload thumbnail")
	}

	// Create database record
	dbImage, err := s.queries.CreateImage(ctx, repository.CreateImageParams{
		InspectionID: inspectionID,
		StorageKey:   storageKey,
		ThumbnailKey: sql.NullString{
			String: thumbnailKey,
			Valid:  true,
		},
		OriginalFilename: sql.NullString{
			String: header.Filename,
			Valid:  true,
		},
		ContentType: contentType,
		SizeBytes:   int32(header.Size),
		Width: sql.NullInt32{
			Int32: int32(width),
			Valid: true,
		},
		Height: sql.NullInt32{
			Int32: int32(height),
			Valid: true,
		},
		AnalysisStatus: sql.NullString{
			String: string(domain.ImageAnalysisStatusPending),
			Valid:  true,
		},
	})
	if err != nil {
		// Clean up storage on database error
		_ = s.storage.Delete(ctx, storageKey)
		_ = s.storage.Delete(ctx, thumbnailKey)
		return nil, domain.Internal(err, op, "failed to create image record")
	}

	// Convert to domain type
	return s.toDomain(dbImage), nil
}

// =============================================================================
// Delete
// =============================================================================

// Delete removes an image from storage and database.
func (s *imageService) Delete(ctx context.Context, imageID, userID uuid.UUID) error {
	const op = "image.delete"

	// Get image with authorization check
	image, err := s.GetByID(ctx, imageID, userID)
	if err != nil {
		return err
	}

	// Delete from storage (both original and thumbnail)
	// Continue even if storage deletion fails - we still want to remove DB record
	if err := s.storage.Delete(ctx, image.StorageKey); err != nil {
		s.logger.Error("failed to delete original image from storage", "error", err, "key", image.StorageKey)
	}
	if err := s.storage.Delete(ctx, image.ThumbnailKey); err != nil {
		s.logger.Error("failed to delete thumbnail from storage", "error", err, "key", image.ThumbnailKey)
	}

	// Delete from database
	if err := s.queries.DeleteImageByID(ctx, imageID); err != nil {
		return domain.Internal(err, op, "failed to delete image record")
	}

	return nil
}

// =============================================================================
// GetByID
// =============================================================================

// GetByID retrieves an image by ID with authorization check.
func (s *imageService) GetByID(ctx context.Context, imageID, userID uuid.UUID) (*domain.Image, error) {
	const op = "image.get"

	// Get image with inspection join for authorization
	row, err := s.queries.GetImageByIDWithInspection(ctx, imageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "image", imageID.String())
		}
		return nil, domain.Internal(err, op, "failed to fetch image")
	}

	// Check authorization
	if row.UserID != userID {
		return nil, domain.NotFound(op, "image", imageID.String())
	}

	// Convert to domain type (use only image fields)
	dbImage := repository.Image{
		ID:                  row.ID,
		InspectionID:        row.InspectionID,
		StorageKey:          row.StorageKey,
		ThumbnailKey:        row.ThumbnailKey,
		OriginalFilename:    row.OriginalFilename,
		ContentType:         row.ContentType,
		SizeBytes:           row.SizeBytes,
		Width:               row.Width,
		Height:              row.Height,
		AnalysisStatus:      row.AnalysisStatus,
		AnalysisCompletedAt: row.AnalysisCompletedAt,
		CreatedAt:           row.CreatedAt,
	}

	return s.toDomain(dbImage), nil
}

// =============================================================================
// ListByInspection
// =============================================================================

// ListByInspection retrieves all images for an inspection.
func (s *imageService) ListByInspection(ctx context.Context, inspectionID, userID uuid.UUID) ([]domain.Image, error) {
	const op = "image.list"

	// Verify inspection exists and user owns it
	_, err := s.queries.GetInspectionByIDAndUserID(ctx, repository.GetInspectionByIDAndUserIDParams{
		ID:     inspectionID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NotFound(op, "inspection", inspectionID.String())
		}
		return nil, domain.Internal(err, op, "failed to fetch inspection")
	}

	// Fetch images
	dbImages, err := s.queries.ListImagesByInspectionID(ctx, inspectionID)
	if err != nil {
		return nil, domain.Internal(err, op, "failed to fetch images")
	}

	// Convert to domain types
	images := make([]domain.Image, len(dbImages))
	for i, dbImage := range dbImages {
		images[i] = *s.toDomain(dbImage)
	}

	return images, nil
}

// =============================================================================
// GetThumbnailURL
// =============================================================================

// GetThumbnailURL returns a presigned/public URL for the image thumbnail.
func (s *imageService) GetThumbnailURL(ctx context.Context, imageID, userID uuid.UUID) (string, error) {
	const op = "image.thumbnail_url"

	// Get image with authorization
	image, err := s.GetByID(ctx, imageID, userID)
	if err != nil {
		return "", err
	}

	// Generate URL with 1 hour expiry
	url, err := s.storage.URL(ctx, image.ThumbnailKey, 1*time.Hour)
	if err != nil {
		return "", domain.Internal(err, op, "failed to generate thumbnail URL")
	}

	return url, nil
}

// =============================================================================
// GetOriginalURL
// =============================================================================

// GetOriginalURL returns a presigned/public URL for the original image.
func (s *imageService) GetOriginalURL(ctx context.Context, imageID, userID uuid.UUID) (string, error) {
	const op = "image.original_url"

	// Get image with authorization
	image, err := s.GetByID(ctx, imageID, userID)
	if err != nil {
		return "", err
	}

	// Generate URL with 1 hour expiry
	url, err := s.storage.URL(ctx, image.StorageKey, 1*time.Hour)
	if err != nil {
		return "", domain.Internal(err, op, "failed to generate original URL")
	}

	return url, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// toDomain converts a repository Image to a domain Image.
func (s *imageService) toDomain(dbImage repository.Image) *domain.Image {
	// Helper to get string from sql.NullString
	getString := func(ns sql.NullString) string {
		if ns.Valid {
			return ns.String
		}
		return ""
	}

	// Helper to get int32 from sql.NullInt32
	getInt32 := func(ni sql.NullInt32) int32 {
		if ni.Valid {
			return ni.Int32
		}
		return 0
	}

	// Helper to get time from sql.NullTime
	getTime := func(nt sql.NullTime) time.Time {
		if nt.Valid {
			return nt.Time
		}
		return time.Time{}
	}

	return &domain.Image{
		ID:               dbImage.ID,
		InspectionID:     dbImage.InspectionID,
		StorageKey:       dbImage.StorageKey,
		ThumbnailKey:     getString(dbImage.ThumbnailKey),
		OriginalFilename: getString(dbImage.OriginalFilename),
		ContentType:      dbImage.ContentType,
		SizeBytes:        int64(dbImage.SizeBytes),
		Width:            getInt32(dbImage.Width),
		Height:           getInt32(dbImage.Height),
		AnalysisStatus:   domain.ImageAnalysisStatus(getString(dbImage.AnalysisStatus)),
		CreatedAt:        getTime(dbImage.CreatedAt),
		UpdatedAt:        time.Time{}, // Not stored in DB (no updated_at column)
		// ThumbnailURL and OriginalURL are populated on demand by the handler
	}
}
