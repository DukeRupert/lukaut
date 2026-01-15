.PHONY: run build dev deps sqlc templ templ\:watch migrate\:up migrate\:down migrate\:create \
        docker\:up docker\:down docker\:build docker\:push docker\:logs \
        test test\:v test\:cover css css\:watch

# ==============================================================================
# Development Commands
# ==============================================================================

# Run the server
run:
	@go run cmd/server/main.go

# Build the server binary
build:
	@go build -o server cmd/server/main.go

# Run with hot reload (requires air)
dev:
	@air

# Install dev dependencies
deps:
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/air-verse/air@latest
	@go install github.com/a-h/templ/cmd/templ@latest
	@go install github.com/templui/templui/cmd/templui@latest
	@npm install

# Generate sqlc code
sqlc:
	@sqlc generate

# Generate templ code
templ:
	@templ generate

# Watch templ files for changes
templ\:watch:
	@templ generate --watch

# ==============================================================================
# Database Commands
# ==============================================================================

migrate\:up:
	@goose -dir internal/migrations postgres "$(DATABASE_URL)" up

migrate\:down:
	@goose -dir internal/migrations postgres "$(DATABASE_URL)" down

migrate\:create:
	@goose -dir internal/migrations create $(name) sql

# ==============================================================================
# Docker Commands (Local Development)
# ==============================================================================

# Start development services (db, mailhog, minio)
docker\:up:
	@docker compose up -d

# Stop development services
docker\:down:
	@docker compose down

# Stop and remove volumes
docker\:clean:
	@docker compose down -v

# View logs
docker\:logs:
	@docker compose logs -f

# Build Docker image locally
docker\:build:
	@docker build -t lukaut:local .

# Build and run locally with Docker
docker\:run: docker\:build
	@docker run --rm -it \
		--network lukaut_lukaut-network \
		-p 8080:8080 \
		-e DATABASE_URL=postgres://lukaut:lukaut@lukaut-db:5432/lukaut?sslmode=disable \
		-e SMTP_HOST=lukaut-mailhog \
		-e SMTP_PORT=1025 \
		-e AI_PROVIDER=mock \
		-e STORAGE_PROVIDER=local \
		lukaut:local

# ==============================================================================
# Production Docker Commands
# ==============================================================================

# Build and push to GitHub Container Registry
docker\:push:
	@docker build -t ghcr.io/$(GITHUB_USER)/lukaut:latest \
		--build-arg VERSION=$$(git describe --tags --always) \
		--build-arg COMMIT=$$(git rev-parse HEAD) \
		.
	@docker push ghcr.io/$(GITHUB_USER)/lukaut:latest

# Deploy to production (pulls latest and restarts)
deploy:
	@ssh $(VPS_USER)@$(VPS_HOST) "cd $(VPS_DEPLOY_PATH) && \
		docker compose -f docker-compose.prod.yml pull && \
		docker compose -f docker-compose.prod.yml up -d"

# ==============================================================================
# Testing
# ==============================================================================

test:
	@go test ./cmd/... ./internal/...

test\:v:
	@go test -v ./cmd/... ./internal/...

test\:cover:
	@go test -coverprofile=coverage.out ./cmd/... ./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ==============================================================================
# CSS (Tailwind)
# ==============================================================================

css:
	@npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/output.css --minify

css\:watch:
	@npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/output.css --watch

# ==============================================================================
# Utilities
# ==============================================================================

# Format code
fmt:
	@go fmt ./...

# Run linter
lint:
	@golangci-lint run

# Clean build artifacts
clean:
	@rm -f server coverage.out coverage.html
	@rm -rf tmp/
