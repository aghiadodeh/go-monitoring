package middleware

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/aghiadodeh/go-monitoring/logwriter"
	"github.com/aghiadodeh/go-monitoring/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

// MiddlewareConfig holds options for the monitoring middleware.
type MiddlewareConfig struct {
	Writer          *logwriter.Writer
	SkipPaths       []string // URL prefixes to skip (default: "/api/monitoring")
	UserContextKey  string   // c.Locals key for user (default: "user")
	MaxBodySize     int      // max body bytes to capture (-1 = unlimited, default: 64KB)
	CaptureReqBody  bool
	CaptureRespBody bool
}

// uuidRe matches standard UUIDs (v4 and similar).
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// numericRe matches purely numeric path segments.
var numericRe = regexp.MustCompile(`^\d+$`)

// New returns a Fiber middleware that captures request/response data
// and sends it to the async log writer. All heavy work (DB write) runs
// in an isolated goroutine via the Writer — the HTTP response is never
// blocked by monitoring.
func New(cfg MiddlewareConfig) fiber.Handler {
	if cfg.UserContextKey == "" {
		cfg.UserContextKey = "user"
	}
	if cfg.MaxBodySize == 0 {
		cfg.MaxBodySize = 64 * 1024
	}

	return func(c *fiber.Ctx) error {
		// Check if this path should be skipped.
		path := c.Path()
		for _, sp := range cfg.SkipPaths {
			if strings.HasPrefix(path, sp) {
				return c.Next()
			}
		}

		// --- Capture request data (synchronous – before handler) ---
		reqIP := c.IP()
		reqMethod := c.Method()
		reqOriginalURL := c.OriginalURL()

		reqHeaders := captureRequestHeaders(c)
		reqParams := c.AllParams()
		reqQueries := c.Queries()

		var reqBody json.RawMessage
		if cfg.CaptureReqBody {
			reqBody = copyBytes(c.Body(), cfg.MaxBodySize)
		}

		// --- Execute the handler (measure only handler duration) ---
		start := time.Now()
		handlerErr := c.Next()
		duration := float64(time.Since(start).Milliseconds())

		// If the handler returned an error (e.g. fiber.NewError(400, "msg")
		// or a raw GORM error), Fiber's ErrorHandler has NOT run yet — the
		// response is still at its default state (status 200, empty body).
		// Invoke the app's ErrorHandler now so it writes the real response
		// (status code + JSON body) before we capture it.
		if handlerErr != nil {
			if err := c.App().Config().ErrorHandler(c, handlerErr); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		// --- Capture response data (synchronous – after handler) ---
		// At this point the response is fully populated: either the handler
		// wrote it directly, or the ErrorHandler just wrote it above.
		statusCode := c.Response().StatusCode()

		// ignore 404 status code
		if statusCode == 404 && !strings.HasPrefix(path, "/api/") {
			return nil
		}

		success := statusCode < 400

		var respBody json.RawMessage
		if cfg.CaptureRespBody {
			respBody = copyBytes(c.Response().Body(), cfg.MaxBodySize)
		}

		// Capture the raw Go error (e.g. GORM errors) for debugging.
		// This preserves the full error chain separately from the
		// client-facing response body written by the ErrorHandler.
		var exception interface{}
		if handlerErr != nil {
			exception = handlerErr.Error()
		}

		respHeaders := captureResponseHeaders(c)

		// Normalized route path (e.g. /api/users/:id).
		routePath := c.Route().Path

		// Full URL including protocol + host.
		fullURL := buildFullURL(c)

		// Authenticated user (if any).
		userJSON := captureUser(c, cfg.UserContextKey)

		// Build JSON fields.
		requestJSON, _ := json.Marshal(map[string]interface{}{
			"ip":       reqIP,
			"headers":  reqHeaders,
			"params":   reqParams,
			"queries":  reqQueries,
			"body":     reqBody,
			"datetime": start,
		})

		responseJSON, _ := json.Marshal(map[string]interface{}{
			"statusCode": statusCode,
			"body":       respBody,
			"exception":  exception,
			"datetime":   time.Now(),
		})

		respHeadersJSON, _ := json.Marshal(respHeaders)

		// Fallback path normalization when route path is empty.
		if routePath == "" {
			routePath = normalizePath(reqOriginalURL)
		}

		entry := models.RequestLog{
			Key:             "apis-traffic",
			Path:            routePath,
			URL:             fullURL,
			Method:          reqMethod,
			User:            datatypes.JSON(userJSON),
			Request:         datatypes.JSON(requestJSON),
			Response:        datatypes.JSON(responseJSON),
			ResponseHeaders: datatypes.JSON(respHeadersJSON),
			Success:         success,
			Duration:        duration,
		}

		// Non-blocking enqueue — all DB work happens in the Writer goroutine.
		cfg.Writer.Write(entry)

		// Return nil — we already invoked the ErrorHandler above,
		// so Fiber must not call it a second time.
		return nil
	}
}

// --- helpers ---

func captureRequestHeaders(c *fiber.Ctx) map[string]string {
	h := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		h[string(key)] = string(value)
	})
	return h
}

func captureResponseHeaders(c *fiber.Ctx) map[string]string {
	h := make(map[string]string)
	c.Response().Header.VisitAll(func(key, value []byte) {
		h[string(key)] = string(value)
	})
	return h
}

func captureUser(c *fiber.Ctx, key string) json.RawMessage {
	u := c.Locals(key)
	if u == nil {
		return json.RawMessage("null")
	}
	b, err := json.Marshal(u)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}

func buildFullURL(c *fiber.Ctx) string {
	protocol := c.Get("X-Forwarded-Proto")
	if protocol == "" {
		protocol = c.Protocol()
	}
	host := c.Hostname()
	return protocol + "://" + host + c.OriginalURL()
}

// normalizePath strips query strings and replaces numeric / UUID
// segments with :id – used as a fallback when c.Route().Path is empty.
func normalizePath(raw string) string {
	path := strings.SplitN(raw, "?", 2)[0]
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if numericRe.MatchString(seg) || uuidRe.MatchString(seg) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

// copyBytes returns a safe copy of src, truncated to maxLen bytes.
// If maxLen < 0 the full slice is copied.
func copyBytes(src []byte, maxLen int) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	n := len(src)
	if maxLen >= 0 && n > maxLen {
		n = maxLen
	}
	dst := make([]byte, n)
	copy(dst, src[:n])
	return dst
}
