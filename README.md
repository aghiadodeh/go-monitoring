# go-monitoring

A high-performance API monitoring library for **Go** applications built with [Fiber](https://gofiber.io) and [GORM](https://gorm.io).

It captures every HTTP request/response hitting your application, stores them **asynchronously** in your database via a batched writer goroutine, and exposes analytics REST endpoints protected by JWT authentication.

---

## Features

- **Non-blocking request capture** — all DB writes happen in an isolated goroutine via a buffered channel + batch INSERT, so monitoring adds near-zero latency to your API responses.
- **Request logging** — method, URL, headers, params, query, body, response, duration, user, success/failure.
- **Job logging** — record background/cron job executions with metadata.
- **Analytics API** — paginated & filtered listing of request logs, per-endpoint duration stats, time-series bucketing.
- **JWT authentication** — login endpoint + guard middleware to protect the analytics API.
- **Database agnostic** — works with any GORM-supported database (PostgreSQL, MySQL, SQLite, SQL Server).
- **Configurable** — tune buffer size, batch size, flush interval, body capture limits, and more via environment variables or a config struct.

---

## Installation

```bash
go get github.com/aghiadodeh/go-monitoring
```

---

## Important Note for Developers

> **Global middlewares must skip the monitoring paths.**
>
> If your application registers global middleware (e.g. a response transformer, custom serializer, etc.), you **must** exclude `/monitoring` and `/api/monitoring` from it. Otherwise the middleware will interfere with the monitoring dashboard and API responses (e.g. wrapping HTML in JSON, altering headers, or breaking the SPA).

```go
app.Use(func(c *fiber.Ctx) error {
    path := c.Path()
    if strings.HasPrefix(path, "/api/monitoring") || strings.HasPrefix(path, "/monitoring") {
        return c.Next()
    }
    // your custom middleware logic
    return myCustomMiddleware(c)
})
```

Make sure these `c.Next()` skips are in place **before** calling `monitoring.Setup(app, db)`.

---

## Quick Start

```go
package main

import (
    "log"

    "github.com/gofiber/fiber/v2"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    monitoring "github.com/aghiadodeh/go-monitoring"
)

func main() {
    // 1. Connect to your database
    dsn := "host=localhost user=app password=secret dbname=myapp port=5432 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create the Fiber app
    app := fiber.New()

    // 3. Setup monitoring (uses env-var defaults)
    // Must be called BEFORE registering your routes.
    m := monitoring.Setup(app, db)

    // 4. Your routes
    app.Get("/", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"message": "hello"})
    })

    // 5. Log a background job (optional)
    m.LogJob("daily-cleanup", true, map[string]any{"deleted": 42})

    log.Fatal(app.Listen(":3000"))
}
```

---

## Database Tables

The library does **not** auto-create tables. You must create them yourself before starting the application.

### `monitoring_request_logs`

| Column             | Type               | Constraints    |
| ------------------ | ------------------ | -------------- |
| `id`               | `CHAR(36)`         | PRIMARY KEY    |
| `key`              | `VARCHAR(255)`     |                |
| `path`             | `VARCHAR(500)`     |                |
| `url`              | `VARCHAR(2048)`    |                |
| `method`           | `VARCHAR(10)`      |                |
| `user`             | `JSON` / `JSONB`   |                |
| `request`          | `JSON` / `JSONB`   |                |
| `response`         | `JSON` / `JSONB`   |                |
| `response_headers` | `JSON` / `JSONB`   |                |
| `success`          | `BOOLEAN`          | DEFAULT `true` |
| `duration`         | `DOUBLE PRECISION` |                |
| `created_at`       | `TIMESTAMP`        | INDEX          |
| `updated_at`       | `TIMESTAMP`        |                |

### `monitoring_job_logs`

| Column       | Type             | Constraints    |
| ------------ | ---------------- | -------------- |
| `id`         | `CHAR(36)`       | PRIMARY KEY    |
| `name`       | `VARCHAR(255)`   | NOT NULL       |
| `success`    | `BOOLEAN`        | DEFAULT `true` |
| `metadata`   | `JSON` / `JSONB` | NOT NULL       |
| `created_at` | `TIMESTAMP`      | INDEX          |
| `updated_at` | `TIMESTAMP`      |                |

#### PostgreSQL migration example

```sql
CREATE TABLE monitoring_request_logs (
    id               CHAR(36) PRIMARY KEY,
    key              VARCHAR(255),
    path             VARCHAR(500),
    url              VARCHAR(2048),
    method           VARCHAR(10),
    "user"           JSONB,
    request          JSONB,
    response         JSONB,
    response_headers JSONB,
    success          BOOLEAN DEFAULT TRUE,
    duration         DOUBLE PRECISION,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_logs_created_at ON monitoring_request_logs (created_at);

CREATE TABLE monitoring_job_logs (
    id         CHAR(36) PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    success    BOOLEAN DEFAULT TRUE,
    metadata   JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_logs_created_at ON monitoring_job_logs (created_at);
```

#### MySQL migration example

```sql
CREATE TABLE monitoring_request_logs (
    id               CHAR(36) PRIMARY KEY,
    `key`            VARCHAR(255),
    path             VARCHAR(500),
    url              VARCHAR(2048),
    method           VARCHAR(10),
    user             JSON,
    request          JSON,
    response         JSON,
    response_headers JSON,
    success          BOOLEAN DEFAULT TRUE,
    duration         DOUBLE,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_request_logs_created_at (created_at)
);

CREATE TABLE monitoring_job_logs (
    id         CHAR(36) PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    success    BOOLEAN DEFAULT TRUE,
    metadata   JSON NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_job_logs_created_at (created_at)
);
```

---

## Configuration

All settings can be controlled via **environment variables** or by passing a `*monitoring.Config` struct to `Setup()`.

| Environment Variable              | Default   | Description                            |
| --------------------------------- | --------- | -------------------------------------- |
| `MONITORING_REQUEST_SAVE_ENABLED` | `true`    | Enable/disable request logging         |
| `MONITORING_DASHBOARD_ENABLED`    | `true`    | Serve the static frontend dashboard    |
| `MONITORING_DASHBOARD_PATH`       | _(empty)_ | Filesystem path to the dashboard build |
| `MONITORING_AUTH_REQUIRED`        | `false`   | Require JWT for analytics API          |
| `MONITORING_APIS_ENABLED`         | `true`    | Enable analytics API endpoints         |
| `MONITORING_USERNAME`             | `admin`   | Dashboard login username               |
| `MONITORING_PASSWORD`             | `admin`   | Dashboard login password               |
| `MONITORING_JWT_SECRET`           | _(empty)_ | JWT signing secret                     |
| `MONITORING_BUFFER_SIZE`          | `10000`   | Log writer channel buffer capacity     |
| `MONITORING_BATCH_SIZE`           | `100`     | Records per batch INSERT               |
| `MONITORING_FLUSH_INTERVAL_MS`    | `5000`    | Max ms between flushes                 |
| `MONITORING_WORKERS`              | `1`       | Number of writer goroutines            |

### Programmatic configuration

```go
cfg := &monitoring.Config{
    RequestSaveEnabled: true,
    AuthRequired:       true,
    JWTSecret:          "my-secret",
    BufferSize:         20000,
    BatchSize:          200,
    Workers:            2,
    MaxBodySize:        128 * 1024, // 128KB
    SkipPaths:          []string{"/api/monitoring", "/health"},
    UserContextKey:     "user",
    CaptureReqBody:     true,
    CaptureRespBody:    true,
}

m := monitoring.Setup(app, db, cfg)
```

---

## API Endpoints

All endpoints are prefixed with `/api/monitoring` and optionally protected by JWT.

### Authentication

| Method | Path                                   | Description        |
| ------ | -------------------------------------- | ------------------ |
| POST   | `/api/monitoring/authentication/login` | Login, returns JWT |

**Request body:**

```json
{ "username": "admin", "password": "admin" }
```

**Response:**

```json
{ "token": "eyJhbGci...", "success": true }
```

### Request Logs

| Method | Path                                | Description                              |
| ------ | ----------------------------------- | ---------------------------------------- |
| GET    | `/api/monitoring/requests`          | List request logs (paginated + filtered) |
| GET    | `/api/monitoring/requests/analyze`  | Request analytics & charts data          |
| GET    | `/api/monitoring/requests/view/:id` | View a single request log                |

**Query parameters for `/requests`:**

`page`, `per_page`, `fromDate`, `toDate`, `sortKey`, `url`, `method`, `exception`, `success`, `durationGt`, `durationLt`, `statusCode`

### Job Logs

| Method | Path                       | Description                          |
| ------ | -------------------------- | ------------------------------------ |
| GET    | `/api/monitoring/jobs`     | List job logs (paginated + filtered) |
| GET    | `/api/monitoring/jobs/:id` | View a single job log                |

**Query parameters for `/jobs`:**

`page`, `per_page`, `fromDate`, `toDate`, `sortKey`, `name`, `success`

### Utilities

| Method | Path                    | Description                |
| ------ | ----------------------- | -------------------------- |
| DELETE | `/api/monitoring/clear` | Delete all monitoring data |

---

## Architecture — Performance Design

```
HTTP Request
     │
     ▼
┌──────────────────────┐
│  Fiber Middleware     │  ← captures req/res data synchronously
│  (near-zero overhead)│
└──────────┬───────────┘
           │ non-blocking channel send
           ▼
┌──────────────────────┐
│  Buffered Channel    │  capacity: 10,000 (configurable)
│  (back-pressure)     │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Writer Goroutine(s) │  ← isolated from request handling
│  batch INSERT every  │
│  100 records or 5s   │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Database            │  PostgreSQL / MySQL / SQLite
└──────────────────────┘
```

- The middleware **never** performs a DB write directly.
- Log entries are sent to a buffered channel (non-blocking; if full, the entry is dropped to protect latency).
- A background goroutine collects entries and flushes them in **batch INSERTs** (single multi-row INSERT statement), dramatically reducing DB round-trips.
- On application shutdown, all remaining entries are flushed automatically via Fiber's `OnShutdown` hook — no manual `m.Shutdown()` call is needed.

---

## License

ISC
