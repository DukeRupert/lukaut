// Package billing provides Stripe billing integration for subscription management.
package billing

import (
	"fmt"

	"github.com/stripe/stripe-go/v79"
	billingportalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/stripe/stripe-go/v79/webhook"
)

// Service defines the interface for billing operations.
type Service interface {
	// CreateCustomer creates a new Stripe customer for the given email.
	CreateCustomer(email, name string) (string, error)

	// CreateCheckoutSession creates a Stripe Checkout session for subscribing.
	// Returns the checkout URL to redirect the user to.
	CreateCheckoutSession(customerID, priceID, successURL, cancelURL string) (string, error)

	// CreatePortalSession creates a Stripe Customer Portal session.
	// Returns the portal URL to redirect the user to.
	CreatePortalSession(customerID, returnURL string) (string, error)

	// GetSubscription retrieves a Stripe subscription by ID.
	GetSubscription(subscriptionID string) (*stripe.Subscription, error)

	// CancelSubscription sets a subscription to cancel at period end.
	CancelSubscription(subscriptionID string) error

	// ReactivateSubscription removes the cancel_at_period_end flag.
	ReactivateSubscription(subscriptionID string) error

	// VerifyWebhookSignature verifies the Stripe webhook signature and returns the event.
	VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error)

	// TierForPriceID returns the subscription tier for a given Stripe price ID.
	TierForPriceID(priceID string) string
}

// PriceConfig holds the Stripe price IDs for each plan.
type PriceConfig struct {
	StarterMonthlyPriceID      string
	StarterYearlyPriceID       string
	ProfessionalMonthlyPriceID string
	ProfessionalYearlyPriceID  string
}

// stripeService is the concrete implementation of Service.
type stripeService struct {
	webhookSecret string
	prices        PriceConfig
	priceToTier   map[string]string // maps price ID -> tier name
}

// NewStripeService creates a new Stripe billing service.
//
// The secretKey is used to authenticate Stripe API calls.
// The webhookSecret is used to verify incoming webhook signatures.
// The prices configure which Stripe price IDs map to which tiers.
func NewStripeService(secretKey, webhookSecret string, prices PriceConfig) Service {
	stripe.Key = secretKey

	priceToTier := make(map[string]string)
	if prices.StarterMonthlyPriceID != "" {
		priceToTier[prices.StarterMonthlyPriceID] = "starter"
	}
	if prices.StarterYearlyPriceID != "" {
		priceToTier[prices.StarterYearlyPriceID] = "starter"
	}
	if prices.ProfessionalMonthlyPriceID != "" {
		priceToTier[prices.ProfessionalMonthlyPriceID] = "professional"
	}
	if prices.ProfessionalYearlyPriceID != "" {
		priceToTier[prices.ProfessionalYearlyPriceID] = "professional"
	}

	return &stripeService{
		webhookSecret: webhookSecret,
		prices:        prices,
		priceToTier:   priceToTier,
	}
}

func (s *stripeService) CreateCustomer(email, name string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create customer: %w", err)
	}
	return c.ID, nil
}

func (s *stripeService) CreateCheckoutSession(customerID, priceID, successURL, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}
	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create checkout session: %w", err)
	}
	return sess.URL, nil
}

func (s *stripeService) CreatePortalSession(customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
	sess, err := billingportalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create portal session: %w", err)
	}
	return sess.URL, nil
}

func (s *stripeService) GetSubscription(subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get subscription: %w", err)
	}
	return sub, nil
}

func (s *stripeService) CancelSubscription(subscriptionID string) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return fmt.Errorf("stripe cancel subscription: %w", err)
	}
	return nil
}

func (s *stripeService) ReactivateSubscription(subscriptionID string) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	_, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return fmt.Errorf("stripe reactivate subscription: %w", err)
	}
	return nil
}

func (s *stripeService) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("stripe webhook signature verification failed: %w", err)
	}
	return event, nil
}

func (s *stripeService) TierForPriceID(priceID string) string {
	if tier, ok := s.priceToTier[priceID]; ok {
		return tier
	}
	return ""
}
