package handler

import (
	"errors"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BugHandler struct {
	db           *gorm.DB
	bugRepo      repository.BugRepo
	storyRepo    repository.StoryRepo
	userRepo     repository.UserRepo
	projectRepo  repository.ProjectRepo
	activityRepo repository.ActivityRepo
	notifier     service.Notifier
	events       service.EventPublisher
}

func NewBugHandler(db *gorm.DB) *BugHandler {
	return &BugHandler{
		db:           db,
		bugRepo:      repository.NewBugRepository(db),
		storyRepo:    repository.NewStoryRepository(db),
		userRepo:     repository.NewUserRepository(db),
		projectRepo:  repository.NewProjectRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
		notifier:     service.NoopNotifier{},
	}
}

// WithNotifier wires in the notification service after construction so the
// router can compose handlers + notifier in one place without forcing every
// caller (tests, scripts) to supply one.
func (h *BugHandler) WithNotifier(n service.Notifier) *BugHandler {
	if n != nil {
		h.notifier = n
	}
	return h
}

// WithEvents wires a WebSocket broadcaster after construction so project-level
// bug events can fan out without changing the existing NewBugHandler signature
// (preserves the *gorm.DB-only constructor used by tests).
func (h *BugHandler) WithEvents(p service.EventPublisher) *BugHandler {
	if p != nil {
		h.events = p
	}
	return h
}

type createBugRequest struct {
	StoryID      *uint  `json:"story_id"`
	Title        string `json:"title" binding:"required,min=2,max=255"`
	Description  string `json:"description"`
	Severity     string `json:"severity" binding:"required,oneof=low medium high critical"`
	AssignedToID *uint  `json:"assigned_to"`
}

type updateBugStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open in_progress resolved closed"`
}

type updateBugRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty,max=5000"`
	Severity    *string `json:"severity" binding:"omitempty,oneof=low medium high critical"`
	Version     int     `json:"version"`
}

type assignBugRequest struct {
	AssignedToID *uint `json:"assigned_to"`
}

var errBugAssigneeRole = errors.New("bug_assignee_role")

func (h *BugHandler) Create(c *gin.Context) {
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可创建缺陷")
		return
	}

	projectID := project.ID

	var req createBugRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if req.StoryID != nil {
		story, err := h.storyRepo.FindByID(*req.StoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "关联故事不存在")
				return
			}
			api.Internal(c, "服务器内部错误")
			return
		}
		if story.ProjectID != projectID {
			api.BadRequest(c, "关联故事不属于当前项目")
			return
		}
	}

	if req.AssignedToID != nil {
		if err := h.ensureAssignableUser(projectID, *req.AssignedToID); err != nil {
			h.handleAssignUserErr(c, err)
			return
		}
	}

	bug := model.BugReport{
		ProjectID:   project.ID,
		StoryID:     req.StoryID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Severity:    req.Severity,
		Status:      model.BugStatusOpen,
		ReportedBy:  userID,
		AssignedTo:  req.AssignedToID,
	}
	if err := h.bugRepo.Create(&bug); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &project.ID, userID, "bug", bug.ID, "created",
		nil, map[string]any{"title": bug.Title, "severity": bug.Severity, "status": bug.Status}),
		"write bug activity log", "bug_id", bug.ID, "action", "created")

	if bug.AssignedTo != nil {
		h.notifier.Notify(c.Request.Context(), *bug.AssignedTo, service.NotificationEvent{
			Type:       model.NotificationBugAssigned,
			EntityType: model.NotificationEntityBug,
			EntityID:   bug.ID,
			ProjectID:  &project.ID,
			ActorID:    &userID,
			Title:      "新缺陷指派给你",
			Body:       bug.Title,
			Metadata: gin.H{
				"bug_id":   bug.ID,
				"severity": bug.Severity,
				"status":   bug.Status,
			},
		})
	}

	api.Success(c, "缺陷创建成功", bug)
}

func (h *BugHandler) List(c *gin.Context) {
	project := middleware.MustProject(c)

	bugs, err := h.bugRepo.ListByProjectUnpaged(project.ID, repository.BugListOptions{
		ListOptions: repository.ListOptions{
			Status: strings.TrimSpace(c.Query("status")),
		},
		Severity: strings.TrimSpace(c.Query("severity")),
		Assignee: strings.TrimSpace(c.Query("assignee")),
	})
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(bugs))
	for _, b := range bugs {
		row := gin.H{
			"id":          b.ID,
			"project_id":  b.ProjectID,
			"story_id":    b.StoryID,
			"title":       b.Title,
			"description": b.Description,
			"severity":    b.Severity,
			"status":      b.Status,
			"resolved_at": b.ResolvedAt,
			"created_at":  b.CreatedAt,
			"updated_at":  b.UpdatedAt,
		}
		if b.Reporter != nil {
			row["reported_by"] = gin.H{"id": b.Reporter.ID, "email": b.Reporter.Email}
		}
		if b.Assignee != nil {
			row["assigned_to"] = gin.H{"id": b.Assignee.ID, "email": b.Assignee.Email}
		}
		items = append(items, row)
	}

	api.Success(c, "success", gin.H{"bugs": items})
}

func (h *BugHandler) Get(c *gin.Context) {
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

	bug, err := h.loadBugWithAccess(bugID, userID)
	if err != nil {
		respondAccessError(c, err, "缺陷不存在")
		return
	}

	data := gin.H{
		"id":          bug.ID,
		"project_id":  bug.ProjectID,
		"story_id":    bug.StoryID,
		"title":       bug.Title,
		"description": bug.Description,
		"severity":    bug.Severity,
		"status":      bug.Status,
		"resolved_at": bug.ResolvedAt,
		"created_at":  bug.CreatedAt,
		"updated_at":  bug.UpdatedAt,
	}
	if bug.Reporter != nil {
		data["reported_by"] = gin.H{"id": bug.Reporter.ID, "email": bug.Reporter.Email}
	}
	if bug.Assignee != nil {
		data["assigned_to"] = gin.H{"id": bug.Assignee.ID, "email": bug.Assignee.Email}
	}

	api.Success(c, "success", data)
}

func (h *BugHandler) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleDeveloper && role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅开发或测试可更新缺陷状态")
		return
	}

	bugID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "缺陷ID无效")
		return
	}

	bug, err := h.loadBugWithAccess(bugID, userID)
	if err != nil {
		respondAccessError(c, err, "缺陷不存在")
		return
	}

	var req updateBugStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if !service.Workflow.CanBugTransit(bug.Status, req.Status) {
		api.BadRequest(c, "非法缺陷状态流转")
		return
	}

	oldStatus := bug.Status
	bug.Status = req.Status
	if bug.Status == model.BugStatusResolved || bug.Status == model.BugStatusClosed {
		now := time.Now()
		bug.ResolvedAt = &now
	} else {
		bug.ResolvedAt = nil
	}

	if err := h.bugRepo.Save(bug); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "status_changed",
		map[string]any{"status": oldStatus}, map[string]any{"status": bug.Status}),
		"write bug activity log", "bug_id", bug.ID, "action", "status_changed")

	api.Success(c, "缺陷状态更新成功", gin.H{
		"id":          bug.ID,
		"status":      bug.Status,
		"resolved_at": bug.ResolvedAt,
		"updated_at":  bug.UpdatedAt,
	})
}

func (h *BugHandler) Assign(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可指派缺陷")
		return
	}

	bugID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "缺陷ID无效")
		return
	}

	bug, err := h.loadBugWithAccess(bugID, userID)
	if err != nil {
		respondAccessError(c, err, "缺陷不存在")
		return
	}

	var req assignBugRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if req.AssignedToID != nil {
		if err := h.ensureAssignableUser(bug.ProjectID, *req.AssignedToID); err != nil {
			h.handleAssignUserErr(c, err)
			return
		}
	}

	oldAssigned := bug.AssignedTo
	bug.AssignedTo = req.AssignedToID
	if err := h.bugRepo.Save(bug); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "assigned",
		map[string]any{"assigned_to": oldAssigned}, map[string]any{"assigned_to": bug.AssignedTo}),
		"write bug activity log", "bug_id", bug.ID, "action", "assigned")

	if bug.AssignedTo != nil && (oldAssigned == nil || *oldAssigned != *bug.AssignedTo) {
		h.notifier.Notify(c.Request.Context(), *bug.AssignedTo, service.NotificationEvent{
			Type:       model.NotificationBugAssigned,
			EntityType: model.NotificationEntityBug,
			EntityID:   bug.ID,
			ProjectID:  &bug.ProjectID,
			ActorID:    &userID,
			Title:      "新缺陷指派给你",
			Body:       bug.Title,
			Metadata: gin.H{
				"bug_id":   bug.ID,
				"severity": bug.Severity,
				"status":   bug.Status,
			},
		})
	}

	api.Success(c, "缺陷指派成功", gin.H{
		"id":          bug.ID,
		"assigned_to": bug.AssignedTo,
		"updated_at":  bug.UpdatedAt,
	})
}

// Update edits a bug's editable fields (title/description/severity) using
// optimistic locking on Version. status/assignee/reporter/projectID are NOT
// mutable here — UpdateStatus and Assign cover those paths.
func (h *BugHandler) Update(c *gin.Context) {
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

	bug, err := h.loadBugWithAccess(bugID, userID)
	if err != nil {
		respondAccessError(c, err, "缺陷不存在")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if bug.ReportedBy != userID && role != model.RoleAdmin && role != model.RoleProduct {
		api.Forbidden(c, "仅缺陷上报人、产品经理或管理员可编辑")
		return
	}

	var req updateBugRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	fields := map[string]any{}
	oldValue := gin.H{}
	newValue := gin.H{}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			api.BadRequest(c, "title 不能为空")
			return
		}
		oldValue["title"] = bug.Title
		newValue["title"] = trimmed
		fields["title"] = trimmed
	}
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		oldValue["description"] = bug.Description
		newValue["description"] = trimmed
		fields["description"] = trimmed
	}
	if req.Severity != nil {
		oldValue["severity"] = bug.Severity
		newValue["severity"] = *req.Severity
		fields["severity"] = *req.Severity
	}
	if len(fields) == 0 {
		api.BadRequest(c, "未提供需要更新的字段")
		return
	}

	if err := h.bugRepo.UpdateWithVersion(bug.ID, req.Version, fields); err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			api.Conflict(c, "缺陷已被他人修改，请刷新后重试")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	updated, err := h.bugRepo.FindByIDWithDetails(bug.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &updated.ProjectID, userID, "bug", updated.ID, "updated",
		oldValue, newValue),
		"write bug activity log", "bug_id", updated.ID, "action", "updated")

	if h.events != nil {
		h.events.BroadcastProject(updated.ProjectID, "bug.updated", gin.H{
			"bug_id":     updated.ID,
			"project_id": updated.ProjectID,
			"updated_by": userID,
		})
	}

	api.Success(c, "缺陷更新成功", gin.H{
		"id":          updated.ID,
		"title":       updated.Title,
		"description": updated.Description,
		"severity":    updated.Severity,
		"status":      updated.Status,
		"version":     updated.Version,
		"updated_at":  updated.UpdatedAt,
	})
}

// Delete soft-deletes a bug. Reporter or admin only.
func (h *BugHandler) Delete(c *gin.Context) {
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

	bug, err := h.loadBugWithAccess(bugID, userID)
	if err != nil {
		respondAccessError(c, err, "缺陷不存在")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if bug.ReportedBy != userID && role != model.RoleAdmin {
		api.Forbidden(c, "仅缺陷上报人或管理员可删除")
		return
	}

	if err := h.bugRepo.Delete(bug.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &bug.ProjectID, userID, "bug", bug.ID, "deleted",
		map[string]any{"title": bug.Title, "status": bug.Status}, nil),
		"write bug activity log", "bug_id", bug.ID, "action", "deleted")

	if h.events != nil {
		h.events.BroadcastProject(bug.ProjectID, "bug.deleted", gin.H{
			"bug_id":     bug.ID,
			"project_id": bug.ProjectID,
			"deleted_by": userID,
		})
	}

	api.Success(c, "缺陷删除成功", gin.H{"id": bug.ID})
}

func (h *BugHandler) loadBugWithAccess(bugID, userID uint) (*model.BugReport, error) {
	bug, err := h.bugRepo.FindByIDWithDetails(bugID)
	if err != nil {
		return nil, err
	}

	if _, _, err := ensureProjectAccess(h.db, bug.ProjectID, userID); err != nil {
		return nil, err
	}
	return bug, nil
}

func (h *BugHandler) ensureAssignableUser(projectID, userID uint) error {
	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		return err
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if user.Role != model.RoleDeveloper && user.Role != model.RoleAdmin {
		return errBugAssigneeRole
	}
	return nil
}

func (h *BugHandler) handleAssignUserErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errForbidden):
		api.BadRequest(c, "指派用户不是项目成员")
	case errors.Is(err, gorm.ErrRecordNotFound):
		api.NotFound(c, "指派用户不存在")
	case errors.Is(err, errBugAssigneeRole):
		api.BadRequest(c, "缺陷仅可指派给开发角色")
	default:
		api.Internal(c, "服务器内部错误")
	}
}
