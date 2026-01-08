# Stage 1: Build
FROM docker.io/library/golang:alpine AS builder

WORKDIR /build

# Install build dependencies (including GCC for CGO)
RUN apk add --no-cache git ca-certificates tzdata build-base musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build binary with CGO enabled (required for SQLite support)
ARG BUILD_VERSION=1.0.0
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-X 'main.AppVersion=${BUILD_VERSION}' -w -s" \
    -a \
    -installsuffix cgo \
    -o go-stats-traefik \
    .

# Stage 2: Runtime (alpine for better compatibility)
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary
COPY --from=builder /build/go-stats-traefik /app/go-stats-traefik

# Make binary executable
RUN chmod +x /app/go-stats-traefik

# Expose default port
EXPOSE 8080

# Create data directory with proper permissions
RUN mkdir -p /data && \
    chmod 755 /data

# Keep running as root to avoid permission issues
# Command
ENTRYPOINT ["/app/go-stats-traefik"]
CMD ["-host", "0.0.0.0", "-port", "8080", "-db", "/data/stats.db"]
