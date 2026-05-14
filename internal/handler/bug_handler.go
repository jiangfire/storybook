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
	db         *gorm.DB
	bugRepo    *repository.BugRepository
	storyRepo  *repository.StoryRepository
	userRepo   *repository.UserRepository
	projectRepo *repository.ProjectRepository
}

func NewBugHandler(db *gorm.DB) *BugHandler {
	return &BugHandler{
		db:          db,
		bugRepo:     repository.NewBugRepository(db),
		storyRepo:   repository.NewStoryRepository(db),
		userRepo:    repository.NewUserRepository(db),
		projectRepo: repository.NewProjectRepository(db),
	}
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

type assignBugRequest struct {
	AssignedToID *uint `json:"assigned_to"`
}

var errBugAssigneeRole = errors.New("bug_assignee_role")

func (h *BugHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可创建缺陷")
		return
	}

	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}

	project, _, err := ensureProjectAccess(h.db, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "项目不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

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

	pid := project.ID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "bug",
		EntityID:   bug.ID,
		Action:     "created",
		UserID:     userID,
		ProjectID:  &pid,
		NewValue:   model.MarshalJSON(gin.H{"title": bug.Title, "severity": bug.Severity, "status": bug.Status}),
	}).Error, "write bug activity log", "bug_id", bug.ID, "action", "created")

	api.Success(c, "缺陷创建成功", bug)
}

func (h *BugHandler) List(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "项目不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	query := h.db.Model(&model.BugReport{}).Where("project_id = ?", projectID)
	status := strings.TrimSpace(c.Query("status"))
	if status != "" {
		query = query.Where("status = ?", status)
	}
	severity := strings.TrimSpace(c.Query("severity"))
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	assignee := strings.TrimSpace(c.Query("assignee"))
	if assignee != "" {
		query = query.Where("assigned_to = ?", assignee)
	}

	var bugs []model.BugReport
	if err := query.Preload("Reporter").Preload("Assignee").Order("created_at DESC").Find(&bugs).Error; err != nil {
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

	pid := bug.ProjectID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "bug",
		EntityID:   bug.ID,
		Action:     "status_changed",
		UserID:     userID,
		ProjectID:  &pid,
		OldValue:   model.MarshalJSON(gin.H{"status": oldStatus}),
		NewValue:   model.MarshalJSON(gin.H{"status": bug.Status}),
	}).Error, "write bug activity log", "bug_id", bug.ID, "action", "status_changed")

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

	pid := bug.ProjectID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "bug",
		EntityID:   bug.ID,
		Action:     "assigned",
		UserID:     userID,
		ProjectID:  &pid,
		OldValue:   model.MarshalJSON(gin.H{"assigned_to": oldAssigned}),
		NewValue:   model.MarshalJSON(gin.H{"assigned_to": bug.AssignedTo}),
	}).Error, "write bug activity log", "bug_id", bug.ID, "action", "assigned")

	api.Success(c, "缺陷指派成功", gin.H{
		"id":          bug.ID,
		"assigned_to": bug.AssignedTo,
		"updated_at":  bug.UpdatedAt,
	})
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

func (h *BugHandler) isProjectMember(projectID, userID uint) bool {
	project, err := h.projectRepo.FindByID(projectID)
	if err != nil {
		return false
	}
	if project.OwnerID == userID {
		return true
	}

	isMember, err := h.projectRepo.IsMember(projectID, userID)
	if err != nil {
		return false
	}
	return isMember
}

func (h *BugHandler) ensureAssignableUser(projectID, userID uint) error {
	if !h.isProjectMember(projectID, userID) {
		return errForbidden
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
