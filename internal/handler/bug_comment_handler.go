package handler

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BugCommentHandler struct {
	db           *gorm.DB
	commentRepo  *repository.BugCommentRepository
	bugRepo      *repository.BugRepository
	activityRepo *repository.ActivityLogRepository
	events       EventPublisher
}

func NewBugCommentHandler(db *gorm.DB) *BugCommentHandler {
	return &BugCommentHandler{
		db:           db,
		commentRepo:  repository.NewBugCommentRepository(db),
		bugRepo:      repository.NewBugRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
	}
}

// WithEvents wires the WebSocket broadcaster post-construction so we can keep
// the constructor signature trivial for tests that don't need realtime fan-out.
func (h *BugCommentHandler) WithEvents(p EventPublisher) *BugCommentHandler {
	if p != nil {
		h.events = p
	}
	return h
}

type createBugCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=5000"`
}

type updateBugCommentRequest struct {
	Body    string `json:"body" binding:"required,min=1,max=5000"`
	Version int    `json:"version"`
}

func (h *BugCommentHandler) loadBugForUser(bugID, userID uint) (*model.BugReport, error) {
	bug, err := h.bugRepo.FindByID(bugID)
	if err != nil {
		return nil, err
	}
	if _, _, err := ensureProjectAccess(h.db, bug.ProjectID, userID); err != nil {
		return nil, err
	}
	return bug, nil
}

func (h *BugCommentHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	bugID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "缺陷ID无效")
		return
	}
	bug, err := h.loadBugForUser(bugID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "缺陷不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var req createBugCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		api.BadRequest(c, "评论内容不能为空")
		return
	}

	comment := model.BugComment{
		BugID:    bug.ID,
		AuthorID: userID,
		Body:     body,
	}
	if err := h.commentRepo.Create(&comment); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// Reload to attach Author for response rendering.
	if fresh, err := h.commentRepo.FindByIDWithAuthor(comment.ID); err == nil {
		comment = *fresh
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "comment_added",
		map[string]any{"comment_id": comment.ID, "preview": truncateForLog(body, 120)}, nil),
		"write bug comment activity log", "bug_id", bug.ID)

	if h.events != nil {
		h.events.BroadcastProject(bug.ProjectID, "bug.comment.created", gin.H{
			"bug_id":     bug.ID,
			"comment_id": comment.ID,
			"author_id":  userID,
		})
	}

	api.Success(c, "评论创建成功", comment)
}

func (h *BugCommentHandler) List(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	bugID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "缺陷ID无效")
		return
	}
	bug, err := h.loadBugForUser(bugID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "缺陷不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	items, err := h.commentRepo.ListByBug(bug.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "success", gin.H{
		"bug_id":   bug.ID,
		"items":    items,
		"total":    len(items),
	})
}

func (h *BugCommentHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	commentID, ok := parseUintParam(c, "commentID")
	if !ok {
		api.BadRequest(c, "评论ID无效")
		return
	}

	comment, err := h.commentRepo.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "评论不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	bug, err := h.loadBugForUser(comment.BugID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "缺陷不存在")
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
	if comment.AuthorID != userID && role != model.RoleAdmin {
		api.Forbidden(c, "仅作者本人或管理员可编辑评论")
		return
	}

	var req updateBugCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		api.BadRequest(c, "评论内容不能为空")
		return
	}

	fields := map[string]any{"body": body}
	if err := h.commentRepo.UpdateWithVersion(comment.ID, req.Version, fields); err != nil {
		if strings.HasPrefix(err.Error(), "version conflict") {
			api.Conflict(c, "评论已被修改,请刷新后重试")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	reloaded, err := h.commentRepo.FindByIDWithAuthor(comment.ID)
	if err == nil {
		comment = reloaded
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "comment_updated",
		map[string]any{"comment_id": comment.ID, "preview": truncateForLog(body, 120)}, nil),
		"write bug comment activity log", "bug_id", bug.ID)

	if h.events != nil {
		h.events.BroadcastProject(bug.ProjectID, "bug.comment.updated", gin.H{
			"bug_id":     bug.ID,
			"comment_id": comment.ID,
			"actor_id":   userID,
		})
	}

	api.Success(c, "评论更新成功", comment)
}

func (h *BugCommentHandler) Delete(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	commentID, ok := parseUintParam(c, "commentID")
	if !ok {
		api.BadRequest(c, "评论ID无效")
		return
	}

	comment, err := h.commentRepo.FindByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "评论不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	bug, err := h.loadBugForUser(comment.BugID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "缺陷不存在")
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
	if comment.AuthorID != userID && role != model.RoleAdmin {
		api.Forbidden(c, "仅作者本人或管理员可删除评论")
		return
	}

	if err := h.commentRepo.Delete(comment.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "comment_deleted",
		map[string]any{"comment_id": comment.ID}, nil),
		"write bug comment activity log", "bug_id", bug.ID)

	if h.events != nil {
		h.events.BroadcastProject(bug.ProjectID, "bug.comment.deleted", gin.H{
			"bug_id":     bug.ID,
			"comment_id": comment.ID,
			"actor_id":   userID,
		})
	}

	api.Success(c, "评论删除成功", gin.H{"id": comment.ID})
}

// truncateForLog clips a string to n runes (UTF-8 safe) so
// activity-log NewValue payloads stay small and readable.
func truncateForLog(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
