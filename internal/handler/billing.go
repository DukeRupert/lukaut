// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements billing/subscription management handlers backed by Stripe.
//
// Routes handled:
//   - GET  /settings/billing           -> ShowBilling
//   - POST /settings/billing/checkout  -> CreateCheckout
//   - POST /settings/billing/portal    -> OpenPortal
//   - POST /settings/billing/cancel    -> CancelSubscription
//   - POST /settings/billing/reactivate -> ReactivateSubscription
//   - GET  /settings/billing/success   -> CheckoutSuccess
package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/billing"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/DukeRupert/lukaut/internal/templ/pages/settings"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
)

// BillingHandler handles billing and subscription management HTTP requests.
type BillingHandler struct {
	billing     billing.Service
	userService service.UserService
	baseURL     string
	prices      billing.PriceConfig
	logger      *slog.Logger
}

// NewBillingHandler creates a new BillingHandler.
// billingService may be nil when Stripe is not configured (development mode).
func NewBillingHandler(billingService billing.Service, userService service.UserService, baseURL string, prices billing.PriceConfig, logger *slog.Logger) *BillingHandler {
	return &BillingHandler{
		billing:     billingService,
		userService: userService,
		baseURL:     baseURL,
		prices:      prices,
		logger:      logger,
	}
}

// RegisterRoutes registers billing routes on the provided mux.
func (h *BillingHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /settings/billing", requireUser(http.HandlerFunc(h.ShowBilling)))
	mux.Handle("GET /settings/billing/success", requireUser(http.HandlerFunc(h.CheckoutSuccess)))
	mux.Handle("POST /settings/billing/checkout", requireUser(http.HandlerFunc(h.CreateCheckout)))
	mux.Handle("POST /settings/billing/portal", requireUser(http.HandlerFunc(h.OpenPortal)))
	mux.Handle("POST /settings/billing/cancel", requireUser(http.HandlerFunc(h.CancelSubscription)))
	mux.Handle("POST /settings/billing/reactivate", requireUser(http.HandlerFunc(h.ReactivateSubscription)))
}

// ShowBilling renders the billing settings page with current subscription info.
func (h *BillingHandler) ShowBilling(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	plan := settings.PlanInfo{
		Tier:   string(user.SubscriptionTier),
		Status: string(user.SubscriptionStatus),
	}

	// Fetch live subscription details from Stripe if available
	if h.billing != nil && user.SubscriptionID != "" {
		sub, err := h.billing.GetSubscription(user.SubscriptionID)
		if err != nil {
			h.logger.Warn("failed to fetch stripe subscription", "error", err, "subscription_id", user.SubscriptionID)
		} else {
			plan.PeriodEnd = time.Unix(sub.CurrentPeriodEnd, 0).Format("January 2, 2006")
			plan.CancelAtEnd = sub.CancelAtPeriodEnd
			plan.Status = string(sub.Status)
		}
	}

	data := settings.BillingPageData{
		CurrentPath: "/settings/billing",
		User:        domainUserToDisplay(user),
		ActiveTab:   settings.TabBilling,
		Plan:        plan,
		Prices: settings.PriceConfig{
			StarterMonthlyPriceID:      h.prices.StarterMonthlyPriceID,
			StarterYearlyPriceID:       h.prices.StarterYearlyPriceID,
			ProfessionalMonthlyPriceID: h.prices.ProfessionalMonthlyPriceID,
			ProfessionalYearlyPriceID:  h.prices.ProfessionalYearlyPriceID,
		},
	}

	var flash *shared.Flash
	if r.URL.Query().Get("updated") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Subscription updated successfully.",
		}
	}
	if r.URL.Query().Get("canceled") == "1" {
		flash = &shared.Flash{
			Type:    shared.FlashSuccess,
			Message: "Your subscription has been canceled. You'll retain access until the end of your billing period.",
		}
	}
	data.Flash = flash

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settings.BillingPage(data).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render billing page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreateCheckout creates a Stripe Checkout session and redirects to it.
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if h.billing == nil {
		h.logger.Warn("checkout attempted but Stripe is not configured")
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	priceID := r.FormValue("price_id")
	if priceID == "" {
		h.logger.Warn("checkout attempted without price_id", "user_id", user.ID)
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	// Ensure user has a Stripe customer
	customerID := user.StripeCustomerID
	if customerID == "" {
		var err error
		customerID, err = h.billing.CreateCustomer(user.Email, user.Name)
		if err != nil {
			h.logger.Error("failed to create stripe customer", "error", err, "user_id", user.ID)
			http.Error(w, "Failed to initialize billing", http.StatusInternalServerError)
			return
		}
		if err := h.userService.UpdateStripeCustomer(r.Context(), user.ID, customerID); err != nil {
			h.logger.Error("failed to save stripe customer ID", "error", err, "user_id", user.ID)
		}
	}

	successURL := fmt.Sprintf("%s/settings/billing/success?session_id={CHECKOUT_SESSION_ID}", h.baseURL)
	cancelURL := fmt.Sprintf("%s/settings/billing", h.baseURL)

	checkoutURL, err := h.billing.CreateCheckoutSession(customerID, priceID, successURL, cancelURL)
	if err != nil {
		h.logger.Error("failed to create checkout session", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, checkoutURL, http.StatusSeeOther)
}

// OpenPortal creates a Stripe Customer Portal session and redirects to it.
func (h *BillingHandler) OpenPortal(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if h.billing == nil {
		h.logger.Warn("portal requested but Stripe is not configured")
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	if user.StripeCustomerID == "" {
		h.logger.Warn("portal requested but user has no stripe customer", "user_id", user.ID)
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	returnURL := fmt.Sprintf("%s/settings/billing", h.baseURL)
	portalURL, err := h.billing.CreatePortalSession(user.StripeCustomerID, returnURL)
	if err != nil {
		h.logger.Error("failed to create portal session", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to open billing portal", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusSeeOther)
}

// CancelSubscription sets the subscription to cancel at period end.
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if h.billing == nil {
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Billing is not configured.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if user.SubscriptionID == "" {
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "No active subscription to cancel.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.billing.CancelSubscription(user.SubscriptionID); err != nil {
		h.logger.Error("failed to cancel subscription", "error", err, "user_id", user.ID)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to cancel subscription. Please try again.", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/settings/billing?canceled=1")
	w.WriteHeader(http.StatusOK)
}

// ReactivateSubscription removes the cancel-at-period-end flag.
func (h *BillingHandler) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if h.billing == nil {
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Billing is not configured.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if user.SubscriptionID == "" {
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "No subscription to reactivate.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.billing.ReactivateSubscription(user.SubscriptionID); err != nil {
		h.logger.Error("failed to reactivate subscription", "error", err, "user_id", user.ID)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to reactivate subscription. Please try again.", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/settings/billing?updated=1")
	w.WriteHeader(http.StatusOK)
}

// CheckoutSuccess handles the return from Stripe Checkout.
// The webhook is the authoritative update path; this just provides a good UX redirect.
func (h *BillingHandler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	h.logger.Info("checkout success return", "user_id", user.ID, "session_id", sessionID)

	http.Redirect(w, r, "/settings/billing?updated=1", http.StatusSeeOther)
}
