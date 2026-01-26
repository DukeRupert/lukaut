// Package handler contains HTTP handlers for the Lukaut application.
//
// This file defines stub handlers for the billing/subscription management flow.
// These handlers will be wired up once the Stripe billing service is implemented.
//
// Routes to be handled:
//   - GET  /settings/billing          -> ShowBilling (display current plan, status, actions)
//   - POST /settings/billing/checkout -> CreateCheckout (create Stripe Checkout session, redirect)
//   - POST /settings/billing/portal   -> OpenPortal (create Stripe Customer Portal session, redirect)
//   - POST /settings/billing/cancel   -> CancelSubscription (cancel at period end via htmx)
//   - POST /settings/billing/reactivate -> ReactivateSubscription (undo pending cancellation via htmx)
//   - GET  /settings/billing/success  -> CheckoutSuccess (return URL after Stripe Checkout)
package handler

import (
	"log/slog"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/auth"
	"github.com/DukeRupert/lukaut/internal/templ/pages/settings"
	"github.com/DukeRupert/lukaut/internal/templ/shared"
)

// BillingHandler handles billing and subscription management HTTP requests.
//
// Dependencies (to be injected when billing service is implemented):
//   - UserService: for reading/updating user subscription data
//   - BillingService: for Stripe API calls (checkout sessions, portal sessions, etc.)
//   - Logger: structured logging
type BillingHandler struct {
	logger *slog.Logger
}

// NewBillingHandler creates a new BillingHandler.
//
// TODO: Add BillingService dependency once internal/billing/stripe.go is implemented.
// Signature will become: NewBillingHandler(billingService billing.Service, logger *slog.Logger)
func NewBillingHandler(logger *slog.Logger) *BillingHandler {
	return &BillingHandler{
		logger: logger,
	}
}

// RegisterRoutes registers billing routes on the provided mux.
// All routes require authentication (requireUser middleware).
func (h *BillingHandler) RegisterRoutes(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	mux.Handle("GET /settings/billing", requireUser(http.HandlerFunc(h.ShowBilling)))
	mux.Handle("GET /settings/billing/success", requireUser(http.HandlerFunc(h.CheckoutSuccess)))

	// POST actions (htmx)
	mux.Handle("POST /settings/billing/checkout", requireUser(http.HandlerFunc(h.CreateCheckout)))
	mux.Handle("POST /settings/billing/portal", requireUser(http.HandlerFunc(h.OpenPortal)))
	mux.Handle("POST /settings/billing/cancel", requireUser(http.HandlerFunc(h.CancelSubscription)))
	mux.Handle("POST /settings/billing/reactivate", requireUser(http.HandlerFunc(h.ReactivateSubscription)))
}

// =============================================================================
// GET /settings/billing - Show Billing Page
// =============================================================================
//
// Purpose: Display the user's current subscription status and available actions.
//
// Inputs:
//   - Authenticated user from context (via auth.GetUser)
//
// Outputs:
//   - Renders the billing settings page template with:
//     - Current plan name and tier (Starter/Professional)
//     - Subscription status (active, trialing, canceled, past_due, inactive)
//     - Current period end date (for active subscriptions)
//     - Available plan options with pricing
//     - Action buttons: Upgrade, Manage (portal), Cancel, Reactivate
//
// Template data: settings.BillingPageData (to be created by UI builder)

func (h *BillingHandler) ShowBilling(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// TODO: Fetch subscription details from Stripe via BillingService
	// subscription, err := h.billingService.GetSubscription(user.SubscriptionID)

	data := settings.BillingPageData{
		CurrentPath: "/settings/billing",
		User:        domainUserToDisplay(user),
		ActiveTab:   settings.TabBilling,
		Plan: settings.PlanInfo{
			Tier:   string(user.SubscriptionTier),
			Status: string(user.SubscriptionStatus),
			// TODO: Populate from Stripe subscription object:
			// PeriodEnd:   subscription.CurrentPeriodEnd
			// CancelAtEnd: subscription.CancelAtPeriodEnd
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

// =============================================================================
// POST /settings/billing/checkout - Create Stripe Checkout Session
// =============================================================================
//
// Purpose: Create a Stripe Checkout session for the selected plan and redirect
//          the user to Stripe's hosted checkout page.
//
// Inputs:
//   - Form value "price_id": The Stripe Price ID for the selected plan
//   - Authenticated user from context
//
// Outputs:
//   - 303 redirect to Stripe Checkout URL on success
//   - Error flash and re-render billing page on failure
//
// Stripe API calls needed:
//   - stripe.Customer.Create (if user has no StripeCustomerID)
//   - stripe.CheckoutSession.Create with:
//     - Mode: "subscription"
//     - SuccessURL: baseURL + "/settings/billing/success?session_id={CHECKOUT_SESSION_ID}"
//     - CancelURL:  baseURL + "/settings/billing"
//     - CustomerEmail or Customer ID
//     - LineItems with the selected price_id
//
// Side effects:
//   - May create a Stripe customer and save StripeCustomerID to user record

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	priceID := r.FormValue("price_id")
	if priceID == "" {
		h.logger.Warn("checkout attempted without price_id", "user_id", user.ID)
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	h.logger.Info("checkout stub called",
		"user_id", user.ID,
		"price_id", priceID,
	)

	// TODO: Implement Stripe Checkout session creation
	// 1. Ensure user has a Stripe customer ID (create if needed)
	// 2. Create checkout session via billingService.CreateCheckoutSession(user, priceID)
	// 3. Redirect to session.URL

	http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
}

// =============================================================================
// POST /settings/billing/portal - Open Stripe Customer Portal
// =============================================================================
//
// Purpose: Create a Stripe Customer Portal session for managing payment methods,
//          viewing invoices, and updating billing details.
//
// Inputs:
//   - Authenticated user with a StripeCustomerID
//
// Outputs:
//   - 303 redirect to Stripe Customer Portal URL
//   - Error flash if user has no Stripe customer (shouldn't happen for subscribers)
//
// Stripe API calls needed:
//   - stripe.BillingPortalSession.Create with:
//     - Customer: user.StripeCustomerID
//     - ReturnURL: baseURL + "/settings/billing"

func (h *BillingHandler) OpenPortal(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.StripeCustomerID == "" {
		h.logger.Warn("portal requested but user has no stripe customer", "user_id", user.ID)
		http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
		return
	}

	h.logger.Info("portal stub called", "user_id", user.ID)

	// TODO: Implement Stripe Customer Portal session
	// 1. Create portal session via billingService.CreatePortalSession(user.StripeCustomerID)
	// 2. Redirect to session.URL

	http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
}

// =============================================================================
// POST /settings/billing/cancel - Cancel Subscription
// =============================================================================
//
// Purpose: Cancel the user's subscription at the end of the current billing period.
//          The user retains access until the period ends.
//
// Inputs:
//   - Authenticated user with an active SubscriptionID
//
// Outputs (htmx):
//   - On success: HX-Trigger toast with cancellation confirmation message
//   - On success: HX-Redirect to /settings/billing?canceled=1
//   - On error: HX-Trigger toast with error message
//
// Stripe API calls needed:
//   - stripe.Subscription.Update with CancelAtPeriodEnd: true
//
// Side effects:
//   - Updates user.SubscriptionStatus to "canceled" in local DB

func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.SubscriptionID == "" {
		h.logger.Warn("cancel requested but user has no subscription", "user_id", user.ID)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "No active subscription to cancel.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Info("cancel subscription stub called",
		"user_id", user.ID,
		"subscription_id", user.SubscriptionID,
	)

	// TODO: Implement subscription cancellation
	// 1. Call billingService.CancelSubscription(user.SubscriptionID)
	// 2. Update user subscription status in DB
	// 3. Return HX-Redirect header

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Subscription cancellation is not yet implemented.", "type": "warning"}}`)
	w.WriteHeader(http.StatusOK)
}

// =============================================================================
// POST /settings/billing/reactivate - Reactivate Canceled Subscription
// =============================================================================
//
// Purpose: Undo a pending cancellation, keeping the subscription active beyond
//          the current period end.
//
// Inputs:
//   - Authenticated user with a canceled (but not yet expired) SubscriptionID
//
// Outputs (htmx):
//   - On success: HX-Trigger toast with reactivation message
//   - On success: HX-Redirect to /settings/billing?updated=1
//   - On error: HX-Trigger toast with error message
//
// Stripe API calls needed:
//   - stripe.Subscription.Update with CancelAtPeriodEnd: false
//
// Side effects:
//   - Updates user.SubscriptionStatus to "active" in local DB

func (h *BillingHandler) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.SubscriptionID == "" {
		h.logger.Warn("reactivate requested but user has no subscription", "user_id", user.ID)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "No subscription to reactivate.", "type": "error"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Info("reactivate subscription stub called",
		"user_id", user.ID,
		"subscription_id", user.SubscriptionID,
	)

	// TODO: Implement subscription reactivation
	// 1. Call billingService.ReactivateSubscription(user.SubscriptionID)
	// 2. Update user subscription status in DB
	// 3. Return HX-Redirect header

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Subscription reactivation is not yet implemented.", "type": "warning"}}`)
	w.WriteHeader(http.StatusOK)
}

// =============================================================================
// GET /settings/billing/success - Checkout Success Return URL
// =============================================================================
//
// Purpose: Handle the return from a successful Stripe Checkout session.
//          Optionally verify the session and update user data before redirecting.
//
// Inputs:
//   - Query param "session_id": The Stripe Checkout session ID (from redirect URL template)
//   - Authenticated user from context
//
// Outputs:
//   - 303 redirect to /settings/billing?updated=1
//
// Stripe API calls (optional, webhooks handle this too):
//   - stripe.CheckoutSession.Get to verify payment
//   - Update user subscription fields if not yet done by webhook
//
// Note: The webhook handler (checkout.session.completed) is the primary mechanism
// for updating subscription data. This handler mainly provides a good UX redirect.

func (h *BillingHandler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	h.logger.Info("checkout success return",
		"user_id", user.ID,
		"session_id", sessionID,
	)

	// TODO: Optionally verify checkout session via billingService
	// This is belt-and-suspenders; the webhook is the authoritative update path.

	http.Redirect(w, r, "/settings/billing?updated=1", http.StatusSeeOther)
}

