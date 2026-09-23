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

func setupAssignStoryRouter(t *testing.T, storyStatus string) (*gin.Engine, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:assign_guard_"+sanitizeDSN(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.ProjectTechLead{}, &model.UserStory{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	story := model.UserStory{
		ProjectID:          1,
		Title:              "待分配故事",
		StoryType:          model.StoryTypeFeature,
		Status:             storyStatus,
		ReviewStatus:       model.ReviewStatusApproved,
		CreatedBy:          1,
		Priority:           2,
		Position:           1,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{}),
		Tags:               model.MarshalJSON([]string{}),
		CodeReferences:     model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	// 开发成员（user_id=11），供分配目标校验
	if err := db.Create(&model.ProjectMember{ProjectID: 1, UserID: 11, RoleInProject: model.RoleDeveloper}).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	h := NewStoryHandler(db, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(1))
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Set(middleware.CtxProjectKey, &model.Project{ID: 1, Name: "P", OwnerID: 1})
		c.Set(middleware.CtxStoryKey, &story)
		c.Next()
	})
	r.POST("/api/stories/:id/assignee", h.AssignStory)
	return r, story.ID
}

func TestAssignStoryRejectsPendingStory(t *testing.T) {
	r, _ := setupAssignStoryRouter(t, model.StoryStatusPending)

	body, _ := json.Marshal(map[string]any{"assigned_to": 11})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/stories/1/assignee", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for pending story, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAssignStoryAllowsBacklogStory(t *testing.T) {
	r, _ := setupAssignStoryRouter(t, model.StoryStatusBacklog)

	body, _ := json.Marshal(map[string]any{"assigned_to": 11})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/stories/1/assignee", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for backlog story, got %d body=%s", w.Code, w.Body.String())
	}
}
