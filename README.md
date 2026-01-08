# API Stats (Real-Time Traffic Statistics)

Simple, lightweight, real-time traffic statistics collector written in Go +
SQLite. Designed to be used as a **Traefik ForwardAuth Middleware**.

## Features

- **Lightweight**: Minimal memory footprint (Golang).
- **Embedded DB**: Uses SQLite, zero-configuration needed.
- **Middleware Mode**: Captures traffic via Traefik ForwardAuth.
- **REST API**: Simple endpoints to view aggregated statistics.
- **Async Logging**: Uses buffered channels to ensure it never blocks your main
  traffic.

---

## Installation & Usage

### 1. Build from Source

```bash
# Initialize module (if not already done)
go mod tidy

# Build binary to bin/ folder
go build -o bin/apistats
```

### 2. Run Locally

```bash
./bin/apistats
```

By default:

- Listens on `0.0.0.0:8080`
- Database: `./stats.db`

**Custom Flags:**

```bash
./apistats -host 127.0.0.1 -port 9000 -db /path/to/my.db
```

**Check Version:**

```bash
./bin/apistats --version
# Output: API Stats Version 1.0
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
    "version": "1.0",
    "timestamp": 1704672000
}
```

This endpoint is useful for:

- Docker/Kubernetes health probes
- Load balancer health checks
- Monitoring systems (Prometheus, Datadog, etc.)

---

## API Reference

### Check Statistics

#### 1. Get All Stats

Returns a list of all paths and their yearly summaries.

```bash
curl http://localhost:8080/api/stats
```

**Response:**

```json
{
    "data": [
        {
            "pathprefix": "/api/v1/users",
            "years": [
                { "year": 2026, "total": 1500, "avg": 125 }
            ]
        }
    ]
}
```

#### 2. Get Yearly Details

Returns detailed monthly statistics for a specific year.

```bash
curl http://localhost:8080/api/stats/2026
```

**Response:**

```json
{
    "data": [
        {
            "pathprefix": "/api/v1/users",
            "year": 2026,
            "total": 1500,
            "avg": 125,
            "months": [
                { "month": 1, "total": 500, "avg": 16.1 },
                { "month": 2, "total": 1000, "avg": 35.7 }
            ]
        }
    ]
}
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
    apistats:
        build: .
        image: my-apistats:latest
        container_name: apistats
        restart: unless-stopped
        volumes:
            - ./data:/data # Persist SQLite database
        command: /app/apistats -db /data/stats.db

    # 2. Your Application (The one you want to track)
    my-app:
        image: nginxdemos/hello
        labels:
            - "traefik.enable=true"
            - "traefik.http.routers.my-app.rule=Host(`myapp.localhost`)"

            # Define the Middleware
            - "traefik.http.middlewares.stats-logger.forwardauth.address=http://apistats:8080"
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
                address: "http://apistats:8080"
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
