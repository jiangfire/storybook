package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetOverviewReturnsZeroWhenAveragePointsIsNull(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:project_handler_overview_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{
		Username:       "owner",
		Email:          "owner@example.com",
		HashedPassword: "hashed-password",
		Role:           model.RoleProduct,
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	project := model.Project{
		Name:        "零点数项目",
		Description: "用于测试概览聚合空值",
		OwnerID:     owner.ID,
		AgileMode:   model.AgileModeKanban,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	h := NewProjectHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.GET("/api/projects/:id/overview", h.GetOverview)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/1/overview", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data missing: %#v", payload)
	}

	stats, ok := data["statistics"].(map[string]any)
	if !ok {
		t.Fatalf("statistics missing: %#v", data)
	}

	avgPoints, ok := stats["avg_story_points"].(float64)
	if !ok {
		t.Fatalf("avg_story_points is not float64: %#v", stats["avg_story_points"])
	}
	if avgPoints != 0 {
		t.Fatalf("expected avg_story_points=0, got %v", avgPoints)
	}
}
