package mock

import (
	"context"
	"log/slog"
	"time"

	"github.com/DukeRupert/lukaut/internal/ai"
	"github.com/google/uuid"
)

// Provider is a mock AI provider for testing and development
type Provider struct {
	logger *slog.Logger

	// Configurable responses for testing
	AnalyzeImageResponse     *ai.AnalysisResult
	AnalyzeImageError        error
	MatchRegulationsResponse []ai.RegulationMatch
	MatchRegulationsError    error

	// Call tracking for testing
	AnalyzeImageCalls     int
	MatchRegulationsCalls int
}

// New creates a new mock AI provider
func New(logger *slog.Logger) *Provider {
	return &Provider{
		logger: logger,
	}
}

// AnalyzeImage returns a canned response with sample violations
func (p *Provider) AnalyzeImage(ctx context.Context, params ai.AnalyzeImageParams) (*ai.AnalysisResult, error) {
	p.AnalyzeImageCalls++

	// If a custom response or error is set, use it
	if p.AnalyzeImageError != nil {
		return nil, p.AnalyzeImageError
	}
	if p.AnalyzeImageResponse != nil {
		return p.AnalyzeImageResponse, nil
	}

	// Default canned response
	return &ai.AnalysisResult{
		Violations: []ai.PotentialViolation{
			{
				Description: "Worker not wearing hard hat in construction zone",
				Location:    "Center-left of image, near scaffolding",
				BoundingBox: &ai.BoundingBox{
					X:      0.25,
					Y:      0.30,
					Width:  0.15,
					Height: 0.25,
				},
				Confidence:           ai.ConfidenceHigh,
				Category:             "Personal Protective Equipment",
				Severity:             ai.SeveritySerious,
				SuggestedRegulations: []string{"1926.100(a)", "1926.100(b)"},
			},
			{
				Description: "Scaffolding appears to lack proper guardrails",
				Location:    "Right side of image, upper platform",
				BoundingBox: &ai.BoundingBox{
					X:      0.60,
					Y:      0.15,
					Width:  0.30,
					Height: 0.40,
				},
				Confidence:           ai.ConfidenceMedium,
				Category:             "Fall Protection",
				Severity:             ai.SeverityCritical,
				SuggestedRegulations: []string{"1926.451(g)(1)", "1926.451(g)(4)"},
			},
			{
				Description: "Construction materials stored near edge without barriers",
				Location:    "Bottom right corner",
				BoundingBox: &ai.BoundingBox{
					X:      0.70,
					Y:      0.75,
					Width:  0.20,
					Height: 0.15,
				},
				Confidence:           ai.ConfidenceMedium,
				Category:             "Housekeeping",
				Severity:             ai.SeverityOther,
				SuggestedRegulations: []string{"1926.250(a)(1)"},
			},
		},
		GeneralObservations: "Active construction site with multiple workers. Scaffolding is prominently featured. Overall site appears moderately organized but has several safety concerns.",
		ImageQualityNotes:   "Image quality is good with clear visibility. Adequate lighting and resolution for safety analysis.",
		Usage: ai.UsageInfo{
			Model:        "mock-ai-v1",
			InputTokens:  1250,
			OutputTokens: 850,
			CostCents:    5,
			Duration:     250 * time.Millisecond,
		},
	}, nil
}

// MatchRegulations returns canned regulation matches
func (p *Provider) MatchRegulations(ctx context.Context, params ai.MatchParams) ([]ai.RegulationMatch, error) {
	p.MatchRegulationsCalls++

	// If a custom response or error is set, use it
	if p.MatchRegulationsError != nil {
		return nil, p.MatchRegulationsError
	}
	if p.MatchRegulationsResponse != nil {
		return p.MatchRegulationsResponse, nil
	}

	// Default canned response based on common categories
	matches := []ai.RegulationMatch{
		{
			RegulationID:   uuid.New(), // Mock UUID
			StandardNumber: "1926.501(b)(1)",
			Title:          "Fall protection - Unprotected sides and edges",
			Category:       "Fall Protection",
			RelevanceScore: 0.95,
			Explanation:    "Highly relevant for fall hazards at elevated work surfaces",
			IsPrimary:      true,
		},
		{
			RegulationID:   uuid.New(),
			StandardNumber: "1926.451(g)(1)",
			Title:          "Scaffolding - Guardrail systems",
			Category:       "Scaffolding",
			RelevanceScore: 0.88,
			Explanation:    "Applies to scaffolding guardrail requirements",
			IsPrimary:      false,
		},
	}

	// Limit results if requested
	if params.MaxResults > 0 && len(matches) > params.MaxResults {
		matches = matches[:params.MaxResults]
	}

	return matches, nil
}

// Reset clears call counters and custom responses for testing
func (p *Provider) Reset() {
	p.AnalyzeImageCalls = 0
	p.MatchRegulationsCalls = 0
	p.AnalyzeImageResponse = nil
	p.AnalyzeImageError = nil
	p.MatchRegulationsResponse = nil
	p.MatchRegulationsError = nil
}
