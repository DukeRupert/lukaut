// Package handler contains HTTP handlers for the Lukaut application.
//
// This file implements the Stripe webhook handler for processing billing events.
//
// Route:
//   - POST /webhooks/stripe -> HandleStripeWebhook
//
// This route is PUBLIC (no auth middleware) because Stripe calls it directly.
// Authentication is via the Stripe webhook signature verification.
package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/DukeRupert/lukaut/internal/billing"
	"github.com/DukeRupert/lukaut/internal/domain"
	"github.com/DukeRupert/lukaut/internal/service"
	"github.com/stripe/stripe-go/v79"
)

// WebhookHandler handles incoming webhook events from Stripe.
type WebhookHandler struct {
	billing     billing.Service
	userService service.UserService
	logger      *slog.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
// billingService may be nil when Stripe is not configured.
func NewWebhookHandler(billingService billing.Service, userService service.UserService, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		billing:     billingService,
		userService: userService,
		logger:      logger,
	}
}

// RegisterRoutes registers webhook routes on the provided mux.
// These routes are PUBLIC — no auth middleware.
func (h *WebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/stripe", h.HandleStripeWebhook)
}

// HandleStripeWebhook processes incoming Stripe webhook events.
func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		h.logger.Warn("stripe webhook received but billing is not configured")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read body (limit to 64KB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify signature
	signature := r.Header.Get("Stripe-Signature")
	event, err := h.billing.VerifyWebhookSignature(body, signature)
	if err != nil {
		h.logger.Warn("webhook signature verification failed", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.logger.Info("stripe webhook received", "type", event.Type, "id", event.ID)

	// Route to event-specific handler
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(event)
	case "customer.subscription.created":
		h.handleSubscriptionCreated(event)
	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(event)
	case "invoice.payment_succeeded":
		h.handlePaymentSucceeded(event)
	case "invoice.payment_failed":
		h.handlePaymentFailed(event)
	default:
		h.logger.Debug("unhandled webhook event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleCheckoutCompleted(event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		h.logger.Error("failed to parse checkout session", "error", err)
		return
	}

	if session.Customer == nil || session.Subscription == nil {
		h.logger.Warn("checkout session missing customer or subscription", "session_id", session.ID)
		return
	}

	customerID := session.Customer.ID
	subscriptionID := session.Subscription.ID

	// Look up user by Stripe customer ID, or by email if customer is new
	user, err := h.userService.GetByStripeCustomerID(r_ctx(), customerID)
	if err != nil {
		h.logger.Info("user not found by customer ID, checkout may update on subscription event",
			"customer_id", customerID, "subscription_id", subscriptionID)
		return
	}

	// Update subscription info
	if err := h.userService.UpdateSubscription(r_ctx(), user.ID, string(domain.SubscriptionStatusActive), "", subscriptionID); err != nil {
		h.logger.Error("failed to update subscription on checkout", "error", err, "user_id", user.ID)
	}
}

func (h *WebhookHandler) handleSubscriptionCreated(event stripe.Event) {
	h.processSubscriptionEvent(event, "created")
}

func (h *WebhookHandler) handleSubscriptionUpdated(event stripe.Event) {
	h.processSubscriptionEvent(event, "updated")
}

func (h *WebhookHandler) processSubscriptionEvent(event stripe.Event, action string) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("failed to parse subscription event", "error", err, "action", action)
		return
	}

	if sub.Customer == nil {
		h.logger.Warn("subscription event missing customer", "subscription_id", sub.ID, "action", action)
		return
	}

	user, err := h.userService.GetByStripeCustomerID(r_ctx(), sub.Customer.ID)
	if err != nil {
		h.logger.Warn("user not found for subscription event",
			"customer_id", sub.Customer.ID, "subscription_id", sub.ID, "action", action)
		return
	}

	// Determine tier from price
	tier := ""
	if len(sub.Items.Data) > 0 && sub.Items.Data[0].Price != nil {
		tier = h.billing.TierForPriceID(sub.Items.Data[0].Price.ID)
	}

	status := string(sub.Status)
	if err := h.userService.UpdateSubscription(r_ctx(), user.ID, status, tier, sub.ID); err != nil {
		h.logger.Error("failed to update subscription", "error", err, "user_id", user.ID, "action", action)
	}

	h.logger.Info("subscription event processed",
		"user_id", user.ID, "action", action, "status", status, "tier", tier)
}

func (h *WebhookHandler) handleSubscriptionDeleted(event stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("failed to parse subscription deleted event", "error", err)
		return
	}

	if sub.Customer == nil {
		h.logger.Warn("subscription deleted event missing customer", "subscription_id", sub.ID)
		return
	}

	user, err := h.userService.GetByStripeCustomerID(r_ctx(), sub.Customer.ID)
	if err != nil {
		h.logger.Warn("user not found for subscription deletion", "customer_id", sub.Customer.ID)
		return
	}

	if err := h.userService.UpdateSubscription(r_ctx(), user.ID, string(domain.SubscriptionStatusInactive), "", ""); err != nil {
		h.logger.Error("failed to deactivate subscription", "error", err, "user_id", user.ID)
	}

	h.logger.Info("subscription deleted", "user_id", user.ID, "subscription_id", sub.ID)
}

func (h *WebhookHandler) handlePaymentSucceeded(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		h.logger.Error("failed to parse invoice payment succeeded event", "error", err)
		return
	}

	if invoice.Customer == nil {
		return
	}

	user, err := h.userService.GetByStripeCustomerID(r_ctx(), invoice.Customer.ID)
	if err != nil {
		h.logger.Debug("user not found for payment succeeded", "customer_id", invoice.Customer.ID)
		return
	}

	// Ensure status is active (recovery from past_due)
	if user.SubscriptionStatus != domain.SubscriptionStatusActive {
		if err := h.userService.UpdateSubscription(r_ctx(), user.ID,
			string(domain.SubscriptionStatusActive), string(user.SubscriptionTier), user.SubscriptionID); err != nil {
			h.logger.Error("failed to reactivate on payment success", "error", err, "user_id", user.ID)
		}
	}
}

func (h *WebhookHandler) handlePaymentFailed(event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		h.logger.Error("failed to parse invoice payment failed event", "error", err)
		return
	}

	if invoice.Customer == nil {
		return
	}

	user, err := h.userService.GetByStripeCustomerID(r_ctx(), invoice.Customer.ID)
	if err != nil {
		h.logger.Debug("user not found for payment failed", "customer_id", invoice.Customer.ID)
		return
	}

	if err := h.userService.UpdateSubscription(r_ctx(), user.ID,
		string(domain.SubscriptionStatusPastDue), string(user.SubscriptionTier), user.SubscriptionID); err != nil {
		h.logger.Error("failed to set past_due on payment failure", "error", err, "user_id", user.ID)
	}

	h.logger.Warn("payment failed", "user_id", user.ID, "customer_id", invoice.Customer.ID)
}

// r_ctx returns a background context for webhook processing.
// Webhooks are async events and don't have a request context from a user session.
func r_ctx() context.Context {
	return context.Background()
}
