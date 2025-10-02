# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# The protobuf files are already copied with the source code

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -a -installsuffix cgo \
    -o file-service ./cmd/server

# Runtime stage
FROM alpine:3.19@sha256:13b7e62e8df80264dbb747995705a986aa530415763a6c58f84a3ca8af9a5bcd

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wget tzdata && \
    update-ca-certificates

# Create non-root user and group
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appuser /app/file-service .

# Copy timezone data
COPY --from=builder --chown=appuser:appuser /usr/share/zoneinfo /usr/share/zoneinfo

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8082 50052

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# Set environment variables
ENV TZ=UTC

# Note: Redis connection is configured via environment variables
# Set REDIS_ENABLED=true and REDIS_ADDR=redis:6379 to enable caching

# Run the application
CMD ["./file-service"]
