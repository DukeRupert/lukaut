package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// =============================================================================
// R2Storage Implementation
// =============================================================================

// R2Storage implements the Storage interface using Cloudflare R2.
// R2 is S3-compatible, so we use the AWS SDK v2 with custom configuration.
type R2Storage struct {
	client       *s3.Client
	presignClient *s3.PresignClient
	bucketName   string
	publicURL    string // Optional public URL (e.g., custom domain)
	logger       *slog.Logger
}

// NewR2Storage creates a new R2Storage instance.
//
// The R2 endpoint URL is automatically constructed from the account ID.
// Returns an error if the AWS SDK client creation fails.
func NewR2Storage(cfg R2Config, logger *slog.Logger) (*R2Storage, error) {
	// Default region for R2
	region := cfg.Region
	if region == "" {
		region = "auto"
	}

	// Construct R2 endpoint URL
	// Format: https://{account_id}.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Create AWS credentials
	creds := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		"", // session token not needed for R2
	)

	// Create custom endpoint resolver
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		},
	)

	// Create AWS config
	awsCfg := aws.Config{
		Region:                      region,
		Credentials:                 creds,
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Create presign client for generating signed URLs
	presignClient := s3.NewPresignClient(client)

	logger.Info("initialized R2 storage",
		"bucket", cfg.BucketName,
		"endpoint", endpoint,
		"public_url", cfg.PublicURL,
	)

	return &R2Storage{
		client:        client,
		presignClient: presignClient,
		bucketName:    cfg.BucketName,
		publicURL:     strings.TrimSuffix(cfg.PublicURL, "/"),
		logger:        logger,
	}, nil
}

// =============================================================================
// Interface Implementation
// =============================================================================

// Put stores data at the specified key.
func (s *R2Storage) Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return &StorageError{Op: "Put", Key: key, Err: err}
	}

	// Check if object exists when overwrite is disabled
	if !opts.Overwrite {
		exists, err := s.Exists(ctx, key)
		if err != nil {
			return &StorageError{Op: "Put", Key: key, Err: fmt.Errorf("failed to check existence: %w", err)}
		}
		if exists {
			return &StorageError{Op: "Put", Key: key, Err: ErrKeyExists}
		}
	}

	// Wrap data with size limit if specified
	var reader io.Reader = data
	if opts.MaxSize > 0 {
		reader = io.LimitReader(data, opts.MaxSize+1)
	}

	// Detect content type if not provided
	contentType := opts.ContentType
	if contentType == "" {
		contentType = DetectContentType("", key, nil)
	}

	// Build PutObject input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	// Set ACL based on Public flag
	if opts.Public {
		input.ACL = types.ObjectCannedACLPublicRead
	}

	// Upload the object
	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		// Check for size limit exceeded
		// Note: This is a best-effort check since we can't know the exact size
		// until after the upload attempt
		if opts.MaxSize > 0 {
			return &StorageError{Op: "Put", Key: key, Err: ErrTooLarge}
		}

		// Wrap other S3 errors
		return &StorageError{Op: "Put", Key: key, Err: s.wrapS3Error(err)}
	}

	s.logger.Debug("stored object in R2",
		"key", key,
		"etag", aws.ToString(result.ETag),
		"content_type", contentType,
	)

	return nil
}

// Get retrieves the data at the specified key.
func (s *R2Storage) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: err}
	}

	// Get the object
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ObjectInfo{}, &StorageError{Op: "Get", Key: key, Err: s.wrapS3Error(err)}
	}

	// Build object info
	info := ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		ContentType:  aws.ToString(result.ContentType),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
	}

	return result.Body, info, nil
}

// Delete removes the object at the specified key.
func (s *R2Storage) Delete(ctx context.Context, key string) error {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return &StorageError{Op: "Delete", Key: key, Err: err}
	}

	// Delete the object (idempotent - S3 doesn't error if key doesn't exist)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return &StorageError{Op: "Delete", Key: key, Err: s.wrapS3Error(err)}
	}

	s.logger.Debug("deleted object from R2", "key", key)

	return nil
}

// URL returns a URL for accessing the object.
// If publicURL is configured and expires is 0, returns a public URL.
// Otherwise, returns a presigned URL valid for the specified duration.
func (s *R2Storage) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return "", &StorageError{Op: "URL", Key: key, Err: err}
	}

	// If public URL is configured and no expiration, return public URL
	if s.publicURL != "" && expires == 0 {
		return fmt.Sprintf("%s/%s", s.publicURL, key), nil
	}

	// Default expiration if not specified
	if expires == 0 {
		expires = 15 * time.Minute
	}

	// Generate presigned URL
	request, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", &StorageError{Op: "URL", Key: key, Err: fmt.Errorf("failed to generate presigned URL: %w", err)}
	}

	return request.URL, nil
}

// Exists checks if an object exists at the specified key.
func (s *R2Storage) Exists(ctx context.Context, key string) (bool, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return false, &StorageError{Op: "Exists", Key: key, Err: err}
	}

	// Use HeadObject to check existence without downloading the object
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if the error is NotFound
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}

		// Check for NoSuchKey error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" {
				return false, nil
			}
		}

		// Other errors are actual failures
		return false, &StorageError{Op: "Exists", Key: key, Err: s.wrapS3Error(err)}
	}

	return true, nil
}

// =============================================================================
// Internal Helpers
// =============================================================================

// validateKey checks if a storage key is valid.
// Rejects empty keys and keys with path traversal attempts.
func (s *R2Storage) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	// Reject keys with path traversal
	if strings.Contains(key, "..") {
		return ErrInvalidKey
	}

	return nil
}

// wrapS3Error converts S3 SDK errors to storage errors.
func (s *R2Storage) wrapS3Error(err error) error {
	if err == nil {
		return nil
	}

	// Check for NotFound errors
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return ErrNotFound
	}

	// Check for NoSuchKey
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return ErrNotFound
	}

	// Check for access denied
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return ErrNotFound
		case "AccessDenied", "Forbidden":
			return ErrAccessDenied
		}

		// Check HTTP status code
		if httpErr, ok := err.(interface{ HTTPStatusCode() int }); ok {
			switch httpErr.HTTPStatusCode() {
			case http.StatusNotFound:
				return ErrNotFound
			case http.StatusForbidden:
				return ErrAccessDenied
			}
		}
	}

	// Return the original error wrapped
	return fmt.Errorf("R2 operation failed: %w", err)
}
