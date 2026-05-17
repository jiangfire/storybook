package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type accessMWFixture struct {
	db       *gorm.DB
	owner    model.User
	member   model.User
	outsider model.User
	project  model.Project
	story    model.UserStory
	bug      model.BugReport
	sprint   model.Sprint
	task     model.Task
	tc       model.TestCase
}

func newAccessMWFixture(t *testing.T) accessMWFixture {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.UserStory{},
		&model.BugReport{},
		&model.Sprint{},
		&model.Task{},
		&model.TestCase{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	users := []model.User{
		{Username: "owner", Email: "owner@example.com", HashedPassword: "hashed", Role: model.RoleProduct},
		{Username: "member", Email: "member@example.com", HashedPassword: "hashed", Role: model.RoleDeveloper},
		{Username: "outsider", Email: "outsider@example.com", HashedPassword: "hashed", Role: model.RoleTester},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	project := model.Project{Name: "Alpha", OwnerID: users[0].ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := db.Create(&[]model.ProjectMember{
		{ProjectID: project.ID, UserID: users[0].ID, RoleInProject: model.RoleProduct},
		{ProjectID: project.ID, UserID: users[1].ID, RoleInProject: model.RoleDeveloper},
	}).Error; err != nil {
		t.Fatalf("create members: %v", err)
	}

	story := model.UserStory{
		ProjectID:          project.ID,
		Title:              "登录故事",
		StoryType:          model.StoryTypeFeature,
		Status:             model.StoryStatusPending,
		ReviewStatus:       model.ReviewStatusPending,
		CreatedBy:          users[0].ID,
		AcceptanceCriteria: datatypes.JSON([]byte("[]")),
		Tags:               datatypes.JSON([]byte("[]")),
		CodeReferences:     datatypes.JSON([]byte("[]")),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	bug := model.BugReport{
		ProjectID:   project.ID,
		StoryID:     &story.ID,
		Title:       "崩溃",
		Description: "点击登录崩溃",
		Status:      model.BugStatusOpen,
		Severity:    model.BugSeverityHigh,
		ReportedBy:  users[0].ID,
	}
	if err := db.Create(&bug).Error; err != nil {
		t.Fatalf("create bug: %v", err)
	}

	sprint := model.Sprint{
		ProjectID: project.ID,
		Name:      "Sprint 1",
		Status:    "planned",
	}
	if err := db.Create(&sprint).Error; err != nil {
		t.Fatalf("create sprint: %v", err)
	}

	task := model.Task{
		ProjectID:      project.ID,
		StoryID:        story.ID,
		Title:          "实现登录",
		Description:    "前端登录表单",
		Status:         model.TaskStatusTodo,
		CreatedBy:      users[0].ID,
		CodeReferences: datatypes.JSON([]byte("[]")),
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	tc := model.TestCase{
		StoryID:        story.ID,
		Title:          "登录回归",
		Steps:          datatypes.JSON([]byte(`["打开页面","输入密码"]`)),
		ExpectedResult: "登录成功",
		Status:         "pending",
		CreatedBy:      users[1].ID,
	}
	if err := db.Create(&tc).Error; err != nil {
		t.Fatalf("create test case: %v", err)
	}

	return accessMWFixture{
		db:       db,
		owner:    users[0],
		member:   users[1],
		outsider: users[2],
		project:  project,
		story:    story,
		bug:      bug,
		sprint:   sprint,
		task:     task,
		tc:       tc,
	}
}

func newAccessRouter(userID uint, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != 0 {
			c.Set(CtxUserIDKey, userID)
			c.Set(CtxRoleKey, role)
		}
		c.Next()
	})
	return r
}

func TestRequireBugAccess(t *testing.T) {
	f := newAccessMWFixture(t)

	t.Run("unauthorized", func(t *testing.T) {
		r := newAccessRouter(0, "")
		r.GET("/bugs/:id", RequireBugAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bugs/%d", f.bug.ID), nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		r := newAccessRouter(f.outsider.ID, f.outsider.Role)
		r.GET("/bugs/:id", RequireBugAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bugs/%d", f.bug.ID), nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		r.GET("/bugs/:id", RequireBugAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bugs/99999", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success and injects context", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		var gotBug *model.BugReport
		var gotProject *model.Project
		r.GET("/bugs/:id", RequireBugAccess(f.db, "id"), func(c *gin.Context) {
			gotBug = MustBug(c)
			gotProject = MustProject(c)
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/bugs/%d", f.bug.ID), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if gotBug == nil || gotBug.ID != f.bug.ID {
			t.Fatal("bug not injected")
		}
		if gotProject == nil || gotProject.ID != f.project.ID {
			t.Fatal("project not injected")
		}
	})
}

func TestRequireSprintAccess(t *testing.T) {
	f := newAccessMWFixture(t)

	t.Run("unauthorized", func(t *testing.T) {
		r := newAccessRouter(0, "")
		r.GET("/sprints/:id", RequireSprintAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sprints/%d", f.sprint.ID), nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		r := newAccessRouter(f.outsider.ID, f.outsider.Role)
		r.GET("/sprints/:id", RequireSprintAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sprints/%d", f.sprint.ID), nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		r.GET("/sprints/:id", RequireSprintAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sprints/99999", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success and injects context", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		var gotSprint *model.Sprint
		r.GET("/sprints/:id", RequireSprintAccess(f.db, "id"), func(c *gin.Context) {
			gotSprint = MustSprint(c)
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sprints/%d", f.sprint.ID), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if gotSprint == nil || gotSprint.ID != f.sprint.ID {
			t.Fatal("sprint not injected")
		}
	})
}

func TestRequireTaskAccess(t *testing.T) {
	f := newAccessMWFixture(t)

	t.Run("unauthorized", func(t *testing.T) {
		r := newAccessRouter(0, "")
		r.GET("/tasks/:id", RequireTaskAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", f.task.ID), nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		r := newAccessRouter(f.outsider.ID, f.outsider.Role)
		r.GET("/tasks/:id", RequireTaskAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", f.task.ID), nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		r.GET("/tasks/:id", RequireTaskAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks/99999", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success and injects context", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		var gotTask *model.Task
		r.GET("/tasks/:id", RequireTaskAccess(f.db, "id"), func(c *gin.Context) {
			gotTask = MustTask(c)
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", f.task.ID), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if gotTask == nil || gotTask.ID != f.task.ID {
			t.Fatal("task not injected")
		}
	})
}

func TestRequireTestCaseAccess(t *testing.T) {
	f := newAccessMWFixture(t)

	t.Run("unauthorized", func(t *testing.T) {
		r := newAccessRouter(0, "")
		r.GET("/test-cases/:id", RequireTestCaseAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test-cases/%d", f.tc.ID), nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		r := newAccessRouter(f.outsider.ID, f.outsider.Role)
		r.GET("/test-cases/:id", RequireTestCaseAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test-cases/%d", f.tc.ID), nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		r.GET("/test-cases/:id", RequireTestCaseAccess(f.db, "id"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test-cases/99999", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success and injects context", func(t *testing.T) {
		r := newAccessRouter(f.member.ID, f.member.Role)
		var gotTC *model.TestCase
		var gotStory *model.UserStory
		r.GET("/test-cases/:id", RequireTestCaseAccess(f.db, "id"), func(c *gin.Context) {
			gotTC = MustTestCase(c)
			gotStory = MustStory(c)
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test-cases/%d", f.tc.ID), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if gotTC == nil || gotTC.ID != f.tc.ID {
			t.Fatal("test case not injected")
		}
		if gotStory == nil || gotStory.ID != f.story.ID {
			t.Fatal("story not injected")
		}
	})
}
