package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// LocalStorage Implementation
// =============================================================================

// LocalStorage implements the Storage interface using the local filesystem.
// It stores files in a base directory and serves them via HTTP.
//
// Security: Path traversal prevention is enforced in resolvePath().
type LocalStorage struct {
	basePath string // Root directory for file storage
	baseURL  string // Base URL for file access
	logger   *slog.Logger
}

// NewLocalStorage creates a new LocalStorage instance.
//
// The base directory is created if it doesn't exist.
// Returns an error if directory creation fails.
func NewLocalStorage(cfg LocalConfig, logger *slog.Logger) (*LocalStorage, error) {
	// Ensure base path is absolute
	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base path: %w", err)
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Ensure baseURL doesn't end with a slash for consistent URL generation
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	logger.Info("initialized local storage",
		"base_path", absPath,
		"base_url", baseURL,
	)

	return &LocalStorage{
		basePath: absPath,
		baseURL:  baseURL,
		logger:   logger,
	}, nil
}

// =============================================================================
// Interface Implementation
// =============================================================================

// Put stores data at the specified key.
func (s *LocalStorage) Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error {
	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Resolve and validate the file path
	filePath, err := s.resolvePath(key)
	if err != nil {
		return &StorageError{Op: "Put", Key: key, Err: err}
	}

	// Check if file exists when overwrite is disabled
	if !opts.Overwrite {
		if _, err := os.Stat(filePath); err == nil {
			return &StorageError{Op: "Put", Key: key, Err: ErrKeyExists}
		}
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &StorageError{Op: "Put", Key: key, Err: fmt.Errorf("failed to create directory: %w", err)}
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return &StorageError{Op: "Put", Key: key, Err: fmt.Errorf("failed to create file: %w", err)}
	}
	defer file.Close()

	// Copy data to file with optional size limit
	var written int64
	if opts.MaxSize > 0 {
		// Use a limited reader to enforce max size
		lr := io.LimitReader(data, opts.MaxSize+1)
		written, err = io.Copy(file, lr)
		if err != nil {
			os.Remove(filePath) // Clean up on error
			return &StorageError{Op: "Put", Key: key, Err: fmt.Errorf("failed to write file: %w", err)}
		}
		if written > opts.MaxSize {
			os.Remove(filePath) // Clean up oversized file
			return &StorageError{Op: "Put", Key: key, Err: ErrTooLarge}
		}
	} else {
		// No size limit, copy everything
		written, err = io.Copy(file, data)
		if err != nil {
			os.Remove(filePath) // Clean up on error
			return &StorageError{Op: "Put", Key: key, Err: fmt.Errorf("failed to write file: %w", err)}
		}
	}

	s.logger.Debug("stored file",
		"key", key,
		"path", filePath,
		"size", written,
		"content_type", opts.ContentType,
	)

	return nil
}

// Get retrieves the data at the specified key.
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	// Check context cancellation
	if ctx.Err() != nil {
		return nil, ObjectInfo{}, ctx.Err()
	}

	// Resolve and validate the file path
	filePath, err := s.resolvePath(key)
	if err != nil {
		return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: err}
	}

	// Get file info
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: ErrNotFound}
		}
		return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: fmt.Errorf("failed to stat file: %w", err)}
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: fmt.Errorf("failed to open file: %w", err)}
	}

	// Detect content type from filename
	contentType := DetectContentType("", key, nil)

	info := ObjectInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  contentType,
		LastModified: stat.ModTime(),
		ETag:         "", // Local storage doesn't generate ETags
	}

	return file, info, nil
}

// Delete removes the object at the specified key.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Resolve and validate the file path
	filePath, err := s.resolvePath(key)
	if err != nil {
		return &StorageError{Op: "Delete", Key: key, Err: err}
	}

	// Remove the file (idempotent - no error if doesn't exist)
	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return &StorageError{Op: "Delete", Key: key, Err: fmt.Errorf("failed to delete file: %w", err)}
	}

	s.logger.Debug("deleted file", "key", key, "path", filePath)

	return nil
}

// URL returns a URL for accessing the object.
// For local storage, this is always a public URL (expires parameter is ignored).
func (s *LocalStorage) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	// Check context cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Validate the key
	if _, err := s.resolvePath(key); err != nil {
		return "", &StorageError{Op: "URL", Key: key, Err: err}
	}

	// Generate public URL
	url := fmt.Sprintf("%s/%s", s.baseURL, key)

	return url, nil
}

// Exists checks if an object exists at the specified key.
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	// Check context cancellation
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Resolve and validate the file path
	filePath, err := s.resolvePath(key)
	if err != nil {
		return false, &StorageError{Op: "Exists", Key: key, Err: err}
	}

	// Check if file exists
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &StorageError{Op: "Exists", Key: key, Err: fmt.Errorf("failed to stat file: %w", err)}
	}

	return true, nil
}

// =============================================================================
// Internal Helpers
// =============================================================================

// resolvePath converts a storage key to an absolute file path.
//
// Security: This function prevents path traversal attacks by:
// 1. Rejecting keys that contain ".." path components
// 2. Ensuring the resolved path is within the base directory
// 3. Cleaning the path to normalize separators
func (s *LocalStorage) resolvePath(key string) (string, error) {
	// Reject empty keys
	if key == "" {
		return "", ErrInvalidKey
	}

	// Clean the key to normalize path separators and remove redundant elements
	cleanKey := filepath.Clean(key)

	// Reject keys that try to escape the base directory
	// filepath.Clean converts ".." to parent directory traversal
	if strings.Contains(cleanKey, "..") {
		return "", ErrInvalidKey
	}

	// Build the absolute path
	absPath := filepath.Join(s.basePath, cleanKey)

	// Ensure the resolved path is still within the base directory
	// This is a defense-in-depth check against path traversal
	if !strings.HasPrefix(absPath, s.basePath) {
		return "", ErrInvalidKey
	}

	return absPath, nil
}
