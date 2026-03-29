package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureAssignableUserMatchesBugAssignRules(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug_handler_assignable_user_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{Username: "owner", Email: "owner@bug-assign.example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	developer := model.User{Username: "dev", Email: "dev@bug-assign.example.com", HashedPassword: "hashed-password", Role: model.RoleDeveloper}
	tester := model.User{Username: "tester", Email: "tester@bug-assign.example.com", HashedPassword: "hashed-password", Role: model.RoleTester}
	outsider := model.User{Username: "outsider", Email: "outsider@bug-assign.example.com", HashedPassword: "hashed-password", Role: model.RoleDeveloper}
	if err := db.Create(&[]model.User{owner, developer, tester, outsider}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	var users []model.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		t.Fatalf("reload users: %v", err)
	}
	owner = users[0]
	developer = users[1]
	tester = users[2]
	outsider = users[3]

	project := model.Project{Name: "缺陷指派测试", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&[]model.ProjectMember{
		{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct},
		{ProjectID: project.ID, UserID: developer.ID, RoleInProject: model.RoleDeveloper},
		{ProjectID: project.ID, UserID: tester.ID, RoleInProject: model.RoleTester},
	}).Error; err != nil {
		t.Fatalf("create project members: %v", err)
	}

	h := NewBugHandler(db)

	if err := h.ensureAssignableUser(project.ID, developer.ID); err != nil {
		t.Fatalf("expected developer to be assignable, got %v", err)
	}
	if err := h.ensureAssignableUser(project.ID, tester.ID); !errors.Is(err, errBugAssigneeRole) {
		t.Fatalf("expected tester to be rejected by role, got %v", err)
	}
	if err := h.ensureAssignableUser(project.ID, outsider.ID); !errors.Is(err, errForbidden) {
		t.Fatalf("expected outsider to be rejected as non-member, got %v", err)
	}
}

type bugHandlerFixture struct {
	db        *gorm.DB
	handler   *BugHandler
	project   model.Project
	owner     model.User
	developer model.User
	tester    model.User
}

func setupBugHandlerFixture(t *testing.T) bugHandlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.BugReport{},
		&model.ActivityLog{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	owner := model.User{Username: "owner-" + t.Name(), Email: "owner+" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	developer := model.User{Username: "dev-" + t.Name(), Email: "dev+" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com", HashedPassword: "hashed-password", Role: model.RoleDeveloper}
	tester := model.User{Username: "tester-" + t.Name(), Email: "tester+" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com", HashedPassword: "hashed-password", Role: model.RoleTester}
	if err := db.Create(&[]model.User{owner, developer, tester}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	var users []model.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		t.Fatalf("reload users: %v", err)
	}
	owner = users[0]
	developer = users[1]
	tester = users[2]

	project := model.Project{Name: "缺陷测试项目", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&[]model.ProjectMember{
		{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct},
		{ProjectID: project.ID, UserID: developer.ID, RoleInProject: model.RoleDeveloper},
		{ProjectID: project.ID, UserID: tester.ID, RoleInProject: model.RoleTester},
	}).Error; err != nil {
		t.Fatalf("create project members: %v", err)
	}

	return bugHandlerFixture{
		db:        db,
		handler:   NewBugHandler(db),
		project:   project,
		owner:     owner,
		developer: developer,
		tester:    tester,
	}
}

func newBugHandlerRouter(userID uint, role string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, userID)
		c.Set(middleware.CtxRoleKey, role)
		c.Next()
	})
	return r
}

func TestCreateRejectsNonDeveloperInitialAssignee(t *testing.T) {
	fixture := setupBugHandlerFixture(t)

	r := newBugHandlerRouter(fixture.tester.ID, model.RoleTester)
	r.POST("/projects/:id/bugs", fixture.handler.Create)

	body := bytes.NewBufferString(fmt.Sprintf(`{"title":"登录缺陷","severity":"high","assigned_to":%d}`, fixture.tester.ID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%d/bugs", fixture.project.ID), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "缺陷仅可指派给开发角色") {
		t.Fatalf("expected role validation error, got body=%s", w.Body.String())
	}

	var count int64
	if err := fixture.db.Model(&model.BugReport{}).Count(&count).Error; err != nil {
		t.Fatalf("count bugs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no bug to be created, got %d", count)
	}
}

func TestAssignRejectsNonDeveloperAssignee(t *testing.T) {
	fixture := setupBugHandlerFixture(t)

	bug := model.BugReport{
		ProjectID:   fixture.project.ID,
		Title:       "待指派缺陷",
		Description: "test",
		Severity:    model.BugSeverityHigh,
		Status:      model.BugStatusOpen,
		ReportedBy:  fixture.tester.ID,
	}
	if err := fixture.db.Create(&bug).Error; err != nil {
		t.Fatalf("create bug: %v", err)
	}

	r := newBugHandlerRouter(fixture.owner.ID, model.RoleProduct)
	r.PATCH("/bugs/:id/assign", fixture.handler.Assign)

	body := bytes.NewBufferString(fmt.Sprintf(`{"assigned_to":%d}`, fixture.tester.ID))
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/bugs/%d/assign", bug.ID), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "缺陷仅可指派给开发角色") {
		t.Fatalf("expected role validation error, got body=%s", w.Body.String())
	}

	var saved model.BugReport
	if err := fixture.db.First(&saved, bug.ID).Error; err != nil {
		t.Fatalf("reload bug: %v", err)
	}
	if saved.AssignedTo != nil {
		t.Fatalf("expected assignee to stay nil, got %v", *saved.AssignedTo)
	}
}

func TestUpdateStatusMaintainsResolvedAtLifecycle(t *testing.T) {
	fixture := setupBugHandlerFixture(t)

	bug := model.BugReport{
		ProjectID:   fixture.project.ID,
		Title:       "状态流转缺陷",
		Description: "test",
		Severity:    model.BugSeverityMedium,
		Status:      model.BugStatusOpen,
		ReportedBy:  fixture.tester.ID,
		AssignedTo:  &fixture.developer.ID,
	}
	if err := fixture.db.Create(&bug).Error; err != nil {
		t.Fatalf("create bug: %v", err)
	}

	r := newBugHandlerRouter(fixture.developer.ID, model.RoleDeveloper)
	r.PATCH("/bugs/:id/status", fixture.handler.UpdateStatus)

	resolveReq := httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/bugs/%d/status", bug.ID),
		bytes.NewBufferString(`{"status":"resolved"}`),
	)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveResp := httptest.NewRecorder()
	r.ServeHTTP(resolveResp, resolveReq)

	if resolveResp.Code != http.StatusOK {
		t.Fatalf("expected resolve request 200, got %d body=%s", resolveResp.Code, resolveResp.Body.String())
	}

	var resolved model.BugReport
	if err := fixture.db.First(&resolved, bug.ID).Error; err != nil {
		t.Fatalf("reload resolved bug: %v", err)
	}
	if resolved.Status != model.BugStatusResolved || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved bug with resolved_at, got status=%s resolved_at=%v", resolved.Status, resolved.ResolvedAt)
	}

	reopenReq := httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/bugs/%d/status", bug.ID),
		bytes.NewBufferString(`{"status":"in_progress"}`),
	)
	reopenReq.Header.Set("Content-Type", "application/json")
	reopenResp := httptest.NewRecorder()
	r.ServeHTTP(reopenResp, reopenReq)

	if reopenResp.Code != http.StatusOK {
		t.Fatalf("expected reopen request 200, got %d body=%s", reopenResp.Code, reopenResp.Body.String())
	}

	var reopened model.BugReport
	if err := fixture.db.First(&reopened, bug.ID).Error; err != nil {
		t.Fatalf("reload reopened bug: %v", err)
	}
	if reopened.Status != model.BugStatusInProgress {
		t.Fatalf("expected reopened bug status=in_progress, got %s", reopened.Status)
	}
	if reopened.ResolvedAt != nil {
		t.Fatalf("expected resolved_at to be cleared, got %v", reopened.ResolvedAt)
	}
}
