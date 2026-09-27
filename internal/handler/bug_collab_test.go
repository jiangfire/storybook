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
	"github.com/jiangfire/storybook/internal/repository"
	"gorm.io/gorm"
)

func setupBugCollabDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:bug_collab_"+sanitizeDSN(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}, &model.BugReport{}, &model.ActivityLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedBugCollabProject(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 项目成员：1=产品（操作者）、11=开发、12=测试
	if err := db.Create(&model.ProjectMember{ProjectID: 1, UserID: 1, RoleInProject: model.RoleProduct}).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if err := db.Create(&model.ProjectMember{ProjectID: 1, UserID: 11, RoleInProject: model.RoleDeveloper}).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if err := db.Create(&model.ProjectMember{ProjectID: 1, UserID: 12, RoleInProject: model.RoleTester}).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
}

func newBugCollabRouter(t *testing.T, role string) (*gin.Engine, *gorm.DB) {
	db := setupBugCollabDB(t)
	seedBugCollabProject(t, db)

	h := NewBugHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(1))
		c.Set(middleware.CtxRoleKey, role)
		c.Set(middleware.CtxProjectKey, &model.Project{ID: 1, Name: "P", OwnerID: 1})
		c.Next()
	})
	r.POST("/api/projects/:id/bugs", h.Create)
	return r, db
}

func TestProductCanCreateBug(t *testing.T) {
	r, db := newBugCollabRouter(t, model.RoleProduct)

	body, _ := json.Marshal(map[string]any{"title": "登录按钮失效", "severity": "high"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects/1/bugs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for product create bug, got %d body=%s", w.Code, w.Body.String())
	}
	var count int64
	db.Model(&model.BugReport{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 bug created, got %d", count)
	}
}

func TestDeveloperCannotCreateBug(t *testing.T) {
	r, _ := newBugCollabRouter(t, model.RoleDeveloper)

	body, _ := json.Marshal(map[string]any{"title": "登录按钮失效", "severity": "high"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects/1/bugs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for developer create bug, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestBugAssignRejectsNonDeveloperMember(t *testing.T) {
	db := setupBugCollabDB(t)
	seedBugCollabProject(t, db)

	bug := model.BugReport{ProjectID: 1, Title: "样式错乱", Severity: model.BugSeverityMedium, Status: model.BugStatusOpen, ReportedBy: 12}
	if err := db.Create(&bug).Error; err != nil {
		t.Fatalf("seed bug: %v", err)
	}

	h := NewBugHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(1))
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Set(middleware.CtxBugKey, &bug)
		c.Next()
	})
	r.PATCH("/api/bugs/:id/assign", h.Assign)

	// 项目内的测试成员不再是合法指派对象（与故事指派同口径：仅开发成员）
	body, _ := json.Marshal(map[string]any{"assigned_to": 12})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/bugs/1/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when assigning to tester, got %d body=%s", w.Code, w.Body.String())
	}

	// 开发成员可以正常指派
	body, _ = json.Marshal(map[string]any{"assigned_to": 11})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/bugs/1/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when assigning to developer, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMeBugsAggregatesOnlyActiveAssignedBugs(t *testing.T) {
	db := setupBugCollabDB(t)

	bugs := []model.BugReport{
		{ProjectID: 1, Title: "进行中的缺陷", Severity: model.BugSeverityHigh, Status: model.BugStatusInProgress, ReportedBy: 12, AssignedTo: &[]uint{7}[0]},
		{ProjectID: 1, Title: "待处理的缺陷", Severity: model.BugSeverityMedium, Status: model.BugStatusOpen, ReportedBy: 12, AssignedTo: &[]uint{7}[0]},
		{ProjectID: 1, Title: "已解决的缺陷", Severity: model.BugSeverityLow, Status: model.BugStatusResolved, ReportedBy: 12, AssignedTo: &[]uint{7}[0]},
		{ProjectID: 1, Title: "别人的缺陷", Severity: model.BugSeverityLow, Status: model.BugStatusOpen, ReportedBy: 12},
	}
	if err := db.Create(&bugs).Error; err != nil {
		t.Fatalf("seed bugs: %v", err)
	}

	h := NewMeHandler(
		repository.NewUserRepository(db),
		repository.NewStoryRepository(db),
		repository.NewTaskRepository(db),
		repository.NewBugRepository(db),
	)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(7))
		c.Next()
	})
	r.GET("/api/me/bugs", h.MyBugs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me/bugs", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Data struct {
			Bugs []struct {
				ID     uint   `json:"id"`
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"bugs"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if payload.Data.Total != 2 {
		t.Fatalf("expected 2 active assigned bugs, got %d", payload.Data.Total)
	}
	for _, bug := range payload.Data.Bugs {
		if bug.Status == model.BugStatusResolved {
			t.Fatalf("resolved bug should be excluded: %+v", bug)
		}
	}
}
