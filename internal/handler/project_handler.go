package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	db *gorm.DB
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
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

type projectItem struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	AgileMode   string      `json:"agile_mode"`
	Owner       interface{} `json:"owner,omitempty"`
	MemberCount int64       `json:"member_count"`
	StoryCount  int64       `json:"story_count"`
	IsOwner     bool        `json:"is_owner"`
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

	var exists int64
	if err := h.db.Model(&model.Project{}).
		Where("owner_id = ? AND name = ?", userID, req.Name).
		Count(&exists).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	if exists > 0 {
		api.Conflict(c, "同一用户不能创建同名项目")
		return
	}

	project := model.Project{
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
		OwnerID:     userID,
		AgileMode:   req.AgileMode,
	}

	tx := h.db.Begin()
	if err := tx.Create(&project).Error; err != nil {
		tx.Rollback()
		api.Internal(c, "服务器内部错误")
		return
	}

	// 创建者自动加入项目成员。
	member := model.ProjectMember{
		ProjectID:     project.ID,
		UserID:        userID,
		RoleInProject: c.GetString(middleware.CtxRoleKey),
	}

	if member.RoleInProject == "" {
		member.RoleInProject = model.RoleDeveloper
	}

	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		api.Internal(c, "服务器内部错误")
		return
	}

	defaultColumns := []model.BoardColumn{
		{ProjectID: project.ID, Name: "待办", Position: 1},
		{ProjectID: project.ID, Name: "就绪", Position: 2},
		{ProjectID: project.ID, Name: "开发中", Position: 3},
		{ProjectID: project.ID, Name: "测试中", Position: 4},
		{ProjectID: project.ID, Name: "已完成", Position: 5},
	}

	if err := tx.Create(&defaultColumns).Error; err != nil {
		tx.Rollback()
		api.Internal(c, "服务器内部错误")
		return
	}

	if err := tx.Commit().Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	var owner model.User
	if err := h.db.First(&owner, userID).Error; err != nil {
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

	query := h.db.Model(&model.Project{})
	if role != model.RoleTechLead && role != model.RoleAdmin {
		query = query.
			Joins("LEFT JOIN project_members pm ON pm.project_id = projects.id").
			Where("projects.owner_id = ? OR pm.user_id = ?", userID, userID).
			Group("projects.id")
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
		var memberCount int64
		_ = h.db.Model(&model.ProjectMember{}).Where("project_id = ?", p.ID).Count(&memberCount).Error

		var storyCount int64
		_ = h.db.Model(&model.UserStory{}).Where("project_id = ?", p.ID).Count(&storyCount).Error

		var owner model.User
		_ = h.db.Select("id, email").First(&owner, p.OwnerID).Error

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

	project, isOwner, err := h.getProjectWithAccess(projectID, userID)
	if err != nil {
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

	var members []model.ProjectMember
	if err := h.db.Preload("User").Where("project_id = ?", project.ID).Find(&members).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	statusBreakdown := map[string]int64{}
	statuses := []string{model.StoryStatusBacklog, model.StoryStatusReady, model.StoryStatusInProgress, model.StoryStatusTest, model.StoryStatusDone}
	var totalStories int64
	for _, status := range statuses {
		var count int64
		_ = h.db.Model(&model.UserStory{}).Where("project_id = ? AND status = ?", project.ID, status).Count(&count).Error
		statusBreakdown[status] = count
		totalStories += count
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

	var project model.Project
	if err := h.db.First(&project, projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "项目不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	if project.OwnerID != userID {
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

		var exists int64
		if err := h.db.Model(&model.Project{}).
			Where("owner_id = ? AND name = ? AND id <> ?", userID, name, project.ID).
			Count(&exists).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		if exists > 0 {
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
		if err := h.db.Save(&project).Error; err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
	}

	var owner model.User
	_ = h.db.Select("id, email").First(&owner, project.OwnerID).Error

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

	project, _, err := h.getProjectWithAccess(projectID, userID)
	if err != nil {
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

	var totalStories int64
	if err := h.db.Model(&model.UserStory{}).Where("project_id = ?", project.ID).Count(&totalStories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	statusBreakdown := map[string]int64{}
	statuses := []string{model.StoryStatusBacklog, model.StoryStatusReady, model.StoryStatusInProgress, model.StoryStatusTest, model.StoryStatusDone}
	for _, status := range statuses {
		var count int64
		_ = h.db.Model(&model.UserStory{}).Where("project_id = ? AND status = ?", project.ID, status).Count(&count).Error
		statusBreakdown[status] = count
	}

	doneCount := statusBreakdown[model.StoryStatusDone]
	completionRate := 0.0
	if totalStories > 0 {
		completionRate = (float64(doneCount) / float64(totalStories)) * 100
	}

	var activeMembers int64
	_ = h.db.Model(&model.ProjectMember{}).Where("project_id = ?", project.ID).Count(&activeMembers).Error

	var avgPoints float64
	_ = h.db.Model(&model.UserStory{}).Where("project_id = ? AND points IS NOT NULL", project.ID).Select("AVG(points)").Scan(&avgPoints).Error

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

	var project model.Project
	if err := h.db.First(&project, projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "项目不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	if project.OwnerID != userID {
		api.Forbidden(c, "仅项目Owner可删除项目")
		return
	}

	if err := h.db.Delete(&project).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "项目删除成功", gin.H{"id": projectID})
}

func (h *ProjectHandler) ListMembers(c *gin.Context) {
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

	project, isOwner, err := h.getProjectWithAccess(projectID, userID)
	if err != nil {
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

	var members []model.ProjectMember
	if err := h.db.Preload("User").Where("project_id = ?", project.ID).Order("id ASC").Find(&members).Error; err != nil {
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

	project, _, err := h.getProjectWithAccess(projectID, userID)
	if err != nil {
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

	if project.OwnerID != userID {
		api.Forbidden(c, "仅项目Owner可管理成员")
		return
	}

	var req addProjectMemberRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var target model.User
	if err := h.db.First(&target, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var exists int64
	if err := h.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", project.ID, target.ID).
		Count(&exists).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists > 0 {
		api.Conflict(c, "该用户已在项目中")
		return
	}

	member := model.ProjectMember{
		ProjectID:     project.ID,
		UserID:        target.ID,
		RoleInProject: req.RoleInProject,
	}
	if err := h.db.Create(&member).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "成员添加成功", gin.H{
		"project_id":      project.ID,
		"user_id":         target.ID,
		"role_in_project": member.RoleInProject,
	})
}

func (h *ProjectHandler) RemoveMember(c *gin.Context) {
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
	targetUserID, ok := parseUintParam(c, "userID")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	project, _, err := h.getProjectWithAccess(projectID, userID)
	if err != nil {
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

	if project.OwnerID != userID {
		api.Forbidden(c, "仅项目Owner可管理成员")
		return
	}

	if targetUserID == project.OwnerID {
		api.BadRequest(c, "不能移除项目Owner")
		return
	}

	result := h.db.Where("project_id = ? AND user_id = ?", project.ID, targetUserID).Delete(&model.ProjectMember{})
	if result.Error != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if result.RowsAffected == 0 {
		api.NotFound(c, "项目成员不存在")
		return
	}

	api.Success(c, "成员移除成功", gin.H{
		"project_id": project.ID,
		"user_id":    targetUserID,
	})
}

var errForbidden = fmt.Errorf("forbidden")

func (h *ProjectHandler) getProjectWithAccess(projectID, userID uint) (*model.Project, bool, error) {
	var project model.Project
	if err := h.db.Preload("Owner").First(&project, projectID).Error; err != nil {
		return nil, false, err
	}

	if project.OwnerID == userID {
		return &project, true, nil
	}

	var user model.User
	if err := h.db.Select("id, role").First(&user, userID).Error; err == nil {
		if user.Role == model.RoleTechLead || user.Role == model.RoleAdmin {
			return &project, false, nil
		}
	} else {
		return nil, false, err
	}

	var exists int64
	if err := h.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", project.ID, userID).
		Count(&exists).Error; err != nil {
		return nil, false, err
	}

	if exists == 0 {
		return nil, false, errForbidden
	}

	return &project, false, nil
}

func (h *ProjectHandler) recentActivities(projectID uint, limit int) []gin.H {
	var logs []model.ActivityLog
	if err := h.db.Where("project_id = ?", projectID).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
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
	var row struct {
		T *time.Time `gorm:"column:t"`
	}
	if err := h.db.Model(target).
		Select(fmt.Sprintf("MAX(%s) AS t", column)).
		Where("project_id = ?", projectID).
		Scan(&row).Error; err != nil {
		return time.Time{}, false
	}

	if row.T == nil || row.T.IsZero() {
		return time.Time{}, false
	}
	return *row.T, true
}

func normalizeAnyTime(raw any) (time.Time, bool) {
	switch v := raw.(type) {
	case nil:
		return time.Time{}, false
	case time.Time:
		return v, !v.IsZero()
	case *time.Time:
		if v == nil || v.IsZero() {
			return time.Time{}, false
		}
		return *v, true
	case sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			return time.Time{}, false
		}
		return v.Time, true
	case string:
		return parseFlexibleTime(v)
	case []byte:
		return parseFlexibleTime(string(v))
	case int64:
		return normalizeUnixTime(v), true
	case int32:
		return normalizeUnixTime(int64(v)), true
	case int:
		return normalizeUnixTime(int64(v)), true
	case float64:
		return normalizeUnixTime(int64(v)), true
	case float32:
		return normalizeUnixTime(int64(v)), true
	default:
		return parseFlexibleTime(fmt.Sprint(v))
	}
}

func parseFlexibleTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "<nil>") {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, !t.IsZero()
		}
	}

	if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return normalizeUnixTime(unix), true
	}

	return time.Time{}, false
}

func normalizeUnixTime(v int64) time.Time {
	// 13位按毫秒，10位按秒。
	if v > 9999999999 {
		return time.UnixMilli(v)
	}
	return time.Unix(v, 0)
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
