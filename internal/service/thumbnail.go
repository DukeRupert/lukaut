// Package service contains business logic for the Lukaut application.
//
// This file implements thumbnail generation for uploaded inspection photos.
package service

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/disintegration/imaging"
)

// =============================================================================
// Interface Definition
// =============================================================================

// ThumbnailProcessor handles thumbnail generation from images.
type ThumbnailProcessor interface {
	// GenerateThumbnail creates a thumbnail from the provided image data.
	// Returns the thumbnail bytes (as JPEG), original width, and original height.
	// The thumbnail will fit within maxWidth x maxHeight while preserving aspect ratio.
	GenerateThumbnail(data io.Reader, maxWidth, maxHeight int) ([]byte, int, int, error)
}

// =============================================================================
// Implementation
// =============================================================================

// imagingProcessor implements ThumbnailProcessor using the imaging library.
type imagingProcessor struct{}

// NewImagingProcessor creates a new thumbnail processor using the imaging library.
func NewImagingProcessor() ThumbnailProcessor {
	return &imagingProcessor{}
}

// GenerateThumbnail creates a thumbnail from the provided image data.
//
// The thumbnail is resized to fit within maxWidth x maxHeight while preserving
// the original aspect ratio. The output is always JPEG format with quality 85.
//
// Returns:
//   - thumbnail bytes (JPEG format)
//   - original image width
//   - original image height
//   - error if generation fails
func (p *imagingProcessor) GenerateThumbnail(data io.Reader, maxWidth, maxHeight int) ([]byte, int, int, error) {
	// Decode the image
	img, format, err := image.Decode(data)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Resize to fit within maxWidth x maxHeight while preserving aspect ratio
	// imaging.Fit will resize the image to fit within the specified dimensions
	// while maintaining aspect ratio
	thumbnail := imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)

	// Encode thumbnail as JPEG
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumbnail, imaging.JPEG, imaging.JPEGQuality(domain.ThumbnailJPEGQuality)); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	_ = format // Acknowledge we read the format even though we don't use it

	return buf.Bytes(), originalWidth, originalHeight, nil
}
