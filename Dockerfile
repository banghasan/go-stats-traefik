# Stage 1: Build
FROM golang:alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build binary with optimization flags
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -a \
    -installsuffix cgo \
    -o go-stats-traefik \
    .

# Stage 2: Runtime (distroless)
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy ca-certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /build/go-stats-traefik /app/go-stats-traefik

# Expose default port
EXPOSE 8080

# Run as non-root user (distroless uses UID 65532)
USER nonroot:nonroot

# Command
ENTRYPOINT ["/app/go-stats-traefik"]
CMD ["-host", "0.0.0.0", "-port", "8080", "-db", "/data/stats.db"]
