// Package service contains the business logic layer.
//
// This file implements the quota service for checking and enforcing
// rate limits based on subscription tier.
package service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/google/uuid"
)

// Job type constants matching the worker package.
const (
	JobTypeAnalyzeInspection = "analyze_inspection"
	JobTypeGenerateReport    = "generate_report"
)

// =============================================================================
// Interface Definition
// =============================================================================

// QuotaService defines operations for checking quota limits.
type QuotaService interface {
	// GetUsage returns the current quota usage for a user.
	GetUsage(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) (*domain.QuotaUsage, error)

	// CheckAnalysisQuota checks if the user has quota remaining for analysis jobs.
	// Returns nil if quota is available, or QuotaExceeded error if not.
	CheckAnalysisQuota(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) error

	// CheckReportQuota checks if the user has quota remaining for report jobs.
	// Returns nil if quota is available, or QuotaExceeded error if not.
	CheckReportQuota(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) error
}

// =============================================================================
// Implementation
// =============================================================================

type quotaService struct {
	queries *repository.Queries
	logger  *slog.Logger
}

// NewQuotaService creates a new QuotaService.
func NewQuotaService(queries *repository.Queries, logger *slog.Logger) QuotaService {
	return &quotaService{
		queries: queries,
		logger:  logger,
	}
}

// GetUsage returns the current quota usage for a user.
func (s *quotaService) GetUsage(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) (*domain.QuotaUsage, error) {
	const op = "quota.get_usage"

	quota := domain.GetTierQuota(tier)

	// If unlimited, return early
	if quota.UnlimitedAnalysis && quota.UnlimitedReports {
		return &domain.QuotaUsage{
			IsUnlimited: true,
		}, nil
	}

	// Get current month boundaries
	startOfMonth, endOfMonth := getCurrentMonthBoundaries()

	// Count analysis jobs
	analysisCount, err := s.queries.CountCompletedJobsByUserAndType(ctx, repository.CountCompletedJobsByUserAndTypeParams{
		JobType:       JobTypeAnalyzeInspection,
		Column2:       userID.String(),
		CompletedAt:   sql.NullTime{Time: startOfMonth, Valid: true},
		CompletedAt_2: sql.NullTime{Time: endOfMonth, Valid: true},
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count analysis jobs")
	}

	// Count report jobs
	reportCount, err := s.queries.CountCompletedJobsByUserAndType(ctx, repository.CountCompletedJobsByUserAndTypeParams{
		JobType:       JobTypeGenerateReport,
		Column2:       userID.String(),
		CompletedAt:   sql.NullTime{Time: startOfMonth, Valid: true},
		CompletedAt_2: sql.NullTime{Time: endOfMonth, Valid: true},
	})
	if err != nil {
		return nil, domain.Internal(err, op, "failed to count report jobs")
	}

	return &domain.QuotaUsage{
		AnalysisUsed:  analysisCount,
		AnalysisLimit: int64(quota.AnalysisPerMonth),
		ReportsUsed:   reportCount,
		ReportsLimit:  int64(quota.ReportsPerMonth),
		IsUnlimited:   false,
	}, nil
}

// CheckAnalysisQuota checks if the user has quota remaining for analysis jobs.
func (s *quotaService) CheckAnalysisQuota(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) error {
	const op = "quota.check_analysis"

	quota := domain.GetTierQuota(tier)

	// Unlimited tier - always allow
	if quota.UnlimitedAnalysis {
		return nil
	}

	// Get current month boundaries
	startOfMonth, endOfMonth := getCurrentMonthBoundaries()

	// Count completed analysis jobs this month
	count, err := s.queries.CountCompletedJobsByUserAndType(ctx, repository.CountCompletedJobsByUserAndTypeParams{
		JobType:       JobTypeAnalyzeInspection,
		Column2:       userID.String(),
		CompletedAt:   sql.NullTime{Time: startOfMonth, Valid: true},
		CompletedAt_2: sql.NullTime{Time: endOfMonth, Valid: true},
	})
	if err != nil {
		return domain.Internal(err, op, "failed to count analysis jobs")
	}

	limit := int64(quota.AnalysisPerMonth)
	if count >= limit {
		s.logger.Info("Analysis quota exceeded",
			"user_id", userID,
			"tier", tier,
			"used", count,
			"limit", limit,
		)
		return domain.QuotaExceeded(op, domain.QuotaTypeAnalysis, count, limit)
	}

	return nil
}

// CheckReportQuota checks if the user has quota remaining for report jobs.
func (s *quotaService) CheckReportQuota(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier) error {
	const op = "quota.check_report"

	quota := domain.GetTierQuota(tier)

	// Unlimited tier - always allow
	if quota.UnlimitedReports {
		return nil
	}

	// Get current month boundaries
	startOfMonth, endOfMonth := getCurrentMonthBoundaries()

	// Count completed report jobs this month
	count, err := s.queries.CountCompletedJobsByUserAndType(ctx, repository.CountCompletedJobsByUserAndTypeParams{
		JobType:       JobTypeGenerateReport,
		Column2:       userID.String(),
		CompletedAt:   sql.NullTime{Time: startOfMonth, Valid: true},
		CompletedAt_2: sql.NullTime{Time: endOfMonth, Valid: true},
	})
	if err != nil {
		return domain.Internal(err, op, "failed to count report jobs")
	}

	limit := int64(quota.ReportsPerMonth)
	if count >= limit {
		s.logger.Info("Report quota exceeded",
			"user_id", userID,
			"tier", tier,
			"used", count,
			"limit", limit,
		)
		return domain.QuotaExceeded(op, domain.QuotaTypeReport, count, limit)
	}

	return nil
}

// getCurrentMonthBoundaries returns the start and end times for the current month in UTC.
func getCurrentMonthBoundaries() (start, end time.Time) {
	now := time.Now().UTC()
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0)
	return start, end
}
