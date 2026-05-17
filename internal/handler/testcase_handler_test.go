package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type testCaseHandlerFixture struct {
	db        *gorm.DB
	handler   *TestCaseHandler
	project   model.Project
	story     model.UserStory
	owner     model.User
	admin     model.User
	techLead  model.User
	tester    model.User
	developer model.User
}

func setupTestCaseHandlerFixture(t *testing.T) testCaseHandlerFixture {
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
		&model.ProjectTechLead{},
		&model.UserStory{},
		&model.TestCase{},
		&model.ActivityLog{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	suffix := strings.ReplaceAll(t.Name(), "/", "-")
	owner := model.User{Username: "owner-" + suffix, Email: "owner+" + suffix + "@example.com", HashedPassword: "hashed-password", Role: model.RoleProduct}
	admin := model.User{Username: "admin-" + suffix, Email: "admin+" + suffix + "@example.com", HashedPassword: "hashed-password", Role: model.RoleAdmin}
	techLead := model.User{Username: "techlead-" + suffix, Email: "techlead+" + suffix + "@example.com", HashedPassword: "hashed-password", Role: model.RoleTechLead}
	tester := model.User{Username: "tester-" + suffix, Email: "tester+" + suffix + "@example.com", HashedPassword: "hashed-password", Role: model.RoleTester}
	developer := model.User{Username: "developer-" + suffix, Email: "developer+" + suffix + "@example.com", HashedPassword: "hashed-password", Role: model.RoleDeveloper}
	if err := db.Create(&[]model.User{owner, admin, techLead, tester, developer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	var users []model.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		t.Fatalf("reload users: %v", err)
	}
	owner = users[0]
	admin = users[1]
	techLead = users[2]
	tester = users[3]
	developer = users[4]

	project := model.Project{Name: "测试用例项目", Description: "test", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := db.Create(&[]model.ProjectMember{
		{ProjectID: project.ID, UserID: owner.ID, RoleInProject: model.RoleProduct},
		{ProjectID: project.ID, UserID: tester.ID, RoleInProject: model.RoleTester},
		{ProjectID: project.ID, UserID: developer.ID, RoleInProject: model.RoleDeveloper},
	}).Error; err != nil {
		t.Fatalf("create project members: %v", err)
	}
	if err := db.Create(&model.ProjectTechLead{ProjectID: project.ID, UserID: techLead.ID}).Error; err != nil {
		t.Fatalf("assign tech lead: %v", err)
	}

	story := model.UserStory{
		ProjectID:          project.ID,
		Title:              "测试故事",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusPending,
		ReviewStatus:       model.ReviewStatusPending,
		CreatedBy:          owner.ID,
		AcceptanceCriteria: datatypes.JSON([]byte("[]")),
		Tags:               datatypes.JSON([]byte("[]")),
		CodeReferences:     datatypes.JSON([]byte("[]")),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	return testCaseHandlerFixture{
		db:        db,
		handler:   NewTestCaseHandler(db),
		project:   project,
		story:     story,
		owner:     owner,
		admin:     admin,
		techLead:  techLead,
		tester:    tester,
		developer: developer,
	}
}

func newTestCaseRouter(userID uint, role string, story *model.UserStory, project *model.Project, testCase *model.TestCase) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, userID)
		c.Set(middleware.CtxRoleKey, role)
		if story != nil {
			c.Set(middleware.CtxStoryKey, story)
		}
		if project != nil {
			c.Set(middleware.CtxProjectKey, project)
		}
		if testCase != nil {
			c.Set(middleware.CtxTestCaseKey, testCase)
		}
		c.Next()
	})
	return r
}

func TestListByStoryAllowsAssignedTechLead(t *testing.T) {
	fixture := setupTestCaseHandlerFixture(t)

	testCase := model.TestCase{
		StoryID:        fixture.story.ID,
		Title:          "登录回归",
		Description:    "验证主流程",
		Steps:          model.MarshalJSON([]string{"打开登录页", "输入账号密码"}),
		ExpectedResult: "登录成功",
		Status:         "pending",
		CreatedBy:      fixture.tester.ID,
	}
	if err := fixture.db.Create(&testCase).Error; err != nil {
		t.Fatalf("create test case: %v", err)
	}

	r := newTestCaseRouter(fixture.techLead.ID, model.RoleTechLead, &fixture.story, &fixture.project, nil)
	r.GET("/stories/:id/test-cases", fixture.handler.ListByStory)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/stories/%d/test-cases", fixture.story.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "登录回归") || !strings.Contains(w.Body.String(), fixture.tester.Email) {
		t.Fatalf("expected response to include test case and creator, got body=%s", w.Body.String())
	}
}

func TestCreateRejectsBlankStepsAfterTrim(t *testing.T) {
	fixture := setupTestCaseHandlerFixture(t)

	r := newTestCaseRouter(fixture.tester.ID, model.RoleTester, &fixture.story, &fixture.project, nil)
	r.POST("/stories/:id/test-cases", fixture.handler.Create)

	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/stories/%d/test-cases", fixture.story.ID),
		bytes.NewBufferString(`{"title":"登录用例","steps":[" ","\t"],"expected_result":"成功"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "steps 至少包含一条有效步骤") {
		t.Fatalf("expected steps validation error, got body=%s", w.Body.String())
	}

	var count int64
	if err := fixture.db.Model(&model.TestCase{}).Count(&count).Error; err != nil {
		t.Fatalf("count test cases: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no test case to be created, got %d", count)
	}
}

func TestUpdateStatusRejectsDeveloperEvenWithProjectAccess(t *testing.T) {
	fixture := setupTestCaseHandlerFixture(t)

	testCase := model.TestCase{
		StoryID:        fixture.story.ID,
		Title:          "待更新用例",
		Steps:          model.MarshalJSON([]string{"执行步骤"}),
		ExpectedResult: "成功",
		Status:         "pending",
		CreatedBy:      fixture.tester.ID,
	}
	if err := fixture.db.Create(&testCase).Error; err != nil {
		t.Fatalf("create test case: %v", err)
	}

	r := newTestCaseRouter(fixture.developer.ID, model.RoleDeveloper, nil, nil, nil)
	r.PATCH("/test-cases/:id/status", fixture.handler.UpdateStatus)

	req := httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/test-cases/%d/status", testCase.ID),
		bytes.NewBufferString(`{"status":"passed"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "仅测试人员可更新测试用例状态") {
		t.Fatalf("expected permission error, got body=%s", w.Body.String())
	}
}

func TestUpdateStatusAllowsAdminAccessAcrossProjects(t *testing.T) {
	fixture := setupTestCaseHandlerFixture(t)

	testCase := model.TestCase{
		StoryID:        fixture.story.ID,
		Title:          "管理员更新用例",
		Steps:          model.MarshalJSON([]string{"执行步骤"}),
		ExpectedResult: "成功",
		Status:         "pending",
		CreatedBy:      fixture.tester.ID,
	}
	if err := fixture.db.Create(&testCase).Error; err != nil {
		t.Fatalf("create test case: %v", err)
	}

	r := newTestCaseRouter(fixture.admin.ID, model.RoleAdmin, &fixture.story, &fixture.project, &testCase)
	r.PATCH("/test-cases/:id/status", fixture.handler.UpdateStatus)

	req := httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/test-cases/%d/status", testCase.ID),
		bytes.NewBufferString(`{"status":"passed"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var saved model.TestCase
	if err := fixture.db.First(&saved, testCase.ID).Error; err != nil {
		t.Fatalf("reload test case: %v", err)
	}
	if saved.Status != "passed" {
		t.Fatalf("expected status=passed, got %s", saved.Status)
	}
}
