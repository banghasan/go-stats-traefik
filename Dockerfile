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
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
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

# Create non-root user
RUN addgroup -S -g 65532 nonroot && \
    adduser -S -u 65532 -G nonroot nonroot

# Change ownership of the binary
RUN chown nonroot:nonroot /app/go-stats-traefik

# Ensure data directory exists with proper permissions
RUN mkdir -p /data && \
    chown -R nonroot:nonroot /data

# Switch to non-root user
USER nonroot:nonroot

# Command
ENTRYPOINT ["/app/go-stats-traefik"]
CMD ["-host", "0.0.0.0", "-port", "8080", "-db", "/data/stats.db"]
