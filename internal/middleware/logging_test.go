package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{}))
	r := gin.New()
	r.Use(RequestLogger(logger))
	r.GET("/ping", func(c *gin.Context) {
		c.Set(CtxUserIDKey, uint(7))
		c.Set(CtxRoleKey, "admin")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	logText := output.String()
	for _, expected := range []string{
		"request completed",
		"method=GET",
		"path=/ping",
		"status=204",
		"user_id=7",
		"role=admin",
	} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
}

func TestRecovery(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{}))
	r := gin.New()
	r.Use(Recovery(logger))
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	logText := output.String()
	for _, expected := range []string{
		"panic recovered",
		"method=GET",
		"path=/panic",
		"panic=boom",
		"status=500",
	} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("expected log to contain %q, got %q", expected, logText)
		}
	}
}
