# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lukaut is an AI-powered SaaS platform for construction safety inspectors. Inspectors upload site photos, AI analyzes them for potential OSHA violations, suggests applicable regulations, and generates print-ready reports (PDF/DOCX).

## Tech Stack

- **Backend:** Go 1.22+ with stdlib router (no framework)
- **Database:** PostgreSQL 16 with sqlc for type-safe queries and goose for migrations
- **Frontend:** Server-rendered HTML templates, htmx, Alpine.js, Tailwind CSS
- **AI:** Anthropic Claude API for image analysis and regulation matching
- **Storage:** Cloudflare R2 (S3-compatible)
- **Payments:** Stripe subscriptions
- **Email:** Postmark

## Development Commands

```bash
# Install dependencies
go mod download

# Run database migrations
go run cmd/migrate/main.go up

# Start development server
go run cmd/server/main.go

# Generate sqlc code (after modifying .sql query files)
sqlc generate

# Build Tailwind CSS (watch mode)
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/output.css --watch
```

## Architecture

### Directory Structure

- `cmd/server/` - Application entry point
- `internal/` - All application code (not importable externally)
  - `ai/` - AI provider abstraction with Anthropic implementation
  - `billing/` - Stripe integration
  - `config/` - Configuration loading
  - `domain/` - Core business types (User, Inspection, Violation, etc.)
  - `email/` - Email service (Postmark)
  - `handler/` - HTTP handlers
  - `jobs/` - Background job definitions
  - `middleware/` - HTTP middleware (auth, subscription checks)
  - `repository/` - sqlc-generated database queries
  - `report/` - PDF and DOCX generation
  - `service/` - Business logic layer
  - `storage/` - File storage abstraction (local dev, R2 prod)
  - `worker/` - Job processing
- `migrations/` - SQL migration files (goose format)
- `sqlc/` - sqlc query definitions
- `web/templates/` - Go HTML templates
- `web/static/` - CSS, JS, images

### Key Patterns

- **Stdlib router:** Go 1.22+ ServeMux with method matching and path parameters
- **Cookie-based sessions:** PostgreSQL-backed, SHA-256 hashed tokens
- **Background jobs:** Database-backed queue with PostgreSQL SKIP LOCKED
- **Provider abstraction:** Interfaces for AI, storage, email allow swapping implementations
- **OSHA regulations:** Local PostgreSQL database with full-text search (no external API)

### Core Workflow

1. Inspector uploads site photos to inspection
2. Background job sends images to Claude API for analysis
3. AI identifies potential violations and suggests OSHA regulations
4. Inspector reviews, accepts/rejects, edits findings
5. One-click report generation (PDF/DOCX)

## Environment Variables

Required for development:
```
DATABASE_URL=postgres://lukaut:password@localhost:5432/lukaut?sslmode=disable
ANTHROPIC_API_KEY=
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET_NAME=lukaut-files
POSTMARK_API_TOKEN=
```

## Brand Colors (Tailwind)

The brand uses a construction-safety inspired color palette:
- `navy` (#1E3A5F) - Primary brand color, headers, nav
- `safety-orange` (#FF6B35) - Accent, CTAs, highlights
- Slate gray tones for text and borders
- Clean white/soft gray for backgrounds
