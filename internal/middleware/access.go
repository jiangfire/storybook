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
