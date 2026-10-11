package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RequestMiddleware interface {
	Log(ctx *gin.Context)
	Timeout(ctx *gin.Context)
}

type requestMiddleware struct {
	timeout time.Duration
}

func NewRequestMiddleware(timeout time.Duration) RequestMiddleware {
	return &requestMiddleware{timeout: timeout}
}

// Log writes one JSON line per request (NFR-05) and tags it with a request ID.
func (m *requestMiddleware) Log(c *gin.Context) {
	start := time.Now()
	requestId := c.GetHeader("X-Request-Id")
	if requestId == "" {
		requestId = uuid.NewString()
	}
	c.Set("request_id", requestId)
	c.Header("X-Request-Id", requestId)

	c.Next()

	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	slog.Info("request",
		"request_id", requestId,
		"method", c.Request.Method,
		"route", route,
		"status", c.Writer.Status(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

// Timeout bounds each request. A request still waiting for a database
// connection when it fires is answered with 503 instead of hanging.
func (m *requestMiddleware) Timeout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), m.timeout)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}
