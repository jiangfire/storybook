package handler

import (
	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MeHandler struct {
	userRepo  repository.UserRepo
	storyRepo *repository.StoryRepository
	taskRepo  repository.TaskRepo
}

func NewMeHandler(db *gorm.DB) *MeHandler {
	return &MeHandler{
		userRepo:  repository.NewUserRepository(db),
		storyRepo: repository.NewStoryRepository(db),
		taskRepo:  repository.NewTaskRepository(db),
	}
}

func (h *MeHandler) Dashboard(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	assignedStories, err := h.storyRepo.ListByAssignee(userID, 20)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	createdStories, err := h.storyRepo.ListByCreator(userID, 20)
	if err != nil {
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

	totalAssigned, err := h.storyRepo.CountByAssignee(userID)
	logging.LogIfErr(err, "count assigned stories failed", "user_id", userID)

	inProgress, err := h.storyRepo.CountByAssigneeAndStatus(userID, model.StoryStatusInProgress)
	logging.LogIfErr(err, "count in-progress stories failed", "user_id", userID)

	completed, err := h.storyRepo.CountByAssigneeAndStatus(userID, model.StoryStatusDone)
	logging.LogIfErr(err, "count completed stories failed", "user_id", userID)

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
