package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/metrics"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		latency := time.Since(startedAt)
		status := c.Writer.Status()
		// Observe HTTP metrics keyed by gin-route (NOT the raw URL) to keep
		// label cardinality bounded.
		routeLabel := c.FullPath()
		if routeLabel == "" {
			// Unmatched routes (404s, websocket upgrades, etc.) collapse into one
			// label so an attacker scanning random URLs cannot blow up cardinality.
			routeLabel = "unmatched"
		}
		statusLabel := strconv.Itoa(status)
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, routeLabel, statusLabel).Observe(latency.Seconds())
		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, routeLabel, statusLabel).Inc()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		if rawQuery := strings.TrimSpace(c.Request.URL.RawQuery); rawQuery != "" {
			attrs = append(attrs, "query", rawQuery)
		}
		if userID, ok := CurrentUserID(c); ok {
			attrs = append(attrs, "user_id", userID)
		}
		if role, ok := CurrentRole(c); ok {
			attrs = append(attrs, "role", role)
		}
		if errText := strings.TrimSpace(c.Errors.String()); errText != "" {
			attrs = append(attrs, "errors", errText)
		}

		switch status := c.Writer.Status(); {
		case status >= http.StatusInternalServerError:
			logger.Error("request completed", attrs...)
		case status >= http.StatusBadRequest:
			logger.Warn("request completed", attrs...)
		default:
			logger.Info("request completed", attrs...)
		}
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", http.StatusInternalServerError,
			"client_ip", c.ClientIP(),
			"panic", fmt.Sprint(recovered),
			"stack", string(debug.Stack()),
		}
		if userID, ok := CurrentUserID(c); ok {
			attrs = append(attrs, "user_id", userID)
		}
		if role, ok := CurrentRole(c); ok {
			attrs = append(attrs, "role", role)
		}

		logger.Error("panic recovered", attrs...)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
