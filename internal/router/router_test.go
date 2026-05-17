package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/auth"
	"github.com/jiangfire/storybook/internal/config"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/wiring"
	"gorm.io/gorm"
)

func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tm := auth.NewTokenManager("router-test-secret", 1, 1)
	container, err := wiring.Build(&config.Config{}, db, slog.Default(), tm)
	if err != nil {
		t.Fatalf("wiring build: %v", err)
	}
	r := New(container)
	return r, db
}

func TestHealth_AlwaysOK(t *testing.T) {
	r, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestReadyz_HealthyDB(t *testing.T) {
	r, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"ready"`) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestReadyz_DBClosed_Returns503(t *testing.T) {
	r, db := newTestRouter(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql.DB: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when DB is closed, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"unavailable"`) {
		t.Errorf("expected unavailable status in body, got: %s", w.Body.String())
	}
}
