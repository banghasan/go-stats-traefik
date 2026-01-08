# API Documentation - Go Stats Traefik

## Base URL
```
http://localhost:8080/
```

## Authentication
No authentication required.

## HTTP Status Codes
- `200`: Success
- `400`: Bad Request
- `404`: Not Found
- `500`: Internal Server Error

---

## Endpoints

### 1. Health Check
#### GET `/health`
Check if the service is running and healthy.

**Response:**
```
{
  "status": "healthy",
  "version": "1.0.1",
  "timestamp": 1767841066
}
```

### 2. Summary Statistics
#### GET `/api/stats`
Get summary statistics for all path prefixes including total hits across all years.

**Response:**
```
{
  "total": 13,
  "data": [
    {
      "pathprefix": "/unknown",
      "total": 1
    },
    {
      "pathprefix": "/v1",
      "total": 5
    },
    {
      "pathprefix": "/v2",
      "total": 6
    }
  ]
}
```

### 3. Path and Year Combinations
#### GET `/api/stats/data`
Get all unique path prefixes and available years in the database.

**Response:**
```
[
  {
    "pathprefix": "/",
    "years": [2025, 2026]
  },
  {
    "pathprefix": "/v1",
    "years": [2026]
  },
  {
    "pathprefix": "/v2",
    "years": [2025, 2026, 2027]
  }
]
```

### 4. Yearly Details
#### GET `/api/stats/{year}`
Get detailed monthly statistics for a specific year.

**Response:**
```
{
  "data": [
    {
      "pathprefix": "/v1",
      "year": 2026,
      "total": 1500,
      "avg": 125,
      "months": [
        {
          "month": 1,
          "total": 500,
          "avg": 16.1
        },
        {
          "month": 2,
          "total": 1000,
          "avg": 35.7
        }
      ]
    }
  ]
}
```

### 5. Path-Specific Data
#### GET `/api/stats/data/{pathprefix}?year={year}`
Get monthly statistics for a specific path prefix. The `year` parameter is optional and defaults to the current year.

**Parameters:**
- `year` (optional): The year to retrieve data for (e.g., `2026`)

**Response:**
```
{
  "data": [
    {
      "pathprefix": "/v2",
      "year": 2026,
      "total": 1,
      "avg": 1,
      "months": [
        {
          "month": 1,
          "total": 1,
          "avg": 1
        }
      ]
    }
  ]
}
```

---

## Middleware Endpoint

### 6. ForwardAuth Middleware
#### GET `/`
This endpoint is used as a Traefik ForwardAuth middleware. It captures traffic statistics and allows the request to proceed by returning a 200 status.

**Headers Used:**
- `X-Forwarded-Uri`: The original path of the request
- `X-Replaced-Path`: Alternative header for the original path

---

## Usage Examples with curlie

### Health Check
```curlie
GET /health
```

### Summary Statistics
```curlie
GET /api/stats
```

### Path and Year Combinations
```curlie
GET /api/stats/data
```

### Yearly Details for 2026
```curlie
GET /api/stats/2026
```

### Path-Specific Data for /v1
```curlie
GET /api/stats/data/v1
```

### Path-Specific Data for /v2 in Year 2026
```curlie
GET /api/stats/data/v2?year=2026
```

### Using as Middleware
```curlie
GET /
Headers:
  X-Forwarded-Uri: /api/v1/test
```