.PHONY: run build dev deps sqlc migrate-up migrate-down migrate-create docker-up docker-down

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

# Generate sqlc code
sqlc:
	@sqlc generate

# Database migrations
migrate-up:
	@goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-create:
	@goose -dir migrations create $(name) sql

# Docker
docker-up:
	@docker compose up -d

docker-down:
	@docker compose down
