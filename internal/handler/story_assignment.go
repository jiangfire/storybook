package handler

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *StoryHandler) AssignStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理、技术负责人或管理员可分配故事")
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

	var req assignStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	oldAssigned := story.AssignedTo

	if req.AssignedTo != nil {
		member, err := h.projectRepo.GetMember(story.ProjectID, *req.AssignedTo)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.BadRequest(c, "被分配人不是项目成员")
				return
			}
			api.Internal(c, "服务器内部错误")
			return
		}
		if member.RoleInProject != model.RoleDeveloper {
			api.BadRequest(c, "故事只能分配给开发角色")
			return
		}
		story.AssignedTo = req.AssignedTo
	} else {
		story.AssignedTo = nil
	}

	if err := h.storyRepo.Save(story); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	pid := story.ProjectID
	logging.LogIfErr(h.activityRepo.Create(&model.ActivityLog{
		EntityType: "story",
		EntityID:   story.ID,
		Action:     "assigned",
		UserID:     userID,
		ProjectID:  &pid,
		OldValue: model.MarshalJSON(gin.H{
			"assigned_to": oldAssigned,
		}),
		NewValue: model.MarshalJSON(gin.H{
			"assigned_to": story.AssignedTo,
		}),
	}), "write story activity log", "story_id", story.ID, "action", "assigned")

	if story.AssignedTo != nil && (oldAssigned == nil || *oldAssigned != *story.AssignedTo) {
		h.notifier.Notify(c.Request.Context(), *story.AssignedTo, service.NotificationEvent{
			Type:       model.NotificationStoryAssigned,
			EntityType: model.NotificationEntityStory,
			EntityID:   story.ID,
			ProjectID:  &pid,
			ActorID:    &userID,
			Title:      "新故事指派给你",
			Body:       story.Title,
			Metadata: gin.H{
				"story_id": story.ID,
				"title":    story.Title,
				"status":   story.Status,
			},
		})
	}

	var assignee any
	if story.AssignedTo != nil {
		user, err := h.userRepo.FindByID(*story.AssignedTo)
		if err == nil {
			assignee = gin.H{"id": user.ID, "email": user.Email}
		}
	}

	api.Success(c, "故事分配成功", gin.H{
		"id":          story.ID,
		"assigned_to": assignee,
		"status":      story.Status,
		"updated_at":  story.UpdatedAt,
	})
}

// ReviewStory 审批故事（技术负责人）
func (h *StoryHandler) ReviewStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "仅技术负责人可审批故事")
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

	// 检查审批权限
	canReview, err := service.CanReviewStory(h.db, story, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !canReview {
		api.Forbidden(c, "无权审批此故事")
		return
	}

	var req reviewStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if req.Approved == nil {
		api.BadRequest(c, "approved字段必填")
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if !*req.Approved && comment == "" {
		api.BadRequest(c, "拒绝审批必须填写原因")
		return
	}

	if err := h.storySvc.Review(story, userID, *req.Approved, comment); err != nil {
		if items, ok := serviceValidationItems(err); ok {
			api.BadRequest(c, "参数验证失败", items...)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "审批成功", gin.H{
		"id":     story.ID,
		"status": story.Status,
		"review_status": func() string {
			if strings.TrimSpace(story.ReviewStatus) == "" {
				return model.ReviewStatusPending
			}
			return story.ReviewStatus
		}(),
		"review_comment": story.ReviewComment,
		"reviewed_by":    story.ReviewedBy,
		"reviewed_at":    story.ReviewedAt,
		"approved":       *req.Approved,
	})
}
