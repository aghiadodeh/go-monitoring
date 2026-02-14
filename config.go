package monitoring

import (
	"os"
	"strconv"
	"time"
)

// Config holds all monitoring configuration loaded from environment variables.
type Config struct {
	// Request logging
	RequestSaveEnabled bool

	// Dashboard
	DashboardEnabled bool
	DashboardPath    string // filesystem path to the browser build

	// Authentication
	AuthRequired bool
	APIsEnabled  bool
	Username     string
	Password     string
	JWTSecret    string

	// Log writer performance tuning
	BufferSize    int           // channel buffer size (default: 10000)
	BatchSize     int           // records per batch insert (default: 100)
	FlushInterval time.Duration // max time between flushes (default: 5s)
	Workers       int           // number of writer goroutines (default: 1)

	// Middleware options
	SkipPaths       []string // URL prefixes to skip logging (default: ["/api/monitoring"])
	UserContextKey  string   // key for user data in c.Locals() (default: "user")
	MaxBodySize     int      // max request/response body bytes to capture (default: 64KB, -1 = unlimited)
	CaptureReqBody  bool     // capture request body (default: true)
	CaptureRespBody bool     // capture response body (default: true)
}

// DefaultConfig returns a Config populated from environment variables with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		RequestSaveEnabled: envBool("MONITORING_REQUEST_SAVE_ENABLED", true),
		DashboardEnabled:   envBool("MONITORING_DASHBOARD_ENABLED", true),
		DashboardPath:      envStr("MONITORING_DASHBOARD_PATH", "./vendor/github.com/aghiadodeh/go-monitoring/browser"),
		AuthRequired:       envBool("MONITORING_AUTH_REQUIRED", false),
		APIsEnabled:        envBool("MONITORING_APIS_ENABLED", true),
		Username:           envStr("MONITORING_USERNAME", "admin"),
		Password:           envStr("MONITORING_PASSWORD", "admin"),
		JWTSecret:          envStr("MONITORING_JWT_SECRET", "monitoring-secret-change-me"),

		BufferSize:    envInt("MONITORING_BUFFER_SIZE", 10000),
		BatchSize:     envInt("MONITORING_BATCH_SIZE", 100),
		FlushInterval: time.Duration(envInt("MONITORING_FLUSH_INTERVAL_MS", 5000)) * time.Millisecond,
		Workers:       envInt("MONITORING_WORKERS", 1),

		SkipPaths:       []string{"/api/monitoring", "/monitoring", "/.well-known"},
		UserContextKey:  "user",
		MaxBodySize:     64 * 1024, // 64KB
		CaptureReqBody:  true,
		CaptureRespBody: true,
	}
}

// --- helpers ---

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envStr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
