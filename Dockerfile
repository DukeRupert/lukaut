# ==============================================================================
# STAGE 1: Build Tailwind CSS
# ==============================================================================
FROM node:20-alpine AS tailwind-builder

WORKDIR /build

# Copy package files and install dependencies
COPY package.json package-lock.json ./
RUN npm ci --production=false

# Copy Tailwind config and source files
COPY tailwind.config.js ./
COPY web/static/css/ ./web/static/css/
COPY web/templates/ ./web/templates/

# Build production CSS
RUN npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/output.css --minify

# ==============================================================================
# STAGE 2: Build Go binary
# ==============================================================================
FROM golang:1.23-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Enable GOTOOLCHAIN=auto to download required Go version
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Copy built CSS from tailwind stage
COPY --from=tailwind-builder /build/web/static/css/output.css ./web/static/css/output.css

# Build the binary with optimizations
# CGO_ENABLED=0 for static binary
# -ldflags to strip debug info and reduce size
# -trimpath to remove file system paths from binary
ARG VERSION=dev
ARG COMMIT=unknown
RUN GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -trimpath \
    -o /build/server \
    ./cmd/server

# ==============================================================================
# STAGE 3: Production runtime
# ==============================================================================
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 lukaut && \
    adduser -u 1000 -G lukaut -s /bin/sh -D lukaut

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/server ./server

# Copy web assets (templates are embedded, but static files are served from filesystem)
COPY --from=go-builder /build/web/templates ./web/templates
COPY --from=go-builder /build/web/static ./web/static

# Create storage directory for local development mode
RUN mkdir -p /app/storage && chown -R lukaut:lukaut /app

# Switch to non-root user
USER lukaut

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the server
ENTRYPOINT ["./server"]
