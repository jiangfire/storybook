package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMCPBatchUpdateACStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:mcp_handler_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "pm", Email: "pm@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "p1", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	criteria := []model.AcceptanceCriterion{
		{ID: "ac-1", Description: "first", Status: model.ACStatusPending, Order: 1},
		{ID: "ac-2", Description: "second", Status: model.ACStatusPending, Order: 2},
	}
	story := model.UserStory{
		ProjectID:          project.ID,
		Title:              "story1",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusBacklog,
		Priority:           1,
		CreatedBy:          owner.ID,
		AcceptanceCriteria: model.MarshalJSON(criteria),
		Tags:               model.MarshalJSON([]string{}),
		CodeReferences:     model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	h := NewMCPHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.POST("/mcp/v1/stories/:id/update-ac-status", h.BatchUpdateACStatus)

	body := map[string]any{
		"updates": []map[string]any{
			{"ac_id": "ac-1", "status": "passed", "evidence": "test"},
			{"ac_id": "ac-2", "status": "failed", "evidence": "test2"},
		},
	}
	payload, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp/v1/stories/1/update-ac-status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var refreshed model.UserStory
	if err := db.First(&refreshed, story.ID).Error; err != nil {
		t.Fatalf("reload story: %v", err)
	}
	after, err := model.ParseAcceptanceCriteria(refreshed.AcceptanceCriteria)
	if err != nil {
		t.Fatalf("parse criteria: %v", err)
	}
	if after[0].Status != model.ACStatusPassed || after[1].Status != model.ACStatusFailed {
		t.Fatalf("unexpected statuses: %+v", after)
	}
}

func TestMCPACCompletionStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:mcp_handler_stats_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "pm2", Email: "pm2@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "p2", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	storyDone := model.UserStory{
		ProjectID: project.ID,
		Title:     "story done",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-d1", Ref: "AC-1", Description: "done", Status: model.ACStatusPassed, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&storyDone).Error; err != nil {
		t.Fatalf("create story done: %v", err)
	}

	storyPending := model.UserStory{
		ProjectID: project.ID,
		Title:     "story pending",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-p1", Ref: "AC-P0-1", Description: "pending high", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&storyPending).Error; err != nil {
		t.Fatalf("create story pending: %v", err)
	}

	h := NewMCPHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.GET("/mcp/v1/stats/ac-completion", h.ACCompletionStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mcp/v1/stats/ac-completion", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := payload["data"].(map[string]any)
	if int(data["total_stories"].(float64)) != 2 {
		t.Fatalf("total_stories mismatch: %#v", data["total_stories"])
	}
	if int(data["completed_stories"].(float64)) != 1 {
		t.Fatalf("completed_stories mismatch: %#v", data["completed_stories"])
	}
	if int(data["pending_high_priority_acs"].(float64)) != 1 {
		t.Fatalf("pending_high_priority_acs mismatch: %#v", data["pending_high_priority_acs"])
	}
}

func TestMCPAnalyzeCodeACRejectPathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:mcp_handler_traversal_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "pm3", Email: "pm3@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "p3", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID: project.ID,
		Title:     "story traversal",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "desc", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	h := NewMCPHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.POST("/mcp/v1/stories/:id/analyze-code-ac", h.AnalyzeCodeAC)

	body := map[string]any{"file_path": "..\\go.mod"}
	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp/v1/stories/1/analyze-code-ac", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
