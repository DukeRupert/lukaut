package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/DukeRupert/lukaut/internal/ai"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/storage"
	"github.com/DukeRupert/lukaut/internal/worker"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// AnalyzeInspectionHandler processes jobs that analyze inspection images for violations.
// It sends images to the AI service and creates violation records based on the results.
type AnalyzeInspectionHandler struct {
	queries    *repository.Queries
	aiProvider ai.AIProvider
	storage    storage.Storage
	logger     *slog.Logger
}

// NewAnalyzeInspectionHandler creates a new handler for inspection analysis jobs.
func NewAnalyzeInspectionHandler(
	queries *repository.Queries,
	aiProvider ai.AIProvider,
	storage storage.Storage,
	logger *slog.Logger,
) *AnalyzeInspectionHandler {
	return &AnalyzeInspectionHandler{
		queries:    queries,
		aiProvider: aiProvider,
		storage:    storage,
		logger:     logger,
	}
}

// Type returns the job type identifier.
func (h *AnalyzeInspectionHandler) Type() string {
	return worker.JobTypeAnalyzeInspection
}

// Handle executes the inspection analysis job.
// It processes all pending images for an inspection, analyzes them with AI,
// and creates violation records with linked regulations.
func (h *AnalyzeInspectionHandler) Handle(ctx context.Context, payload []byte) error {
	// Unmarshal the payload
	var p worker.AnalyzeInspectionPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return worker.NewPermanentError(fmt.Errorf("invalid payload: %w", err))
	}

	h.logger.Info("Analyzing inspection",
		"inspection_id", p.InspectionID,
		"user_id", p.UserID,
	)

	// 1. Fetch and validate inspection
	inspection, err := h.queries.GetInspectionByID(ctx, p.InspectionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return worker.NewPermanentError(fmt.Errorf("inspection not found: %w", err))
		}
		// Database error - retryable
		return fmt.Errorf("fetch inspection: %w", err)
	}

	// Verify ownership
	if inspection.UserID != p.UserID {
		return worker.NewPermanentError(fmt.Errorf("inspection does not belong to user"))
	}

	// Check valid status (draft or analyzing)
	status := domain.InspectionStatus(inspection.Status)
	if status != domain.InspectionStatusDraft && status != domain.InspectionStatusAnalyzing {
		return worker.NewPermanentError(fmt.Errorf("invalid inspection status: %s (expected draft or analyzing)", status))
	}

	// 2. Update inspection status to "analyzing"
	if status == domain.InspectionStatusDraft {
		err = h.queries.UpdateInspectionStatus(ctx, repository.UpdateInspectionStatusParams{
			ID:     p.InspectionID,
			Status: domain.InspectionStatusAnalyzing.String(),
		})
		if err != nil {
			return fmt.Errorf("update inspection status to analyzing: %w", err)
		}
		h.logger.Info("Updated inspection status to analyzing", "inspection_id", p.InspectionID)
	}

	// 3. Fetch pending images
	images, err := h.queries.ListPendingImagesByInspectionID(ctx, p.InspectionID)
	if err != nil {
		return fmt.Errorf("fetch pending images: %w", err)
	}

	h.logger.Info("Found pending images", "inspection_id", p.InspectionID, "count", len(images))

	// 4. Process each image
	successCount := 0
	failCount := 0

	for _, img := range images {
		imgLogger := h.logger.With("image_id", img.ID, "inspection_id", p.InspectionID)
		imgLogger.Info("Processing image", "storage_key", img.StorageKey)

		// Mark image as analyzing
		err = h.queries.UpdateImageAnalysisStatus(ctx, repository.UpdateImageAnalysisStatusParams{
			ID:             img.ID,
			AnalysisStatus: sql.NullString{String: domain.ImageAnalysisStatusAnalyzing.String(), Valid: true},
		})
		if err != nil {
			imgLogger.Error("Failed to mark image as analyzing", "error", err)
			// Continue with other images
			continue
		}

		// Analyze the image
		if err := h.analyzeImage(ctx, img, p.InspectionID, p.UserID, imgLogger); err != nil {
			imgLogger.Error("Image analysis failed", "error", err)
			failCount++

			// Mark image as failed
			if markErr := h.markImageFailed(ctx, img.ID); markErr != nil {
				imgLogger.Error("Failed to mark image as failed", "error", markErr)
			}
			// Continue processing other images (partial failure is acceptable)
			continue
		}

		// Mark image as completed
		if err := h.markImageCompleted(ctx, img.ID); err != nil {
			imgLogger.Error("Failed to mark image as completed", "error", err)
			// Don't fail the whole job for this
		}

		successCount++
		imgLogger.Info("Image analysis completed successfully")
	}

	// 5. Update inspection status to "review"
	err = h.queries.UpdateInspectionStatus(ctx, repository.UpdateInspectionStatusParams{
		ID:     p.InspectionID,
		Status: domain.InspectionStatusReview.String(),
	})
	if err != nil {
		return fmt.Errorf("update inspection status to review: %w", err)
	}

	h.logger.Info("Inspection analysis completed",
		"inspection_id", p.InspectionID,
		"total_images", len(images),
		"success", successCount,
		"failed", failCount,
	)

	return nil
}

// analyzeImage downloads and analyzes a single image, creating violation records.
func (h *AnalyzeInspectionHandler) analyzeImage(
	ctx context.Context,
	img repository.Image,
	inspectionID uuid.UUID,
	userID uuid.UUID,
	logger *slog.Logger,
) error {
	// Download image from storage
	reader, objInfo, err := h.storage.Get(ctx, img.StorageKey)
	if err != nil {
		return fmt.Errorf("download image from storage: %w", err)
	}
	defer reader.Close()

	// Read image data into memory
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read image data: %w", err)
	}

	logger.Info("Downloaded image from storage",
		"size_bytes", len(imageData),
		"content_type", objInfo.ContentType,
	)

	// Call AI provider to analyze the image
	analysisResult, err := h.aiProvider.AnalyzeImage(ctx, ai.AnalyzeImageParams{
		ImageData:    imageData,
		ContentType:  objInfo.ContentType,
		Context:      "", // Optional: could include inspector notes from inspection
		ImageID:      img.ID,
		InspectionID: inspectionID,
		UserID:       userID,
	})
	if err != nil {
		// Check if this is a retryable AI error
		if ai.IsRetryable(err) {
			// Retryable errors like rate limits should propagate up
			return fmt.Errorf("ai analysis (retryable): %w", err)
		}
		// Invalid image or content policy violations are permanent
		if errors.Is(err, ai.EAIInvalidImage) || errors.Is(err, ai.EAIContentPolicy) {
			return worker.NewPermanentError(fmt.Errorf("ai analysis (permanent): %w", err))
		}
		// Other AI errors
		return fmt.Errorf("ai analysis: %w", err)
	}

	logger.Info("AI analysis completed",
		"violations_found", len(analysisResult.Violations),
		"input_tokens", analysisResult.Usage.InputTokens,
		"output_tokens", analysisResult.Usage.OutputTokens,
		"cost_cents", analysisResult.Usage.CostCents,
	)

	// Store each violation
	for i, violation := range analysisResult.Violations {
		if err := h.storeViolation(ctx, violation, img.ID, inspectionID, i+1, logger); err != nil {
			// Log but don't fail the whole image analysis
			logger.Error("Failed to store violation", "error", err, "violation_index", i)
		}
	}

	return nil
}

// storeViolation creates a violation record and links it to relevant regulations.
func (h *AnalyzeInspectionHandler) storeViolation(
	ctx context.Context,
	violation ai.PotentialViolation,
	imageID uuid.UUID,
	inspectionID uuid.UUID,
	sortOrder int,
	logger *slog.Logger,
) error {
	// Convert bounding box to JSON if present
	var boundingBoxJSON pqtype.NullRawMessage
	if violation.BoundingBox != nil {
		data, err := json.Marshal(violation.BoundingBox)
		if err != nil {
			return fmt.Errorf("marshal bounding box: %w", err)
		}
		boundingBoxJSON = pqtype.NullRawMessage{
			RawMessage: data,
			Valid:      true,
		}
	}

	// Create the violation record
	createdViolation, err := h.queries.CreateViolation(ctx, repository.CreateViolationParams{
		InspectionID: inspectionID,
		ImageID:      uuid.NullUUID{UUID: imageID, Valid: true},
		Description:  violation.Description,
		AiDescription: sql.NullString{
			String: violation.Description + " (Location: " + violation.Location + ")",
			Valid:  true,
		},
		Confidence: sql.NullString{
			String: string(violation.Confidence),
			Valid:  true,
		},
		BoundingBox: boundingBoxJSON,
		Status:      "pending", // Inspector needs to review
		Severity: sql.NullString{
			String: string(violation.Severity),
			Valid:  true,
		},
		InspectorNotes: sql.NullString{Valid: false}, // No notes yet
		SortOrder: sql.NullInt32{
			Int32: int32(sortOrder),
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("create violation: %w", err)
	}

	logger.Info("Created violation record",
		"violation_id", createdViolation.ID,
		"description", violation.Description,
		"confidence", violation.Confidence,
		"severity", violation.Severity,
	)

	// Link regulations to the violation
	if err := h.linkRegulations(ctx, createdViolation.ID, violation, logger); err != nil {
		// Log but don't fail - violation was created successfully
		logger.Error("Failed to link regulations", "error", err, "violation_id", createdViolation.ID)
	}

	return nil
}

// linkRegulations searches for and links relevant OSHA regulations to a violation.
func (h *AnalyzeInspectionHandler) linkRegulations(
	ctx context.Context,
	violationID uuid.UUID,
	violation ai.PotentialViolation,
	logger *slog.Logger,
) error {
	// Build search query from violation description and category
	searchQuery := violation.Description
	if violation.Category != "" {
		searchQuery = violation.Category + " " + searchQuery
	}

	// Search for matching regulations using full-text search
	regulations, err := h.queries.SearchRegulations(ctx, repository.SearchRegulationsParams{
		WebsearchToTsquery: searchQuery,
		Limit:              5, // Limit to top 5 most relevant regulations
	})
	if err != nil {
		return fmt.Errorf("search regulations: %w", err)
	}

	logger.Info("Found matching regulations",
		"violation_id", violationID,
		"count", len(regulations),
		"query", searchQuery,
	)

	// Link each regulation to the violation
	for i, reg := range regulations {
		isPrimary := i == 0 // First (highest ranked) regulation is primary

		// Convert float32 rank to string for storage
		rankStr := fmt.Sprintf("%.6f", reg.Rank)

		_, err := h.queries.CreateViolationRegulation(ctx, repository.CreateViolationRegulationParams{
			ViolationID:  violationID,
			RegulationID: reg.ID,
			RelevanceScore: sql.NullString{
				String: rankStr,
				Valid:  true,
			},
			AiExplanation: sql.NullString{
				String: fmt.Sprintf("AI identified this as a %s violation", violation.Category),
				Valid:  violation.Category != "",
			},
			IsPrimary: sql.NullBool{
				Bool:  isPrimary,
				Valid: true,
			},
		})
		if err != nil {
			logger.Error("Failed to link regulation",
				"error", err,
				"violation_id", violationID,
				"regulation_id", reg.ID,
			)
			// Continue linking other regulations
			continue
		}

		logger.Info("Linked regulation to violation",
			"violation_id", violationID,
			"regulation_id", reg.ID,
			"standard_number", reg.StandardNumber,
			"is_primary", isPrimary,
		)
	}

	return nil
}

// markImageFailed updates an image's analysis status to failed.
func (h *AnalyzeInspectionHandler) markImageFailed(ctx context.Context, imageID uuid.UUID) error {
	return h.queries.UpdateImageAnalysisStatus(ctx, repository.UpdateImageAnalysisStatusParams{
		ID:             imageID,
		AnalysisStatus: sql.NullString{String: domain.ImageAnalysisStatusFailed.String(), Valid: true},
	})
}

// markImageCompleted updates an image's analysis status to completed.
func (h *AnalyzeInspectionHandler) markImageCompleted(ctx context.Context, imageID uuid.UUID) error {
	return h.queries.UpdateImageAnalysisStatus(ctx, repository.UpdateImageAnalysisStatusParams{
		ID:             imageID,
		AnalysisStatus: sql.NullString{String: domain.ImageAnalysisStatusCompleted.String(), Valid: true},
	})
}
