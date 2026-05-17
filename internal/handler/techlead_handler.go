package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/logging"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/repository"
	"github.com/jiangfire/storybook/internal/service"
	"gorm.io/gorm"
)

type TechLeadHandler struct {
	db           *gorm.DB
	storyRepo    repository.StoryRepo
	projectRepo  repository.ProjectRepo
	userRepo     repository.UserRepo
	taskRepo     repository.TaskRepo
	activityRepo repository.ActivityRepo
}

func NewTechLeadHandler(db *gorm.DB) *TechLeadHandler {
	return &TechLeadHandler{
		db:           db,
		storyRepo:    repository.NewStoryRepository(db),
		projectRepo:  repository.NewProjectRepository(db),
		userRepo:     repository.NewUserRepository(db),
		taskRepo:     repository.NewTaskRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
	}
}

// ListPendingStories 获取待审批故事列表（技术负责人）
func (h *TechLeadHandler) ListPendingStories(c *gin.Context) {
	userID := middleware.MustUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	query := h.storyRepo.DB().Model(&model.UserStory{}).Where("status = ?", model.StoryStatusPending)
	if role == model.RoleTechLead {
		projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
		if err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		if len(projectIDs) == 0 {
			api.Success(c, "success", gin.H{"stories": []gin.H{}, "total": 0, "page": 1, "limit": 20})
			return
		}
		query = query.Where("project_id IN ?", projectIDs)
	}

	// 支持按项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 支持搜索
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// 分页
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var stories []model.UserStory
	if err := query.Preload("Project").Preload("Creator").Order("created_at DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(stories))
	for _, story := range stories {
		item := gin.H{
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
			"created_at": story.CreatedAt,
		}

		if story.Project != nil {
			item["project"] = gin.H{
				"id":   story.Project.ID,
				"name": story.Project.Name,
			}
		}

		if story.Creator != nil {
			item["created_by"] = gin.H{
				"id":    story.Creator.ID,
				"email": story.Creator.Email,
			}
		}

		items = append(items, item)
	}

	api.Success(c, "success", gin.H{
		"stories": items,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// ListWorkload 获取开发人员工作负载
func (h *TechLeadHandler) ListWorkload(c *gin.Context) {
	userID := middleware.MustUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	// 获取项目ID（可选，用于筛选特定项目的负载）
	projectID := uint(0)
	if pid := c.Query("project_id"); pid != "" {
		if id, err := strconv.ParseUint(pid, 10, 64); err == nil {
			projectID = uint(id)
		}
	}

	visibleProjectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if role == model.RoleTechLead && len(visibleProjectIDs) == 0 {
		api.Success(c, "success", gin.H{"workloads": []gin.H{}})
		return
	}
	if projectID > 0 && role == model.RoleTechLead {
		allowed := false
		for _, id := range visibleProjectIDs {
			if id == projectID {
				allowed = true
				break
			}
		}
		if !allowed {
			api.Forbidden(c, "无权查看该项目负载")
			return
		}
	}

	// 获取开发人员列表
	var developerIDs []uint
	if projectID > 0 {
		// 获取特定项目的开发人员
		h.projectRepo.DB().Model(&model.ProjectMember{}).
			Where("project_id = ? AND role_in_project = ?", projectID, model.RoleDeveloper).
			Pluck("user_id", &developerIDs)
	} else {
		if role == model.RoleAdmin {
			developerIDs, _ = h.userRepo.ListIDsByRole(model.RoleDeveloper)
		} else {
			h.projectRepo.DB().Model(&model.ProjectMember{}).
				Where("project_id IN ? AND role_in_project = ?", visibleProjectIDs, model.RoleDeveloper).
				Distinct().
				Pluck("user_id", &developerIDs)
		}
	}

	if len(developerIDs) == 0 {
		api.Success(c, "success", gin.H{"workloads": []gin.H{}})
		return
	}

	workloads := make([]gin.H, 0, len(developerIDs))
	for _, devID := range developerIDs {
		if wl := h.buildDeveloperWorkload(devID, projectID, visibleProjectIDs, role); wl != nil {
			workloads = append(workloads, wl)
		}
	}

	api.Success(c, "success", gin.H{
		"workloads": workloads,
	})
}

// applyProjectFilter adds project_id scope to a GORM query when projectID or
// visibleProjectIDs / role are present. It mutates and returns the query chain.
func applyProjectFilter(q *gorm.DB, projectID uint, visibleProjectIDs []uint, role string) *gorm.DB {
	if projectID > 0 {
		return q.Where("project_id = ?", projectID)
	}
	if role == model.RoleTechLead {
		return q.Where("project_id IN ?", visibleProjectIDs)
	}
	return q
}

// buildDeveloperWorkload aggregates metrics for a single developer.
func (h *TechLeadHandler) buildDeveloperWorkload(devID, projectID uint, visibleProjectIDs []uint, role string) gin.H {
	user, err := h.userRepo.FindByID(devID)
	if err != nil {
		return nil
	}

	var scopeProjects []uint
	if projectID > 0 {
		scopeProjects = []uint{projectID}
	} else if role == model.RoleTechLead {
		scopeProjects = visibleProjectIDs
	}

	var activeStories int64
	q := h.storyRepo.DB().Model(&model.UserStory{}).
		Where("assigned_to = ? AND status IN ?", devID, []string{model.StoryStatusInProgress, model.StoryStatusReady})
	q = applyProjectFilter(q, projectID, visibleProjectIDs, role)
	q.Count(&activeStories)

	activeTasks, _ := h.taskRepo.CountTasksByAssignee(devID, []string{model.TaskStatusTodo, model.TaskStatusInProgress}, scopeProjects, time.Time{})

	var totalPoints float64
	q = h.storyRepo.DB().Model(&model.UserStory{}).
		Where("assigned_to = ? AND points IS NOT NULL AND status != ?", devID, model.StoryStatusDone)
	q = applyProjectFilter(q, projectID, visibleProjectIDs, role)
	q.Select("COALESCE(SUM(points), 0)").Scan(&totalPoints)

	estimatedHours, _ := h.taskRepo.SumEstimatedHoursByAssignee(devID, model.TaskStatusDone, scopeProjects)

	var completedStories, totalAssigned int64
	q1 := h.storyRepo.DB().Model(&model.UserStory{}).Where("assigned_to = ? AND status = ?", devID, model.StoryStatusDone)
	q1 = applyProjectFilter(q1, projectID, visibleProjectIDs, role)
	q1.Count(&completedStories)

	q2 := h.storyRepo.DB().Model(&model.UserStory{}).Where("assigned_to = ?", devID)
	q2 = applyProjectFilter(q2, projectID, visibleProjectIDs, role)
	q2.Count(&totalAssigned)

	completionRate := 0.0
	if totalAssigned > 0 {
		completionRate = float64(completedStories) / float64(totalAssigned) * 100
	}

	return gin.H{
		"user":               gin.H{"id": user.ID, "email": user.Email},
		"active_stories":     activeStories,
		"active_tasks":       activeTasks,
		"total_story_points": totalPoints,
		"estimated_hours":    estimatedHours,
		"completion_rate":    completionRate,
	}
}

// ListMyProjects 获取技术负责人负责的项目列表
func (h *TechLeadHandler) ListMyProjects(c *gin.Context) {
	userID := middleware.MustUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var projects []model.Project
	query := h.projectRepo.DB().Model(&model.Project{})
	if role != model.RoleAdmin {
		if len(projectIDs) == 0 {
			api.Success(c, "success", gin.H{"projects": []gin.H{}})
			return
		}
		query = query.Where("id IN ?", projectIDs)
	}
	if err := query.Find(&projects).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		// 统计待审批故事数
		var pendingCount int64
		h.storyRepo.DB().Model(&model.UserStory{}).Where("project_id = ? AND status = ?", p.ID, model.StoryStatusPending).Count(&pendingCount)

		items = append(items, gin.H{
			"id":              p.ID,
			"name":            p.Name,
			"agile_mode":      p.AgileMode,
			"pending_stories": pendingCount,
		})
	}

	api.Success(c, "success", gin.H{"projects": items})
}

// AddTechLead 为项目指定技术负责人（admin）
func (h *TechLeadHandler) AddTechLead(c *gin.Context) {
	project := middleware.MustProject(c)
	userID := middleware.MustUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可指定技术负责人")
		return
	}

	projectID := project.ID

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if !middleware.BindJSON(c, &req) {
		return
	}

	// 检查用户是否存在且是tech_lead角色
	user, err := h.userRepo.FindByID(req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}
	if user.Role != model.RoleTechLead {
		api.BadRequest(c, "该用户不是技术负责人角色")
		return
	}

	// 检查是否已存在
	exists, err := h.projectRepo.HasTechLead(projectID, req.UserID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists {
		api.Conflict(c, "该技术负责人已在项目中")
		return
	}

	// 创建关联
	techLead := model.ProjectTechLead{
		ProjectID:  projectID,
		UserID:     req.UserID,
		AssignedAt: time.Now(),
	}
	if err := h.projectRepo.AddTechLead(&techLead); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// 记录活动日志
	logging.LogIfErr(service.WriteActivityLog(h.db, &projectID, userID, "project", projectID, "techlead_added",
		nil, map[string]any{"user_id": req.UserID}),
		"write project activity log", "project_id", projectID, "action", "techlead_added")

	api.Success(c, "技术负责人添加成功", gin.H{
		"project_id": projectID,
		"user_id":    req.UserID,
	})
}

// RemoveTechLead 移除项目技术负责人（admin）
func (h *TechLeadHandler) RemoveTechLead(c *gin.Context) {
	project := middleware.MustProject(c)
	userID := middleware.MustUserID(c)

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可移除技术负责人")
		return
	}

	projectID := project.ID

	targetUserID, ok := parseUintParam(c, "userID")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	affected, err := h.projectRepo.RemoveTechLead(projectID, targetUserID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if affected == 0 {
		api.NotFound(c, "技术负责人不存在")
		return
	}

	// 记录活动日志
	logging.LogIfErr(service.WriteActivityLog(h.db, &projectID, userID, "project", projectID, "techlead_removed",
		map[string]any{"user_id": targetUserID}, nil),
		"write project activity log", "project_id", projectID, "action", "techlead_removed")

	api.Success(c, "技术负责人移除成功", gin.H{
		"project_id": projectID,
		"user_id":    targetUserID,
	})
}

// ListProjectTechLeads 获取项目技术负责人列表
func (h *TechLeadHandler) ListProjectTechLeads(c *gin.Context) {
	project := middleware.MustProject(c)
	projectID := project.ID

	techLeads, err := h.projectRepo.ListTechLeads(projectID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(techLeads))
	for _, tl := range techLeads {
		item := gin.H{
			"id":          tl.ID,
			"assigned_at": tl.AssignedAt,
		}
		if tl.User != nil {
			item["user"] = gin.H{
				"id":    tl.User.ID,
				"email": tl.User.Email,
			}
		}
		items = append(items, item)
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"tech_leads": items,
	})
}
