package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AIProvider defines the interface for AI-powered construction site analysis
type AIProvider interface {
	// AnalyzeImage analyzes a construction site photo for OSHA violations
	AnalyzeImage(ctx context.Context, params AnalyzeImageParams) (*AnalysisResult, error)

	// MatchRegulations finds relevant OSHA regulations for a violation description
	MatchRegulations(ctx context.Context, params MatchParams) ([]RegulationMatch, error)
}

// AnalyzeImageParams contains parameters for image analysis
type AnalyzeImageParams struct {
	ImageData    []byte    // Raw image bytes
	ContentType  string    // MIME type (e.g., "image/jpeg")
	Context      string    // Optional context provided by inspector
	ImageID      uuid.UUID // Image ID for tracking
	InspectionID uuid.UUID // Inspection ID for tracking
	UserID       uuid.UUID // User ID for usage tracking
}

// MatchParams contains parameters for regulation matching
type MatchParams struct {
	ViolationDescription string    // Description of the violation to match
	Category             string    // Optional category filter
	MaxResults           int       // Maximum number of results to return
	UserID               uuid.UUID // User ID for tracking
	InspectionID         uuid.UUID // Inspection ID for tracking
}

// AnalysisResult contains the complete analysis of a construction site image
type AnalysisResult struct {
	Violations          []PotentialViolation // Identified potential violations
	GeneralObservations string               // General safety observations
	ImageQualityNotes   string               // Notes about image quality/usability
	Usage               UsageInfo            // Token usage and cost information
}

// PotentialViolation represents a single identified safety concern
type PotentialViolation struct {
	Description          string       // What the violation is
	Location             string       // Where in the image (human-readable)
	BoundingBox          *BoundingBox // Optional coordinates in image
	Confidence           Confidence   // How confident the AI is
	Category             string       // OSHA category (e.g., "Fall Protection")
	Severity             Severity     // Estimated severity level
	SuggestedRegulations []string     // Suggested OSHA regulation numbers
}

// BoundingBox represents coordinates in an image (normalized 0-1)
type BoundingBox struct {
	X      float64 // Left edge (0 = left, 1 = right)
	Y      float64 // Top edge (0 = top, 1 = bottom)
	Width  float64 // Width (0-1)
	Height float64 // Height (0-1)
}

// RegulationMatch represents a matched OSHA regulation
type RegulationMatch struct {
	RegulationID   uuid.UUID // Database ID
	StandardNumber string    // OSHA standard number (e.g., "1926.501(b)(1)")
	Title          string    // Regulation title
	Category       string    // Category
	RelevanceScore float64   // How relevant (0-1, from full-text search rank)
	Explanation    string    // Why this regulation is relevant
	IsPrimary      bool      // Is this the primary applicable regulation?
}

// UsageInfo tracks API usage for billing and monitoring
type UsageInfo struct {
	Model        string        // AI model used
	InputTokens  int           // Tokens in the request
	OutputTokens int           // Tokens in the response
	CostCents    int           // Estimated cost in cents
	Duration     time.Duration // Request duration
}

// Confidence levels for violation detection
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"   // 90%+ confident
	ConfidenceMedium Confidence = "medium" // 60-90% confident
	ConfidenceLow    Confidence = "low"    // 30-60% confident
)

// Valid checks if the confidence level is valid
func (c Confidence) Valid() bool {
	switch c {
	case ConfidenceHigh, ConfidenceMedium, ConfidenceLow:
		return true
	default:
		return false
	}
}

// Severity levels for violations (matches domain.ViolationSeverity)
type Severity string

const (
	SeverityCritical       Severity = "critical"       // Imminent danger
	SeveritySerious        Severity = "serious"        // Serious hazard with potential for severe injury
	SeverityOther          Severity = "other"          // Violation that doesn't fit serious category
	SeverityRecommendation Severity = "recommendation" // Best practice, may not be regulatory violation
)

// Valid checks if the severity level is valid
func (s Severity) Valid() bool {
	switch s {
	case SeverityCritical, SeveritySerious, SeverityOther, SeverityRecommendation:
		return true
	default:
		return false
	}
}

// ProviderConfig contains common configuration for AI providers
type ProviderConfig struct {
	MaxRetries     int           // Maximum retry attempts for transient errors
	RetryBaseDelay time.Duration // Base delay for exponential backoff
	RequestTimeout time.Duration // Timeout for individual requests
}

// Error codes for AI provider operations
var (
	// EAIRateLimit indicates the API rate limit has been exceeded
	EAIRateLimit = errors.New("ai provider rate limit exceeded")

	// EAIInvalidImage indicates the image format or content is invalid
	EAIInvalidImage = errors.New("invalid image format or content")

	// EAIContentPolicy indicates the image violates content policy
	EAIContentPolicy = errors.New("image violates content policy")

	// EAITimeout indicates the request timed out
	EAITimeout = errors.New("ai request timed out")

	// EAIUnavailable indicates the AI service is temporarily unavailable
	EAIUnavailable = errors.New("ai service temporarily unavailable")

	// EAIUnauthorized indicates invalid API credentials
	EAIUnauthorized = errors.New("ai provider authentication failed")
)

// IsRetryable returns true if the error is a transient error that can be retried
func IsRetryable(err error) bool {
	return errors.Is(err, EAIRateLimit) ||
		errors.Is(err, EAITimeout) ||
		errors.Is(err, EAIUnavailable)
}

// WrapError wraps an error with context about the AI operation
func WrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("ai %s: %w", operation, err)
}
