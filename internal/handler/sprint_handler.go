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

type SprintHandler struct {
	db         *gorm.DB
	sprintRepo *repository.SprintRepository
	notifier   service.Notifier
}

func NewSprintHandler(db *gorm.DB) *SprintHandler {
	return &SprintHandler{
		db:         db,
		sprintRepo: repository.NewSprintRepository(db),
		notifier:   service.NoopNotifier{},
	}
}

func (h *SprintHandler) WithNotifier(n service.Notifier) *SprintHandler {
	if n != nil {
		h.notifier = n
	}
	return h
}

type createSprintRequest struct {
	Name      string `json:"name" binding:"required,min=2,max=120"`
	Goal      string `json:"goal" binding:"max=500"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type updateSprintStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=planned active completed"`
}

type assignStorySprintRequest struct {
	SprintID *uint `json:"sprint_id"`
}

func (h *SprintHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可创建冲刺")
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

	var req createSprintRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	start, err := parseDate(req.StartDate)
	if err != nil {
		api.BadRequest(c, "start_date格式错误，支持 YYYY-MM-DD 或 RFC3339")
		return
	}
	end, err := parseDate(req.EndDate)
	if err != nil {
		api.BadRequest(c, "end_date格式错误，支持 YYYY-MM-DD 或 RFC3339")
		return
	}
	if end.Before(start) {
		api.BadRequest(c, "结束时间必须晚于开始时间")
		return
	}

	sprint := model.Sprint{
		ProjectID: projectID,
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: start,
		EndDate:   end,
		Status:    model.SprintStatusPlanned,
		CreatedBy: userID,
	}
	if err := h.sprintRepo.Create(&sprint); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	pid := projectID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "sprint",
		EntityID:   sprint.ID,
		Action:     "created",
		UserID:     userID,
		ProjectID:  &pid,
		NewValue:   model.MarshalJSON(gin.H{"name": sprint.Name, "status": sprint.Status}),
	}).Error, "write sprint activity log", "sprint_id", sprint.ID, "action", "created")

	api.Success(c, "冲刺创建成功", sprint)
}

func (h *SprintHandler) List(c *gin.Context) {
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

	sprints, err := h.sprintRepo.ListByProjectDesc(projectID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(sprints))
	for _, s := range sprints {
		var totalStories int64
		logging.LogIfErr(h.db.Model(&model.UserStory{}).Where("sprint_id = ?", s.ID).Count(&totalStories).Error, "count sprint stories failed", "sprint_id", s.ID)
		var doneStories int64
		logging.LogIfErr(h.db.Model(&model.UserStory{}).Where("sprint_id = ? AND status = ?", s.ID, model.StoryStatusDone).Count(&doneStories).Error, "count done stories in sprint failed", "sprint_id", s.ID)

		items = append(items, gin.H{
			"id":            s.ID,
			"project_id":    s.ProjectID,
			"name":          s.Name,
			"goal":          s.Goal,
			"start_date":    s.StartDate,
			"end_date":      s.EndDate,
			"status":        s.Status,
			"total_stories": totalStories,
			"done_stories":  doneStories,
			"created_at":    s.CreatedAt,
			"updated_at":    s.UpdatedAt,
		})
	}

	api.Success(c, "success", gin.H{"sprints": items})
}

func (h *SprintHandler) UpdateStatus(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可变更冲刺状态")
		return
	}

	sprintID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "冲刺ID无效")
		return
	}

	sprint, err := h.sprintRepo.FindByID(sprintID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "冲刺不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	if _, _, err := ensureProjectAccess(h.db, sprint.ProjectID, userID); err != nil {
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var req updateSprintStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	if !service.Workflow.CanSprintTransit(sprint.Status, req.Status) {
		api.BadRequest(c, "非法冲刺状态流转")
		return
	}

	oldStatus := sprint.Status
	sprint.Status = req.Status
	if err := h.sprintRepo.Save(sprint); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	pid := sprint.ProjectID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "sprint",
		EntityID:   sprint.ID,
		Action:     "status_changed",
		UserID:     userID,
		ProjectID:  &pid,
		OldValue:   model.MarshalJSON(gin.H{"status": oldStatus}),
		NewValue:   model.MarshalJSON(gin.H{"status": sprint.Status}),
	}).Error, "write sprint activity log", "sprint_id", sprint.ID, "action", "status_changed")

	if sprint.Status == model.SprintStatusActive || sprint.Status == model.SprintStatusCompleted {
		notifType := model.NotificationSprintStarted
		title := "冲刺已启动"
		if sprint.Status == model.SprintStatusCompleted {
			notifType = model.NotificationSprintCompleted
			title = "冲刺已完成"
		}
		h.notifier.NotifyProjectMembers(c.Request.Context(), sprint.ProjectID, service.NotificationEvent{
			Type:       notifType,
			EntityType: model.NotificationEntitySprint,
			EntityID:   sprint.ID,
			ProjectID:  &pid,
			ActorID:    &userID,
			Title:      title,
			Body:       sprint.Name,
			Metadata: gin.H{
				"sprint_id":   sprint.ID,
				"sprint_name": sprint.Name,
				"status":      sprint.Status,
			},
		})
	}

	api.Success(c, "冲刺状态更新成功", gin.H{
		"id":         sprint.ID,
		"status":     sprint.Status,
		"updated_at": sprint.UpdatedAt,
	})
}

func (h *SprintHandler) AssignStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可规划冲刺")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, project, _, err := ensureStoryAccess(h.db, storyID, userID)
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

	var req assignStorySprintRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	oldSprintID := story.SprintID

	if req.SprintID != nil {
		sprint, err := h.sprintRepo.FindByID(*req.SprintID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				api.NotFound(c, "冲刺不存在")
				return
			}
			api.Internal(c, "服务器内部错误")
			return
		}
		if sprint.ProjectID != story.ProjectID {
			api.BadRequest(c, "冲刺不属于当前项目")
			return
		}
		story.SprintID = req.SprintID
	} else {
		story.SprintID = nil
	}

	if err := h.db.Save(story).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	pid := project.ID
	logging.LogIfErr(h.db.Create(&model.ActivityLog{
		EntityType: "story",
		EntityID:   story.ID,
		Action:     "sprint_assigned",
		UserID:     userID,
		ProjectID:  &pid,
		OldValue:   model.MarshalJSON(gin.H{"sprint_id": oldSprintID}),
		NewValue:   model.MarshalJSON(gin.H{"sprint_id": story.SprintID}),
	}).Error, "write story sprint-assignment log", "story_id", story.ID, "sprint_id", story.SprintID)

	api.Success(c, "故事冲刺规划成功", gin.H{
		"story_id":   story.ID,
		"sprint_id":  story.SprintID,
		"updated_at": story.UpdatedAt,
	})
}

func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("empty date")
	}

	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, raw)
}
