package middleware

import (
	"errors"
	"strconv"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	CtxProjectKey  = "project"
	CtxStoryKey    = "story"
	CtxIsOwnerKey  = "is_owner"
	CtxBugKey      = "bug"
	CtxSprintKey   = "sprint"
	CtxTaskKey     = "task"
	CtxTestCaseKey = "test_case"
)

// RequireProjectAccess parses paramKey as a project ID, verifies the current
// user can access it, and injects *model.Project plus an owner flag into the
// Gin context. On failure it aborts with the appropriate 401/403/404 response.
func RequireProjectAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		projectID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "项目ID无效")
			c.Abort()
			return
		}

		project, isOwner, err := service.EnsureProjectAccess(db, uint(projectID), userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "项目不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// RequireStoryAccess parses paramKey as a story ID, verifies the current user
// can access its project, and injects *model.UserStory, *model.Project and an
// owner flag into the Gin context.
func RequireStoryAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		storyID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "故事ID无效")
			c.Abort()
			return
		}

		story, project, isOwner, err := service.EnsureStoryAccess(db, uint(storyID), userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "故事不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxStoryKey, story)
		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// MustProject returns the project injected by RequireProjectAccess. It panics
// if the middleware was not applied — this is a programmer error, not a
// runtime user error.
func MustProject(c *gin.Context) *model.Project {
	v, ok := c.Get(CtxProjectKey)
	if !ok {
		panic("middleware.MustProject: no project in context; forgot RequireProjectAccess?")
	}
	return v.(*model.Project)
}

// MustStory returns the story injected by RequireStoryAccess. It panics if the
// middleware was not applied.
func MustStory(c *gin.Context) *model.UserStory {
	v, ok := c.Get(CtxStoryKey)
	if !ok {
		panic("middleware.MustStory: no story in context; forgot RequireStoryAccess?")
	}
	return v.(*model.UserStory)
}

// IsProjectOwner returns true when the current user owns the project that was
// injected by RequireProjectAccess or RequireStoryAccess.
func IsProjectOwner(c *gin.Context) bool {
	v, ok := c.Get(CtxIsOwnerKey)
	if !ok {
		return false
	}
	return v.(bool)
}

// RequireBugAccess 把对象级 bug 路由的"取 bugID → 查项目 → 项目成员校验"链路
// 集中到中间件:旧实现里 5 个 handler 方法各自先 parseUintParam,再 loadBugWithAccess,
// 现在统一前置成 *Access。bug 直接持有 ProjectID,所以无需经故事中转。
// 注入:*model.BugReport / *model.Project / is_owner。
func RequireBugAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		bugID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "缺陷ID无效")
			c.Abort()
			return
		}

		// Preload Reporter/Assignee/Story 与原 BugRepository.FindByIDWithDetails 一致:
		// 让 handler 直接渲染响应不必再回查一次。
		var bug model.BugReport
		if err := db.Preload("Reporter").Preload("Assignee").Preload("Story").First(&bug, uint(bugID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "缺陷不存在")
			} else {
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		project, isOwner, err := service.EnsureProjectAccess(db, bug.ProjectID, userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "缺陷不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxBugKey, &bug)
		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// RequireSprintAccess 把对象级 sprint 路由的鉴权链路统一抽到中间件。
// sprint 直接持有 ProjectID,所以无需经故事中转。
// 注入:*model.Sprint / *model.Project / is_owner。
func RequireSprintAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		sprintID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "冲刺ID无效")
			c.Abort()
			return
		}

		var sprint model.Sprint
		if err := db.First(&sprint, uint(sprintID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "冲刺不存在")
			} else {
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		project, isOwner, err := service.EnsureProjectAccess(db, sprint.ProjectID, userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "冲刺不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxSprintKey, &sprint)
		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// RequireTaskAccess 把对象级 task 路由的鉴权链路统一抽到中间件。
// task 直接持有 ProjectID,所以无需经故事中转。Preload Assignee/Creator/Story
// 与原 TaskRepository.FindByIDWithDetails 一致。
// 注入:*model.Task / *model.Project / is_owner。
func RequireTaskAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		taskID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "任务ID无效")
			c.Abort()
			return
		}

		var task model.Task
		if err := db.Preload("Assignee").Preload("Creator").Preload("Story").First(&task, uint(taskID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "任务不存在")
			} else {
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		project, isOwner, err := service.EnsureProjectAccess(db, task.ProjectID, userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "任务不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxTaskKey, &task)
		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// RequireTestCaseAccess 把对象级 test-case 路由的鉴权链路统一抽到中间件。
// test_case 仅持有 StoryID,先取故事再做项目鉴权,与旧 loadStoryWithAccess 路径等价。
// 注入:*model.TestCase / *model.UserStory / *model.Project / is_owner。
func RequireTestCaseAccess(db *gorm.DB, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			api.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		raw := c.Param(paramKey)
		tcID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			api.BadRequest(c, "测试用例ID无效")
			c.Abort()
			return
		}

		var tc model.TestCase
		if err := db.First(&tc, uint(tcID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "测试用例不存在")
			} else {
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		// 经故事走项目鉴权,跟原 TestCaseHandler.loadStoryWithAccess 一致;
		// 故事或项目缺失都按"用户故事不存在"返回,与 P1 行为对齐。
		story, project, isOwner, err := service.EnsureStoryAccess(db, tc.StoryID, userID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				api.NotFound(c, "用户故事不存在")
			case errors.Is(err, service.ErrForbidden):
				api.Forbidden(c, "非项目成员无法访问")
			default:
				api.Internal(c, "服务器内部错误")
			}
			c.Abort()
			return
		}

		c.Set(CtxTestCaseKey, &tc)
		c.Set(CtxStoryKey, story)
		c.Set(CtxProjectKey, project)
		c.Set(CtxIsOwnerKey, isOwner)
		c.Next()
	}
}

// MustBug returns the bug injected by RequireBugAccess. Panics if missing —
// this is a programmer error (route not wired with RequireBugAccess).
func MustBug(c *gin.Context) *model.BugReport {
	v, ok := c.Get(CtxBugKey)
	if !ok {
		panic("middleware.MustBug: no bug in context; forgot RequireBugAccess?")
	}
	return v.(*model.BugReport)
}

// MustSprint returns the sprint injected by RequireSprintAccess.
func MustSprint(c *gin.Context) *model.Sprint {
	v, ok := c.Get(CtxSprintKey)
	if !ok {
		panic("middleware.MustSprint: no sprint in context; forgot RequireSprintAccess?")
	}
	return v.(*model.Sprint)
}

// MustTask returns the task injected by RequireTaskAccess.
func MustTask(c *gin.Context) *model.Task {
	v, ok := c.Get(CtxTaskKey)
	if !ok {
		panic("middleware.MustTask: no task in context; forgot RequireTaskAccess?")
	}
	return v.(*model.Task)
}

// MustTestCase returns the test case injected by RequireTestCaseAccess.
func MustTestCase(c *gin.Context) *model.TestCase {
	v, ok := c.Get(CtxTestCaseKey)
	if !ok {
		panic("middleware.MustTestCase: no test_case in context; forgot RequireTestCaseAccess?")
	}
	return v.(*model.TestCase)
}
