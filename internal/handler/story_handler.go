package handler

import (
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

type StoryHandler struct {
	db             *gorm.DB
	events         EventPublisher
	storySvc       *service.StoryService
	userRepo       repository.UserRepo
	storyRepo      repository.StoryRepo
	projectRepo    repository.ProjectRepo
	activityRepo   repository.ActivityRepo
	boardColumnRepo *repository.BoardColumnRepository
	notifier       service.Notifier
}

type EventPublisher interface {
	BroadcastProject(projectID uint, eventType string, data any)
	BroadcastUser(userID uint, eventType string, data any)
}

func NewStoryHandler(db *gorm.DB, events EventPublisher) *StoryHandler {
	return NewStoryHandlerWithVector(db, events, nil)
}

func NewStoryHandlerWithVector(db *gorm.DB, events EventPublisher, vectorSvc service.VectorService) *StoryHandler {
	return &StoryHandler{
		db:              db,
		events:          events,
		storySvc:        service.NewStoryServiceWithVector(db, events, vectorSvc),
		userRepo:        repository.NewUserRepository(db),
		storyRepo:       repository.NewStoryRepository(db),
		projectRepo:     repository.NewProjectRepository(db),
		activityRepo:    repository.NewActivityLogRepository(db),
		boardColumnRepo: repository.NewBoardColumnRepository(db),
		notifier:        service.NoopNotifier{},
	}
}

func (h *StoryHandler) WithNotifier(n service.Notifier) *StoryHandler {
	if n != nil {
		h.notifier = n
		h.storySvc.WithNotifier(n)
	}
	return h
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

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		respondAccessError(c, err, "项目不存在")
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

	creator, err := h.userRepo.FindByID(userID)
	if err != nil {
		logging.LogIfErr(err, "load story creator failed", "user_id", userID)
		creator = &model.User{}
	}

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

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		respondAccessError(c, err, "项目不存在")
		return
	}

	query := h.storyRepo.DB().Model(&model.UserStory{}).Where("project_id = ?", projectID)
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

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		respondAccessError(c, err, "项目不存在")
		return
	}

	columns, err := h.boardColumnRepo.ListByProject(projectID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	byStatus := map[string][]gin.H{
		model.StoryStatusPending:    {},
		model.StoryStatusBacklog:    {},
		model.StoryStatusReady:      {},
		model.StoryStatusInProgress: {},
		model.StoryStatusTest:       {},
		model.StoryStatusDone:       {},
	}

	stories, err := h.storyRepo.ListBoardByProjectWithAssignee(projectID)
	if err != nil {
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
		{Pos: 0, Status: model.StoryStatusPending},
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
	story := middleware.MustStory(c)

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
		"code_references":     service.ParseStringArrayJSON(story.CodeReferences),
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
	story := middleware.MustStory(c)
	userID, _ := middleware.CurrentUserID(c)

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

func (h *StoryHandler) DeleteStory(c *gin.Context) {
	story := middleware.MustStory(c)
	userID, _ := middleware.CurrentUserID(c)

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
