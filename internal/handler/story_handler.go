package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoryHandler struct {
	db       *gorm.DB
	events   EventPublisher
	storySvc *service.StoryService
}

type EventPublisher interface {
	Broadcast(eventType string, data any)
}

func NewStoryHandler(db *gorm.DB, events EventPublisher) *StoryHandler {
	return &StoryHandler{
		db:       db,
		events:   events,
		storySvc: service.NewStoryService(db, events),
	}
}

type createStoryACItem struct {
	ID          string `json:"id"`
	Ref         string `json:"ref"`
	Description string `json:"description" binding:"required"`
	Order       int    `json:"order"`
}

type createStoryRequest struct {
	Title              string              `json:"title" binding:"required,min=2,max=200"`
	Description        string              `json:"description" binding:"max=2000"`
	StoryType          string              `json:"story_type" binding:"required,oneof=feature bug chore"`
	Priority           int                 `json:"priority" binding:"min=0,max=4"`
	StoryPoints        *int                `json:"story_points"`
	AcceptanceCriteria []createStoryACItem `json:"acceptance_criteria"`
	Tags               []string            `json:"tags"`
}

type updateStoryRequest struct {
	Title              *string             `json:"title"`
	Description        *string             `json:"description"`
	StoryType          *string             `json:"story_type"`
	Priority           *int                `json:"priority"`
	StoryPoints        *int                `json:"story_points"`
	AcceptanceCriteria []createStoryACItem `json:"acceptance_criteria"`
	Tags               *[]string           `json:"tags"`
}

type updateStatusRequest struct {
	Status   string  `json:"status" binding:"required,oneof=backlog ready in_progress test done"`
	Position float64 `json:"position"`
}

type updateACStatusRequest struct {
	Status   string `json:"status" binding:"required,oneof=pending passed failed"`
	Evidence string `json:"evidence"`
	Notes    string `json:"notes"`
}

type addCodeRefRequest struct {
	Reference string `json:"reference" binding:"required,max=500"`
}

type assignStoryRequest struct {
	AssignedTo *uint `json:"assigned_to"`
}

type reviewStoryRequest struct {
	Approved *bool  `json:"approved"`
	Comment  string `json:"comment"`
}

func (h *StoryHandler) CreateStory(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足，只有产品经理可以创建用户故事")
		return
	}

	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}

	if err := h.ensureProjectMember(projectID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
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

	var req createStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	criteria := normalizeAC(req.AcceptanceCriteria)
	story, err := h.storySvc.Create(service.CreateStoryInput{
		ProjectID:          projectID,
		UserID:             userID,
		Title:              req.Title,
		Description:        req.Description,
		StoryType:          req.StoryType,
		Priority:           req.Priority,
		StoryPoints:        req.StoryPoints,
		AcceptanceCriteria: criteria,
		Tags:               req.Tags,
		Position:           float64(time.Now().UnixNano()),
	})
	if err != nil {
		if items, ok := serviceValidationItems(err); ok {
			api.BadRequest(c, "参数验证失败", items...)
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var creator model.User
	_ = h.db.Select("id, email").First(&creator, userID).Error

	payload := gin.H{
		"id":                  story.ID,
		"project_id":          story.ProjectID,
		"title":               story.Title,
		"description":         story.Description,
		"story_type":          story.StoryType,
		"status":              story.Status,
		"priority":            story.Priority,
		"story_points":        story.Points,
		"acceptance_criteria": criteria,
		"created_by": gin.H{
			"id":    creator.ID,
			"email": creator.Email,
		},
		"created_at": story.CreatedAt,
	}

	api.Success(c, "用户故事创建成功", payload)
}

func (h *StoryHandler) ListStories(c *gin.Context) {
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

	if err := h.ensureProjectMember(projectID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
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

	query := h.db.Model(&model.UserStory{}).Where("project_id = ?", projectID)
	includeArchived := strings.EqualFold(strings.TrimSpace(c.DefaultQuery("include_archived", "false")), "true")
	if !includeArchived {
		query = query.Where("archived = ?", false)
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" {
		query = query.Where("status = ?", status)
	}

	assignee := strings.TrimSpace(c.Query("assignee"))
	if assignee != "" {
		query = query.Where("assigned_to = ?", assignee)
	}

	sortBy := strings.TrimSpace(c.DefaultQuery("sort_by", "priority"))
	order := strings.ToLower(strings.TrimSpace(c.DefaultQuery("order", "desc")))
	if order != "asc" {
		order = "desc"
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var stories []model.UserStory
	query = query.Preload("Assignee")
	switch sortBy {
	case "assignee":
		query = query.Order("assigned_to " + order).Order("priority DESC").Order("position ASC")
	case "position":
		query = query.Order("position " + order).Order("priority DESC")
	default:
		query = query.Order("priority " + order).Order("position ASC")
	}

	if err := query.Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(stories))
	for _, story := range stories {
		criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
		totalAC, passedAC, pendingAC, _ := summarizeAC(criteria)
		completion := 0.0
		if totalAC > 0 {
			completion = float64(passedAC) / float64(totalAC) * 100
		}

		row := gin.H{
			"id":         story.ID,
			"title":      story.Title,
			"story_type": story.StoryType,
			"status":     story.Status,
			"review_status": func() string {
				if strings.TrimSpace(story.ReviewStatus) == "" {
					return model.ReviewStatusPending
				}
				return story.ReviewStatus
			}(),
			"review_comment": story.ReviewComment,
			"priority":       story.Priority,
			"story_points": func() any {
				if story.Points == nil {
					return nil
				}
				return *story.Points
			}(),
			"position": story.Position,
			"acceptance_criteria_summary": gin.H{
				"total":                 totalAC,
				"passed":                passedAC,
				"pending":               pendingAC,
				"completion_percentage": completion,
			},
		}

		if story.Assignee != nil {
			row["assigned_to"] = gin.H{
				"id":    story.Assignee.ID,
				"email": story.Assignee.Email,
			}
		}

		items = append(items, row)
	}

	api.Success(c, "success", gin.H{
		"stories":          items,
		"total":            total,
		"sort_by":          sortBy,
		"order":            order,
		"include_archived": includeArchived,
	})
}

func (h *StoryHandler) GetBoard(c *gin.Context) {
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

	if err := h.ensureProjectMember(projectID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
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

	var columns []model.BoardColumn
	if err := h.db.Where("project_id = ?", projectID).Order("position ASC").Find(&columns).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	byStatus := map[string][]gin.H{
		model.StoryStatusBacklog:    {},
		model.StoryStatusReady:      {},
		model.StoryStatusInProgress: {},
		model.StoryStatusTest:       {},
		model.StoryStatusDone:       {},
	}

	var stories []model.UserStory
	if err := h.db.Where("project_id = ? AND archived = ?", projectID, false).Preload("Assignee").Order("position ASC, priority DESC").Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	for _, s := range stories {
		criteria, _ := model.ParseAcceptanceCriteria(s.AcceptanceCriteria)
		totalAC, passedAC, pendingAC, _ := summarizeAC(criteria)
		completion := 0.0
		if totalAC > 0 {
			completion = float64(passedAC) / float64(totalAC) * 100
		}

		item := gin.H{
			"id":         s.ID,
			"project_id": s.ProjectID,
			"title":      s.Title,
			"story_type": s.StoryType,
			"status":     s.Status,
			"review_status": func() string {
				if strings.TrimSpace(s.ReviewStatus) == "" {
					return model.ReviewStatusPending
				}
				return s.ReviewStatus
			}(),
			"review_comment": s.ReviewComment,
			"priority":       s.Priority,
			"position":       s.Position,
			"created_by":     s.CreatedBy,
			"created_at":     s.CreatedAt,
			"updated_at":     s.UpdatedAt,
			"acceptance_criteria_summary": gin.H{
				"total":                 totalAC,
				"passed":                passedAC,
				"pending":               pendingAC,
				"completion_percentage": completion,
			},
			"story_points": func() any {
				if s.Points == nil {
					return nil
				}
				return *s.Points
			}(),
		}
		if s.Assignee != nil {
			assignee := gin.H{"id": s.Assignee.ID, "email": s.Assignee.Email}
			item["assignee"] = assignee
			item["assigned_to"] = assignee
		}
		byStatus[s.Status] = append(byStatus[s.Status], item)
	}

	type statusMapping struct {
		Pos    int
		Status string
	}
	mapping := []statusMapping{
		{Pos: 1, Status: model.StoryStatusBacklog},
		{Pos: 2, Status: model.StoryStatusReady},
		{Pos: 3, Status: model.StoryStatusInProgress},
		{Pos: 4, Status: model.StoryStatusTest},
		{Pos: 5, Status: model.StoryStatusDone},
	}

	result := make([]gin.H, 0, len(mapping))
	for _, m := range mapping {
		name := defaultColumnName(m.Pos)
		for _, c := range columns {
			if c.Position == m.Pos {
				name = c.Name
				break
			}
		}

		storiesInCol := byStatus[m.Status]
		result = append(result, gin.H{
			"position": m.Pos,
			"name":     name,
			"status":   m.Status,
			"stories":  storiesInCol,
			"count":    len(storiesInCol),
		})
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"columns":    result,
	})
}

func (h *StoryHandler) GetStory(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		api.Internal(c, "AC数据损坏")
		return
	}

	data := gin.H{
		"id":          story.ID,
		"project_id":  story.ProjectID,
		"project":     gin.H{"id": story.ProjectID, "name": story.Project.Name},
		"title":       story.Title,
		"description": story.Description,
		"story_type":  story.StoryType,
		"status":      story.Status,
		"review_status": func() string {
			if strings.TrimSpace(story.ReviewStatus) == "" {
				return model.ReviewStatusPending
			}
			return story.ReviewStatus
		}(),
		"review_comment": story.ReviewComment,
		"priority":       story.Priority,
		"story_points": func() any {
			if story.Points == nil {
				return nil
			}
			return *story.Points
		}(),
		"acceptance_criteria": criteria,
		"tags":                story.Tags,
		"code_references":     parseStringArrayJSON(story.CodeReferences),
		"sprint_id":           story.SprintID,
		"created_at":          story.CreatedAt,
		"updated_at":          story.UpdatedAt,
	}

	if story.Assignee != nil {
		data["assigned_to"] = gin.H{"id": story.Assignee.ID, "email": story.Assignee.Email}
	}
	if story.Creator != nil {
		data["created_by"] = gin.H{"id": story.Creator.ID, "email": story.Creator.Email}
	}

	api.Success(c, "success", data)
}

func (h *StoryHandler) UpdateStory(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "只有产品经理和故事创建者可以编辑")
		return
	}

	var req updateStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var criteria *[]model.AcceptanceCriterion
	if req.AcceptanceCriteria != nil {
		normalized := normalizeAC(req.AcceptanceCriteria)
		criteria = &normalized
	}

	changed, err := h.storySvc.Update(story, userID, service.UpdateStoryInput{
		Title:              req.Title,
		Description:        req.Description,
		StoryType:          req.StoryType,
		Priority:           req.Priority,
		StoryPoints:        req.StoryPoints,
		AcceptanceCriteria: criteria,
		Tags:               req.Tags,
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
		api.Success(c, "success", gin.H{
			"id":      story.ID,
			"message": "无字段变更",
		})
		return
	}

	api.Success(c, "用户故事更新成功", gin.H{
		"id":          story.ID,
		"title":       story.Title,
		"description": story.Description,
		"story_type":  story.StoryType,
		"priority":    story.Priority,
		"story_points": func() any {
			if story.Points == nil {
				return nil
			}
			return *story.Points
		}(),
		"status":     story.Status,
		"updated_at": story.UpdatedAt,
	})
}

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
		if err == gorm.ErrRecordNotFound {
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
		if err == gorm.ErrRecordNotFound {
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
	_ = h.db.Select("id, email").First(&user, userID).Error

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
		if err == gorm.ErrRecordNotFound {
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
		if err == gorm.ErrRecordNotFound {
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
		var member model.ProjectMember
		if err := h.db.Where("project_id = ? AND user_id = ?", story.ProjectID, *req.AssignedTo).First(&member).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
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

	if err := h.db.Save(story).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	pid := story.ProjectID
	_ = h.db.Create(&model.ActivityLog{
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
	}).Error

	var assignee any
	if story.AssignedTo != nil {
		var user model.User
		if err := h.db.Select("id, email").First(&user, *story.AssignedTo).Error; err == nil {
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
		if err == gorm.ErrRecordNotFound {
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
		if err == gorm.ErrRecordNotFound {
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

func (h *StoryHandler) DeleteStory(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可删除故事")
		return
	}

	if err := h.storySvc.Delete(story, userID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "用户故事删除成功", gin.H{"id": story.ID})
}

func (h *StoryHandler) ArchiveStory(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可归档故事")
		return
	}

	changed, err := h.storySvc.Archive(story, userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !changed {
		api.Success(c, "success", gin.H{"id": story.ID, "archived": true})
		return
	}
	api.Success(c, "故事归档成功", gin.H{"id": story.ID, "archived": story.Archived, "updated_at": story.UpdatedAt})
}

func (h *StoryHandler) RestoreStory(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	role, _ := middleware.CurrentRole(c)
	if denyTechLeadStoryMutation(c, role) {
		return
	}
	if role != model.RoleProduct && role != model.RoleAdmin && story.CreatedBy != userID {
		api.Forbidden(c, "仅产品经理、管理员或创建者可恢复故事")
		return
	}

	changed, err := h.storySvc.Restore(story, userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if !changed {
		api.Success(c, "success", gin.H{"id": story.ID, "archived": false})
		return
	}
	api.Success(c, "故事恢复成功", gin.H{"id": story.ID, "archived": story.Archived, "updated_at": story.UpdatedAt})
}

func (h *StoryHandler) GetActivities(c *gin.Context) {
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

	story, err := h.getStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
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

	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := h.db.Model(&model.ActivityLog{}).
		Where("entity_type = ? AND entity_id = ?", "story", story.ID)

	action := strings.TrimSpace(c.Query("action"))
	if action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var logs []model.ActivityLog
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	activities := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		oldValue := any(nil)
		if len(log.OldValue) > 0 {
			parsed, _ := model.ParseJSONMap(log.OldValue)
			oldValue = parsed
		}

		newValue := any(nil)
		if len(log.NewValue) > 0 {
			parsed, _ := model.ParseJSONMap(log.NewValue)
			newValue = parsed
		}

		item := gin.H{
			"id":          log.ID,
			"entity_type": log.EntityType,
			"entity_id":   log.EntityID,
			"action":      log.Action,
			"old_value":   oldValue,
			"new_value":   newValue,
			"created_at":  log.CreatedAt,
		}

		if log.User != nil {
			item["user"] = gin.H{
				"id":         log.User.ID,
				"email":      log.User.Email,
				"avatar_url": log.User.AvatarURL,
			}
		}

		activities = append(activities, item)
	}

	api.Success(c, "success", gin.H{
		"activities": activities,
		"total":      total,
		"page":       page,
		"limit":      limit,
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
		if err == gorm.ErrRecordNotFound {
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

func (h *StoryHandler) getStoryWithAccess(storyID, userID uint) (*model.UserStory, error) {
	story, err := h.storySvc.GetWithAccess(storyID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return nil, errForbidden
		}
		return nil, err
	}
	return story, nil
}

func (h *StoryHandler) ensureProjectMember(projectID, userID uint) error {
	err := h.storySvc.EnsureProjectMember(projectID, userID)
	if errors.Is(err, service.ErrForbidden) {
		return errForbidden
	}
	return err
}

func normalizeAC(items []createStoryACItem) []model.AcceptanceCriterion {
	if len(items) == 0 {
		return []model.AcceptanceCriterion{}
	}

	normalized := make([]model.AcceptanceCriterion, 0, len(items))
	for i, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("ac-%d", i+1)
		}

		order := item.Order
		if order <= 0 {
			order = i + 1
		}

		normalized = append(normalized, model.AcceptanceCriterion{
			ID:          id,
			Ref:         strings.TrimSpace(item.Ref),
			Description: strings.TrimSpace(item.Description),
			Status:      model.ACStatusPending,
			Order:       order,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].Order < normalized[j].Order
	})

	return normalized
}

func summarizeAC(items []model.AcceptanceCriterion) (total, passed, pending, failed int) {
	total = len(items)
	for _, ac := range items {
		switch ac.Status {
		case model.ACStatusPassed:
			passed++
		case model.ACStatusFailed:
			failed++
		default:
			pending++
		}
	}
	return
}

func actorFromContext(c *gin.Context, userID uint) gin.H {
	return gin.H{
		"id":    userID,
		"email": c.GetString(middleware.CtxEmailKey),
	}
}

func denyTechLeadStoryMutation(c *gin.Context, role string) bool {
	if role == model.RoleTechLead {
		api.Forbidden(c, "技术负责人仅可审批用户故事")
		return true
	}
	return false
}

func parseStringArrayJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	return out
}

func defaultColumnName(position int) string {
	switch position {
	case 1:
		return "待办"
	case 2:
		return "就绪"
	case 3:
		return "开发中"
	case 4:
		return "测试中"
	case 5:
		return "已完成"
	default:
		return "未命名列"
	}
}
