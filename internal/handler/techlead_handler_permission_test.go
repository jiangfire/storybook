package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

// setupTechLeadPermissionRouter 构造带项目上下文的 TechLead 路由，验证角色门禁。
func setupTechLeadPermissionRouter(t *testing.T, role string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:techlead_perm_"+sanitizeDSN(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.ProjectTechLead{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	h := NewTechLeadHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(1))
		c.Set(middleware.CtxRoleKey, role)
		c.Set(middleware.CtxProjectKey, &model.Project{ID: 1, Name: "P", OwnerID: 1})
		c.Next()
	})
	r.POST("/api/projects/:id/techleads", h.AddTechLead)
	r.DELETE("/api/projects/:id/techleads/:userID", h.RemoveTechLead)
	return r, db
}

// sanitizeDSN 把测试名转成可用于 sqlite DSN 的安全字符串（仅保留字母数字）。
func sanitizeDSN(s string) string {
	out := make([]rune, 0, len(s))
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			out = append(out, ch)
		}
	}
	return string(out)
}

func TestProductCanAddTechLead(t *testing.T) {
	r, db := setupTechLeadPermissionRouter(t, model.RoleProduct)

	if err := db.Create(&model.User{ID: 7, Username: "tl-7", Email: "tl7@example.com", Role: model.RoleTechLead}).Error; err != nil {
		t.Fatalf("create tl user: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"user_id": 7})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects/1/techleads", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for product, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeveloperCannotAddTechLead(t *testing.T) {
	r, db := setupTechLeadPermissionRouter(t, model.RoleDeveloper)

	if err := db.Create(&model.User{ID: 7, Username: "tl-7", Email: "tl7@example.com", Role: model.RoleTechLead}).Error; err != nil {
		t.Fatalf("create tl user: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"user_id": 7})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects/1/techleads", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for developer, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeveloperCannotRemoveTechLead(t *testing.T) {
	r, db := setupTechLeadPermissionRouter(t, model.RoleDeveloper)

	if err := db.Create(&model.ProjectTechLead{ProjectID: 1, UserID: 7}).Error; err != nil {
		t.Fatalf("seed tech lead: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/1/techleads/7", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for developer, got %d body=%s", w.Code, w.Body.String())
	}
}
