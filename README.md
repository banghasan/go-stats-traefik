# API Stats (Real-Time Traffic Statistics)

Simple, lightweight, real-time traffic statistics collector written in Go +
SQLite. Designed to be used as a **Traefik ForwardAuth Middleware**.

## Features

- **Lightweight**: Minimal memory footprint (Golang).
- **Embedded DB**: Uses SQLite, zero-configuration needed.
- **Middleware Mode**: Captures traffic via Traefik ForwardAuth (path `/`).
- **REST API**: Simple endpoints to view aggregated statistics.
  - `/api/`: Main statistics (grouped by Host).
  - `/api/:host`: Statistics for specific host.

---

## Installation & Usage

### 1. Build from Source

```bash
# Initialize module (if not already done)
go mod tidy

# Build binary to bin/ folder
go build -o bin/go-stats-traefik
```

### 2. Run Locally

```bash
./bin/go-stats-traefik
```

By default:

- Listens on `0.0.0.0:8080`
- Database: `./stats.db`

**Custom Flags:**

```bash
./bin/go-stats-traefik -host 127.0.0.1 -port 9000 -db /path/to/my.db -tz "Asia/Jakarta"
```

- `-tz`: Set timezone for logging (default: "UTC"). Example: "Asia/Jakarta"

**Check Version:**

```bash
./bin/go-stats-traefik --version
# Output: API Stats Version 2.0.0
```

### 3. Run with Docker

Pull the pre-built image from GitHub Container Registry:

```bash
docker pull ghcr.io/banghasan/go-stats-traefik:latest
```

Run with Docker:

```bash
docker run -d \
  --name go-stats-traefik \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  ghcr.io/banghasan/go-stats-traefik:latest
```

Or use Docker Compose (`docker-compose.yml` provided):

```bash
docker-compose up -d
```

---

## Development

### Building with Podman

This project prefers `podman` for local container builds:

```bash
podman build . -t go-stats-traefik
```

---

## Health Check

The application provides a health check endpoint for monitoring:

```bash
curl http://localhost:8080/health
```

**Response:**

```json
{
  "status": "healthy",
  "version": "2.0.0",
  "timestamp": 1704672000
}
```

This endpoint is useful for:

- Docker/Kubernetes health probes
- Load balancer health checks
- Monitoring systems (Prometheus, Datadog, etc.)

---

## API Reference

> **Note**: The application automatically extracts and stores only the **first
> path segment** (prefix) from incoming requests.

### Path Prefix Extraction Examples

| Original Path (X-Forwarded-Uri) | Stored As | Description                  |
| ------------------------------- | --------- | ---------------------------- |
| `/v3/cal/today`                 | `/v3`     | Extracts first segment       |
| `/v3/tools/ip`                  | `/v3`     | Same prefix, combined stats  |
| `/v2/quran/ayat/acak`           | `/v2`     | Different prefix             |
| `/v2/quran/surah/1`             | `/v2`     | Same prefix, combined stats  |
| `/api/users/123`                | `/api`    | Works with any path          |
| `/`                             | `/`       | Root path stored as-is       |
| `/hello`                        | `/hello`  | Single segment kept complete |

This aggregation allows you to track traffic by API version or major route
segments instead of individual endpoints.

---

## API Reference

- **GET /api/**: Returns stats for ALL hosts.
- **GET /api/:host**: Returns stats for a specific host (e.g., `/api.test.com`).

### Query Parameters

| Parameter | Default | Description                                                  |
| --------- | ------- | ------------------------------------------------------------ |
| `year`    | (All)   | Filter by specific year (e.g., `?year=2025`)                 |
| `prefix`  | (All)   | Filter by path prefix (e.g., `?prefix=/v1`)                  |
| `all`     | `0`     | If `1` or `true`, show ALL paths. Default shows Top 20 only. |

### Response Format

```json
[
  {
    "host": "api.test.com",
    "total": 1500,
    "data": [
      {
        "prefix": "/v1",
        "total": 500,
        "tahun": [2024, 2025]
      },
      ...
    ]
  }
]
```

---

## Traefik Integration (ForwardAuth)

This application works by intercepting requests via Traefik's `forwardAuth`
middleware. It logs the request metadata (headers) and immediately returns
`200 OK` so the request can proceed to your actual backend.

### Docker Compose Example

Add the `apistats` service and configure the middleware in your
`docker-compose.yml`.

```yaml
version: "3.8"

services:
  # 1. The Stats Service
  go-stats-traefik:
    image: ghcr.io/banghasan/go-stats-traefik:latest
    container_name: go-stats-traefik
    restart: unless-stopped
    volumes:
      - ./data:/data # Persist SQLite database
      - /usr/share/zoneinfo:/usr/share/zoneinfo:ro # Optional: Mount host timezone data
    command: [
      "-host",
      "0.0.0.0",
      "-port",
      "8080",
      "-db",
      "/data/stats.db",
      "-tz",
      "Asia/Jakarta",
    ]

  # 2. Your Application (The one you want to track)
  my-app:
    image: nginxdemos/hello
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.my-app.rule=Host(`myapp.localhost`)"

      # Define the Middleware
      - "traefik.http.middlewares.stats-logger.forwardauth.address=http://go-stats-traefik:8080/"
      # IMPORTANT: Ensure Traefik passes the URI
      - "traefik.http.middlewares.stats-logger.forwardauth.trustForwardHeader=true"

      # Apply the Middleware
      - "traefik.http.routers.my-app.middlewares=stats-logger"
```

### Using Traefik File Provider (rules.yml)

If you prefer using Traefik's file provider instead of Docker labels, create a
`rules.yml` file:

```yaml
http:
  middlewares:
    stats-logger:
      forwardAuth:
        address: "http://go-stats-traefik:8080/"
        trustForwardHeader: true

  routers:
    my-app:
      rule: "Host(`myapp.localhost`)"
      service: my-app-service
      middlewares:
        - stats-logger

  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://my-app:80"
```

Then reference this file in your Traefik configuration:

```yaml
# traefik.yml or docker-compose.yml
providers:
  file:
    filename: /etc/traefik/rules.yml
    watch: true
```

### How it works

1. **User** makes a request to `myapp.localhost/api/v1`.
2. **Traefik** hits the `stats-logger` middleware (our `apistats` service).
3. **API Stats** records the path (`X-Forwarded-Uri`) and timestamp to SQLite.
4. **API Stats** responds with `200 OK`.
5. **Traefik** allows the request to reach `my-app`.
