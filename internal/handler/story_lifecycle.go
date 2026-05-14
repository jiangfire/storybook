package handler

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *StoryHandler) ArchiveStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可归档故事")
		return
	}

	changed, err := h.storySvc.Archive(story, userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !changed {
		api.Success(c, "success", gin.H{"id": story.ID, "archived": true})
		return
	}
	api.Success(c, "故事归档成功", gin.H{"id": story.ID, "archived": story.Archived, "updated_at": story.UpdatedAt})
}

func (h *StoryHandler) RestoreStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可恢复故事")
		return
	}

	changed, err := h.storySvc.Restore(story, userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !changed {
		api.Success(c, "success", gin.H{"id": story.ID, "archived": false})
		return
	}
	api.Success(c, "故事恢复成功", gin.H{"id": story.ID, "archived": story.Archived, "updated_at": story.UpdatedAt})
}

func (h *StoryHandler) GetActivities(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := h.db.Model(&model.ActivityLog{}).
		Where("entity_type = ? AND entity_id = ?", "story", story.ID)

	action := strings.TrimSpace(c.Query("action"))
	if action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var logs []model.ActivityLog
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	activities := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		oldValue := any(nil)
		if len(log.OldValue) > 0 {
			parsed, _ := model.ParseJSONMap(log.OldValue)
			oldValue = parsed
		}

		newValue := any(nil)
		if len(log.NewValue) > 0 {
			parsed, _ := model.ParseJSONMap(log.NewValue)
			newValue = parsed
		}

		item := gin.H{
			"id":          log.ID,
			"entity_type": log.EntityType,
			"entity_id":   log.EntityID,
			"action":      log.Action,
			"old_value":   oldValue,
			"new_value":   newValue,
			"created_at":  log.CreatedAt,
		}

		if log.User != nil {
			item["user"] = gin.H{
				"id":         log.User.ID,
				"email":      log.User.Email,
				"avatar_url": log.User.AvatarURL,
			}
		}

		activities = append(activities, item)
	}

	api.Success(c, "success", gin.H{
		"activities": activities,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}
