# Lukaut Technical Architecture

## Overview

This document records the technical decisions for Lukaut, including the rationale for each choice. The guiding principles are: simplicity, maintainability by a solo developer, minimal operational overhead, and alignment with patterns established in Hiri.

**What Lukaut Does:**
1. Inspector uploads site photos
2. AI analyzes images for potential safety violations
3. AI suggests applicable OSHA regulations for each violation
4. Inspector reviews, accepts/rejects, and annotates findings
5. One-click report generation (PDF/DOCX)

---

## Language & Runtime

### Backend: Go

**Choice:** Go (latest stable, 1.22+)

**Rationale:**
- Single binary deployment eliminates dependency management in production
- Strong standard library reduces external dependencies
- Excellent performance characteristics without tuning
- Static typing catches errors at compile time
- Straightforward concurrency model for background tasks (AI processing, report generation)
- Consistent with Hiri codebase; shared patterns and learnings

---

## Web Framework

### Choice: Standard Library with Thin Router Wrapper

**Package:** None (stdlib `net/http` only)

**Rationale:**
- Go 1.22+ added method matching and path parameters to `http.ServeMux`
- Same ~50 line wrapper as Hiri provides chi-like ergonomics
- Zero external dependencies for routing
- Full compatibility with any `net/http` middleware

**Implementation:**
Identical to Hiri router wrapper pattern. See Hiri TECHNICAL.md for implementation details.

```go
// Route registration examples
r.Get("/inspections", listInspectionsHandler)
r.Post("/inspections", createInspectionHandler)
r.Get("/inspections/{id}", getInspectionHandler)
r.Post("/inspections/{id}/analyze", analyzeInspectionHandler)
```

---

## Database

### Choice: PostgreSQL

**Version:** 16 (latest stable)

**Rationale:**
- Robust, battle-tested relational database
- Excellent JSON/JSONB support for AI response storage and flexible metadata
- Array types for storing lists (violation IDs, image references)
- Full-text search for regulation lookup
- Consistent with Hiri; shared operational knowledge

**What PostgreSQL Handles:**
- All application data (users, inspections, violations, reports)
- Session storage
- Background job queue
- OSHA regulations database with full-text search
- AI response caching (optional, for cost optimization)

---

## Database Access

### Choice: sqlc + pgx

**Packages:**
- `github.com/sqlc-dev/sqlc` (code generation)
- `github.com/jackc/pgx/v5` (PostgreSQL driver)

**Rationale:**
- Write plain SQL, get type-safe Go code
- No ORM abstraction to fight or debug
- Queries are explicit and optimizable
- Compile-time verification of SQL syntax and types
- Consistent with Hiri

**Workflow:**
1. Define schema in SQL migration files
2. Write queries in `.sql` files with sqlc annotations
3. Run `sqlc generate` to produce Go code
4. Call generated functions from application code

---

## Schema Migrations

### Choice: Goose

**Package:** `github.com/pressly/goose/v3`

**Rationale:**
- Simple, file-based migrations
- SQL migrations for transparency
- Embeddable in the application binary
- Consistent with Hiri

**Migration Strategy:**
- Sequential, timestamped migration files
- All migrations in `/migrations` directory
- Migrations run automatically on application startup
- Down migrations provided for development

---

## Frontend

### Primary: Server-Rendered HTML + htmx + Alpine.js + Tailwind CSS

**Rationale:**
- Server-side rendering simplifies state management
- htmx enables dynamic updates without JavaScript build step
- Alpine.js handles UI interactions (modals, image galleries, form state)
- Tailwind CSS provides utility-first styling
- No Node.js required in production
- Fast initial page loads

**When This Approach Applies:**
- Dashboard and inspection list
- Inspection creation and photo upload
- Violation review and annotation interface
- Report preview and generation
- Account and settings pages

### Secondary: Svelte (If Needed)

**When to Reach for Svelte:**
- Complex image annotation interface (if basic htmx proves insufficient)
- Side-by-side image comparison with synchronized zoom/pan
- Drag-and-drop violation reordering with complex state

**Current Assessment:** htmx + Alpine.js should handle MVP requirements. Svelte is a fallback if the violation review interface becomes unwieldy.

---

## htmx Patterns

### Form Submission with Partial Swaps

For forms that should update in place without a full page refresh (showing validation errors, etc.), use this pattern:

**1. Create a partial template** (`web/templates/partials/{form_name}.html`):
```html
{{define "login_form"}}
<form id="login-form" action="/login" method="POST"
      hx-post="/login"
      hx-swap="outerHTML"
      hx-target="#login-form">

    {{/* Flash message inside form for htmx swaps */}}
    {{if .Flash}}
    <div class="rounded-md p-4 {{if eq .Flash.Type "error"}}bg-red-50{{end}}">
        <p>{{.Flash.Message}}</p>
    </div>
    {{end}}

    {{/* Form fields... */}}
</form>
{{end}}
```

**2. Include partial in the page template** (`web/templates/pages/auth/login.html`):
```html
{{define "content"}}
{{template "login_form" .}}
{{end}}
```

**3. Handler checks for htmx and returns partial on error**:
```go
func (h *AuthHandler) renderLoginError(w http.ResponseWriter, r *http.Request, ...) {
    data := AuthPageData{...}

    // For htmx requests, return just the form partial
    if r.Header.Get("HX-Request") == "true" {
        h.renderer.RenderPartial(w, "login_form", data)
        return
    }

    // For regular requests, return full page
    h.renderer.RenderHTTP(w, "auth/login", data)
}
```

**4. Handler uses HX-Redirect for successful redirects**:
```go
// For htmx requests, use HX-Redirect header
if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("HX-Redirect", redirectURL)
    w.WriteHeader(http.StatusOK)
    return
}

http.Redirect(w, r, redirectURL, http.StatusSeeOther)
```

### Key Points

- **Partial naming**: File `partials/foo.html` with `{{define "foo"}}` is rendered via `RenderPartial(w, "foo", data)`
- **Form ID required**: htmx needs `hx-target` to reference the form by ID for swapping
- **Flash in form**: Include flash/error messages inside the form so they appear on partial swaps
- **Graceful degradation**: Keep `action` and `method` on form so it works without JavaScript

### Renderer Configuration

Partials are automatically parsed into all layouts (public, auth, app) so they can be used with `{{template "partial_name" .}}` in page templates.

---

## CSS Framework

### Choice: Tailwind CSS

**Rationale:**
- Utility-first approach speeds up development
- Consistent with Hiri; shared component patterns
- Small production bundle with purging

**Build Process:**
- Tailwind CLI in watch mode during development
- Production build with minification and purging
- Output to `/web/static/css/`

**Color Configuration:**
Tailwind config extended with Lukaut brand colors:

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        'navy': {
          DEFAULT: '#1E3A5F',
          50: '#F0F4F8',
          // ... full scale
          900: '#102A43',
          950: '#0A1929',
        },
        'safety-orange': {
          DEFAULT: '#FF6B35',
          50: '#FFF4F0',
          // ... full scale
          900: '#7A2D14',
          950: '#4A1B0C',
        },
      }
    }
  }
}
```

---

## Authentication

### Choice: Cookie-Based Sessions

**Packages:**
- `github.com/gorilla/sessions` (session management)
- Custom middleware for auth checks

**Rationale:**
- Simpler than JWT for server-rendered applications
- Automatic handling by browsers
- Easy to invalidate
- Works seamlessly with htmx
- Consistent with Hiri

**Session Storage:** PostgreSQL

**Authentication Flow:**
1. Email/password registration with email verification
2. Email/password login
3. Password reset via email token
4. Session stored in database with SHA-256 hashed token

**Security Implementation:**
- bcrypt for password hashing
- 32-byte cryptographically secure session tokens
- Tokens stored as SHA-256 hashes
- HttpOnly, Secure, SameSite cookies
- Rate limiting on auth endpoints

**Middleware Stack:**
```go
// WithUser loads user from session cookie (if present)
func WithUser(userService service.UserService) func(http.Handler) http.Handler

// RequireUser blocks unauthenticated requests
func RequireUser(next http.Handler) http.Handler

// RequireActiveSubscription checks Stripe subscription status
func RequireActiveSubscription(next http.Handler) http.Handler
```

---

## AI Integration

### Choice: Anthropic Claude API

**Package:** `github.com/anthropics/anthropic-sdk-go`

**Rationale:**
- Best-in-class vision capabilities for image analysis
- Strong reasoning for regulation matching
- Developer's preferred AI provider
- Well-documented API with Go SDK

### Architecture

**Provider Abstraction:**
```go
// internal/ai/ai.go
type ImageAnalyzer interface {
    AnalyzeImage(ctx context.Context, params AnalyzeParams) (*AnalysisResult, error)
}

type RegulationMatcher interface {
    MatchRegulations(ctx context.Context, violation string) ([]Regulation, error)
}

type ReportDrafter interface {
    DraftViolationDescription(ctx context.Context, params DraftParams) (string, error)
}

// Combine into single provider interface
type AIProvider interface {
    ImageAnalyzer
    RegulationMatcher
    ReportDrafter
}
```

**Implementation:**
```go
// internal/ai/anthropic/provider.go
type AnthropicProvider struct {
    client *anthropic.Client
    model  string
    logger *slog.Logger
}

func NewAnthropicProvider(cfg Config) (*AnthropicProvider, error) {
    client := anthropic.NewClient(cfg.APIKey)
    return &AnthropicProvider{
        client: client,
        model:  "claude-sonnet-4-20250514", // Configurable
        logger: cfg.Logger,
    }, nil
}
```

### Image Analysis Flow

**Step 1: Upload and Store**
```go
// User uploads images to inspection
POST /inspections/{id}/images
// Images stored in R2, metadata in PostgreSQL
```

**Step 2: Trigger Analysis**
```go
// User clicks "Analyze" or automatic on upload
POST /inspections/{id}/analyze

// Handler enqueues background job
func (h *InspectionHandler) Analyze(w http.ResponseWriter, r *http.Request) {
    inspectionID := r.PathValue("id")
    
    // Enqueue analysis job
    err := h.jobQueue.Enqueue(ctx, jobs.AnalyzeInspection{
        InspectionID: inspectionID,
        UserID:       middleware.GetUserID(ctx),
    })
    
    // Return immediately, show "analyzing" state
    // htmx polls for completion or uses SSE
}
```

**Step 3: Background Analysis Job**
```go
// internal/jobs/analyze_inspection.go
func (j *AnalyzeInspectionJob) Execute(ctx context.Context) error {
    inspection, _ := j.queries.GetInspection(ctx, j.InspectionID)
    images, _ := j.queries.GetInspectionImages(ctx, j.InspectionID)
    
    for _, img := range images {
        // Download image from R2
        imageData, _ := j.storage.Get(ctx, img.StorageKey)
        
        // Analyze with Claude
        result, _ := j.ai.AnalyzeImage(ctx, ai.AnalyzeParams{
            ImageData:   imageData,
            ContentType: img.ContentType,
            Context:     "Construction site safety inspection",
        })
        
        // Store potential violations
        for _, violation := range result.PotentialViolations {
            // Match regulations
            regulations, _ := j.ai.MatchRegulations(ctx, violation.Description)
            
            // Store in database
            j.queries.CreatePotentialViolation(ctx, repository.CreatePotentialViolationParams{
                InspectionID:     j.InspectionID,
                ImageID:          img.ID,
                Description:      violation.Description,
                Confidence:       violation.Confidence,
                BoundingBox:      violation.BoundingBox, // JSONB
                SuggestedRegulations: regulationIDs,     // UUID array
                Status:           "pending_review",
            })
        }
    }
    
    // Update inspection status
    j.queries.UpdateInspectionStatus(ctx, j.InspectionID, "analysis_complete")
    return nil
}
```

### Prompt Engineering

**Image Analysis System Prompt:**
```
You are an expert construction safety inspector assistant. Analyze the provided 
construction site image and identify potential OSHA safety violations.

For each potential violation found:
1. Describe the specific hazard observed
2. Indicate the location in the image (provide bounding box coordinates if possible)
3. Rate confidence level (high, medium, low)
4. Suggest the category of OSHA regulation that may apply

Focus on common construction hazards:
- Fall protection (guardrails, holes, scaffolding)
- Personal protective equipment (hard hats, safety glasses, high-visibility vests)
- Electrical hazards (exposed wiring, improper grounding)
- Housekeeping (debris, blocked exits, trip hazards)
- Ladder safety (improper setup, damaged equipment)
- Scaffolding (missing planks, improper bracing)
- Excavation (shoring, access, spoil placement)

Respond in JSON format:
{
  "violations": [
    {
      "description": "Worker on elevated platform without fall protection",
      "location": "Upper right quadrant, near scaffolding",
      "bounding_box": {"x": 0.6, "y": 0.1, "width": 0.3, "height": 0.4},
      "confidence": "high",
      "category": "fall_protection",
      "severity": "serious"
    }
  ],
  "general_observations": "Site appears to be in active construction phase...",
  "image_quality_notes": "Image is clear and well-lit"
}
```

**Regulation Matching System Prompt:**
```
You are an OSHA regulations expert. Given a safety violation description, 
identify the most applicable OSHA construction standards (29 CFR 1926).

Violation: {violation_description}

Provide the top 3 most relevant regulations with:
1. Standard number (e.g., 1926.501(b)(1))
2. Standard title
3. Brief explanation of why this regulation applies
4. Direct quote of the relevant requirement (if possible)

Respond in JSON format:
{
  "regulations": [
    {
      "standard_number": "1926.501(b)(1)",
      "title": "Duty to have fall protection",
      "relevance": "Requires fall protection for workers on surfaces with unprotected edges 6 feet or more above lower level",
      "requirement_text": "Each employee on a walking/working surface with an unprotected side or edge which is 6 feet or more above a lower level shall be protected..."
    }
  ]
}
```

### Cost Management

**Strategies:**
- Cache AI responses for identical/similar images (hash-based lookup)
- Use claude-haiku for initial screening, claude-sonnet for detailed analysis
- Batch multiple images in single API call where possible
- Set per-user daily/monthly API cost limits
- Track usage in database for billing reconciliation

**Usage Tracking:**
```sql
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    inspection_id UUID REFERENCES inspections(id),
    model VARCHAR(50) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    cost_cents INTEGER NOT NULL,  -- Calculated at time of request
    request_type VARCHAR(50) NOT NULL,  -- 'image_analysis', 'regulation_match', 'description_draft'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for billing queries
CREATE INDEX idx_ai_usage_user_month ON ai_usage (user_id, date_trunc('month', created_at));
```

---

## File Storage

### Choice: Cloudflare R2 with S3-Compatible Abstraction

**Package:** `github.com/aws/aws-sdk-go-v2/service/s3`

**Rationale:**
- S3-compatible API allows provider swapping
- Zero egress fees (critical for image-heavy application)
- Consistent with Hiri architecture
- Generous free tier (10GB storage, unlimited egress)

**Architecture:**
```go
// internal/storage/storage.go
type Storage interface {
    Put(ctx context.Context, key string, data io.Reader, opts PutOptions) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    URL(ctx context.Context, key string) (string, error)  // Presigned or public URL
    Exists(ctx context.Context, key string) (bool, error)
}

// Implementations
type LocalStorage struct { ... }  // Development
type R2Storage struct { ... }     // Production
```

**Storage Key Structure:**
```
users/{user_id}/inspections/{inspection_id}/images/{image_id}.{ext}
users/{user_id}/inspections/{inspection_id}/reports/{report_id}.pdf
users/{user_id}/inspections/{inspection_id}/reports/{report_id}.docx
```

**Image Processing:**
- Accept common formats: JPEG, PNG, HEIC
- Convert HEIC to JPEG on upload (iOS photos)
- Generate thumbnails for list views (200x200)
- Store original for AI analysis
- Validate file size limits (20MB per image)

```go
// internal/service/image.go
type ImageService struct {
    storage   storage.Storage
    processor ImageProcessor
}

func (s *ImageService) Upload(ctx context.Context, params UploadParams) (*Image, error) {
    // Validate file type and size
    // Convert HEIC if needed
    // Generate thumbnail
    // Upload original and thumbnail to storage
    // Create database record
    // Return image metadata
}
```

---

## Payment Processing

### Choice: Stripe Subscriptions

**Package:** `github.com/stripe/stripe-go/v76`

**Rationale:**
- Industry standard
- Handles subscription lifecycle
- Consistent with Hiri
- Customer portal for self-service billing management

**Subscription Tiers:**
```go
// Defined in Stripe, referenced by price ID
var SubscriptionTiers = map[string]Tier{
    "starter": {
        Name:           "Starter",
        PriceID:        "price_starter_monthly",
        ReportsPerMonth: 20,
        PriceCents:     9900,  // $99/month
    },
    "professional": {
        Name:           "Professional", 
        PriceID:        "price_professional_monthly",
        ReportsPerMonth: -1,  // Unlimited
        PriceCents:     14900,  // $149/month
    },
}
```

**Webhook Events:**
- `customer.subscription.created` — Activate account
- `customer.subscription.updated` — Handle plan changes
- `customer.subscription.deleted` — Deactivate account
- `invoice.payment_failed` — Notify user, grace period
- `invoice.paid` — Reset monthly usage counters

**Usage Enforcement:**
```go
// Middleware to check subscription status and usage limits
func RequireActiveSubscription(queries *repository.Queries) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := middleware.GetUserFromContext(r.Context())
            
            // Check subscription status
            if user.SubscriptionStatus != "active" && user.SubscriptionStatus != "trialing" {
                // Redirect to billing page
                http.Redirect(w, r, "/settings/billing", http.StatusSeeOther)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Check before report generation
func (s *ReportService) CanGenerateReport(ctx context.Context, userID uuid.UUID) (bool, error) {
    user, _ := s.queries.GetUser(ctx, userID)
    
    if user.SubscriptionTier == "professional" {
        return true, nil  // Unlimited
    }
    
    // Count reports this month
    count, _ := s.queries.CountReportsThisMonth(ctx, userID)
    limit := SubscriptionTiers[user.SubscriptionTier].ReportsPerMonth
    
    return count < limit, nil
}
```

---

## Email

### Choice: Postmark

**Package:** Custom HTTP client (Postmark API is simple)

**Rationale:**
- Best deliverability reputation
- Simple API
- Consistent with Hiri

**Email Types:**
- Welcome / Email verification
- Password reset
- Report ready for download
- Subscription confirmation
- Payment failed notification
- Usage limit warning (approaching monthly cap)

**Template System:**
- Go HTML templates in `/web/templates/email/`
- Both HTML and plain text versions
- Lukaut branding (Forest Deep header, Paradise Gold accents)

---

## Background Jobs

### Choice: Database-Backed Job Queue

**Rationale:**
- No additional infrastructure
- PostgreSQL SKIP LOCKED provides reliable processing
- Sufficient for workload scale
- Consistent with Hiri

**Job Types:**
```go
// High priority - user waiting
const (
    JobTypeAnalyzeInspection = "analyze_inspection"
    JobTypeGenerateReport    = "generate_report"
)

// Normal priority - background
const (
    JobTypeSendEmail         = "send_email"
    JobTypeCleanupExpiredTokens = "cleanup_tokens"
    JobTypeSyncStripeUsage   = "sync_stripe_usage"
)
```

**Job Schema:**
```sql
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, processing, completed, failed
    priority INTEGER NOT NULL DEFAULT 0,  -- Higher = more urgent
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_pending ON jobs (priority DESC, scheduled_at ASC) 
    WHERE status = 'pending';
```

**Worker Configuration:**
- Poll interval: 1 second
- Concurrency: 5 workers (configurable)
- Exponential backoff on failure
- Dead letter handling after max attempts

---

## Report Generation

### Choice: Go Libraries for PDF and DOCX

**Packages:**
- `github.com/johnfercher/maroto/v2` (PDF generation)
- `github.com/nguyenthenguyen/docx` or `baliance.com/gooxml` (DOCX generation)

**Rationale:**
- Pure Go implementations, no external dependencies
- maroto provides clean API for structured documents
- DOCX allows user editing post-generation

### Report Structure

```
┌─────────────────────────────────────────────────────────────┐
│  [LUKAUT LOGO]                                              │
│                                                             │
│  SAFETY INSPECTION REPORT                                   │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Site Information                                           │
│  ────────────────                                           │
│  Address: 123 Construction Ave, Miami, FL 33101             │
│  Date: December 15, 2025                                    │
│  Inspector: John Smith                                      │
│  Weather: Clear, 75°F                                       │
│                                                             │
│  Executive Summary                                          │
│  ─────────────────                                          │
│  This inspection identified 5 violations requiring          │
│  immediate attention, 3 violations requiring correction     │
│  within 7 days, and 2 recommendations for improvement.      │
│                                                             │
│  Violations                                                 │
│  ──────────                                                 │
│                                                             │
│  1. FALL PROTECTION VIOLATION                    [SERIOUS]  │
│     ┌──────────────────┐                                    │
│     │   [PHOTO]        │  Worker observed on elevated       │
│     │                  │  platform without fall protection  │
│     │                  │  equipment.                        │
│     └──────────────────┘                                    │
│                                                             │
│     Applicable Regulations:                                 │
│     • OSHA 1926.501(b)(1) - Duty to have fall protection   │
│     • OSHA 1926.502(d) - Personal fall arrest systems      │
│                                                             │
│     Required Action: Immediately provide fall protection    │
│     for all workers above 6 feet.                          │
│                                                             │
│  [... additional violations ...]                            │
│                                                             │
│  ─────────────────────────────────────────────────────────  │
│  Inspector Signature: ____________________  Date: ________  │
│                                                             │
│  Generated by Lukaut | lukaut.com                          │
└─────────────────────────────────────────────────────────────┘
```

### Report Generation Service

```go
// internal/service/report.go
type ReportService struct {
    queries   *repository.Queries
    storage   storage.Storage
    pdfGen    ReportGenerator
    docxGen   ReportGenerator
}

type ReportGenerator interface {
    Generate(ctx context.Context, data ReportData) (io.Reader, error)
}

type ReportData struct {
    Inspection  Inspection
    Site        Site
    Inspector   User
    Violations  []ViolationWithRegulations
    GeneratedAt time.Time
}

func (s *ReportService) Generate(ctx context.Context, params GenerateParams) (*Report, error) {
    // Gather all data
    data, err := s.gatherReportData(ctx, params.InspectionID)
    if err != nil {
        return nil, err
    }
    
    // Generate PDF
    pdfReader, err := s.pdfGen.Generate(ctx, data)
    if err != nil {
        return nil, err
    }
    
    pdfKey := fmt.Sprintf("users/%s/inspections/%s/reports/%s.pdf",
        params.UserID, params.InspectionID, reportID)
    s.storage.Put(ctx, pdfKey, pdfReader, storage.PutOptions{
        ContentType: "application/pdf",
    })
    
    // Generate DOCX
    docxReader, err := s.docxGen.Generate(ctx, data)
    if err != nil {
        return nil, err
    }
    
    docxKey := fmt.Sprintf("users/%s/inspections/%s/reports/%s.docx",
        params.UserID, params.InspectionID, reportID)
    s.storage.Put(ctx, docxKey, docxReader, storage.PutOptions{
        ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    })
    
    // Save report record
    report, err := s.queries.CreateReport(ctx, repository.CreateReportParams{
        ID:           reportID,
        InspectionID: params.InspectionID,
        UserID:       params.UserID,
        PDFStorageKey:  pdfKey,
        DOCXStorageKey: docxKey,
        ViolationCount: len(data.Violations),
    })
    
    return report, nil
}
```

---

## OSHA Regulations Database

### Choice: Local PostgreSQL Database with Full-Text Search

**Rationale:**
- No external API dependency (OSHA doesn't have a real-time API)
- Full-text search for regulation lookup
- Can be updated periodically from OSHA website
- Allows AI to reference exact regulation text

### Schema

```sql
-- OSHA regulation standards
CREATE TABLE regulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    standard_number VARCHAR(50) NOT NULL UNIQUE,  -- e.g., "1926.501(b)(1)"
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,  -- e.g., "Fall Protection"
    subcategory VARCHAR(100),
    full_text TEXT NOT NULL,
    summary TEXT,  -- AI-friendly summary
    severity_typical VARCHAR(20),  -- serious, willful, repeat, other
    parent_standard VARCHAR(50),  -- For hierarchy, e.g., "1926.501"
    effective_date DATE,
    last_updated DATE,
    search_vector TSVECTOR,  -- Full-text search
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Full-text search index
CREATE INDEX idx_regulations_search ON regulations USING GIN (search_vector);

-- Category index for browsing
CREATE INDEX idx_regulations_category ON regulations (category, subcategory);

-- Trigger to update search vector
CREATE OR REPLACE FUNCTION update_regulation_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.standard_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.category, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.summary, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.full_text, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER regulation_search_vector_update
    BEFORE INSERT OR UPDATE ON regulations
    FOR EACH ROW EXECUTE FUNCTION update_regulation_search_vector();
```

### Regulation Categories (29 CFR 1926 - Construction)

```go
var RegulationCategories = []string{
    "General Safety and Health Provisions",  // Subpart C
    "Occupational Health and Environmental Controls",  // Subpart D
    "Personal Protective Equipment",  // Subpart E
    "Fire Protection and Prevention",  // Subpart F
    "Signs, Signals, and Barricades",  // Subpart G
    "Materials Handling and Storage",  // Subpart H
    "Tools - Hand and Power",  // Subpart I
    "Welding and Cutting",  // Subpart J
    "Electrical",  // Subpart K
    "Scaffolds",  // Subpart L
    "Fall Protection",  // Subpart M
    "Cranes and Derricks",  // Subpart N (new) / CC (old)
    "Motor Vehicles and Mechanized Equipment",  // Subpart O
    "Excavations",  // Subpart P
    "Concrete and Masonry Construction",  // Subpart Q
    "Steel Erection",  // Subpart R
    "Underground Construction",  // Subpart S
    "Demolition",  // Subpart T
    "Stairways and Ladders",  // Subpart X
}
```

### Data Population

**Initial Seeding:**
- Parse OSHA website or eCFR (Electronic Code of Federal Regulations)
- Script to extract 29 CFR 1926 (Construction Industry)
- Store in migrations as SQL INSERT statements
- ~500-1000 individual standards for construction

**Update Strategy:**
- Quarterly manual review of OSHA updates
- Migration file for each update batch
- Version tracking for audit trail

### Search Implementation

```go
// internal/service/regulation.go
type RegulationService struct {
    queries *repository.Queries
}

func (s *RegulationService) Search(ctx context.Context, query string, limit int) ([]Regulation, error) {
    // Full-text search with ranking
    return s.queries.SearchRegulations(ctx, repository.SearchRegulationsParams{
        Query: query,
        Limit: limit,
    })
}

func (s *RegulationService) GetByCategory(ctx context.Context, category string) ([]Regulation, error) {
    return s.queries.GetRegulationsByCategory(ctx, category)
}

func (s *RegulationService) GetByStandardNumber(ctx context.Context, number string) (*Regulation, error) {
    return s.queries.GetRegulationByStandardNumber(ctx, number)
}
```

**sqlc Query:**
```sql
-- name: SearchRegulations :many
SELECT 
    id, standard_number, title, category, summary,
    ts_rank(search_vector, plainto_tsquery('english', @query)) as rank
FROM regulations
WHERE search_vector @@ plainto_tsquery('english', @query)
ORDER BY rank DESC
LIMIT @limit;
```

---

## Observability & Telemetry

### Choice: Prometheus + Sentry

**Rationale:**
- Industry standard for metrics
- Consistent with Hiri
- Free tier sufficient for MVP

### Business Metrics

```go
// internal/telemetry/metrics.go
var (
    InspectionsCreated = promauto.NewCounter(prometheus.CounterOpts{
        Name: "lukaut_inspections_created_total",
        Help: "Total number of inspections created",
    })
    
    ImagesAnalyzed = promauto.NewCounter(prometheus.CounterOpts{
        Name: "lukaut_images_analyzed_total",
        Help: "Total number of images analyzed by AI",
    })
    
    ViolationsDetected = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "lukaut_violations_detected_total",
        Help: "Total violations detected by category",
    }, []string{"category", "confidence"})
    
    ReportsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "lukaut_reports_generated_total",
        Help: "Total reports generated by format",
    }, []string{"format"})  // pdf, docx
    
    AIRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "lukaut_ai_request_duration_seconds",
        Help:    "AI API request duration",
        Buckets: []float64{1, 2, 5, 10, 30, 60},
    }, []string{"request_type"})
    
    AITokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "lukaut_ai_tokens_used_total",
        Help: "Total AI tokens consumed",
    }, []string{"model", "token_type"})  // input, output
)
```

### Error Tracking (Sentry)

- Disabled by default for development
- Automatic user context via middleware
- AI-specific error categorization

---

## Deployment

### Choice: Docker + Caddy on VPS

**Components:**
- Single Go binary in Docker container
- PostgreSQL in Docker container (or managed database)
- Caddy as reverse proxy with automatic TLS

**Rationale:**
- Simple, reproducible deployment
- Consistent with Hiri
- Easy to migrate to any VPS provider

**Docker Compose:**
```yaml
version: '3.8'

services:
  app:
    build: .
    environment:
      - DATABASE_URL=postgres://lukaut:password@db:5432/lukaut?sslmode=disable
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - R2_ACCESS_KEY_ID=${R2_ACCESS_KEY_ID}
      - R2_SECRET_ACCESS_KEY=${R2_SECRET_ACCESS_KEY}
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:16
    environment:
      - POSTGRES_USER=lukaut
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=lukaut
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
    depends_on:
      - app
    restart: unless-stopped

volumes:
  postgres_data:
  caddy_data:
```

---

## Project Structure

```
lukaut/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── domain/                  # Core business types
│   │   ├── user.go
│   │   ├── inspection.go
│   │   ├── violation.go
│   │   ├── regulation.go
│   │   └── report.go
│   ├── ai/                      # AI provider abstraction
│   │   ├── ai.go                # Interface definition
│   │   └── anthropic/           # Anthropic implementation
│   │       ├── provider.go
│   │       ├── image_analysis.go
│   │       └── regulation_match.go
│   ├── storage/                 # File storage abstraction
│   │   ├── storage.go           # Interface definition
│   │   ├── local.go             # Local filesystem (dev)
│   │   └── r2.go                # Cloudflare R2 (prod)
│   ├── billing/                 # Stripe integration
│   │   ├── billing.go
│   │   └── stripe.go
│   ├── email/                   # Email abstraction
│   │   ├── email.go
│   │   └── postmark.go
│   ├── report/                  # Report generation
│   │   ├── report.go            # Interface
│   │   ├── pdf.go               # PDF generator
│   │   └── docx.go              # DOCX generator
│   ├── repository/              # Database queries (sqlc generated)
│   ├── service/                 # Business logic
│   │   ├── user.go
│   │   ├── inspection.go
│   │   ├── violation.go
│   │   ├── regulation.go
│   │   └── report.go
│   ├── handler/                 # HTTP handlers
│   │   ├── auth.go
│   │   ├── inspection.go
│   │   ├── violation.go
│   │   ├── report.go
│   │   ├── settings.go
│   │   └── webhook/
│   │       └── stripe.go
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go
│   │   ├── subscription.go
│   │   └── logging.go
│   ├── jobs/                    # Background job definitions
│   │   ├── analyze_inspection.go
│   │   ├── generate_report.go
│   │   └── send_email.go
│   ├── worker/                  # Job processing
│   └── telemetry/               # Prometheus + Sentry
├── migrations/                  # SQL migration files
├── sqlc/                        # sqlc query files
│   ├── queries/
│   │   ├── users.sql
│   │   ├── inspections.sql
│   │   ├── violations.sql
│   │   ├── regulations.sql
│   │   └── reports.sql
│   └── schema.sql
├── web/
│   ├── templates/               # Go HTML templates
│   │   ├── layouts/
│   │   ├── partials/
│   │   ├── pages/
│   │   └── email/
│   └── static/
│       ├── css/
│       ├── js/
│       └── images/
├── docs/                        # Documentation
├── scripts/                     # Dev and deployment scripts
├── sqlc.yaml
├── tailwind.config.js
├── docker-compose.yml
├── Dockerfile
└── Caddyfile
```

---

## Core Data Model

### Entity Relationship

```
┌─────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    User     │       │   Inspection    │       │     Image       │
├─────────────┤       ├─────────────────┤       ├─────────────────┤
│ id          │──────<│ id              │──────<│ id              │
│ email       │       │ user_id         │       │ inspection_id   │
│ password    │       │ site_id         │       │ storage_key     │
│ name        │       │ status          │       │ thumbnail_key   │
│ subscription│       │ weather         │       │ content_type    │
│ stripe_id   │       │ notes           │       │ analysis_status │
└─────────────┘       │ created_at      │       │ created_at      │
                      └─────────────────┘       └─────────────────┘
                              │                         │
                              │                         │
                              ▼                         │
                      ┌─────────────────┐               │
                      │    Violation    │               │
                      ├─────────────────┤               │
                      │ id              │<──────────────┘
                      │ inspection_id   │
                      │ image_id        │
                      │ description     │
                      │ ai_description  │
                      │ confidence      │
                      │ bounding_box    │
                      │ status          │  ┌─────────────────┐
                      │ severity        │  │   Regulation    │
                      │ inspector_notes │  ├─────────────────┤
                      │ created_at      │  │ id              │
                      └────────┬────────┘  │ standard_number │
                               │           │ title           │
                               │           │ category        │
                               ▼           │ full_text       │
                      ┌─────────────────┐  │ summary         │
                      │ violation_      │  └────────┬────────┘
                      │ regulations     │           │
                      ├─────────────────┤           │
                      │ violation_id    │───────────┘
                      │ regulation_id   │
                      │ relevance_score │
                      │ ai_explanation  │
                      └─────────────────┘

┌─────────────┐       ┌─────────────────┐
│    Site     │       │     Report      │
├─────────────┤       ├─────────────────┤
│ id          │       │ id              │
│ user_id     │       │ inspection_id   │
│ name        │       │ user_id         │
│ address     │       │ pdf_storage_key │
│ city        │       │ docx_storage_key│
│ state       │       │ violation_count │
│ zip         │       │ generated_at    │
│ client_name │       └─────────────────┘
│ client_email│
└─────────────┘
```

### Key Tables Schema

```sql
-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255),
    phone VARCHAR(50),
    stripe_customer_id VARCHAR(255),
    subscription_status VARCHAR(50) DEFAULT 'inactive',  -- inactive, trialing, active, past_due, canceled
    subscription_tier VARCHAR(50),  -- starter, professional
    subscription_id VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sites (reusable inspection locations)
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    client_name VARCHAR(255),
    client_email VARCHAR(255),
    client_phone VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Inspections
CREATE TABLE inspections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    site_id UUID REFERENCES sites(id),
    title VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',  -- draft, analyzing, review, completed
    inspection_date DATE NOT NULL DEFAULT CURRENT_DATE,
    weather_conditions VARCHAR(100),
    temperature VARCHAR(20),
    inspector_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Images
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspection_id UUID NOT NULL REFERENCES inspections(id) ON DELETE CASCADE,
    storage_key VARCHAR(500) NOT NULL,
    thumbnail_key VARCHAR(500),
    original_filename VARCHAR(255),
    content_type VARCHAR(100) NOT NULL,
    size_bytes INTEGER NOT NULL,
    width INTEGER,
    height INTEGER,
    analysis_status VARCHAR(50) DEFAULT 'pending',  -- pending, analyzing, completed, failed
    analysis_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Violations (potential issues identified)
CREATE TABLE violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspection_id UUID NOT NULL REFERENCES inspections(id) ON DELETE CASCADE,
    image_id UUID REFERENCES images(id),
    description TEXT NOT NULL,  -- Final description (may be edited by inspector)
    ai_description TEXT,  -- Original AI-generated description
    confidence VARCHAR(20),  -- high, medium, low
    bounding_box JSONB,  -- {x, y, width, height} in relative coordinates
    status VARCHAR(50) NOT NULL DEFAULT 'pending_review',  -- pending_review, confirmed, rejected, edited
    severity VARCHAR(50),  -- critical, serious, other, recommendation
    inspector_notes TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Violation-Regulation junction
CREATE TABLE violation_regulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    violation_id UUID NOT NULL REFERENCES violations(id) ON DELETE CASCADE,
    regulation_id UUID NOT NULL REFERENCES regulations(id),
    relevance_score DECIMAL(3,2),  -- 0.00 to 1.00
    ai_explanation TEXT,  -- Why this regulation applies
    is_primary BOOLEAN DEFAULT FALSE,  -- Main applicable regulation
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(violation_id, regulation_id)
);

-- Reports
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspection_id UUID NOT NULL REFERENCES inspections(id),
    user_id UUID NOT NULL REFERENCES users(id),
    pdf_storage_key VARCHAR(500),
    docx_storage_key VARCHAR(500),
    violation_count INTEGER NOT NULL DEFAULT 0,
    generated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sessions
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- AI Usage tracking
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    inspection_id UUID REFERENCES inspections(id),
    model VARCHAR(50) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    cost_cents INTEGER NOT NULL,
    request_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_inspections_user_id ON inspections(user_id);
CREATE INDEX idx_inspections_status ON inspections(status);
CREATE INDEX idx_images_inspection_id ON images(inspection_id);
CREATE INDEX idx_violations_inspection_id ON violations(inspection_id);
CREATE INDEX idx_violations_status ON violations(status);
CREATE INDEX idx_ai_usage_user_month ON ai_usage(user_id, date_trunc('month', created_at));
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

---

## MVP Scope

### Phase 1: Core Flow (Weeks 1-4)

**Authentication:**
- [ ] User registration with email verification
- [ ] Login/logout
- [ ] Password reset
- [ ] Basic session management

**Inspection Management:**
- [ ] Create inspection (title, date, site info)
- [ ] Upload images (multiple, drag-and-drop)
- [ ] Image storage in R2
- [ ] Thumbnail generation
- [ ] List inspections (paginated)
- [ ] View inspection detail

**AI Analysis:**
- [ ] Anthropic API integration
- [ ] Background job for image analysis
- [ ] Store potential violations with confidence
- [ ] Basic regulation matching

### Phase 2: Review & Reports (Weeks 5-8)

**Violation Review:**
- [ ] List AI-detected violations
- [ ] Accept/reject violations
- [ ] Edit violation descriptions
- [ ] Add manual violations
- [ ] View/change linked regulations
- [ ] Add inspector notes

**Report Generation:**
- [ ] PDF report generation
- [ ] DOCX report generation
- [ ] Download reports
- [ ] Email report to client

**Regulations:**
- [ ] Seed OSHA 1926 database
- [ ] Full-text search
- [ ] Browse by category

### Phase 3: Polish & Launch (Weeks 9-12)

**Billing:**
- [ ] Stripe subscription integration
- [ ] Starter and Professional tiers
- [ ] Usage tracking and limits
- [ ] Customer portal access

**Sites Management:**
- [ ] Create/edit reusable sites
- [ ] Client information storage
- [ ] Site history (past inspections)

**User Experience:**
- [ ] Dashboard with stats
- [ ] Quick actions
- [ ] Usage indicators
- [ ] Mobile-responsive polish

**Operations:**
- [ ] Prometheus metrics
- [ ] Sentry error tracking
- [ ] Logging and monitoring
- [ ] Backup strategy

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-15 | Go + stdlib router | Consistency with Hiri, simplicity |
| 2024-12-15 | PostgreSQL + sqlc | Consistency with Hiri, type safety |
| 2024-12-15 | Anthropic Claude for AI | Best vision capabilities, developer preference |
| 2024-12-15 | Cloudflare R2 for storage | Zero egress, S3-compatible, consistency with Hiri |
| 2024-12-15 | Local OSHA database | No official API, enables offline regulation lookup |
| 2024-12-15 | PDF + DOCX output | PDF for printing, DOCX for editing |
| 2024-12-15 | Single-tenant MVP | Reduce complexity for initial launch |
| 2024-12-15 | Web-responsive only | Native mobile deferred until market validation |

---

## Review Schedule

This document should be reviewed:
- Before each major phase begins
- When a significant technical challenge is encountered
- Every 3 months during active development