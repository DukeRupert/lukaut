// Package report provides PDF and DOCX report generation for safety inspections.
//
// This package defines a Generator interface implemented by PDFGenerator and
// DOCXGenerator, along with common helpers for formatting and styling reports
// in the Lukaut brand style.
package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
)

// =============================================================================
// Generator Interface
// =============================================================================

// Generator defines the interface for report generators.
// Implementations handle the specifics of each format (PDF, DOCX).
type Generator interface {
	// Generate creates a report and writes it to the provided writer.
	// Returns the number of bytes written and any error.
	Generate(ctx context.Context, data *domain.ReportData, w io.Writer) (int64, error)

	// Format returns the output format of this generator.
	Format() domain.ReportFormat
}

// =============================================================================
// Brand Colors
// =============================================================================

// BrandColors defines the color palette for reports.
// These match the Lukaut brand colors from Tailwind config.
var BrandColors = struct {
	Navy         string // Primary brand color
	SafetyOrange string // Accent color for CTAs
	TextDark     string // Primary text
	TextMuted    string // Secondary text
	Border       string // Borders and dividers
	Background   string // Light background
	White        string // White
}{
	Navy:         "#1E3A5F",
	SafetyOrange: "#FF6B35",
	TextDark:     "#1F2937",
	TextMuted:    "#6B7280",
	Border:       "#E5E7EB",
	Background:   "#F9FAFB",
	White:        "#FFFFFF",
}

// =============================================================================
// Severity Colors
// =============================================================================

// SeverityColors maps severity levels to display colors.
var SeverityColors = map[domain.ViolationSeverity]string{
	domain.ViolationSeverityCritical:       "#DC2626", // Red-600
	domain.ViolationSeveritySerious:        "#F59E0B", // Amber-500
	domain.ViolationSeverityOther:          "#3B82F6", // Blue-500
	domain.ViolationSeverityRecommendation: "#6B7280", // Gray-500
}

// SeverityColor returns the color for a severity level.
func SeverityColor(severity domain.ViolationSeverity) string {
	if color, ok := SeverityColors[severity]; ok {
		return color
	}
	return BrandColors.TextMuted
}

// SeverityLabel returns a human-readable label for severity.
func SeverityLabel(severity domain.ViolationSeverity) string {
	switch severity {
	case domain.ViolationSeverityCritical:
		return "Critical"
	case domain.ViolationSeveritySerious:
		return "Serious"
	case domain.ViolationSeverityOther:
		return "Other"
	case domain.ViolationSeverityRecommendation:
		return "Recommendation"
	default:
		return string(severity)
	}
}

// =============================================================================
// Color Conversion Helpers
// =============================================================================

// HexToRGB converts a hex color string to RGB values.
// Input format: "#RRGGBB" or "RRGGBB"
func HexToRGB(hex string) (r, g, b int) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 0, 0, 0
	}

	r = hexToDec(hex[0:2])
	g = hexToDec(hex[2:4])
	b = hexToDec(hex[4:6])
	return
}

// hexToDec converts a 2-character hex string to decimal.
func hexToDec(hex string) int {
	val := 0
	for _, c := range hex {
		val *= 16
		switch {
		case c >= '0' && c <= '9':
			val += int(c - '0')
		case c >= 'a' && c <= 'f':
			val += int(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			val += int(c - 'A' + 10)
		}
	}
	return val
}

// =============================================================================
// Text Formatting Helpers
// =============================================================================

// TruncateText truncates text to a maximum length, adding ellipsis if needed.
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// FormatDate formats a date for display in reports.
func FormatDate(t interface{ Format(string) string }) string {
	return t.Format("January 2, 2006")
}

// FormatDateTime formats a datetime for display in reports.
func FormatDateTime(t interface{ Format(string) string }) string {
	return t.Format("January 2, 2006 at 3:04 PM")
}

// =============================================================================
// Image Download
// =============================================================================

// ImageData holds downloaded image data for embedding in reports.
type ImageData struct {
	Data        []byte
	ContentType string
}

// ImageDownloader abstracts image fetching for report generation.
// This allows testing report generation without network I/O.
type ImageDownloader interface {
	Download(ctx context.Context, url string) (*ImageData, error)
}

// HTTPImageDownloader fetches images over HTTP.
type HTTPImageDownloader struct {
	client *http.Client
}

// NewHTTPImageDownloader creates an ImageDownloader that fetches images over HTTP.
func NewHTTPImageDownloader() *HTTPImageDownloader {
	return &HTTPImageDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Download fetches an image from a URL and returns its data.
// Returns nil, nil if the URL is empty.
func (d *HTTPImageDownloader) Download(ctx context.Context, url string) (*ImageData, error) {
	if url == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("read image data: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg" // Default fallback
	}

	return &ImageData{
		Data:        buf.Bytes(),
		ContentType: contentType,
	}, nil
}
