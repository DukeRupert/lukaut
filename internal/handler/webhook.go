// Package handler contains HTTP handlers for the Lukaut application.
//
// This file defines the stub webhook handler for Stripe events.
// Stripe sends events to this endpoint to notify us of subscription changes,
// payment outcomes, and other billing events.
//
// Route:
//   - POST /webhooks/stripe -> HandleStripeWebhook
//
// This route is PUBLIC (no auth middleware) because Stripe calls it directly.
// Authentication is via the Stripe webhook signature verification.
package handler

import (
	"log/slog"
	"net/http"
)

// WebhookHandler handles incoming webhook events from external services.
//
// Dependencies (to be injected when billing service is implemented):
//   - BillingService: for verifying webhook signatures and processing events
//   - UserService: for updating user subscription data in response to events
//   - WebhookSecret: the Stripe webhook signing secret for signature verification
type WebhookHandler struct {
	logger *slog.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
//
// TODO: Add dependencies once billing service is implemented.
// Signature will become:
//
//	NewWebhookHandler(billingService billing.Service, userService service.UserService, webhookSecret string, logger *slog.Logger)
func NewWebhookHandler(logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		logger: logger,
	}
}

// RegisterRoutes registers webhook routes on the provided mux.
// These routes are PUBLIC — no auth middleware. Stripe must be able to reach them.
func (h *WebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/stripe", h.HandleStripeWebhook)
}

// =============================================================================
// POST /webhooks/stripe - Handle Stripe Webhook Events
// =============================================================================
//
// Purpose: Receive and process Stripe webhook events to keep local subscription
//          data in sync with Stripe's authoritative state.
//
// Inputs:
//   - Request body: raw JSON payload from Stripe
//   - Header "Stripe-Signature": HMAC signature for verification
//
// Processing steps:
//  1. Read request body (limit to 64KB for safety)
//  2. Verify webhook signature using the signing secret
//  3. Parse the event JSON
//  4. Route to event-specific handler based on event.Type
//
// Events to handle:
//
//   - checkout.session.completed
//     Purpose: User completed Stripe Checkout — activate their subscription.
//     Data: session.customer, session.subscription, session.customer_email
//     Side effects: Save StripeCustomerID, SubscriptionID, set status=active
//
//   - customer.subscription.created
//     Purpose: New subscription created (may overlap with checkout.session.completed).
//     Data: subscription.id, subscription.status, subscription.items[].price.lookup_key
//     Side effects: Update SubscriptionStatus, SubscriptionTier, SubscriptionID
//
//   - customer.subscription.updated
//     Purpose: Subscription changed (plan change, renewal, cancellation scheduled).
//     Data: subscription.id, subscription.status, subscription.cancel_at_period_end
//     Side effects: Update SubscriptionStatus, SubscriptionTier
//
//   - customer.subscription.deleted
//     Purpose: Subscription fully expired/terminated.
//     Data: subscription.id, subscription.customer
//     Side effects: Set SubscriptionStatus=inactive, clear SubscriptionTier
//
//   - invoice.payment_succeeded
//     Purpose: Payment went through — subscription continues.
//     Data: invoice.customer, invoice.subscription, invoice.amount_paid
//     Side effects: Ensure SubscriptionStatus=active (recovery from past_due)
//
//   - invoice.payment_failed
//     Purpose: Payment failed — subscription may be at risk.
//     Data: invoice.customer, invoice.subscription, invoice.attempt_count
//     Side effects: Set SubscriptionStatus=past_due, optionally send email alert
//
// Outputs:
//   - 200 OK on successful processing (Stripe will retry on non-2xx)
//   - 400 Bad Request if signature verification fails
//   - 200 OK for unhandled event types (acknowledge receipt, do nothing)

func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("stripe webhook received (stub)",
		"content_length", r.ContentLength,
	)

	// TODO: Implement webhook processing
	//
	// 1. Read body:
	//    body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	//
	// 2. Verify signature:
	//    event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	//
	// 3. Switch on event.Type and dispatch to handlers
	//
	// 4. Return 200 OK

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "stub - not yet implemented"}`))
}
