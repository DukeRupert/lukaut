# Lukaut MVP Roadmap

## Document Purpose

This roadmap provides a comprehensive, prioritized development plan for reaching Lukaut's Minimum Viable Product. It is designed for a solo developer and emphasizes incremental delivery with clear dependencies.

**Last Updated:** December 15, 2025

---

## Current State Assessment

### What Exists

The foundation layer is complete:

| Component | Status | Location |
|-----------|--------|----------|
| Configuration | Done | `/workspaces/lukaut/internal/config.go` |
| Logger | Done | `/workspaces/lukaut/internal/logger.go` |
| Database Connection | Done | `/workspaces/lukaut/cmd/server/main.go` |
| Migrations Framework | Done | `/workspaces/lukaut/internal/migrate.go` |
| Initial Schema | Done | `/workspaces/lukaut/internal/migrations/00001_initial_schema.sql` |
| Repository Layer (sqlc) | Done | `/workspaces/lukaut/internal/repository/*.go` |
| Server Bootstrap | Done | Basic HTTP server with health check |

### What is Missing for MVP

- Authentication system (registration, login, sessions)
- Frontend templating and static assets
- Image upload and storage
- AI integration (Anthropic Claude)
- Background job worker
- Violation review interface
- Report generation (PDF/DOCX)
- Billing integration (Stripe)
- Email service (Postmark)
- OSHA regulations database seed

---

## Planning Document Analysis

### Consistency Check

The three planning documents (BUSINESS.md, TECHNICAL.md, BRAND.md) are well-aligned:

| Aspect | Finding |
|--------|---------|
| Pricing | Consistent ($99 Starter, $149 Professional) |
| Target Market | Consistent (Independent FL inspectors) |
| Tech Stack | Fully specified and consistent |
| Brand Identity | Comprehensive and actionable |
| Color Palette | Ready for Tailwind implementation |

### Identified Gaps

1. **Email Verification Flow** - TECHNICAL.md mentions it but lacks token table schema
2. **Password Reset Flow** - Mentioned but no token management specified
3. **Rate Limiting** - Mentioned in auth section but no implementation details
4. **OSHA Data Source** - No concrete plan for initial data seeding
5. **HEIC Conversion** - Mentioned but no library specified
6. **Image Thumbnails** - Mentioned but no library specified
7. **Report Templates** - Layout shown but no template files
8. **Deployment Details** - Basic docker-compose exists but needs production config

### Dependency Analysis

```
                         +------------------+
                         |   Foundation     |
                         | (COMPLETE)       |
                         +--------+---------+
                                  |
              +-------------------+-------------------+
              |                                       |
     +--------v---------+                    +--------v---------+
     |  Authentication  |                    |    Templates     |
     | (Blocks all UI)  |                    | (Blocks all UI)  |
     +--------+---------+                    +--------+---------+
              |                                       |
              +-------------------+-------------------+
                                  |
                         +--------v---------+
                         | Inspection CRUD  |
                         | (Core feature)   |
                         +--------+---------+
                                  |
              +-------------------+-------------------+
              |                                       |
     +--------v---------+                    +--------v---------+
     |  Image Upload    |                    |    Storage       |
     |                  +<-------------------+  (R2/Local)      |
     +--------+---------+                    +------------------+
              |
     +--------v---------+
     |    AI Analysis   +<---+Background Jobs
     |  (Claude API)    |
     +--------+---------+
              |
     +--------v---------+
     | Violation Review |
     +--------+---------+
              |
     +--------v---------+
     | Report Generation|
     +--------+---------+
              |
     +--------v---------+
     |     Billing      |
     | (Launch blocker) |
     +------------------+
```

---

## MVP Definition

### Core Value Proposition
An inspector can:
1. Create an inspection
2. Upload site photos
3. Receive AI-identified potential violations
4. Review and accept/reject findings
5. Generate a professional PDF report
6. Pay for the service

### Out of Scope for MVP
- DOCX generation (PDF sufficient for validation)
- Sites management (can enter site info per inspection)
- Email reports to clients (download only)
- Team features
- Advanced dashboard analytics
- Mobile-optimized upload (responsive web is sufficient)

---

## Prioritized Todo List

### Priority Definitions

| Priority | Meaning | Timeline |
|----------|---------|----------|
| **P0** | Blocks other work or core functionality | Week 1-2 |
| **P1** | Required for MVP launch | Week 2-6 |
| **P2** | Important but can ship without | Week 6-8 |
| **P3** | Nice-to-have, post-MVP | Post-launch |

---

## Phase 1: Foundation & Authentication (Week 1-2)

### P0-001: Template Engine Setup
**Blocks:** All UI work

Create the HTML template infrastructure with base layouts, partials, and Tailwind CSS integration.

**Tasks:**
- [x] Create `/workspaces/lukaut/web/templates/layouts/base.html` with HTML5 boilerplate
- [x] Create `/workspaces/lukaut/web/templates/layouts/app.html` (authenticated layout)
- [x] Create `/workspaces/lukaut/web/templates/layouts/auth.html` (login/register layout)
- [x] Set up Tailwind CSS with brand colors from BRAND.md
- [x] Create `/workspaces/lukaut/tailwind.config.js` with forest, gold, clay, cream colors
- [x] Create `/workspaces/lukaut/web/static/css/input.css` with Tailwind directives
- [x] Add template parsing to server startup
- [x] Create template helper functions (csrf, formatDate, etc.)

**Files to create:**
```
web/
  templates/
    layouts/
      base.html
      app.html
      auth.html
    partials/
      nav.html
      footer.html
      flash.html
    pages/
      home.html
  static/
    css/
      input.css
    js/
      app.js
```

**Acceptance Criteria:**
- [x] Server renders a styled home page at `/`
- [x] Tailwind build produces output.css
- [x] Brand colors are available as Tailwind classes

**Status: COMPLETE**

---

### P0-002: Authentication Service Layer
**Blocks:** All authenticated features

Implement the user service with password hashing, session management, and middleware.

**Tasks:**
- [x] Create `/workspaces/lukaut/internal/service/user.go` with UserService interface
- [x] Implement password hashing with bcrypt
- [x] Implement session token generation (32 bytes, crypto/rand)
- [x] Implement session token hashing (SHA-256)
- [x] Create `/workspaces/lukaut/internal/middleware/auth.go`
- [x] Implement `WithUser` middleware (loads user from session cookie)
- [x] Implement `RequireUser` middleware (blocks unauthenticated requests)
- [x] Add cookie configuration (HttpOnly, Secure, SameSite=Lax)

**Interface Definition:**
```go
type UserService interface {
    Register(ctx context.Context, params RegisterParams) (*User, error)
    Login(ctx context.Context, email, password string) (*Session, error)
    Logout(ctx context.Context, token string) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    GetBySessionToken(ctx context.Context, token string) (*User, error)
}
```

**Acceptance Criteria:**
- [x] Passwords are hashed with bcrypt cost 12
- [x] Session tokens are cryptographically secure
- [x] Cookies are properly configured for security

**Status: COMPLETE**

---

### P0-003: Authentication Handlers & Pages
**Blocks:** User testing

Create the HTTP handlers and templates for registration, login, and logout.

**Tasks:**
- [x] Create `/workspaces/lukaut/internal/handler/auth.go`
- [x] Implement GET/POST `/register` handler
- [x] Implement GET/POST `/login` handler
- [x] Implement POST `/logout` handler
- [x] Create `/workspaces/lukaut/web/templates/pages/auth/register.html`
- [x] Create `/workspaces/lukaut/web/templates/pages/auth/login.html`
- [x] Add form validation with error display
- [ ] Add CSRF protection (documented, SameSite=Lax provides baseline)
- [x] Add flash messages for success/error states
- [x] Wire up auth routes in main.go
- [x] Add WithUser/RequireUser middleware to protected routes

**Routes:**
```
GET  /register    -> Show registration form
POST /register    -> Process registration
GET  /login       -> Show login form
POST /login       -> Process login
POST /logout      -> Clear session, redirect to login
```

**Acceptance Criteria:**
- [x] User can register with email/password/name
- [x] User can log in with email/password
- [x] User can log out
- [x] Invalid credentials show appropriate error
- [x] Session persists across page refreshes

**Status: COMPLETE** (CSRF token generation deferred, SameSite=Lax cookie provides baseline protection)

---

### P0-004: Email Verification Tokens
**Dependency:** P0-002

Add database table and logic for email verification tokens.

**Tasks:**
- [x] Create migration `/workspaces/lukaut/internal/migrations/00002_email_tokens.sql`
- [x] Add `email_verification_tokens` table
- [x] Add `password_reset_tokens` table
- [x] Create sqlc queries for token CRUD
- [x] Implement token generation and validation in UserService
- [x] Add handlers for /verify-email and /resend-verification
- [x] Create verification templates (verify_email.html, resend_verification.html)
- [ ] Add verification status check in login flow (deferred to P0-005)

**Schema Addition:**
```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Acceptance Criteria:**
- [x] Tokens are stored as SHA-256 hashes
- [x] Tokens expire after configurable duration (24h verification, 1h reset)
- [x] Only one active token per user per type

**Status: COMPLETE** (Email sending deferred to P0-005)

---

### P0-005: Email Service Integration
**Dependency:** P0-004

Integrate email service for transactional emails (SMTP for Mailhog in dev, Postmark SMTP in prod).

**Tasks:**
- [x] Create `/home/dukerupert/Repos/lukaut/internal/email/email.go` with EmailService interface
- [x] Create `/home/dukerupert/Repos/lukaut/internal/email/smtp.go` SMTP implementation
- [x] Add SMTP configuration to config.go (defaults to Mailhog for dev)
- [x] Add email templates for verification, password reset, and report ready
- [x] Create email template files in `/home/dukerupert/Repos/lukaut/web/templates/email/`
- [x] Wire up email service in main.go
- [x] Update auth handler to send verification emails on registration
- [x] Update auth handler to send verification emails on resend request

**Interface Definition:**
```go
type EmailService interface {
    SendVerificationEmail(ctx context.Context, to, name, token string) error
    SendPasswordResetEmail(ctx context.Context, to, name, token string) error
    SendReportReadyEmail(ctx context.Context, to, name, reportURL string) error
}
```

**Acceptance Criteria:**
- [x] Emails send successfully via SMTP (Mailhog in dev)
- [x] Development mode uses Mailhog (localhost:1025)
- [x] Email templates use brand styling (forest, gold, cream, clay colors)
- [x] Verification emails sent on registration
- [x] Verification emails sent on resend request

**Status: COMPLETE**

---

### P0-006: Password Reset Flow
**Dependency:** P0-004, P0-005

Implement forgot password and reset password functionality.

**Tasks:**
- [x] Implement GET/POST `/forgot-password` handler
- [x] Implement GET/POST `/reset-password` handler
- [x] Create password reset templates (forgot_password, forgot_password_sent, reset_password, reset_password_invalid)
- [x] Send reset email with secure token
- [x] Validate token and allow password change
- [x] Invalidate all sessions on password change (handled by UserService.ResetPassword)

**Routes:**
```
GET  /forgot-password           -> Show forgot password form
POST /forgot-password           -> Send reset email
GET  /reset-password?token=xxx  -> Show reset form
POST /reset-password            -> Process password change
```

**Acceptance Criteria:**
- [x] User can request password reset via email
- [x] Reset link works for 1 hour (token expiry handled by P0-004)
- [x] Password change invalidates existing sessions

**Status: COMPLETE**

---

## Phase 2: Core Inspection Flow (Week 3-4)

### P1-001: Storage Service
**Blocks:** Image upload

Implement file storage abstraction with local and R2 backends.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/storage/storage.go` interface
- [ ] Create `/workspaces/lukaut/internal/storage/local.go` for development
- [ ] Create `/workspaces/lukaut/internal/storage/r2.go` for production
- [ ] Add R2 configuration to config.go
- [ ] Implement Put, Get, Delete, URL methods
- [ ] Add content-type detection
- [ ] Add file size validation

**Interface Definition:**
```go
type Storage interface {
    Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    URL(ctx context.Context, key string, expires time.Duration) (string, error)
    Exists(ctx context.Context, key string) (bool, error)
}

type PutOptions struct {
    ContentType string
    Metadata    map[string]string
}
```

**Acceptance Criteria:**
- Local storage works for development
- R2 storage works for production
- Files can be uploaded and retrieved
- Presigned URLs work for downloads

---

### P1-002: Dashboard Page
**Dependency:** P0-003

Create the main dashboard that users see after login.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/dashboard.go`
- [x] Create `/workspaces/lukaut/web/templates/pages/dashboard.html`
- [ ] Show recent inspections (last 10)
- [ ] Show quick stats (total inspections, reports generated)
- [x] Add "New Inspection" CTA button
- [x] Implement empty state for new users

**Route:**
```
GET /dashboard  -> Show user dashboard (requires auth)
```

**Acceptance Criteria:**
- Dashboard shows after login
- Recent inspections are listed
- New users see helpful empty state

---

### P1-003: Inspection CRUD Handlers
**Dependency:** P1-002

Implement create, read, update, delete for inspections.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/inspection.go`
- [ ] Create `/workspaces/lukaut/internal/service/inspection.go`
- [ ] Implement inspection list page with pagination
- [ ] Implement inspection create form
- [ ] Implement inspection detail view
- [ ] Implement inspection edit form
- [ ] Implement inspection delete with confirmation
- [ ] Add status transitions (draft -> analyzing -> review -> completed)

**Routes:**
```
GET  /inspections              -> List user's inspections
GET  /inspections/new          -> Show create form
POST /inspections              -> Create inspection
GET  /inspections/{id}         -> Show inspection detail
GET  /inspections/{id}/edit    -> Show edit form
PUT  /inspections/{id}         -> Update inspection
DELETE /inspections/{id}       -> Delete inspection
```

**Templates:**
```
pages/inspections/
  index.html      (list view)
  new.html        (create form)
  show.html       (detail view)
  edit.html       (edit form)
```

**Acceptance Criteria:**
- User can create inspection with title, date, site info
- User can view list of their inspections
- User can edit inspection details
- User can delete inspection (with confirmation)
- Pagination works for users with many inspections

---

### P1-004: Image Upload Handler
**Dependency:** P1-001, P1-003

Implement image upload for inspections with htmx.

**Tasks:**
- [ ] Add image upload section to inspection detail page
- [ ] Create `/workspaces/lukaut/internal/handler/image.go`
- [ ] Create `/workspaces/lukaut/internal/service/image.go`
- [ ] Implement multi-file upload (drag-and-drop)
- [ ] Validate file types (JPEG, PNG, HEIC)
- [ ] Validate file size (max 20MB)
- [ ] Generate thumbnails (200x200)
- [ ] Store originals and thumbnails in storage
- [ ] Update image list via htmx after upload
- [ ] Show upload progress

**Routes:**
```
POST   /inspections/{id}/images     -> Upload images
DELETE /inspections/{id}/images/{imageId}  -> Delete image
GET    /images/{id}/thumbnail       -> Serve thumbnail
GET    /images/{id}/original        -> Serve original (presigned URL)
```

**Library Recommendation:**
```go
// For image processing
"github.com/disintegration/imaging"

// For HEIC conversion (if needed)
"github.com/nicksimmons53/heic"
```

**Acceptance Criteria:**
- User can upload multiple images at once
- Drag-and-drop works
- Invalid files are rejected with clear error
- Thumbnails display in inspection view
- Original images can be viewed full-size

---

### P1-005: Background Job Worker
**Blocks:** AI analysis

Implement the database-backed job queue worker.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/worker/worker.go`
- [ ] Implement job dequeue with SKIP LOCKED
- [ ] Implement job execution with timeout
- [ ] Implement retry logic with exponential backoff
- [ ] Add graceful shutdown handling
- [ ] Create job type registry pattern
- [ ] Add worker configuration (concurrency, poll interval)
- [ ] Start worker goroutines in main.go

**Worker Interface:**
```go
type JobHandler interface {
    Type() string
    Handle(ctx context.Context, payload json.RawMessage) error
}

type Worker struct {
    queries     *repository.Queries
    handlers    map[string]JobHandler
    concurrency int
    pollInterval time.Duration
}
```

**Acceptance Criteria:**
- Worker processes jobs from database
- Failed jobs retry with backoff
- Dead jobs are marked as failed after max attempts
- Worker shuts down gracefully

---

## Phase 3: AI Integration (Week 5-6)

### P1-006: Anthropic AI Provider
**Dependency:** P1-005

Integrate Claude API for image analysis.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/ai/ai.go` interface
- [ ] Create `/workspaces/lukaut/internal/ai/anthropic/provider.go`
- [ ] Implement image analysis prompt from TECHNICAL.md
- [ ] Implement regulation matching prompt
- [ ] Add API key configuration
- [ ] Add retry logic for API failures
- [ ] Add usage tracking (tokens, cost)
- [ ] Create mock provider for testing

**Interface Definition:**
```go
type AIProvider interface {
    AnalyzeImage(ctx context.Context, params AnalyzeParams) (*AnalysisResult, error)
    MatchRegulations(ctx context.Context, violation string) ([]RegulationMatch, error)
}

type AnalyzeParams struct {
    ImageData   []byte
    ContentType string
    Context     string
}

type AnalysisResult struct {
    Violations          []PotentialViolation
    GeneralObservations string
    ImageQualityNotes   string
}

type PotentialViolation struct {
    Description  string
    Location     string
    BoundingBox  *BoundingBox
    Confidence   string // high, medium, low
    Category     string
    Severity     string
}
```

**Acceptance Criteria:**
- Claude analyzes construction site images
- Violations are identified with confidence levels
- Usage is tracked in database
- Errors are handled gracefully

---

### P1-007: Analysis Background Job
**Dependency:** P1-005, P1-006

Create the job that analyzes inspection images.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/jobs/analyze_inspection.go`
- [ ] Fetch inspection images from storage
- [ ] Send each image to Claude for analysis
- [ ] Parse and store potential violations
- [ ] Match regulations for each violation
- [ ] Update inspection status (draft -> analyzing -> review)
- [ ] Track AI usage per request
- [ ] Handle partial failures gracefully

**Job Flow:**
```
1. Get inspection and images
2. For each image:
   a. Download from storage
   b. Analyze with Claude
   c. For each violation found:
      i.  Store violation record
      ii. Match regulations
      iii. Store regulation associations
   d. Update image analysis_status
3. Update inspection status to "review"
```

**Acceptance Criteria:**
- All images in inspection are analyzed
- Violations are created with AI descriptions
- Regulations are matched and linked
- Inspection moves to "review" status
- Partial failures don't lose successful analysis

---

### P1-008: Trigger Analysis UI
**Dependency:** P1-007

Add UI to trigger AI analysis on an inspection.

**Tasks:**
- [ ] Add "Analyze Images" button to inspection detail
- [ ] Create `/inspections/{id}/analyze` POST handler
- [ ] Enqueue analysis job
- [ ] Show "Analyzing..." state with progress
- [ ] Use htmx polling or SSE to update when complete
- [ ] Handle case of no images to analyze
- [ ] Prevent re-analysis while in progress

**Route:**
```
POST /inspections/{id}/analyze  -> Enqueue analysis job, return status
GET  /inspections/{id}/status   -> Get current analysis status (for polling)
```

**Acceptance Criteria:**
- User can trigger analysis with one click
- UI shows analysis in progress
- UI updates when analysis completes
- Button is disabled during analysis

---

## Phase 4: Review & Reports (Week 7-8)

### P1-009: Violation Review Interface
**Dependency:** P1-007

Create the interface for reviewing AI-detected violations.

**Tasks:**
- [ ] Create `/workspaces/lukaut/web/templates/pages/inspections/review.html`
- [ ] Display violations with associated images
- [ ] Show confidence level and severity
- [ ] Show matched regulations
- [ ] Implement accept/reject buttons (htmx)
- [ ] Implement edit violation description
- [ ] Implement add inspector notes
- [ ] Implement add manual violation
- [ ] Implement violation reordering (drag-and-drop)
- [ ] Implement change linked regulations

**Routes:**
```
GET  /inspections/{id}/review              -> Show review interface
POST /inspections/{id}/violations          -> Add manual violation
PUT  /violations/{id}                      -> Update violation
PUT  /violations/{id}/status               -> Accept/reject violation
DELETE /violations/{id}                    -> Remove violation
POST /violations/{id}/regulations          -> Add regulation link
DELETE /violations/{id}/regulations/{regId} -> Remove regulation link
```

**Acceptance Criteria:**
- User sees all detected violations
- User can accept or reject each violation
- User can edit violation descriptions
- User can add their own violations
- User can modify linked regulations
- Changes persist immediately (htmx)

---

### P1-010: OSHA Regulations Seed Data
**Blocks:** Regulation display

Seed the regulations table with OSHA 1926 construction standards.

**Tasks:**
- [ ] Research and compile key OSHA 1926 regulations
- [ ] Focus on most common violation categories:
  - Fall Protection (Subpart M)
  - Scaffolding (Subpart L)
  - Ladders and Stairways (Subpart X)
  - Electrical (Subpart K)
  - Personal Protective Equipment (Subpart E)
  - Excavations (Subpart P)
- [ ] Create seed migration with INSERT statements
- [ ] Include standard_number, title, category, full_text, summary
- [ ] Verify full-text search works

**Target: 100-200 key regulations for MVP**

**Data Sources:**
- https://www.osha.gov/laws-regs/regulations/standardnumber/1926
- eCFR (Electronic Code of Federal Regulations)

**Acceptance Criteria:**
- Key construction safety regulations are in database
- Full-text search returns relevant results
- Category browsing works

---

### P1-011: Regulation Search & Browse
**Dependency:** P1-010

UI for searching and browsing regulations.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/regulation.go`
- [ ] Create `/workspaces/lukaut/web/templates/pages/regulations/index.html`
- [ ] Implement full-text search with htmx
- [ ] Implement category filtering
- [ ] Show regulation detail in modal/slide-out
- [ ] Allow adding regulation to violation from search

**Routes:**
```
GET /regulations                -> List/search regulations
GET /regulations/{id}           -> Get regulation detail (JSON for modals)
GET /regulations/search?q=xxx   -> Search endpoint (htmx partial)
```

**Acceptance Criteria:**
- User can search regulations by keyword
- User can browse by category
- User can view full regulation text
- User can add regulation to violation from search

---

### P1-012: PDF Report Generation
**Dependency:** P1-009

Generate PDF inspection reports.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/report/report.go` interface
- [ ] Create `/workspaces/lukaut/internal/report/pdf.go` using maroto
- [ ] Implement report layout from TECHNICAL.md specification
- [ ] Include Lukaut branding (logo, colors)
- [ ] Include site information
- [ ] Include executive summary
- [ ] Include violations with photos and regulations
- [ ] Include signature line
- [ ] Store generated PDF in storage
- [ ] Create report record in database

**Report Structure (from TECHNICAL.md):**
```
- Header with logo
- Site Information section
- Executive Summary (violation counts by severity)
- Violations (numbered, with photo, description, regulations)
- Signature line
- Footer with branding
```

**Library:**
```go
"github.com/johnfercher/maroto/v2"
```

**Acceptance Criteria:**
- PDF generates with all inspection data
- Photos are embedded
- Regulations are cited
- Branding matches brand guidelines
- PDF downloads correctly

---

### P1-013: Report Generation Handler
**Dependency:** P1-012

HTTP handlers for generating and downloading reports.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/report.go`
- [ ] Create `/workspaces/lukaut/internal/jobs/generate_report.go`
- [ ] Add "Generate Report" button to review page
- [ ] Enqueue report generation job
- [ ] Show generation progress
- [ ] List generated reports for inspection
- [ ] Implement download endpoint

**Routes:**
```
POST /inspections/{id}/reports      -> Generate new report
GET  /inspections/{id}/reports      -> List reports for inspection
GET  /reports/{id}/download         -> Download report file
```

**Acceptance Criteria:**
- User can generate report from review page
- Report generation happens in background
- User can download generated PDF
- Multiple reports can be generated per inspection

---

## Phase 5: Billing & Launch Prep (Week 9-10)

### P1-014: Stripe Subscription Integration
**Blocks:** Production launch

Integrate Stripe for subscription billing.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/billing/billing.go` interface
- [ ] Create `/workspaces/lukaut/internal/billing/stripe.go`
- [ ] Add Stripe configuration to config.go
- [ ] Create Stripe products and prices (Starter $99, Professional $149)
- [ ] Implement checkout session creation
- [ ] Implement customer portal redirect
- [ ] Create webhook handler for subscription events
- [ ] Update user subscription status on webhook
- [ ] Implement `RequireActiveSubscription` middleware

**Routes:**
```
GET  /settings/billing              -> Show billing page
POST /billing/checkout              -> Create Stripe checkout session
GET  /billing/portal                -> Redirect to Stripe customer portal
POST /webhooks/stripe               -> Handle Stripe webhooks
```

**Webhook Events to Handle:**
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.paid`
- `invoice.payment_failed`

**Acceptance Criteria:**
- User can subscribe to Starter or Professional plan
- User can manage subscription via Stripe portal
- Subscription status updates via webhooks
- Inactive users are blocked from core features

---

### P1-015: Usage Limits & Enforcement
**Dependency:** P1-014

Implement report limits for Starter tier.

**Tasks:**
- [ ] Track reports generated per month per user
- [ ] Add `CountReportsThisMonth` query
- [ ] Check limit before allowing report generation
- [ ] Show usage indicator in UI
- [ ] Show upgrade prompt when approaching/at limit
- [ ] Reset usage tracking monthly (via webhook on invoice.paid)

**Starter Limit:** 20 reports/month
**Professional:** Unlimited

**Acceptance Criteria:**
- Starter users cannot exceed 20 reports/month
- Users see their current usage
- Clear upgrade path when limit reached
- Professional users have no limits

---

### P1-016: Settings & Profile Pages
**Dependency:** P0-003

User settings and profile management.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/settings.go`
- [ ] Create profile edit page (name, company, phone)
- [ ] Create password change page
- [ ] Create billing page (subscription status, portal link)
- [ ] Create account deletion page (with confirmation)

**Routes:**
```
GET  /settings              -> Settings overview
GET  /settings/profile      -> Profile edit form
POST /settings/profile      -> Update profile
GET  /settings/password     -> Password change form
POST /settings/password     -> Change password
GET  /settings/billing      -> Billing/subscription info
POST /settings/delete       -> Delete account
```

**Acceptance Criteria:**
- User can update profile information
- User can change password
- User can view subscription status
- User can delete account

---

### P2-001: Trial Period Support
**Priority:** P2 (can launch without)

Allow users to try before subscribing.

**Tasks:**
- [ ] Add 14-day trial on registration
- [ ] Set subscription_status to "trialing"
- [ ] Track trial end date
- [ ] Show trial countdown in UI
- [ ] Allow full access during trial
- [ ] Prompt to subscribe when trial ends

**Acceptance Criteria:**
- New users get 14-day trial automatically
- Trial users have full Professional features
- Clear indication of trial status and end date
- Smooth conversion to paid subscription

---

### P2-002: DOCX Report Generation
**Priority:** P2 (PDF sufficient for launch)

Add Word document report format.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/report/docx.go`
- [ ] Match PDF report structure
- [ ] Allow user to choose format on download
- [ ] Store both formats when generating

**Library:**
```go
"github.com/nguyenthenguyen/docx"
// or
"baliance.com/gooxml"
```

**Acceptance Criteria:**
- DOCX generates with same content as PDF
- User can choose PDF or DOCX download
- DOCX is editable in Word

---

### P2-003: Sites Management
**Priority:** P2 (inline site entry sufficient for launch)

Reusable site management.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/handler/site.go`
- [ ] Site CRUD pages
- [ ] Site selector in inspection form
- [ ] View past inspections for a site
- [ ] Client information management

**Routes:**
```
GET  /sites              -> List sites
GET  /sites/new          -> Create form
POST /sites              -> Create site
GET  /sites/{id}         -> Site detail with inspection history
GET  /sites/{id}/edit    -> Edit form
PUT  /sites/{id}         -> Update site
DELETE /sites/{id}       -> Delete site
```

**Acceptance Criteria:**
- User can save frequently-used sites
- Site can be selected when creating inspection
- Past inspections for site are visible

---

### P2-004: Email Report to Client
**Priority:** P2 (download sufficient for launch)

Send report directly to client.

**Tasks:**
- [ ] Add client email field to inspection
- [ ] Add "Email to Client" button on report
- [ ] Create email template with download link
- [ ] Generate presigned URL for download
- [ ] Track email sent status

**Acceptance Criteria:**
- User can email report to client
- Email includes download link
- Link expires after reasonable time

---

## Phase 6: Operations & Polish (Week 11-12)

### P2-005: Prometheus Metrics
**Priority:** P2 (can launch without monitoring)

Add application metrics.

**Tasks:**
- [ ] Create `/workspaces/lukaut/internal/telemetry/metrics.go`
- [ ] Add metrics from TECHNICAL.md specification
- [ ] Expose `/metrics` endpoint
- [ ] Add Grafana dashboard (if using)

**Key Metrics:**
- `lukaut_inspections_created_total`
- `lukaut_images_analyzed_total`
- `lukaut_violations_detected_total`
- `lukaut_reports_generated_total`
- `lukaut_ai_request_duration_seconds`
- `lukaut_ai_tokens_used_total`

---

### P2-006: Sentry Error Tracking
**Priority:** P2 (can use logs initially)

Integrate Sentry for error tracking.

**Tasks:**
- [ ] Add Sentry SDK
- [ ] Configure in production only
- [ ] Add user context to errors
- [ ] Add request context
- [ ] Test error capture

---

### P2-007: Rate Limiting
**Priority:** P2 (can launch without)

Add rate limiting to auth endpoints.

**Tasks:**
- [ ] Implement token bucket rate limiter
- [ ] Apply to `/login`, `/register`, `/forgot-password`
- [ ] Use IP-based limiting
- [ ] Return 429 with Retry-After header

---

### P2-008: Production Docker Configuration
**Priority:** P2 (needed for production deploy)

Finalize production deployment configuration.

**Tasks:**
- [ ] Create multi-stage Dockerfile
- [ ] Create production docker-compose.yml
- [ ] Create Caddyfile for reverse proxy
- [ ] Add health check endpoint improvements
- [ ] Document deployment process

---

### P3-001: Image Annotation
**Priority:** P3 (post-MVP)

Allow inspector to draw on images.

---

### P3-002: Violation Bounding Box Display
**Priority:** P3 (post-MVP)

Show AI-detected violation locations on images.

---

### P3-003: Dashboard Analytics
**Priority:** P3 (post-MVP)

Enhanced dashboard with charts and trends.

---

### P3-004: Inspection Templates
**Priority:** P3 (post-MVP)

Save and reuse inspection configurations.

---

## Implementation Schedule

### Week 1-2: Foundation
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P0-001: Template Engine | 8 | None |
| P0-002: Auth Service | 12 | P0-001 |
| P0-003: Auth Handlers | 8 | P0-002 |
| P0-004: Email Tokens | 4 | P0-002 |
| P0-005: Email Service | 6 | P0-004 |
| P0-006: Password Reset | 6 | P0-005 |
| **Total** | **44** | |

### Week 3-4: Core Flow
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P1-001: Storage Service | 8 | None |
| P1-002: Dashboard | 4 | P0-003 |
| P1-003: Inspection CRUD | 12 | P1-002 |
| P1-004: Image Upload | 12 | P1-001, P1-003 |
| P1-005: Job Worker | 10 | None |
| **Total** | **46** | |

### Week 5-6: AI Integration
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P1-006: Anthropic Provider | 12 | P1-005 |
| P1-007: Analysis Job | 10 | P1-006 |
| P1-008: Analysis UI | 6 | P1-007 |
| P1-010: Regulations Seed | 8 | None |
| **Total** | **36** | |

### Week 7-8: Review & Reports
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P1-009: Violation Review | 16 | P1-007 |
| P1-011: Regulation Search | 8 | P1-010 |
| P1-012: PDF Generation | 12 | P1-009 |
| P1-013: Report Handler | 6 | P1-012 |
| **Total** | **42** | |

### Week 9-10: Billing & Launch
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P1-014: Stripe Integration | 16 | P0-003 |
| P1-015: Usage Limits | 6 | P1-014 |
| P1-016: Settings Pages | 8 | P0-003 |
| P2-001: Trial Period | 4 | P1-014 |
| **Total** | **34** | |

### Week 11-12: Polish
| Task | Est. Hours | Dependencies |
|------|------------|--------------|
| P2-002: DOCX Generation | 8 | P1-012 |
| P2-003: Sites Management | 10 | P1-003 |
| P2-005: Metrics | 6 | None |
| P2-006: Sentry | 4 | None |
| P2-008: Docker Config | 6 | None |
| Testing & Bug Fixes | 20 | All |
| **Total** | **54** | |

---

## Architectural Decisions Needed

### Decision 1: Session Store
**Question:** Use gorilla/sessions or build custom session handling?

**Options:**
- A) gorilla/sessions with PostgreSQL store
  - Pro: Well-tested, handles edge cases
  - Con: Additional dependency
- B) Custom implementation
  - Pro: No dependency, full control
  - Con: More code to maintain

**Recommendation:** Custom implementation - the pattern is simple, already have DB schema, and consistent with minimal dependencies approach.

### Decision 2: HEIC Conversion
**Question:** How to handle HEIC images from iPhones?

**Options:**
- A) Convert server-side using libheif
  - Pro: Works for all images
  - Con: Requires CGO/system library
- B) Convert client-side with JavaScript
  - Pro: No server dependency
  - Con: Browser compatibility, larger client bundle
- C) Reject HEIC, require JPEG/PNG
  - Pro: Simplest
  - Con: Poor UX for iPhone users

**Recommendation:** Option A for launch, or Option C for MVP simplicity with plan to add HEIC later. Most iPhone users can configure camera to use JPEG.

### Decision 3: Real-time Analysis Updates
**Question:** How to notify user when analysis completes?

**Options:**
- A) htmx polling (every 5 seconds)
  - Pro: Simple, reliable
  - Con: Slight delay, unnecessary requests
- B) Server-Sent Events (SSE)
  - Pro: Real-time, efficient
  - Con: Connection management complexity
- C) WebSockets
  - Pro: Bi-directional
  - Con: Overkill for this use case

**Recommendation:** Option A for MVP - htmx polling is simple and the 5-second delay is acceptable. Can upgrade to SSE later if needed.

### Decision 4: AI Cost Management
**Question:** How to handle AI cost spikes or abuse?

**Recommendations:**
- Set per-user daily limit (e.g., 50 images/day)
- Track and display usage in settings
- Alert on unusual usage patterns
- Consider image-based pricing in future tiers

---

## Risk Mitigation

### Technical Risks

| Risk | Mitigation |
|------|------------|
| AI analysis inaccurate | Position as "assistant" - inspector always reviews. Improve prompts based on feedback. |
| Large images slow to process | Resize images before AI analysis. Process in background. |
| Storage costs | Monitor usage. R2 free tier generous. Add limits if needed. |
| Database performance | Proper indexing already in schema. Monitor slow queries. |

### Business Risks

| Risk | Mitigation |
|------|------------|
| Low initial adoption | Start with single domain expert. Get testimonials. |
| Competitors copy | Move fast. Focus on construction niche. Build relationships. |
| Regulation changes | Design for easy updates. Monitor OSHA news. |

---

## Success Criteria for MVP Launch

### Functional Criteria
- [ ] User can register, verify email, login, logout
- [ ] User can create inspection and upload photos
- [ ] AI analyzes images and suggests violations
- [ ] User can review, accept/reject violations
- [ ] User can generate PDF report
- [ ] User can subscribe via Stripe
- [ ] Starter tier enforces 20 report/month limit

### Non-Functional Criteria
- [ ] Page load times < 2 seconds
- [ ] AI analysis completes within 60 seconds per image
- [ ] Report generation completes within 30 seconds
- [ ] Works on desktop Chrome, Firefox, Safari
- [ ] Works on mobile browsers (responsive)

### Testing Criteria
- [ ] Domain expert completes 5 inspections
- [ ] Domain expert validates AI accuracy is "helpful"
- [ ] Domain expert validates reports are professional
- [ ] Domain expert would pay $99-149/month

---

## Next Steps

1. **Immediate:** Begin P0-001 (Template Engine Setup)
2. **This Week:** Complete Phase 1 foundation
3. **Week 2:** Begin core inspection flow
4. **Ongoing:** Weekly check-ins with domain expert for feedback

---

## Appendix: File Creation Checklist

### Internal Packages to Create
```
internal/
  service/
    user.go
    inspection.go
    image.go
    violation.go
    regulation.go
    report.go
  handler/
    auth.go
    dashboard.go
    inspection.go
    image.go
    violation.go
    regulation.go
    report.go
    settings.go
    webhook/
      stripe.go
  middleware/
    auth.go
    subscription.go
    logging.go
    csrf.go
  ai/
    ai.go
    anthropic/
      provider.go
      image_analysis.go
      regulation_match.go
    mock/
      provider.go
  storage/
    storage.go
    local.go
    r2.go
  email/
    email.go
    postmark.go
    mock.go
  billing/
    billing.go
    stripe.go
  report/
    report.go
    pdf.go
    docx.go
  jobs/
    analyze_inspection.go
    generate_report.go
    send_email.go
  worker/
    worker.go
  telemetry/
    metrics.go
```

### Template Files to Create
```
web/templates/
  layouts/
    base.html
    app.html
    auth.html
  partials/
    nav.html
    footer.html
    flash.html
    pagination.html
  pages/
    home.html
    dashboard.html
    auth/
      register.html
      login.html
      forgot_password.html
      reset_password.html
      verify_email.html
    inspections/
      index.html
      new.html
      show.html
      edit.html
      review.html
    regulations/
      index.html
    settings/
      index.html
      profile.html
      password.html
      billing.html
  email/
    verification.html
    password_reset.html
    report_ready.html
```

### Migration Files to Create
```
internal/migrations/
  00002_email_tokens.sql
  00003_regulation_seed.sql
```
