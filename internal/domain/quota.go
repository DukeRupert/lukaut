// Package domain contains core business types and interfaces.
//
// This file defines quota types for rate limiting job enqueueing based on subscription tier.
package domain

// QuotaType identifies the type of quota being checked.
type QuotaType string

const (
	QuotaTypeAnalysis QuotaType = "analysis"
	QuotaTypeReport   QuotaType = "report"
)

// TierQuota defines the monthly limits for a subscription tier.
type TierQuota struct {
	AnalysisPerMonth  int
	ReportsPerMonth   int
	UnlimitedAnalysis bool
	UnlimitedReports  bool
}

// TierQuotas maps subscription tiers to their quota limits.
// Free tier has strict limits; paid tiers are unlimited.
var TierQuotas = map[SubscriptionTier]TierQuota{
	SubscriptionTierFree: {
		AnalysisPerMonth: 3,
		ReportsPerMonth:  2,
	},
	SubscriptionTierStarter: {
		UnlimitedAnalysis: true,
		UnlimitedReports:  true,
	},
	SubscriptionTierProfessional: {
		UnlimitedAnalysis: true,
		UnlimitedReports:  true,
	},
}

// QuotaUsage represents current usage against quota limits.
type QuotaUsage struct {
	AnalysisUsed  int64
	AnalysisLimit int64
	ReportsUsed   int64
	ReportsLimit  int64
	IsUnlimited   bool
}

// GetTierQuota returns the quota for a tier, defaulting to free tier for unknown tiers.
func GetTierQuota(tier SubscriptionTier) TierQuota {
	if quota, ok := TierQuotas[tier]; ok {
		return quota
	}
	return TierQuotas[SubscriptionTierFree]
}
