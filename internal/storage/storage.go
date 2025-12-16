// Package storage provides file storage abstraction for the Lukaut application.
//
// This package defines a Storage interface with implementations for:
// - LocalStorage: File system storage for development
// - R2Storage: Cloudflare R2 (S3-compatible) storage for production
//
// The storage service handles uploading inspection images, thumbnails, and
// generated reports with automatic content type detection and validation.
package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Interface Definition
// =============================================================================

// Storage defines the interface for file storage operations.
//
// Implementations:
// - LocalStorage: Stores files on the local filesystem
// - R2Storage: Stores files in Cloudflare R2 object storage
//
// All methods are context-aware for timeout and cancellation support.
type Storage interface {
	// Put stores data at the specified key with the given options.
	// Returns an error if the operation fails or if the key already exists
	// (unless overwrite is enabled in opts).
	Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error

	// Get retrieves the data at the specified key.
	// Returns the data as an io.ReadCloser (caller must close), object metadata,
	// and an error. Returns ErrNotFound if the key doesn't exist.
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)

	// Delete removes the object at the specified key.
	// This operation is idempotent - no error is returned if the key doesn't exist.
	Delete(ctx context.Context, key string) error

	// URL returns a URL for accessing the object at the specified key.
	// For public objects, this is a permanent URL.
	// For private objects, this is a presigned URL valid for the specified duration.
	// Returns an error if the key doesn't exist or URL generation fails.
	URL(ctx context.Context, key string, expires time.Duration) (string, error)

	// Exists checks if an object exists at the specified key.
	// Returns true if the object exists, false otherwise.
	Exists(ctx context.Context, key string) (bool, error)
}

// =============================================================================
// Data Types
// =============================================================================

// PutOptions configures how an object is stored.
type PutOptions struct {
	// ContentType specifies the MIME type of the object.
	// If empty, it will be auto-detected from the file extension or content.
	ContentType string

	// MaxSize specifies the maximum allowed size in bytes.
	// If the data exceeds this size, ErrTooLarge is returned.
	// A value of 0 means no limit.
	MaxSize int64

	// Overwrite allows replacing an existing object at the same key.
	// If false and the key exists, ErrKeyExists is returned.
	Overwrite bool

	// Public determines if the object should be publicly accessible.
	// For R2, this sets the ACL to public-read.
	// For local storage, this is informational only.
	Public bool
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string    // Object key/path
	Size         int64     // Size in bytes
	ContentType  string    // MIME type
	LastModified time.Time // Last modification time
	ETag         string    // Entity tag (if available)
}

// =============================================================================
// Configuration Types
// =============================================================================

// LocalConfig holds configuration for local filesystem storage.
type LocalConfig struct {
	// BasePath is the root directory where files are stored.
	// Example: "./storage" or "/var/lib/lukaut/files"
	BasePath string

	// BaseURL is the public URL prefix for accessing files.
	// Example: "http://localhost:8080/files"
	BaseURL string
}

// R2Config holds configuration for Cloudflare R2 storage.
type R2Config struct {
	// AccountID is your Cloudflare account ID.
	AccountID string

	// AccessKeyID is the R2 API access key ID.
	AccessKeyID string

	// SecretAccessKey is the R2 API secret key.
	SecretAccessKey string

	// BucketName is the name of the R2 bucket to use.
	BucketName string

	// PublicURL is the public URL for the bucket (if using a custom domain).
	// Example: "https://files.lukaut.com"
	// If empty, presigned URLs will be used for all access.
	PublicURL string

	// Region is the AWS region to use (required by AWS SDK).
	// For R2, this can be any valid region string as R2 is globally distributed.
	// Default: "auto"
	Region string
}

// =============================================================================
// Provider Constants
// =============================================================================

const (
	// ProviderLocal identifies the local filesystem storage provider.
	ProviderLocal = "local"

	// ProviderR2 identifies the Cloudflare R2 storage provider.
	ProviderR2 = "r2"
)

// =============================================================================
// Key Generation Helpers
// =============================================================================

// ImageKey generates a storage key for an uploaded inspection image.
// Format: inspections/{inspectionID}/images/{uuid}.{ext}
//
// Parameters:
//   - inspectionID: UUID of the inspection
//   - filename: Original filename (used to extract extension)
//
// Example: "inspections/123e4567-e89b-12d3-a456-426614174000/images/987fcdeb-51a2-43f1-b9c4-12345678abcd.jpg"
func ImageKey(inspectionID uuid.UUID, filename string) string {
	ext := filepath.Ext(filename)
	imageID := uuid.New()
	return fmt.Sprintf("inspections/%s/images/%s%s", inspectionID, imageID, ext)
}

// ThumbnailKey generates a storage key for an image thumbnail.
// Format: inspections/{inspectionID}/thumbnails/{uuid}.{ext}
//
// Parameters:
//   - inspectionID: UUID of the inspection
//   - filename: Original filename (used to extract extension)
//
// Example: "inspections/123e4567-e89b-12d3-a456-426614174000/thumbnails/987fcdeb-51a2-43f1-b9c4-12345678abcd.jpg"
func ThumbnailKey(inspectionID uuid.UUID, filename string) string {
	ext := filepath.Ext(filename)
	thumbnailID := uuid.New()
	return fmt.Sprintf("inspections/%s/thumbnails/%s%s", inspectionID, thumbnailID, ext)
}

// ReportKey generates a storage key for a generated report.
// Format: inspections/{inspectionID}/reports/{uuid}.{ext}
//
// Parameters:
//   - inspectionID: UUID of the inspection
//   - format: Report format ("pdf" or "docx")
//
// Example: "inspections/123e4567-e89b-12d3-a456-426614174000/reports/987fcdeb-51a2-43f1-b9c4-12345678abcd.pdf"
func ReportKey(inspectionID uuid.UUID, format string) string {
	reportID := uuid.New()
	return fmt.Sprintf("inspections/%s/reports/%s.%s", inspectionID, reportID, format)
}
