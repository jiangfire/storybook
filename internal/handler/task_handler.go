package handler

import (
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskHandler struct {
	db      *gorm.DB
	events  EventPublisher
	taskSvc *service.TaskService
}

func NewTaskHandler(db *gorm.DB, events EventPublisher) *TaskHandler {
	return &TaskHandler{
		db:      db,
		events:  events,
		taskSvc: service.NewTaskService(db, events),
	}
}

type createTaskRequest struct {
	Title          string  `json:"title" binding:"required,min=2,max=255"`
	Description    string  `json:"description" binding:"max=2000"`
	Priority       int     `json:"priority" binding:"min=0,max=4"`
	EstimatedHours float64 `json:"estimated_hours" binding:"min=0,max=500"`
}

type updateTaskRequest struct {
	Title          *string  `json:"title"`
	Description    *string  `json:"description"`
	Priority       *int     `json:"priority"`
	EstimatedHours *float64 `json:"estimated_hours"`
}

type updateTaskStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=todo in_progress blocked done"`
}

type updateTaskProgressRequest struct {
	Progress int `json:"progress" binding:"min=0,max=100"`
}

type addTaskCodeRefRequest struct {
	Reference string `json:"reference" binding:"required,max=500"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleDeveloper && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品或开发可创建子任务")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, project, _, err := ensureStoryAccess(h.db, storyID, userID)
	if err != nil {
		h.handleStoryAccessErr(c, err)
		return
	}

	var req createTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	task, err := h.taskSvc.Create(service.CreateTaskInput{
		ProjectID:      project.ID,
		StoryID:        story.ID,
		CreatedBy:      userID,
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		EstimatedHours: req.EstimatedHours,
	})
	if err != nil {
		if items, ok := serviceValidationItems(err); ok {
			api.BadRequest(c, "参数验证失败", items...)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "子任务创建成功", task)
}

func (h *TaskHandler) SplitFromAC(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可拆分子任务")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, project, _, err := ensureStoryAccess(h.db, storyID, userID)
	if err != nil {
		h.handleStoryAccessErr(c, err)
		return
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		api.Internal(c, "AC数据损坏")
		return
	}

	created, err := h.taskSvc.SplitFromAC(story, project.ID, userID, criteria)
	if err != nil {
		if errors.Is(err, service.ErrNoSplittableAC) {
			api.BadRequest(c, "当前故事无可拆分的AC")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "AC拆分子任务完成", gin.H{
		"story_id":      story.ID,
		"created_count": len(created),
		"tasks":         created,
	})
}

func (h *TaskHandler) ListByStory(c *gin.Context) {
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

	story, _, _, err := ensureStoryAccess(h.db, storyID, userID)
	if err != nil {
		h.handleStoryAccessErr(c, err)
		return
	}

	query := h.db.Model(&model.Task{}).Where("story_id = ?", story.ID)
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if assignee := strings.TrimSpace(c.Query("assignee")); assignee != "" {
		query = query.Where("assigned_to = ?", assignee)
	}

	var tasks []model.Task
	if err := query.Preload("Assignee").Order("priority DESC, id ASC").Find(&tasks).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		item := gin.H{
			"id":              t.ID,
			"project_id":      t.ProjectID,
			"story_id":        t.StoryID,
			"title":           t.Title,
			"description":     t.Description,
			"status":          t.Status,
			"priority":        t.Priority,
			"progress":        t.Progress,
			"estimated_hours": t.EstimatedHours,
			"code_references": parseStringArrayJSON(t.CodeReferences),
			"created_at":      t.CreatedAt,
			"updated_at":      t.UpdatedAt,
		}
		if t.Assignee != nil {
			item["assigned_to"] = gin.H{"id": t.Assignee.ID, "email": t.Assignee.Email}
		}
		items = append(items, item)
	}

	api.Success(c, "success", gin.H{"tasks": items})
}

func (h *TaskHandler) Get(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}

	task, _, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	api.Success(c, "success", h.taskPayload(task))
}

func (h *TaskHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}

	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin && task.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可编辑任务")
		return
	}

	var req updateTaskRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	changed, err := h.taskSvc.Update(task, project.ID, userID, service.UpdateTaskInput{
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		EstimatedHours: req.EstimatedHours,
	})
	if err != nil {
		if items, ok := serviceValidationItems(err); ok {
			api.BadRequest(c, "参数验证失败", items...)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	if !changed {
		api.Success(c, "success", h.taskPayload(task))
		return
	}
	api.Success(c, "任务更新成功", h.taskPayload(task))
}

func (h *TaskHandler) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	role, _ := middleware.CurrentRole(c)

	var req updateTaskStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := h.taskSvc.UpdateStatus(task, project.ID, userID, role, req.Status, actorFromContext(c, userID)); err != nil {
		if errors.Is(err, service.ErrNoStatusPermission) {
			api.Forbidden(c, "仅负责人、产品经理或管理员可更新任务状态")
			return
		}
		if errors.Is(err, service.ErrInvalidTransition) {
			api.BadRequest(c, "非法任务状态流转")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "任务状态更新成功", h.taskPayload(task))
}

func (h *TaskHandler) UpdateProgress(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	role, _ := middleware.CurrentRole(c)

	var req updateTaskProgressRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if err := h.taskSvc.UpdateProgress(task, project.ID, userID, role, req.Progress, actorFromContext(c, userID)); err != nil {
		if errors.Is(err, service.ErrNoProgressPermission) {
			api.Forbidden(c, "仅负责人、产品经理或管理员可更新任务进度")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "任务进度更新成功", h.taskPayload(task))
}

func (h *TaskHandler) Claim(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleDeveloper && role != model.RoleAdmin {
		api.Forbidden(c, "仅开发人员可领取任务")
		return
	}

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	if err := h.taskSvc.Claim(task, project.ID, userID); err != nil {
		if errors.Is(err, service.ErrAlreadyClaimed) {
			api.Conflict(c, "任务已被其他人领取")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "任务领取成功", h.taskPayload(task))
}

func (h *TaskHandler) Release(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	role, _ := middleware.CurrentRole(c)
	if err := h.taskSvc.Release(task, project.ID, userID, role); err != nil {
		if errors.Is(err, service.ErrNotClaimed) {
			api.BadRequest(c, "任务未被领取")
			return
		}
		if errors.Is(err, service.ErrNoReleasePermission) {
			api.Forbidden(c, "无权释放该任务")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "任务释放成功", h.taskPayload(task))
}

func (h *TaskHandler) AddCodeReference(c *gin.Context) {
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

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	var req addTaskCodeRefRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	refs, err := h.taskSvc.AddCodeReference(task, project.ID, userID, req.Reference)
	if err != nil {
		if items, ok := serviceValidationItems(err); ok && len(items) > 0 {
			api.BadRequest(c, items[0].Message)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "任务代码关联成功", gin.H{"task_id": task.ID, "code_references": refs})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	taskID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "任务ID无效")
		return
	}
	task, project, err := h.loadTaskWithAccess(taskID, userID)
	if err != nil {
		h.handleTaskAccessErr(c, err)
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin && task.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可删除任务")
		return
	}

	if err := h.taskSvc.Delete(task, project.ID, userID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	api.Success(c, "任务删除成功", gin.H{"id": task.ID})
}

func (h *TaskHandler) loadTaskWithAccess(taskID, userID uint) (*model.Task, *model.Project, error) {
	task, project, err := h.taskSvc.GetWithAccess(taskID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return nil, nil, errForbidden
		}
		return nil, nil, err
	}
	return task, project, nil
}

func (h *TaskHandler) taskPayload(task *model.Task) gin.H {
	payload := gin.H{
		"id":              task.ID,
		"project_id":      task.ProjectID,
		"story_id":        task.StoryID,
		"title":           task.Title,
		"description":     task.Description,
		"status":          task.Status,
		"priority":        task.Priority,
		"progress":        task.Progress,
		"estimated_hours": task.EstimatedHours,
		"code_references": parseStringArrayJSON(task.CodeReferences),
		"created_at":      task.CreatedAt,
		"updated_at":      task.UpdatedAt,
	}
	if task.AssignedTo != nil {
		payload["assigned_to"] = *task.AssignedTo
	}
	if task.Assignee != nil {
		payload["assignee"] = gin.H{"id": task.Assignee.ID, "email": task.Assignee.Email}
	}
	if task.Creator != nil {
		payload["created_by"] = gin.H{"id": task.Creator.ID, "email": task.Creator.Email}
	} else {
		payload["created_by"] = task.CreatedBy
	}
	return payload
}

func (h *TaskHandler) handleStoryAccessErr(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		api.NotFound(c, "用户故事不存在")
		return
	}
	if errors.Is(err, errForbidden) {
		api.Forbidden(c, "非项目成员无法访问")
		return
	}
	api.Internal(c, "服务器内部错误")
}

func (h *TaskHandler) handleTaskAccessErr(c *gin.Context, err error) {
	if err == gorm.ErrRecordNotFound {
		api.NotFound(c, "任务不存在")
		return
	}
	if errors.Is(err, errForbidden) {
		api.Forbidden(c, "非项目成员无法访问")
		return
	}
	api.Internal(c, "服务器内部错误")
}
