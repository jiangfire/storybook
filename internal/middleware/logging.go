package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
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
