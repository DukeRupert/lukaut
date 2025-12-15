# Lukaut

AI-powered safety inspection reports for construction sites. You inspect. We'll handle the rest.

## About

Lukaut is a SaaS platform that transforms construction safety inspections. AI analyzes site photos to identify potential OSHA violations, suggests applicable regulations, and generates print-ready reports—cutting report time from 1-2 hours to 15-30 minutes.

## Features

- **AI Image Analysis** — Upload site photos and get AI-identified potential safety violations
- **Regulation Matching** — Automatic suggestions of applicable OSHA standards (29 CFR 1926)
- **Violation Review** — Accept, reject, or edit AI findings with full inspector control
- **Report Generation** — One-click PDF and DOCX reports ready for print or email
- **Site Management** — Save client and location details for recurring inspections

## Tech Stack

- **Backend:** Go (stdlib router)
- **Database:** PostgreSQL with sqlc
- **Frontend:** Server-rendered HTML, htmx, Alpine.js, Tailwind CSS
- **AI:** Anthropic Claude API
- **Storage:** Cloudflare R2 (S3-compatible)
- **Payments:** Stripe

## Development

### Prerequisites

- Go 1.22+
- PostgreSQL 16
- Node.js (for Tailwind CLI)
- Docker (optional)

### Setup

```bash
# Clone the repository
git clone https://github.com/DukeRupert/lukaut.git
cd lukaut

# Copy environment template
cp .env.example .env

# Install dependencies
go mod download

# Run database migrations
go run cmd/migrate/main.go up

# Start development server
go run cmd/server/main.go
```

### Environment Variables

```bash
DATABASE_URL=postgres://lukaut:password@localhost:5432/lukaut?sslmode=disable
ANTHROPIC_API_KEY=your_api_key
STRIPE_SECRET_KEY=your_stripe_key
STRIPE_WEBHOOK_SECRET=your_webhook_secret
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_access_key
R2_SECRET_ACCESS_KEY=your_secret_key
R2_BUCKET_NAME=lukaut-files
POSTMARK_API_TOKEN=your_postmark_token
```

## Project Structure

```
lukaut/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── ai/              # AI provider abstraction
│   ├── billing/         # Stripe integration
│   ├── config/          # Configuration
│   ├── domain/          # Core business types
│   ├── email/           # Email service
│   ├── handler/         # HTTP handlers
│   ├── jobs/            # Background jobs
│   ├── middleware/      # HTTP middleware
│   ├── repository/      # Database queries (sqlc)
│   ├── report/          # PDF/DOCX generation
│   ├── service/         # Business logic
│   ├── storage/         # File storage
│   └── worker/          # Job processing
├── migrations/          # SQL migrations
├── sqlc/                # sqlc queries
└── web/
    ├── static/          # CSS, JS, images
    └── templates/       # Go HTML templates
```

## Documentation

- [Business Plan](docs/BUSINESS_PLAN.md)
- [Brand Guidelines](docs/BRAND_GUIDELINES.md)
- [Technical Architecture](docs/TECHNICAL_ARCHITECTURE.md)

## License

All rights reserved. This software is proprietary and confidential. Unauthorized copying, distribution, or use of this software is strictly prohibited.

Copyright 2025 Lukaut.