package anthropic

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/DukeRupert/lukaut/internal/ai"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

const (
	// APIBaseURL is the base URL for the Anthropic API
	APIBaseURL = "https://api.anthropic.com/v1/messages"

	// APIVersion is the Anthropic API version
	APIVersion = "2023-06-01"

	// DefaultModel is the default Claude model to use
	DefaultModel = "claude-3-5-sonnet-20241022"

	// MaxImageSize is the maximum image size in bytes (20MB)
	MaxImageSize = 20 * 1024 * 1024

	// Pricing in cents per 1M tokens for claude-3-5-sonnet
	PricingInputCents  = 300  // $3 per 1M input tokens
	PricingOutputCents = 1500 // $15 per 1M output tokens
)

// Config contains configuration for the Anthropic provider
type Config struct {
	APIKey         string
	Model          string
	ProviderConfig ai.ProviderConfig
}

// Provider implements the AIProvider interface using Anthropic's Claude API
type Provider struct {
	config  Config
	client  *http.Client
	queries *repository.Queries
	logger  *slog.Logger
}

// New creates a new Anthropic AI provider
func New(config Config, queries *repository.Queries, logger *slog.Logger) (*Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}

	// Set defaults
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.ProviderConfig.MaxRetries == 0 {
		config.ProviderConfig.MaxRetries = 3
	}
	if config.ProviderConfig.RetryBaseDelay == 0 {
		config.ProviderConfig.RetryBaseDelay = 1 * time.Second
	}
	if config.ProviderConfig.RequestTimeout == 0 {
		config.ProviderConfig.RequestTimeout = 60 * time.Second
	}

	return &Provider{
		config: config,
		client: &http.Client{
			Timeout: config.ProviderConfig.RequestTimeout,
		},
		queries: queries,
		logger:  logger,
	}, nil
}

// AnalyzeImage analyzes a construction site image for OSHA violations using Claude
func (p *Provider) AnalyzeImage(ctx context.Context, params ai.AnalyzeImageParams) (*ai.AnalysisResult, error) {
	startTime := time.Now()

	// Validate input
	if err := p.validateImageParams(params); err != nil {
		return nil, ai.WrapError("analyze image", err)
	}

	// Build the request
	req, err := p.buildAnalyzeImageRequest(ctx, params)
	if err != nil {
		return nil, ai.WrapError("build request", err)
	}

	// Execute with retry logic
	resp, err := p.executeWithRetry(ctx, req)
	if err != nil {
		return nil, ai.WrapError("execute request", err)
	}

	// Parse the response
	result, err := p.parseAnalysisResponse(resp)
	if err != nil {
		return nil, ai.WrapError("parse response", err)
	}

	// Calculate cost
	duration := time.Since(startTime)
	result.Usage = ai.UsageInfo{
		Model:        p.config.Model,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		CostCents:    p.calculateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens),
		Duration:     duration,
	}

	// Track usage in database
	if err := p.trackUsage(ctx, params.UserID, params.InspectionID, result.Usage, "analyze_image"); err != nil {
		// Log but don't fail the request
		p.logger.Error("Failed to track AI usage", "error", err)
	}

	return result, nil
}

// MatchRegulations finds relevant OSHA regulations using PostgreSQL full-text search
func (p *Provider) MatchRegulations(ctx context.Context, params ai.MatchParams) ([]ai.RegulationMatch, error) {
	// Default max results
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	// Search regulations using full-text search
	results, err := p.queries.SearchRegulations(ctx, repository.SearchRegulationsParams{
		WebsearchToTsquery: params.ViolationDescription,
		Limit:              int32(maxResults),
	})
	if err != nil {
		return nil, fmt.Errorf("search regulations: %w", err)
	}

	// Convert to domain types
	matches := make([]ai.RegulationMatch, 0, len(results))
	for i, result := range results {
		match := ai.RegulationMatch{
			RegulationID:   result.ID,
			StandardNumber: result.StandardNumber,
			Title:          result.Title,
			Category:       result.Category,
			RelevanceScore: float64(result.Rank),
			Explanation:    "", // No explanation in MVP - could add AI-generated explanation later
			IsPrimary:      i == 0, // First result is primary
		}
		matches = append(matches, match)
	}

	return matches, nil
}

// validateImageParams validates the image analysis parameters
func (p *Provider) validateImageParams(params ai.AnalyzeImageParams) error {
	if len(params.ImageData) == 0 {
		return ai.EAIInvalidImage
	}
	if len(params.ImageData) > MaxImageSize {
		return fmt.Errorf("%w: image size %d exceeds maximum %d", ai.EAIInvalidImage, len(params.ImageData), MaxImageSize)
	}
	if params.ContentType == "" {
		return fmt.Errorf("%w: content type is required", ai.EAIInvalidImage)
	}
	// Validate content type
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !validTypes[params.ContentType] {
		return fmt.Errorf("%w: unsupported content type %s", ai.EAIInvalidImage, params.ContentType)
	}
	return nil
}

// buildAnalyzeImageRequest builds the HTTP request for image analysis
func (p *Provider) buildAnalyzeImageRequest(ctx context.Context, params ai.AnalyzeImageParams) (*http.Request, error) {
	// Encode image to base64
	imageB64 := base64.StdEncoding.EncodeToString(params.ImageData)

	// Build the request body
	reqBody := apiRequest{
		Model:     p.config.Model,
		MaxTokens: 4096,
		Messages: []apiMessage{
			{
				Role: "user",
				Content: []apiContent{
					{
						Type: "image",
						Source: &apiImageSource{
							Type:      "base64",
							MediaType: params.ContentType,
							Data:      imageB64,
						},
					},
					{
						Type: "text",
						Text: buildImageAnalysisPrompt(params.Context),
					},
				},
			},
		},
	}

	// Marshal to JSON
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", APIBaseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", APIVersion)

	return req, nil
}

// executeWithRetry executes an HTTP request with exponential backoff retry
func (p *Provider) executeWithRetry(ctx context.Context, req *http.Request) (*apiResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= p.config.ProviderConfig.MaxRetries; attempt++ {
		resp, err := p.executeRequest(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Only retry on retryable errors
		if !ai.IsRetryable(err) {
			return nil, err
		}

		// Don't retry if we've exhausted attempts
		if attempt >= p.config.ProviderConfig.MaxRetries {
			break
		}

		// Calculate backoff delay (exponential: base * 2^(attempt-1))
		delay := p.config.ProviderConfig.RetryBaseDelay * time.Duration(1<<(attempt-1))
		p.logger.Info("Retrying AI request", "attempt", attempt, "delay", delay, "error", err)

		// Wait before retry
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Need to recreate request body for retry since it was consumed
		// This is safe because we're only retrying transient errors
		if req.Body != nil {
			// Get body bytes from original request
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("read request body for retry: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return nil, lastErr
}

// executeRequest executes a single HTTP request
func (p *Provider) executeRequest(ctx context.Context, req *http.Request) (*apiResponse, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		// Network errors are typically retryable
		return nil, ai.EAIUnavailable
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Check for errors based on status code
	if resp.StatusCode != http.StatusOK {
		return nil, p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	// Parse successful response
	var apiResp apiResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}

// mapHTTPError maps HTTP status codes to domain errors
func (p *Provider) mapHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp apiErrorResponse
	_ = json.Unmarshal(body, &errResp)

	switch statusCode {
	case http.StatusUnauthorized:
		return ai.EAIUnauthorized
	case http.StatusTooManyRequests:
		return ai.EAIRateLimit
	case http.StatusRequestTimeout:
		return ai.EAITimeout
	case http.StatusBadRequest:
		// Check if it's an image-related error
		if errResp.Error.Type == "invalid_request_error" {
			return ai.EAIInvalidImage
		}
		return fmt.Errorf("bad request: %s", errResp.Error.Message)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return ai.EAIUnavailable
	default:
		return fmt.Errorf("API error (status %d): %s", statusCode, errResp.Error.Message)
	}
}

// parseAnalysisResponse parses the API response into an AnalysisResult
func (p *Provider) parseAnalysisResponse(resp *apiResponse) (*ai.AnalysisResult, error) {
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	// Get the text content
	var textContent string
	for _, content := range resp.Content {
		if content.Type == "text" {
			textContent = content.Text
			break
		}
	}

	if textContent == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	// Parse JSON from text content
	var output analysisOutput
	if err := json.Unmarshal([]byte(textContent), &output); err != nil {
		return nil, fmt.Errorf("parse analysis output: %w", err)
	}

	// Convert to domain types
	result := &ai.AnalysisResult{
		Violations:          make([]ai.PotentialViolation, 0, len(output.Violations)),
		GeneralObservations: output.GeneralObservations,
		ImageQualityNotes:   output.ImageQualityNotes,
	}

	for _, v := range output.Violations {
		violation := ai.PotentialViolation{
			Description:          v.Description,
			Location:             v.Location,
			Confidence:           ai.Confidence(v.Confidence),
			Category:             v.Category,
			Severity:             ai.Severity(v.Severity),
			SuggestedRegulations: v.SuggestedRegulations,
		}

		// Add bounding box if present
		if v.BoundingBox != nil {
			violation.BoundingBox = &ai.BoundingBox{
				X:      v.BoundingBox.X,
				Y:      v.BoundingBox.Y,
				Width:  v.BoundingBox.Width,
				Height: v.BoundingBox.Height,
			}
		}

		// Validate and set defaults
		if !violation.Confidence.Valid() {
			violation.Confidence = ai.ConfidenceMedium
		}
		if !violation.Severity.Valid() {
			violation.Severity = ai.SeverityMedium
		}

		result.Violations = append(result.Violations, violation)
	}

	return result, nil
}

// calculateCost calculates the cost in cents for the given token usage
func (p *Provider) calculateCost(inputTokens, outputTokens int) int {
	inputCost := (inputTokens * PricingInputCents) / 1_000_000
	outputCost := (outputTokens * PricingOutputCents) / 1_000_000
	return inputCost + outputCost
}

// trackUsage records AI usage in the database
func (p *Provider) trackUsage(ctx context.Context, userID, inspectionID uuid.UUID, usage ai.UsageInfo, requestType string) error {
	_, err := p.queries.CreateAIUsage(ctx, repository.CreateAIUsageParams{
		UserID: userID,
		InspectionID: uuid.NullUUID{
			UUID:  inspectionID,
			Valid: inspectionID != uuid.Nil,
		},
		Model:        usage.Model,
		InputTokens:  int32(usage.InputTokens),
		OutputTokens: int32(usage.OutputTokens),
		CostCents:    int32(usage.CostCents),
		RequestType:  requestType,
	})
	return err
}

// API request/response types

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string       `json:"role"`
	Content []apiContent `json:"content"`
}

type apiContent struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	Source *apiImageSource `json:"source,omitempty"`
}

type apiImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type apiResponse struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []apiContentOutput `json:"content"`
	Model   string             `json:"model"`
	Usage   apiUsage           `json:"usage"`
}

type apiContentOutput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type apiErrorResponse struct {
	Type  string   `json:"type"`
	Error apiError `json:"error"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// analysisOutput represents the JSON structure returned by Claude
type analysisOutput struct {
	Violations          []outputViolation `json:"violations"`
	GeneralObservations string            `json:"general_observations"`
	ImageQualityNotes   string            `json:"image_quality_notes"`
}

type outputViolation struct {
	Description          string              `json:"description"`
	Location             string              `json:"location"`
	BoundingBox          *outputBoundingBox  `json:"bounding_box,omitempty"`
	Confidence           string              `json:"confidence"`
	Category             string              `json:"category"`
	Severity             string              `json:"severity"`
	SuggestedRegulations []string            `json:"suggested_regulations"`
}

type outputBoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
