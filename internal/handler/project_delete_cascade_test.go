package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type capturedEvent struct {
	projectID uint
	eventType string
}

type mockProjectEvents struct {
	events []capturedEvent
}

func (m *mockProjectEvents) BroadcastProject(projectID uint, eventType string, _ any) {
	m.events = append(m.events, capturedEvent{projectID: projectID, eventType: eventType})
}
func (m *mockProjectEvents) BroadcastUser(_ uint, _ string, _ any) {}

// setupCascadeEnv 建两个项目：p1 挂满各类子资源并删除，p2 只挂一个故事用于验证隔离。
func setupCascadeEnv(t *testing.T) (*gorm.DB, *mockProjectEvents, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.ProjectTechLead{},
		&model.UserStory{}, &model.Sprint{}, &model.Task{}, &model.BugReport{},
		&model.TestCase{}, &model.Notification{}, &model.ActivityLog{}, &model.BoardColumn{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "owner", Email: "owner@test.dev", HashedPassword: "x", Role: model.RoleAdmin}
	other := model.User{Username: "other", Email: "other@test.dev", HashedPassword: "x", Role: model.RoleProduct}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other: %v", err)
	}

	p1 := model.Project{Name: "p1", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	p2 := model.Project{Name: "p2", OwnerID: other.ID, AgileMode: model.AgileModeKanban}
	// 逐个 Create：range 循环里 &p 是副本，Create 回填的 ID 不会写回 p1/p2
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("seed p1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("seed p2: %v", err)
	}

	seed := []any{
		&model.ProjectMember{ProjectID: p1.ID, UserID: owner.ID, RoleInProject: model.RoleAdmin},
		&model.ProjectTechLead{ProjectID: p1.ID, UserID: owner.ID, AssignedAt: db.NowFunc()},
		&model.Sprint{ProjectID: p1.ID, Name: "s1", Goal: "g", StartDate: db.NowFunc(), EndDate: db.NowFunc(), Status: "planned"},
		&model.UserStory{ProjectID: p1.ID, Title: "story-p1", StoryType: "feature", Status: model.StoryStatusBacklog, Priority: 1, CreatedBy: owner.ID, AcceptanceCriteria: []byte(`[]`), Tags: []byte(`[]`), CodeReferences: []byte(`[]`)},
		&model.UserStory{ProjectID: p2.ID, Title: "story-p2", StoryType: "feature", Status: model.StoryStatusBacklog, Priority: 1, CreatedBy: other.ID, AcceptanceCriteria: []byte(`[]`), Tags: []byte(`[]`), CodeReferences: []byte(`[]`)},
	}
	for _, item := range seed {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("seed %+v: %v", item, err)
		}
	}
	var p1Story model.UserStory
	if err := db.Where("project_id = ?", p1.ID).First(&p1Story).Error; err != nil {
		t.Fatalf("load p1 story: %v", err)
	}
	more := []any{
		&model.Task{ProjectID: p1.ID, StoryID: p1Story.ID, Title: "t1", Status: "todo", Priority: 1, CreatedBy: owner.ID, CodeReferences: []byte(`[]`)},
		&model.BugReport{ProjectID: p1.ID, StoryID: &p1Story.ID, Title: "b1", Severity: "low", Status: "open", ReportedBy: owner.ID},
		&model.TestCase{StoryID: p1Story.ID, Title: "tc1", Status: "draft", Steps: []byte(`[]`), CreatedBy: owner.ID},
		&model.Notification{UserID: owner.ID, Type: "story.assigned", EntityType: "story", EntityID: p1Story.ID, ProjectID: &p1.ID, Title: "n1"},
	}
	for _, item := range more {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("seed %+v: %v", item, err)
		}
	}

	events := &mockProjectEvents{}
	h := NewProjectHandler(db).WithEvents(events)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, owner.ID)
		c.Set(middleware.CtxRoleKey, model.RoleAdmin)
		var project model.Project
		if err := db.Where("id = ?", p1.ID).First(&project).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Set(middleware.CtxProjectKey, &project)
		c.Set(middleware.CtxIsOwnerKey, true)
		c.Next()
	})
	r.DELETE("/api/projects/:id", h.DeleteProject)

	return db, events, r
}

func TestDeleteProjectCascadesSoftDelete(t *testing.T) {
	db, _, r := setupCascadeEnv(t)

	var p1 model.Project
	if err := db.Where("name = ?", "p1").First(&p1).Error; err != nil {
		t.Fatalf("load p1: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%d", p1.ID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("delete failed: %d %s", w.Code, w.Body.String())
	}

	// 过滤查询：p1 的全部子资源对业务不可见；p2 的故事必须幸存
	cases := []struct {
		name  string
		model any
	}{
		{"成员", &model.ProjectMember{}},
		{"技术负责人", &model.ProjectTechLead{}},
		{"冲刺", &model.Sprint{}},
		{"故事", &model.UserStory{}},
		{"任务", &model.Task{}},
		{"缺陷", &model.BugReport{}},
		{"通知", &model.Notification{}},
	}
	for _, tc := range cases {
		var count int64
		if err := db.Model(tc.model).Where("project_id = ?", p1.ID).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", tc.name, err)
		}
		if count != 0 {
			t.Fatalf("%s 应级联软删，仍剩 %d 条", tc.name, count)
		}
	}

	var p2Stories int64
	if err := db.Model(&model.UserStory{}).Where("project_id = ?", p1.ID+1).Count(&p2Stories).Error; err != nil {
		t.Fatalf("count p2 stories: %v", err)
	}
	if p2Stories != 1 {
		t.Fatalf("p2 故事不应被误删，剩 %d 条", p2Stories)
	}

	// 测试用例挂在故事下（无 project_id）：按 p1 故事的 ID 断言
	var p1StoryIDs []uint
	if err := db.Unscoped().Model(&model.UserStory{}).Where("project_id = ?", p1.ID).Pluck("id", &p1StoryIDs).Error; err != nil {
		t.Fatalf("pluck p1 story ids: %v", err)
	}
	var tcCount int64
	if err := db.Unscoped().Model(&model.TestCase{}).Where("story_id IN ?", p1StoryIDs).Count(&tcCount).Error; err != nil {
		t.Fatalf("count testcases: %v", err)
	}
	if tcCount != 1 {
		t.Fatalf("测试用例应保留软删行供审计，实际 %d 条", tcCount)
	}

	var unscopedStories int64
	if err := db.Unscoped().Model(&model.UserStory{}).Where("project_id = ?", p1.ID).Count(&unscopedStories).Error; err != nil {
		t.Fatalf("unscoped count: %v", err)
	}
	if unscopedStories != 1 {
		t.Fatalf("应保留软删行供审计，实际 %d 条", unscopedStories)
	}
}

func TestDeleteProjectWritesAuditLogAndBroadcastsEvent(t *testing.T) {
	db, events, r := setupCascadeEnv(t)

	var p1 model.Project
	if err := db.Unscoped().Where("name = ?", "p1").First(&p1).Error; err != nil {
		t.Fatalf("load p1: %v", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%d", p1.ID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("delete failed: %d %s", w.Code, w.Body.String())
	}

	var logs int64
	if err := db.Model(&model.ActivityLog{}).Where("project_id = ? AND action = ?", p1.ID, "deleted").Count(&logs).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logs != 1 {
		t.Fatalf("应写入一条 deleted 审计日志，实际 %d 条", logs)
	}

	if len(events.events) != 1 || events.events[0].eventType != "project.deleted" || events.events[0].projectID != p1.ID {
		t.Fatalf("应广播一次 project.deleted 事件，实际 %+v", events.events)
	}
}
