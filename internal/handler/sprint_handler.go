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
	"git.neolidy.top/neo/storybook/internal/util/dateparse"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SprintHandler struct {
	db          *gorm.DB
	sprintRepo  repository.SprintRepo
	storyRepo   repository.StoryRepo
	activityRepo repository.ActivityRepo
	notifier    service.Notifier
	events      service.EventPublisher
}

func NewSprintHandler(db *gorm.DB) *SprintHandler {
	return &SprintHandler{
		db:           db,
		sprintRepo:   repository.NewSprintRepository(db),
		storyRepo:    repository.NewStoryRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
		notifier:     service.NoopNotifier{},
	}
}

func (h *SprintHandler) WithNotifier(n service.Notifier) *SprintHandler {
	if n != nil {
		h.notifier = n
	}
	return h
}

// WithEvents wires the project-scope broadcaster post-construction so existing
// callers that only have *gorm.DB stay green.
func (h *SprintHandler) WithEvents(p service.EventPublisher) *SprintHandler {
	if p != nil {
		h.events = p
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
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可创建冲刺")
		return
	}

	projectID := project.ID

	var req createSprintRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	start, err := dateparse.Parse(req.StartDate)
	if err != nil {
		api.BadRequest(c, "start_date格式错误，支持 YYYY-MM-DD 或 RFC3339")
		return
	}
	end, err := dateparse.Parse(req.EndDate)
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

	logging.LogIfErr(service.WriteActivityLog(h.db, &projectID, userID, "sprint", sprint.ID, "created",
		nil, map[string]any{"name": sprint.Name, "status": sprint.Status}),
		"write sprint activity log", "sprint_id", sprint.ID, "action", "created")

	api.Success(c, "冲刺创建成功", sprint)
}

func (h *SprintHandler) List(c *gin.Context) {
	project := middleware.MustProject(c)

	sprints, err := h.sprintRepo.ListByProjectDesc(project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(sprints))
	for _, s := range sprints {
		totalStories, _ := h.storyRepo.CountBySprint(s.ID)
		doneStories, _ := h.storyRepo.CountBySprintAndStatus(s.ID, model.StoryStatusDone)

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
		respondAccessError(c, err, "项目不存在")
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

	logging.LogIfErr(service.WriteActivityLog(h.db, &sprint.ProjectID, userID, "sprint", sprint.ID, "status_changed",
		map[string]any{"status": oldStatus}, map[string]any{"status": sprint.Status}),
		"write sprint activity log", "sprint_id", sprint.ID, "action", "status_changed")

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
			ProjectID:  &sprint.ProjectID,
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
	story := middleware.MustStory(c)
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可规划冲刺")
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

	if err := h.storyRepo.Save(story); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &project.ID, userID, "story", story.ID, "sprint_assigned",
		map[string]any{"sprint_id": oldSprintID}, map[string]any{"sprint_id": story.SprintID}),
		"write story sprint-assignment log", "story_id", story.ID, "sprint_id", story.SprintID)

	api.Success(c, "故事冲刺规划成功", gin.H{
		"story_id":   story.ID,
		"sprint_id":  story.SprintID,
		"updated_at": story.UpdatedAt,
	})
}

// Close finalizes an active sprint: status active → completed, and any story
// still attached to this sprint that is not yet done gets reverted to backlog
// (sprint_id = NULL) so the next iteration can re-plan it.
func (h *SprintHandler) Close(c *gin.Context) {
	h.terminate(c, "close")
}

// Cancel halts a sprint (planned or active) — status → cancelled and ALL
// attached stories revert sprint_id to NULL regardless of their progress.
func (h *SprintHandler) Cancel(c *gin.Context) {
	h.terminate(c, "cancel")
}

// Delete soft-deletes a planned (draft) sprint. Refuses anything other than
// planned so callers must explicitly cancel/close active sprints first.
func (h *SprintHandler) Delete(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理或管理员可删除冲刺")
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	if sprint.Status != model.SprintStatusPlanned {
		api.BadRequest(c, "仅未启动的冲刺可删除，请先取消或完成")
		return
	}

	if err := h.sprintRepo.DeleteWithClearStories(sprint.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &sprint.ProjectID, userID, "sprint", sprint.ID, "deleted",
		map[string]any{"name": sprint.Name, "status": sprint.Status}, nil),
		"write sprint activity log", "sprint_id", sprint.ID, "action", "deleted")

	if h.events != nil {
		h.events.BroadcastProject(sprint.ProjectID, "sprint.deleted", gin.H{
			"sprint_id":  sprint.ID,
			"project_id": sprint.ProjectID,
			"deleted_by": userID,
		})
	}

	api.Success(c, "冲刺删除成功", gin.H{"id": sprint.ID})
}

// terminate is the shared close/cancel implementation. Both transitions revert
// associated stories' sprint_id within the same transaction so a partial
// failure leaves the sprint+stories consistent.
func (h *SprintHandler) terminate(c *gin.Context, action string) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理或管理员可变更冲刺状态")
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	var (
		newStatus  string
		notifType  string
		notifTitle string
		successMsg string
		wsEvent    string
	)
	switch action {
	case "close":
		if sprint.Status != model.SprintStatusActive {
			api.BadRequest(c, "仅活跃中的冲刺可完成")
			return
		}
		newStatus = model.SprintStatusCompleted
		notifType = model.NotificationSprintCompleted
		notifTitle = "冲刺已完成"
		successMsg = "冲刺已完成"
		wsEvent = "sprint.closed"
	case "cancel":
		if sprint.Status != model.SprintStatusPlanned && sprint.Status != model.SprintStatusActive {
			api.BadRequest(c, "仅计划或活跃中的冲刺可取消")
			return
		}
		newStatus = model.SprintStatusCancelled
		notifType = model.NotificationSprintCompleted
		notifTitle = "冲刺已取消"
		successMsg = "冲刺已取消"
		wsEvent = "sprint.cancelled"
	default:
		api.BadRequest(c, "未知冲刺操作")
		return
	}

	oldStatus := sprint.Status
	if err := h.sprintRepo.CloseOrCancel(sprint.ID, newStatus, action == "close"); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	sprint.Status = newStatus
	logging.LogIfErr(service.WriteActivityLog(h.db, &sprint.ProjectID, userID, "sprint", sprint.ID, "status_changed",
		map[string]any{"status": oldStatus}, map[string]any{"status": newStatus}),
		"write sprint activity log", "sprint_id", sprint.ID, "action", action)

	h.notifier.NotifyProjectMembers(c.Request.Context(), sprint.ProjectID, service.NotificationEvent{
		Type:       notifType,
		EntityType: model.NotificationEntitySprint,
		EntityID:   sprint.ID,
		ProjectID:  &sprint.ProjectID,
		ActorID:    &userID,
		Title:      notifTitle,
		Body:       sprint.Name,
		Metadata: gin.H{
			"sprint_id":   sprint.ID,
			"sprint_name": sprint.Name,
			"status":      newStatus,
		},
	})

	if h.events != nil {
		h.events.BroadcastProject(sprint.ProjectID, wsEvent, gin.H{
			"sprint_id":  sprint.ID,
			"project_id": sprint.ProjectID,
			"status":     newStatus,
			"actor_id":   userID,
		})
	}

	api.Success(c, successMsg, gin.H{
		"id":         sprint.ID,
		"status":     sprint.Status,
		"updated_at": sprint.UpdatedAt,
	})
}

type reorderEntry struct {
	StoryID  uint    `json:"story_id" binding:"required"`
	Position float64 `json:"position" binding:"required"`
}

type reorderRequest struct {
	Orders []reorderEntry `json:"orders" binding:"required,min=1,dive"`
}

// Reorder bulk-updates story positions within a sprint so a drag-drop UI can
// flush a whole new ordering in a single round-trip. Validates every story
// belongs to this sprint before any UPDATE fires so a malicious payload can't
// touch unrelated rows; the entire batch runs in one transaction.
func (h *SprintHandler) Reorder(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理或管理员可重新排序冲刺")
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
		respondAccessError(c, err, "项目不存在")
		return
	}

	var req reorderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	ids := make([]uint, 0, len(req.Orders))
	positions := make(map[uint]float64, len(req.Orders))
	for _, o := range req.Orders {
		if _, dup := positions[o.StoryID]; dup {
			api.BadRequest(c, "orders 中出现重复 story_id")
			return
		}
		ids = append(ids, o.StoryID)
		positions[o.StoryID] = o.Position
	}

	// Verify every story is actually attached to this sprint before applying.
	count, err := h.storyRepo.CountBySprintAndIDs(sprint.ID, ids)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if int(count) != len(ids) {
		api.BadRequest(c, "存在不属于该冲刺的故事 ID")
		return
	}

	if err := h.storyRepo.UpdatePositionsBatch(sprint.ID, positions); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &sprint.ProjectID, userID, "sprint", sprint.ID, "reordered",
		nil, map[string]any{"order_count": len(req.Orders)}),
		"write sprint activity log", "sprint_id", sprint.ID, "action", "reordered")

	if h.events != nil {
		h.events.BroadcastProject(sprint.ProjectID, "sprint.reordered", gin.H{
			"sprint_id":  sprint.ID,
			"project_id": sprint.ProjectID,
			"orders":     req.Orders,
			"actor_id":   userID,
		})
	}

	api.Success(c, "冲刺排序更新成功", gin.H{
		"sprint_id":   sprint.ID,
		"order_count": len(req.Orders),
	})
}

