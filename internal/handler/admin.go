package handler

import (
	"log/slog"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/repository"
	"github.com/DukeRupert/lukaut/internal/templ/pages/admin"
	"github.com/google/uuid"
)

// AdminHandler handles admin panel HTTP requests.
type AdminHandler struct {
	repo   *repository.Queries
	logger *slog.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(repo *repository.Queries, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		repo:   repo,
		logger: logger,
	}
}

// RegisterRoutes registers admin routes with the provided middleware.
func (h *AdminHandler) RegisterRoutes(
	mux *http.ServeMux,
	requireAdmin func(http.Handler) http.Handler,
) {
	mux.Handle("GET /admin", requireAdmin(http.HandlerFunc(h.Dashboard)))
	mux.Handle("GET /admin/users", requireAdmin(http.HandlerFunc(h.UsersList)))
	mux.Handle("GET /admin/users/{id}", requireAdmin(http.HandlerFunc(h.UserDetail)))
}

// Dashboard renders the admin dashboard with platform stats.
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch platform stats
	stats, err := h.repo.AdminGetPlatformStats(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch platform stats", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Fetch users with AI usage
	users, err := h.repo.AdminListUsers(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch users", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Convert to template data
	userRows := make([]admin.UserRow, 0, len(users))
	for _, u := range users {
		userRows = append(userRows, admin.UserRow{
			ID:                 u.ID,
			Email:              u.Email,
			Name:               u.Name,
			SubscriptionStatus: domain.NullStringValue(u.SubscriptionStatus),
			EmailVerified:      u.EmailVerified.Valid && u.EmailVerified.Bool,
			CreatedAt:          u.CreatedAt.Time,
			TotalCostCents:     u.TotalCostCents,
			InspectionCount:    u.InspectionCount,
		})
	}

	// Convert interface{} to int64 for cost fields
	totalAICost := toInt64(stats.TotalAiCostCents)
	aiCostThisMonth := toInt64(stats.AiCostThisMonthCents)

	data := admin.DashboardData{
		AdminEmail:           user.Email,
		TotalUsers:           stats.TotalUsers,
		NewUsersThisMonth:    stats.NewUsersThisMonth,
		TotalInspections:     stats.TotalInspections,
		InspectionsThisMonth: stats.InspectionsThisMonth,
		TotalReports:         stats.TotalReports,
		ReportsThisMonth:     stats.ReportsThisMonth,
		TotalAICostCents:     totalAICost,
		AICostThisMonthCents: aiCostThisMonth,
		Users:                userRows,
	}

	if err := admin.DashboardPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render admin dashboard", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// UsersList renders the users list page.
func (h *AdminHandler) UsersList(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.AdminListUsers(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch users", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userRows := make([]admin.UserRow, 0, len(users))
	for _, u := range users {
		userRows = append(userRows, admin.UserRow{
			ID:                 u.ID,
			Email:              u.Email,
			Name:               u.Name,
			SubscriptionStatus: domain.NullStringValue(u.SubscriptionStatus),
			EmailVerified:      u.EmailVerified.Valid && u.EmailVerified.Bool,
			CreatedAt:          u.CreatedAt.Time,
			TotalCostCents:     u.TotalCostCents,
			InspectionCount:    u.InspectionCount,
		})
	}

	if err := admin.UsersPage(userRows).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render users page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// UserDetail renders the user detail page.
func (h *AdminHandler) UserDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.repo.AdminGetUserByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to fetch user", "error", err, "user_id", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Fetch user's inspections
	inspections, err := h.repo.AdminGetUserInspections(r.Context(), repository.AdminGetUserInspectionsParams{
		UserID: id,
		Limit:  20,
	})
	if err != nil {
		h.logger.Warn("failed to fetch user inspections", "error", err, "user_id", id)
	}

	// Fetch user's AI usage history
	aiUsage, err := h.repo.AdminGetUserAIUsageHistory(r.Context(), repository.AdminGetUserAIUsageHistoryParams{
		UserID: id,
		Limit:  20,
	})
	if err != nil {
		h.logger.Warn("failed to fetch AI usage", "error", err, "user_id", id)
	}

	inspectionRows := make([]admin.InspectionRow, 0, len(inspections))
	for _, i := range inspections {
		inspectionRows = append(inspectionRows, admin.InspectionRow{
			ID:             i.ID,
			Title:          i.Title,
			Status:         i.Status,
			InspectionDate: i.InspectionDate,
			CreatedAt:      i.CreatedAt.Time,
		})
	}

	aiUsageRows := make([]admin.AIUsageRow, 0, len(aiUsage))
	for _, a := range aiUsage {
		aiUsageRows = append(aiUsageRows, admin.AIUsageRow{
			Model:        a.Model,
			InputTokens:  a.InputTokens,
			OutputTokens: a.OutputTokens,
			CostCents:    a.CostCents,
			RequestType:  a.RequestType,
			CreatedAt:    a.CreatedAt.Time,
		})
	}

	data := admin.UserDetailData{
		ID:                 user.ID,
		Email:              user.Email,
		Name:               user.Name,
		SubscriptionStatus: domain.NullStringValue(user.SubscriptionStatus),
		SubscriptionTier:   domain.NullStringValue(user.SubscriptionTier),
		EmailVerified:      user.EmailVerified.Valid && user.EmailVerified.Bool,
		CreatedAt:          user.CreatedAt.Time,
		TotalCostCents:     user.TotalCostCents,
		TotalInputTokens:   user.TotalInputTokens,
		TotalOutputTokens:  user.TotalOutputTokens,
		InspectionCount:    user.InspectionCount,
		ReportCount:        user.ReportCount,
		Inspections:        inspectionRows,
		AIUsageHistory:     aiUsageRows,
	}

	if err := admin.UserDetailPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render user detail page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// toInt64 safely converts an interface{} to int64.
// This handles the case where sqlc returns interface{} for COALESCE(SUM(...)).
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
