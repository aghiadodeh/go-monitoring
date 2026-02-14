// Package monitoring provides a drop-in API monitoring library for Go Fiber + GORM applications.
//
// It captures every HTTP request/response, stores them asynchronously in a database via a
// high-performance batched log writer, and exposes analytics API endpoints protected by JWT.
//
// Usage:
//
//	m := monitoring.Setup(app, db)          // uses env-var defaults
//	m := monitoring.Setup(app, db, &cfg)    // custom config
//	defer m.Shutdown()
//
//	m.LogJob("send-emails", true, map[string]any{"count": 42})
package monitoring

import (
	"github.com/aghiadodeh/go-monitoring/auth"
	"github.com/aghiadodeh/go-monitoring/handlers"
	"github.com/aghiadodeh/go-monitoring/logwriter"
	"github.com/aghiadodeh/go-monitoring/middleware"
	"github.com/aghiadodeh/go-monitoring/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Monitor is the main handle returned by Setup.
// Use it to log jobs and to shut down gracefully.
type Monitor struct {
	config     *Config
	writer     *logwriter.Writer
	jobService *services.JobService
}

// Setup initializes the monitoring system:
//   - registers the request-capture middleware (async, non-blocking)
//   - registers the analytics API routes under /api/monitoring
//   - optionally serves the frontend dashboard
//
// Pass nil for cfg to use DefaultConfig() (reads from env vars).
func Setup(app *fiber.App, db *gorm.DB, cfg ...*Config) *Monitor {
	var c *Config
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	} else {
		c = DefaultConfig()
	}

	// ---- async log writer ----
	w := logwriter.New(db, logwriter.Options{
		BufferSize:    c.BufferSize,
		BatchSize:     c.BatchSize,
		FlushInterval: c.FlushInterval,
		Workers:       c.Workers,
	})

	// ---- request monitoring middleware (applied globally) ----
	if c.RequestSaveEnabled {
		app.Use(middleware.New(middleware.MiddlewareConfig{
			Writer:          w,
			SkipPaths:       c.SkipPaths,
			UserContextKey:  c.UserContextKey,
			MaxBodySize:     c.MaxBodySize,
			CaptureReqBody:  c.CaptureReqBody,
			CaptureRespBody: c.CaptureRespBody,
		}))
	}

	// ---- services ----
	reqService := &services.RequestService{DB: db}
	jobService := &services.JobService{DB: db}

	// ---- handlers ----
	reqHandler := &handlers.RequestHandler{Service: reqService}
	jobHandler := &handlers.JobHandler{Service: jobService}

	// ---- routes ----
	api := app.Group("/api/monitoring")

	// Public: authentication
	api.Post("/authentication/login", auth.LoginHandler(c.Username, c.Password, c.JWTSecret))

	// Protected: analytics
	protected := api.Group("", auth.Guard(c.AuthRequired, c.APIsEnabled, c.JWTSecret))

	// Request logs
	protected.Get("/requests", reqHandler.FindAll)
	protected.Get("/requests/analyze", reqHandler.Analyze)
	protected.Get("/requests/view/:id", reqHandler.FindByID)

	// Job logs
	protected.Get("/jobs", jobHandler.FindAll)
	protected.Get("/jobs/:id", jobHandler.FindByID)

	// Clear all
	protected.Delete("/clear", jobHandler.ClearAll)

	// ---- optional static dashboard ----
	if c.DashboardEnabled && c.DashboardPath != "" {
		app.Static("/monitoring", c.DashboardPath)
	}

	return &Monitor{
		config:     c,
		writer:     w,
		jobService: jobService,
	}
}

// LogJob records a background / cron job execution.
func (m *Monitor) LogJob(name string, success bool, metadata interface{}) error {
	return m.jobService.Create(name, success, metadata)
}

// ClearAll deletes all monitoring data from the database.
func (m *Monitor) ClearAll() error {
	return m.jobService.ClearAll()
}

// Shutdown flushes all pending log entries and stops background workers.
// Call this when your application is shutting down.
func (m *Monitor) Shutdown() {
	m.writer.Shutdown()
}
