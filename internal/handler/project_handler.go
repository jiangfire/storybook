package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

type ProjectHandler struct {
	db           *gorm.DB
	projectRepo  *repository.ProjectRepository
	userRepo     *repository.UserRepository
	storyRepo    *repository.StoryRepository
	activityRepo repository.ActivityRepo
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{
		db:           db,
		projectRepo:  repository.NewProjectRepository(db),
		userRepo:     repository.NewUserRepository(db),
		storyRepo:    repository.NewStoryRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
	}
}

type createProjectRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
	AgileMode   string `json:"agile_mode" binding:"required,oneof=scrum kanban"`
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	AgileMode   *string `json:"agile_mode"`
}

type addProjectMemberRequest struct {
	UserID        uint   `json:"user_id" binding:"required"`
	RoleInProject string `json:"role_in_project" binding:"required,oneof=product developer tester"`
}

var projectStoryStatuses = []string{
	model.StoryStatusPending,
	model.StoryStatusBacklog,
	model.StoryStatusReady,
	model.StoryStatusInProgress,
	model.StoryStatusTest,
	model.StoryStatusDone,
}

type projectItem struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	AgileMode   string      `json:"agile_mode"`
	Owner       interface{} `json:"owner,omitempty"`
	MemberCount int64       `json:"member_count"`
	StoryCount  int64       `json:"story_count"`
	IsOwner     bool        `json:"is_owner"`
	Archived    bool        `json:"archived"`
	ArchivedAt  *time.Time  `json:"archived_at,omitempty"`
	CreatedAt   any         `json:"created_at"`
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	var req createProjectRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	exists, err := h.projectRepo.ExistsByOwnerAndName(userID, req.Name)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists {
		api.Conflict(c, "同一用户不能创建同名项目")
		return
	}

	project := model.Project{
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
		OwnerID:     userID,
		AgileMode:   req.AgileMode,
	}

	member := model.ProjectMember{
		ProjectID:     project.ID,
		UserID:        userID,
		RoleInProject: c.GetString(middleware.CtxRoleKey),
	}
	if member.RoleInProject == "" {
		member.RoleInProject = model.RoleDeveloper
	}

	defaultColumns := []model.BoardColumn{
		{ProjectID: project.ID, Name: "待办", Position: 1},
		{ProjectID: project.ID, Name: "就绪", Position: 2},
		{ProjectID: project.ID, Name: "开发中", Position: 3},
		{ProjectID: project.ID, Name: "测试中", Position: 4},
		{ProjectID: project.ID, Name: "已完成", Position: 5},
	}

	if err := h.projectRepo.CreateWithTransaction(&project, &member, defaultColumns); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	owner, err := h.userRepo.FindByID(userID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "项目创建成功", gin.H{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"agile_mode":  project.AgileMode,
		"owner": gin.H{
			"id":    owner.ID,
			"email": owner.Email,
		},
		"created_at": project.CreatedAt,
	})
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)

	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	search := strings.TrimSpace(c.Query("search"))

	query := h.projectRepo.DB().Model(&model.Project{})
	if role != model.RoleAdmin {
		projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
		if err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		if len(projectIDs) == 0 {
			api.Success(c, "success", gin.H{
				"projects": []projectItem{},
				"total":    0,
				"page":     page,
				"limit":    limit,
			})
			return
		}
		query = query.Where("projects.id IN ?", projectIDs)
	}

	// Hide archived projects by default so the dashboard stays clean. Callers
	// can opt back in via ?include_archived=true (any tabs that show archived
	// projects explicitly) or ?archived_only=true to enumerate just the archive.
	includeArchived := strings.EqualFold(strings.TrimSpace(c.Query("include_archived")), "true")
	archivedOnly := strings.EqualFold(strings.TrimSpace(c.Query("archived_only")), "true")
	switch {
	case archivedOnly:
		query = query.Where("projects.archived = ?", true)
	case !includeArchived:
		query = query.Where("projects.archived = ?", false)
	}

	if search != "" {
		like := fmt.Sprintf("%%%s%%", search)
		query = query.Where("projects.name LIKE ?", like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var projects []model.Project
	if err := query.
		Order("projects.created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&projects).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]projectItem, 0, len(projects))
	for _, p := range projects {
		memberCount, _ := h.projectRepo.CountMembers(p.ID)

		var storyCount int64
		logging.LogIfErr(h.storyRepo.DB().Model(&model.UserStory{}).Where("project_id = ?", p.ID).Count(&storyCount).Error, "count project stories failed", "project_id", p.ID)

		owner, err := h.userRepo.FindByID(p.OwnerID)
		if err != nil {
			logging.LogIfErr(err, "load project owner failed", "project_id", p.ID, "owner_id", p.OwnerID)
		}

		items = append(items, projectItem{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			AgileMode:   p.AgileMode,
			Owner: gin.H{
				"id":    owner.ID,
				"email": owner.Email,
			},
			MemberCount: memberCount,
			StoryCount:  storyCount,
			IsOwner:     p.OwnerID == userID,
			Archived:    p.Archived,
			ArchivedAt:  p.ArchivedAt,
			CreatedAt:   p.CreatedAt,
		})
	}

	api.Success(c, "success", gin.H{
		"projects": items,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	project := middleware.MustProject(c)
	isOwner := middleware.IsProjectOwner(c)

	members, err := h.projectRepo.ListMembers(project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	statusBreakdown, totalStories, err := h.storyStats(project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	doneCount := statusBreakdown[model.StoryStatusDone]
	completionRate := 0.0
	if totalStories > 0 {
		completionRate = (float64(doneCount) / float64(totalStories)) * 100
	}

	memberPayload := make([]gin.H, 0, len(members))
	for _, m := range members {
		item := gin.H{"role_in_project": m.RoleInProject}
		if m.User != nil {
			item["user"] = gin.H{"id": m.User.ID, "email": m.User.Email}
		}
		memberPayload = append(memberPayload, item)
	}

	owner := gin.H{}
	if project.Owner != nil {
		owner["id"] = project.Owner.ID
		owner["email"] = project.Owner.Email
	}

	api.Success(c, "success", gin.H{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"agile_mode":  project.AgileMode,
		"owner":       owner,
		"members":     memberPayload,
		"statistics": gin.H{
			"total_stories":    totalStories,
			"status_breakdown": statusBreakdown,
			"completion_rate":  completionRate,
		},
		"is_owner":   isOwner,
		"created_at": project.CreatedAt,
	})
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	if !middleware.IsProjectOwner(c) {
		api.Forbidden(c, "仅项目Owner可更新项目")
		return
	}

	var req updateProjectRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	changed := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len([]rune(name)) < 2 || len([]rune(name)) > 100 {
			api.BadRequest(c, "项目名称长度应在2-100字符之间")
			return
		}

		exists, err := h.projectRepo.ExistsByOwnerAndName(userID, name, project.ID)
		if err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		if exists {
			api.Conflict(c, "同一用户不能创建同名项目")
			return
		}

		if project.Name != name {
			project.Name = name
			changed = true
		}
	}

	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if len([]rune(description)) > 500 {
			api.BadRequest(c, "项目描述最多500字符")
			return
		}
		if project.Description != description {
			project.Description = description
			changed = true
		}
	}

	if req.AgileMode != nil {
		mode := strings.TrimSpace(*req.AgileMode)
		if mode != model.AgileModeScrum && mode != model.AgileModeKanban {
			api.BadRequest(c, "agile_mode仅支持 scrum/kanban")
			return
		}
		if project.AgileMode != mode {
			project.AgileMode = mode
			changed = true
		}
	}

	if changed {
		if err := h.projectRepo.Save(project); err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
	}

	owner, err := h.userRepo.FindByID(project.OwnerID)
	if err != nil {
		logging.LogIfErr(err, "load project owner failed", "project_id", project.ID, "owner_id", project.OwnerID)
	}

	api.Success(c, "项目更新成功", gin.H{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"agile_mode":  project.AgileMode,
		"owner": gin.H{
			"id":    owner.ID,
			"email": owner.Email,
		},
		"updated_at": project.UpdatedAt,
	})
}

func (h *ProjectHandler) GetOverview(c *gin.Context) {
	project := middleware.MustProject(c)

	statusBreakdown, totalStories, err := h.storyStats(project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	doneCount := statusBreakdown[model.StoryStatusDone]
	completionRate := 0.0
	if totalStories > 0 {
		completionRate = (float64(doneCount) / float64(totalStories)) * 100
	}

	activeMembers, _ := h.projectRepo.CountMembers(project.ID)

	var avgPoints float64
	logging.LogIfErr(h.storyRepo.DB().Model(&model.UserStory{}).
		Where("project_id = ? AND points IS NOT NULL", project.ID).
		Select("COALESCE(AVG(points), 0)").
		Scan(&avgPoints).Error, "compute avg story points failed", "project_id", project.ID)

	api.Success(c, "success", gin.H{
		"project": gin.H{
			"id":   project.ID,
			"name": project.Name,
		},
		"statistics": gin.H{
			"total_stories":    totalStories,
			"status_breakdown": statusBreakdown,
			"completion_rate":  completionRate,
			"active_members":   activeMembers,
			"avg_story_points": avgPoints,
		},
		"recent_activities": h.recentActivities(project.ID, 10),
		"created_at":        project.CreatedAt,
		"last_updated_at":   h.lastUpdatedAt(project.ID, project.UpdatedAt),
	})
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	project := middleware.MustProject(c)

	if !middleware.IsProjectOwner(c) {
		api.Forbidden(c, "仅项目Owner可删除项目")
		return
	}

	if err := h.projectRepo.Delete(project.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "项目删除成功", gin.H{"id": project.ID})
}

func (h *ProjectHandler) ListMembers(c *gin.Context) {
	project := middleware.MustProject(c)
	isOwner := middleware.IsProjectOwner(c)

	members, err := h.projectRepo.ListMembers(project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	list := make([]gin.H, 0, len(members))
	for _, m := range members {
		entry := gin.H{
			"id":              m.ID,
			"user_id":         m.UserID,
			"role_in_project": m.RoleInProject,
			"joined_at":       m.JoinedAt,
			"is_owner":        project.OwnerID == m.UserID,
		}
		if m.User != nil {
			entry["user"] = gin.H{
				"id":         m.User.ID,
				"email":      m.User.Email,
				"avatar_url": m.User.AvatarURL,
			}
		}
		list = append(list, entry)
	}

	api.Success(c, "success", gin.H{
		"project_id": project.ID,
		"is_owner":   isOwner,
		"members":    list,
	})
}

func (h *ProjectHandler) AddMember(c *gin.Context) {
	project := middleware.MustProject(c)

	if !middleware.IsProjectOwner(c) {
		api.Forbidden(c, "仅项目Owner可管理成员")
		return
	}

	var req addProjectMemberRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	target, err := h.userRepo.FindByID(req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	isMember, err := h.projectRepo.IsMember(project.ID, target.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if isMember {
		api.Conflict(c, "该用户已在项目中")
		return
	}

	member := model.ProjectMember{
		ProjectID:     project.ID,
		UserID:        target.ID,
		RoleInProject: req.RoleInProject,
	}
	if err := h.projectRepo.AddMember(&member); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "成员添加成功", gin.H{
		"project_id":      project.ID,
		"user_id":         target.ID,
		"role_in_project": member.RoleInProject,
	})
}

func (h *ProjectHandler) ListMemberCandidates(c *gin.Context) {
	project := middleware.MustProject(c)

	if !middleware.IsProjectOwner(c) {
		api.Forbidden(c, "仅项目Owner可管理成员")
		return
	}

	subQuery := h.projectRepo.DB().Model(&model.ProjectMember{}).
		Select("user_id").
		Where("project_id = ?", project.ID)

	var users []model.User
	if err := h.userRepo.DB().
		Select("id, email, role, avatar_url, created_at").
		Where("id <> ?", project.OwnerID).
		Where("id NOT IN (?)", subQuery).
		Order("email ASC").
		Find(&users).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(users))
	for _, user := range users {
		items = append(items, gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"avatar_url": user.AvatarURL,
			"created_at": user.CreatedAt,
		})
	}

	api.Success(c, "success", gin.H{
		"project_id": project.ID,
		"users":      items,
	})
}

func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	project := middleware.MustProject(c)

	targetUserID, ok := parseUintParam(c, "userID")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	if !middleware.IsProjectOwner(c) {
		api.Forbidden(c, "仅项目Owner可管理成员")
		return
	}

	if targetUserID == project.OwnerID {
		api.BadRequest(c, "不能移除项目Owner")
		return
	}

	rowsAffected, err := h.projectRepo.RemoveMember(project.ID, targetUserID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if rowsAffected == 0 {
		api.NotFound(c, "项目成员不存在")
		return
	}

	api.Success(c, "成员移除成功", gin.H{
		"project_id": project.ID,
		"user_id":    targetUserID,
	})
}

var errForbidden = fmt.Errorf("forbidden")

// ArchiveProject hides a project from default listings without deleting it.
// Owner-only. Stories/sprints/bugs stay intact and accessible via direct ID
// access; ListProjects filters archived projects out unless include_archived
// or archived_only is set.
func (h *ProjectHandler) ArchiveProject(c *gin.Context) {
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	if !middleware.IsProjectOwner(c) {
		role, _ := middleware.CurrentRole(c)
		if role != model.RoleAdmin {
			api.Forbidden(c, "仅项目Owner或管理员可归档项目")
			return
		}
	}

	if project.Archived {
		api.Success(c, "项目已处于归档状态", gin.H{"id": project.ID, "archived": true})
		return
	}

	now := time.Now()
	project.Archived = true
	project.ArchivedAt = &now
	if err := h.projectRepo.Save(project); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &project.ID, userID, "project", project.ID, "archived",
		nil, map[string]any{"archived": true, "archived_at": now}),
		"write project activity log", "project_id", project.ID, "action", "archived")

	api.Success(c, "项目归档成功", gin.H{
		"id":          project.ID,
		"archived":    project.Archived,
		"archived_at": project.ArchivedAt,
	})
}

// UnarchiveProject restores an archived project so it appears in default
// listings again. Owner or admin only.
func (h *ProjectHandler) UnarchiveProject(c *gin.Context) {
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	if !middleware.IsProjectOwner(c) {
		role, _ := middleware.CurrentRole(c)
		if role != model.RoleAdmin {
			api.Forbidden(c, "仅项目Owner或管理员可还原项目")
			return
		}
	}

	if !project.Archived {
		api.Success(c, "项目未处于归档状态", gin.H{"id": project.ID, "archived": false})
		return
	}

	project.Archived = false
	project.ArchivedAt = nil
	if err := h.projectRepo.Save(project); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &project.ID, userID, "project", project.ID, "unarchived",
		nil, map[string]any{"archived": false}),
		"write project activity log", "project_id", project.ID, "action", "unarchived")

	api.Success(c, "项目还原成功", gin.H{
		"id":       project.ID,
		"archived": project.Archived,
	})
}

// ExportProject returns a JSON snapshot covering project metadata, members,
// stories, sprints, bugs, and test cases so the owner can back the project up
// or migrate it elsewhere. Only the project owner / admin can export.
func (h *ProjectHandler) ExportProject(c *gin.Context) {
	project := middleware.MustProject(c)
	userID, _ := middleware.CurrentUserID(c)

	if !middleware.IsProjectOwner(c) {
		role, _ := middleware.CurrentRole(c)
		if role != model.RoleAdmin {
			api.Forbidden(c, "仅项目Owner或管理员可导出项目")
			return
		}
	}

	snapshot, err := service.ExportSnapshot(h.db, project.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	memberPayload := make([]gin.H, 0, len(snapshot.Members))
	for _, m := range snapshot.Members {
		entry := gin.H{
			"user_id":         m.UserID,
			"role_in_project": m.RoleInProject,
			"joined_at":       m.JoinedAt,
		}
		if m.User != nil {
			entry["email"] = m.User.Email
		}
		memberPayload = append(memberPayload, entry)
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &project.ID, userID, "project", project.ID, "exported",
		nil, map[string]any{"stories": len(snapshot.Stories), "sprints": len(snapshot.Sprints), "bugs": len(snapshot.Bugs)}),
		"write project activity log", "project_id", project.ID, "action", "exported")

	api.Success(c, "项目导出成功", gin.H{
		"format_version": snapshot.FormatVersion,
		"exported_at":    snapshot.ExportedAt,
		"project": gin.H{
			"id":          snapshot.Project.ID,
			"name":        snapshot.Project.Name,
			"description": snapshot.Project.Description,
			"agile_mode":  snapshot.Project.AgileMode,
			"owner_id":    snapshot.Project.OwnerID,
			"archived":    snapshot.Project.Archived,
			"created_at":  snapshot.Project.CreatedAt,
		},
		"members":    memberPayload,
		"stories":    snapshot.Stories,
		"sprints":    snapshot.Sprints,
		"bugs":       snapshot.Bugs,
		"tasks":      snapshot.Tasks,
		"test_cases": snapshot.TestCases,
	})
}

func (h *ProjectHandler) storyStats(projectID uint) (map[string]int64, int64, error) {
	var totalStories int64
	if err := h.storyRepo.DB().Model(&model.UserStory{}).Where("project_id = ?", projectID).Count(&totalStories).Error; err != nil {
		return nil, 0, err
	}

	statusBreakdown := make(map[string]int64, len(projectStoryStatuses))
	for _, status := range projectStoryStatuses {
		var count int64
		if err := h.storyRepo.DB().Model(&model.UserStory{}).
			Where("project_id = ? AND status = ?", projectID, status).
			Count(&count).Error; err != nil {
			return nil, 0, err
		}
		statusBreakdown[status] = count
	}

	return statusBreakdown, totalStories, nil
}

func (h *ProjectHandler) recentActivities(projectID uint, limit int) []gin.H {
	logs, err := h.activityRepo.ListRecentByProject(projectID, limit)
	if err != nil {
		return []gin.H{}
	}

	items := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		userLabel := "系统"
		if log.User != nil && log.User.Email != "" {
			userLabel = log.User.Email
		}

		items = append(items, gin.H{
			"action":      log.Action,
			"user":        userLabel,
			"description": fmt.Sprintf("%s %s %s#%d", userLabel, actionText(log.Action), log.EntityType, log.EntityID),
			"created_at":  log.CreatedAt,
		})
	}
	return items
}

func (h *ProjectHandler) lastUpdatedAt(projectID uint, fallback time.Time) time.Time {
	last := fallback

	if t, ok := h.queryProjectMaxTime(&model.UserStory{}, "updated_at", projectID); ok && t.After(last) {
		last = t
	}

	if t, ok := h.queryProjectMaxTime(&model.ActivityLog{}, "created_at", projectID); ok && t.After(last) {
		last = t
	}

	return last
}

func (h *ProjectHandler) queryProjectMaxTime(target any, column string, projectID uint) (time.Time, bool) {
	var raw any
	row := h.projectRepo.DB().Model(target).
		Select(fmt.Sprintf("MAX(%s) AS t", column)).
		Where("project_id = ?", projectID).
		Row()
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, false
		}
		return time.Time{}, false
	}

	t, ok := dateparse.ParseAny(raw)
	if !ok || t.IsZero() {
		return time.Time{}, false
	}
	return t, true
}

func actionText(action string) string {
	switch action {
	case "created":
		return "创建了"
	case "updated":
		return "更新了"
	case "status_changed":
		return "变更了状态"
	case "claimed":
		return "领取了"
	case "released":
		return "释放了"
	case "ac_updated":
		return "更新了AC"
	default:
		return "操作了"
	}
}
