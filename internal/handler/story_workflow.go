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
)

func (h *StoryHandler) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	var req updateStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := h.storySvc.UpdateStatus(story, userID, req.Status, req.Position, actorFromContext(c, userID)); err != nil {
		if errors.Is(err, service.ErrInvalidTransition) {
			api.BadRequest(c, "非法状态流转")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "状态更新成功", gin.H{
		"id":         story.ID,
		"status":     story.Status,
		"position":   story.Position,
		"updated_at": story.UpdatedAt,
	})
}

func (h *StoryHandler) ClaimStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleDeveloper && role != model.RoleAdmin {
		api.Forbidden(c, "只有开发人员可以领取故事")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	if err := h.storySvc.Claim(story, userID); err != nil {
		if errors.Is(err, service.ErrAlreadyClaimed) {
			api.Conflict(c, "故事已被其他人领取")
			return
		}
		if errors.Is(err, service.ErrClaimNotAllowed) {
			api.Conflict(c, "当前状态不允许领取")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var user model.User
	if u, err := h.userRepo.FindByID(userID); err != nil {
		logging.LogIfErr(err, "load story user failed", "user_id", userID)
	} else {
		user = *u
	}

	api.Success(c, "故事领取成功", gin.H{
		"id":          story.ID,
		"assigned_to": gin.H{"id": user.ID, "email": user.Email},
		"status":      story.Status,
		"updated_at":  story.UpdatedAt,
	})
}

func (h *StoryHandler) ReleaseStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	if err := h.storySvc.Release(story, userID, role); err != nil {
		if errors.Is(err, service.ErrNotClaimed) {
			api.BadRequest(c, "故事未被领取")
			return
		}
		if errors.Is(err, service.ErrNoReleasePermission) {
			api.Forbidden(c, "无权释放该故事")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "故事释放成功", gin.H{
		"id":          story.ID,
		"assigned_to": nil,
		"status":      story.Status,
		"updated_at":  story.UpdatedAt,
	})
}

func (h *StoryHandler) UpdateACStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	acID := strings.TrimSpace(c.Param("acID"))
	if acID == "" {
		api.BadRequest(c, "ac_id不能为空")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	var req updateACStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	now, err := h.storySvc.UpdateACStatus(story, userID, acID, req.Status, req.Evidence, req.Notes, actorFromContext(c, userID))
	if err != nil {
		if errors.Is(err, service.ErrACNotFound) {
			api.NotFound(c, "AC不存在")
			return
		}
		if errors.Is(err, service.ErrACCorrupted) {
			api.Internal(c, "AC数据损坏")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "AC状态更新成功", gin.H{
		"id":         acID,
		"status":     req.Status,
		"evidence":   req.Evidence,
		"updated_at": now,
	})
}

func (h *StoryHandler) AddCodeReference(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleDeveloper && role != model.RoleAdmin {
		api.Forbidden(c, "仅开发人员可关联代码")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	var req addCodeRefRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	refs, err := h.storySvc.AddCodeReference(story, userID, req.Reference)
	if err != nil {
		if items, ok := serviceValidationItems(err); ok && len(items) > 0 {
			api.BadRequest(c, items[0].Message)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "代码关联成功", gin.H{"story_id": story.ID, "code_references": refs})
}

type addACRequest struct {
	Description string `json:"description" binding:"required,min=1,max=500"`
	Ref         string `json:"ref" binding:"omitempty,max=200"`
	Notes       string `json:"notes" binding:"omitempty,max=1000"`
}

type updateACRequest struct {
	Description *string `json:"description" binding:"omitempty,min=1,max=500"`
	Ref         *string `json:"ref" binding:"omitempty,max=200"`
	Notes       *string `json:"notes" binding:"omitempty,max=1000"`
	Order       *int    `json:"order" binding:"omitempty,min=1"`
}

// AddAC appends a new acceptance criterion to a story.
func (h *StoryHandler) AddAC(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	var req addACRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	ac, err := h.storySvc.AddAC(story, userID, req.Description, req.Ref, req.Notes, actorFromContext(c, userID))
	if err != nil {
		if items, ok := serviceValidationItems(err); ok && len(items) > 0 {
			api.BadRequest(c, items[0].Message)
			return
		}
		if errors.Is(err, service.ErrACCorrupted) {
			api.Internal(c, "AC数据损坏")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "AC新增成功", ac)
}

// UpdateAC selectively edits AC content (description/ref/notes/order). Status
// changes still go through UpdateACStatus.
func (h *StoryHandler) UpdateAC(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	acID := strings.TrimSpace(c.Param("acID"))
	if acID == "" {
		api.BadRequest(c, "ac_id不能为空")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	var req updateACRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if req.Description == nil && req.Ref == nil && req.Notes == nil && req.Order == nil {
		api.BadRequest(c, "未提供需要更新的字段")
		return
	}

	ac, err := h.storySvc.UpdateAC(story, userID, acID, req.Description, req.Ref, req.Notes, req.Order, actorFromContext(c, userID))
	if err != nil {
		if items, ok := serviceValidationItems(err); ok && len(items) > 0 {
			api.BadRequest(c, items[0].Message)
			return
		}
		if errors.Is(err, service.ErrACNotFound) {
			api.NotFound(c, "AC不存在")
			return
		}
		if errors.Is(err, service.ErrACCorrupted) {
			api.Internal(c, "AC数据损坏")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "AC更新成功", ac)
}

// DeleteAC removes a single criterion by ID.
func (h *StoryHandler) DeleteAC(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	acID := strings.TrimSpace(c.Param("acID"))
	if acID == "" {
		api.BadRequest(c, "ac_id不能为空")
		return
	}

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		respondAccessError(c, err, "用户故事不存在")
		return
	}

	if err := h.storySvc.RemoveAC(story, userID, acID, actorFromContext(c, userID)); err != nil {
		if errors.Is(err, service.ErrACNotFound) {
			api.NotFound(c, "AC不存在")
			return
		}
		if errors.Is(err, service.ErrACCorrupted) {
			api.Internal(c, "AC数据损坏")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "AC删除成功", gin.H{"story_id": story.ID, "ac_id": acID})
}
