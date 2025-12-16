package storage

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// =============================================================================
// Content Type Detection
// =============================================================================

// DetectContentType determines the MIME type of a file.
//
// Detection priority:
// 1. If providedType is non-empty, use it directly
// 2. Try to detect from file extension using mime.TypeByExtension
// 3. Sniff content from the first 512 bytes of data (if available)
// 4. Fall back to "application/octet-stream"
//
// Parameters:
//   - providedType: Explicitly provided content type (e.g., from HTTP header)
//   - filename: File name used to extract extension for MIME lookup
//   - data: Optional reader for content sniffing (only first 512 bytes are read)
//
// Returns the detected MIME type.
func DetectContentType(providedType, filename string, data io.Reader) string {
	// 1. Use provided type if available
	if providedType != "" {
		return providedType
	}

	// 2. Try extension-based detection
	ext := strings.ToLower(filepath.Ext(filename))
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}

	// 3. Try content sniffing if data is available
	if data != nil {
		// Read up to 512 bytes for sniffing (http.DetectContentType requirement)
		buffer := make([]byte, 512)
		n, err := io.ReadFull(data, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			// If we can't read, fall through to default
		} else {
			// DetectContentType always returns a valid MIME type
			return http.DetectContentType(buffer[:n])
		}
	}

	// 4. Fall back to generic binary type
	return "application/octet-stream"
}

// =============================================================================
// Content Type Validation
// =============================================================================

// AllowedImageTypes defines the MIME types accepted for inspection images.
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true, // Some systems use this instead of image/jpeg
	"image/png":  true,
	"image/webp": true,
	"image/heic": true, // iPhone photos
	"image/heif": true, // High Efficiency Image Format
}

// IsAllowedImageType checks if a content type is an allowed image format
// for inspection photo uploads.
func IsAllowedImageType(contentType string) bool {
	// Normalize the content type (remove parameters like charset)
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))
	return AllowedImageTypes[baseType]
}

// IsImage returns true if the content type is any image format.
func IsImage(contentType string) bool {
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))
	return strings.HasPrefix(baseType, "image/")
}

// IsPDF returns true if the content type is a PDF document.
func IsPDF(contentType string) bool {
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))
	return baseType == "application/pdf"
}

// IsDocument returns true if the content type is a document format
// (PDF, Word, etc.).
func IsDocument(contentType string) bool {
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))

	documentTypes := map[string]bool{
		"application/pdf": true,
		// Microsoft Word
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		// LibreOffice/OpenOffice
		"application/vnd.oasis.opendocument.text": true,
	}

	return documentTypes[baseType]
}

// =============================================================================
// File Extension Helpers
// =============================================================================

// extensionForContentType returns a common file extension for a MIME type.
// This is useful when generating filenames from content types.
func extensionForContentType(contentType string) string {
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))

	// Common mappings
	extensions := map[string]string{
		"image/jpeg":      ".jpg",
		"image/jpg":       ".jpg",
		"image/png":       ".png",
		"image/webp":      ".webp",
		"image/heic":      ".heic",
		"image/heif":      ".heif",
		"application/pdf": ".pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/msword": ".doc",
	}

	if ext, ok := extensions[baseType]; ok {
		return ext
	}

	// Fall back to using mime package's reverse lookup
	// Get all extensions for this type and return the first one
	exts, err := mime.ExtensionsByType(contentType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}

	return ".bin"
}
