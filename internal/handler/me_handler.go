package handler

import (
	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MeHandler struct {
	db *gorm.DB
}

func NewMeHandler(db *gorm.DB) *MeHandler {
	return &MeHandler{db: db}
}

func (h *MeHandler) Dashboard(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	var user model.User
	if err := h.db.Select("id, email, role").First(&user, userID).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var assignedStories []model.UserStory
	if err := h.db.
		Preload("Project").
		Where("assigned_to = ?", userID).
		Order("updated_at DESC").
		Limit(20).
		Find(&assignedStories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var createdStories []model.UserStory
	if err := h.db.
		Preload("Project").
		Where("created_by = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&createdStories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	assignedPayload := make([]gin.H, 0, len(assignedStories))
	for _, s := range assignedStories {
		assignedPayload = append(assignedPayload, storySummaryWithProject(s))
	}

	createdPayload := make([]gin.H, 0, len(createdStories))
	for _, s := range createdStories {
		createdPayload = append(createdPayload, storySummaryWithProject(s))
	}

	var totalAssigned int64
	_ = h.db.Model(&model.UserStory{}).Where("assigned_to = ?", userID).Count(&totalAssigned).Error

	var inProgress int64
	_ = h.db.Model(&model.UserStory{}).Where("assigned_to = ? AND status = ?", userID, model.StoryStatusInProgress).Count(&inProgress).Error

	var completed int64
	_ = h.db.Model(&model.UserStory{}).Where("assigned_to = ? AND status = ?", userID, model.StoryStatusDone).Count(&completed).Error

	api.Success(c, "success", gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
		"my_stories": gin.H{
			"assigned": assignedPayload,
			"created":  createdPayload,
		},
		"statistics": gin.H{
			"total_assigned": totalAssigned,
			"in_progress":    inProgress,
			"completed":      completed,
		},
	})
}

func storySummaryWithProject(story model.UserStory) gin.H {
	item := gin.H{
		"id":       story.ID,
		"title":    story.Title,
		"status":   story.Status,
		"priority": story.Priority,
	}

	if story.Project != nil {
		item["project"] = story.Project.Name
	}

	return item
}
