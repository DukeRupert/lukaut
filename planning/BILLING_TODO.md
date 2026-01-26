# Billing & Subscription — Implementation TODO

This document tracks the remaining work to make billing fully functional. The scaffolding (stub handlers, routes, settings tab, placeholder template) is already in place and compiling.

---

## What exists today

| File | What it does |
|---|---|
| `internal/handler/billing.go` | Stub handlers for 6 billing routes — logs calls, returns placeholders |
| `internal/handler/webhook.go` | Stub Stripe webhook endpoint — accepts POST, returns 200 |
| `internal/templ/pages/settings/billing.templ` | Placeholder billing page with current-plan card, cancel/reactivate htmx buttons, and a dashed placeholder for plan comparison cards |
| `internal/templ/pages/settings/types.go` | `BillingPageData`, `PlanInfo`, `TabBilling` types |
| `internal/templ/pages/settings/components.templ` | Billing tab in the settings tab bar (credit card icon) |
| `internal/config.go` | `StripeSecretKey` and `StripeWebhookSecret` config fields (loaded from env, default empty) |
| `cmd/server/main.go` | `BillingHandler` and `WebhookHandler` instantiated and routes registered |

---

## TODO

### 1. Create `internal/billing/stripe.go` — Stripe service

Create a `billing.Service` interface and a `stripeService` implementation wrapping the Stripe Go SDK.

**Interface methods needed:**

```go
type Service interface {
    // CreateCustomer creates a Stripe customer for a user who doesn't have one yet.
    // Input:  user email, user name
    // Output: Stripe customer ID (e.g. "cus_xxx")
    CreateCustomer(ctx context.Context, email, name string) (string, error)

    // CreateCheckoutSession creates a Stripe Checkout session for subscribing.
    // Input:  Stripe customer ID, Stripe price ID, success URL, cancel URL
    // Output: Checkout session URL to redirect the user to
    CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error)

    // CreatePortalSession creates a Stripe Customer Portal session.
    // Input:  Stripe customer ID, return URL
    // Output: Portal session URL to redirect the user to
    CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)

    // GetSubscription retrieves the current subscription from Stripe.
    // Input:  Stripe subscription ID
    // Output: Subscription details (status, tier, period end, cancel_at_period_end)
    GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionInfo, error)

    // CancelSubscription sets cancel_at_period_end=true on a Stripe subscription.
    // Input:  Stripe subscription ID
    // Output: error
    CancelSubscription(ctx context.Context, subscriptionID string) error

    // ReactivateSubscription sets cancel_at_period_end=false on a Stripe subscription.
    // Input:  Stripe subscription ID
    // Output: error
    ReactivateSubscription(ctx context.Context, subscriptionID string) error

    // VerifyWebhookSignature parses and verifies a Stripe webhook event.
    // Input:  raw request body, Stripe-Signature header value
    // Output: parsed Stripe event
    VerifyWebhookSignature(payload []byte, signature string) (*stripe.Event, error)
}
```

**Stripe SDK:** `github.com/stripe/stripe-go/v79` (or latest)

**Price ID mapping:** The service needs a way to map tier names ("starter", "professional") and intervals ("monthly", "yearly") to Stripe Price IDs. These should come from environment variables:
- `STRIPE_STARTER_MONTHLY_PRICE_ID`
- `STRIPE_STARTER_YEARLY_PRICE_ID`
- `STRIPE_PROFESSIONAL_MONTHLY_PRICE_ID`
- `STRIPE_PROFESSIONAL_YEARLY_PRICE_ID`

---

### 2. Add UserService methods for subscription data updates

The `service.UserService` interface (`internal/service/user.go`) needs three new methods. These are called by the billing and webhook handlers to keep local user records in sync with Stripe.

```go
// UpdateStripeCustomer saves the Stripe customer ID to the user record.
// Called after creating a Stripe customer during first checkout.
// Input:  user ID (uuid.UUID), Stripe customer ID (string)
// Output: error
UpdateStripeCustomer(ctx context.Context, userID uuid.UUID, customerID string) error

// UpdateUserSubscription updates subscription fields on the user record.
// Called by webhook handler when subscription status changes.
// Input:  user ID, status (domain.SubscriptionStatus), tier (domain.SubscriptionTier), subscription ID (string)
// Output: error
UpdateUserSubscription(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, tier domain.SubscriptionTier, subscriptionID string) error

// GetByStripeCustomerID looks up a user by their Stripe customer ID.
// Called by webhook handler to find which user a Stripe event belongs to.
// Input:  Stripe customer ID (string)
// Output: *domain.User, error
GetByStripeCustomerID(ctx context.Context, customerID string) (*domain.User, error)
```

**sqlc queries needed** (add to `sqlc/queries/users.sql`):
- `UpdateStripeCustomerID` — `UPDATE users SET stripe_customer_id = $2 WHERE id = $1`
- `UpdateSubscription` — `UPDATE users SET subscription_status = $2, subscription_tier = $3, subscription_id = $4 WHERE id = $1`
- `GetUserByStripeCustomerID` — `SELECT * FROM users WHERE stripe_customer_id = $1`

After adding queries, run `sqlc generate` to regenerate repository code.

**Mock updates:** Any mock `UserService` implementations in test files (`internal/handler/auth_test.go`, `internal/middleware/auth_test.go`) must add stub methods for the new interface methods to keep compiling.

---

### 3. Wire billing service into handlers

Once tasks 1 and 2 are complete, update the handler constructors to accept real dependencies:

**`internal/handler/billing.go`:**
- Add `billingService billing.Service` and `userService service.UserService` fields to `BillingHandler`
- Update `NewBillingHandler` signature: `NewBillingHandler(billingService billing.Service, userService service.UserService, baseURL string, logger *slog.Logger)`
- Add `baseURL string` field for constructing Stripe return URLs
- Replace each TODO block in the 6 handlers with the actual Stripe calls (see inline comments in each handler for the exact steps)

**`internal/handler/webhook.go`:**
- Add `billingService billing.Service`, `userService service.UserService`, `webhookSecret string` fields
- Update `NewWebhookHandler` signature accordingly
- Implement the event processing switch (see the doc comment on `HandleStripeWebhook` for all 6 event types and their side effects)

**`cmd/server/main.go`:**
- Conditionally create `billing.NewStripeService(cfg.StripeSecretKey)` if key is configured
- Pass billing service into handler constructors
- Pass `cfg.StripeWebhookSecret` and `userService` into webhook handler

---

### 4. Build the full billing page UI

The current `billing.templ` is a functional placeholder. The full page needs:

**Plan comparison cards** (replace the dashed placeholder):
- Two cards side by side: Starter ($29/mo or $290/yr) and Professional ($79/mo or $790/yr)
- Each card shows: plan name, price, feature list, "Choose Plan" button
- Monthly/yearly toggle (Alpine.js)
- The active plan card is highlighted; other card shows "Upgrade" or "Downgrade"
- "Choose Plan" button submits a form: `POST /settings/billing/checkout` with `price_id` field

**Styling requirements:**
- Brand colors: navy (#1E3A5F) for headers, safety-orange (#FF6B35) for CTAs
- Follow existing settings page card patterns (`FormCard`, `PageHeader`)
- Responsive: cards stack vertically on mobile

**htmx wiring already in place:**
- Cancel button: `hx-post="/settings/billing/cancel"` with `hx-confirm`
- Reactivate button: `hx-post="/settings/billing/reactivate"`
- Manage Billing button: standard form POST to `/settings/billing/portal`

---

### 5. Stripe Dashboard setup (production)

Before billing can work in production:

1. Create Stripe products and prices:
   - Product: "Lukaut Starter" with monthly ($29) and yearly ($290) prices
   - Product: "Lukaut Professional" with monthly ($79) and yearly ($790) prices
2. Note the Price IDs and set them as environment variables
3. Configure the Stripe Customer Portal (allowed actions, branding)
4. Create a webhook endpoint pointing to `{BASE_URL}/webhooks/stripe`
5. Select events to send: `checkout.session.completed`, `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`, `invoice.payment_succeeded`, `invoice.payment_failed`
6. Copy the webhook signing secret to `STRIPE_WEBHOOK_SECRET` env var

---

### 6. Enforce subscription middleware

The `RequireActiveSubscription` middleware already exists in `internal/middleware/` but is not applied to any routes. Once billing is working:

- Apply it to routes that should be gated (report generation, AI analysis, etc.)
- Show a banner on the dashboard when subscription is inactive or past_due, linking to `/settings/billing`
- Consider a grace period for past_due status before fully blocking access

---

## Dependency order

```
1. Stripe service (internal/billing/)
2. UserService methods + sqlc queries (can be done in parallel with 1)
3. Wire into handlers (depends on 1 + 2)
4. Full billing page UI (can be done in parallel with 1-3)
5. Stripe Dashboard setup (can be done in parallel with 1-4)
6. Enforce subscription middleware (depends on 3 being deployed)
```

Tasks 1+2 and 4+5 can each be done in parallel. Task 3 is the integration point. Task 6 is a follow-up after the billing flow is live and tested.
