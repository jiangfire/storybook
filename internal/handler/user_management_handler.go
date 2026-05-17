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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserManagementHandler struct {
	db         *gorm.DB
	userRepo   repository.UserRepo
	storyRepo  repository.StoryRepo
	taskRepo   repository.TaskRepo
	activityRepo repository.ActivityRepo
}

func NewUserManagementHandler(db *gorm.DB) *UserManagementHandler {
	return &UserManagementHandler{
		db:         db,
		userRepo:   repository.NewUserRepository(db),
		storyRepo:  repository.NewStoryRepository(db),
		taskRepo:   repository.NewTaskRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
	}
}

// ListUsers 获取所有用户列表
func (h *UserManagementHandler) ListUsers(c *gin.Context) {
	_, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	roleFilter := strings.TrimSpace(c.Query("role"))
	search := strings.TrimSpace(c.Query("search"))

	// 分页
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	users, total, err := h.userRepo.ListFiltered(roleFilter, search, page, limit)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(users))
	for _, user := range users {
		items = append(items, gin.H{
			"id":            user.ID,
			"email":         user.Email,
			"username":      user.Username,
			"role":          user.Role,
			"avatar_url":    user.AvatarURL,
			"created_at":    user.CreatedAt,
			"last_login_at": user.LastLoginAt,
		})
	}

	api.Success(c, "success", gin.H{
		"users": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreateUser 创建用户（admin/tech_lead）
func (h *UserManagementHandler) CreateUser(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required,min=2,max=100"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role" binding:"required"`
	}
	if !middleware.BindJSON(c, &req) {
		return
	}

	if !isStrongPassword(req.Password) {
		api.BadRequest(c, "密码至少8位，包含字母和数字")
		return
	}

	// 验证角色
	validRoles := map[string]struct{}{
		model.RoleProduct:   {},
		model.RoleDeveloper: {},
		model.RoleTester:    {},
		model.RoleTechLead:  {},
	}
	if _, valid := validRoles[req.Role]; !valid {
		api.BadRequest(c, "无效的用户角色")
		return
	}

	// 只有admin可以创建tech_lead
	if req.Role == model.RoleTechLead && role != model.RoleAdmin {
		api.Forbidden(c, "只有管理员可以创建技术负责人")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.TrimSpace(req.Username)

	// 检查邮箱是否已存在
	exists, err := h.userRepo.ExistsByEmail(email)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists {
		api.Conflict(c, "邮箱已被注册")
		return
	}

	// 检查用户名是否已存在
	exists, err = h.userRepo.ExistsByUsername(username, nil)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}
	if exists {
		api.Conflict(c, "用户名已被使用")
		return
	}

	// 创建用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	user := model.User{
		Email:          email,
		Username:       username,
		HashedPassword: string(hashedPassword),
		Role:           req.Role,
	}

	if err := h.userRepo.Create(&user); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// 记录活动
	logging.LogIfErr(service.WriteActivityLog(h.db, nil, userID, "user", user.ID, "created",
		nil, map[string]any{"email": user.Email, "username": user.Username, "role": user.Role}),
		"write user activity log", "user_id", user.ID, "action", "created")

	api.Success(c, "用户创建成功", gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
	})
}

// UpdateUser 更新用户信息
func (h *UserManagementHandler) UpdateUser(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	// 不能修改自己以外的admin用户（除非自己是admin）
	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	// tech_lead不能修改admin用户
	if targetUser.Role == model.RoleAdmin && role != model.RoleAdmin {
		api.Forbidden(c, "无权修改管理员用户")
		return
	}

	var req struct {
		Username *string `json:"username"`
		Role     *string `json:"role"`
		Password *string `json:"password"`
	}
	if !middleware.BindJSON(c, &req) {
		return
	}

	changed := false
	oldValues := map[string]any{}
	newValues := map[string]any{}

	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if len(username) < 2 || len(username) > 100 {
			api.BadRequest(c, "用户名长度应在2-100字符之间")
			return
		}
		// 检查用户名是否已被使用
		exists, err := h.userRepo.ExistsByUsername(username, &targetUserID)
		if err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		if exists {
			api.Conflict(c, "用户名已被使用")
			return
		}
		if targetUser.Username != username {
			oldValues["username"] = targetUser.Username
			newValues["username"] = username
			targetUser.Username = username
			changed = true
		}
	}

	if req.Role != nil {
		validRoles := map[string]struct{}{
			model.RoleProduct:   {},
			model.RoleDeveloper: {},
			model.RoleTester:    {},
			model.RoleTechLead:  {},
			model.RoleAdmin:     {},
		}
		if _, valid := validRoles[*req.Role]; !valid {
			api.BadRequest(c, "无效的用户角色")
			return
		}
		// 只有admin可以修改为tech_lead或admin
		if (*req.Role == model.RoleTechLead || *req.Role == model.RoleAdmin) && role != model.RoleAdmin {
			api.Forbidden(c, "只有管理员可以设置该角色")
			return
		}
		if targetUser.Role != *req.Role {
			oldValues["role"] = targetUser.Role
			newValues["role"] = *req.Role
			targetUser.Role = *req.Role
			changed = true
		}
	}

	if req.Password != nil {
		if !isStrongPassword(*req.Password) {
			api.BadRequest(c, "密码至少8位，包含字母和数字")
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 10)
		if err != nil {
			api.Internal(c, "服务器内部错误")
			return
		}
		targetUser.HashedPassword = string(hashedPassword)
		changed = true
	}

	if !changed {
		api.Success(c, "success", gin.H{
			"id":      targetUser.ID,
			"message": "无字段变更",
		})
		return
	}

	if err := h.userRepo.Save(targetUser); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// 记录活动
	logging.LogIfErr(service.WriteActivityLog(h.db, nil, userID, "user", targetUser.ID, "updated",
		oldValues, newValues),
		"write user activity log", "user_id", targetUser.ID, "action", "updated")

	api.Success(c, "用户更新成功", gin.H{
		"id":         targetUser.ID,
		"username":   targetUser.Username,
		"role":       targetUser.Role,
		"updated_at": targetUser.UpdatedAt,
	})
}

// GetUserWorkload 获取用户工作负载详情
func (h *UserManagementHandler) GetUserWorkload(c *gin.Context) {
	_, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	user, err := h.userRepo.FindByID(targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	// 获取活跃故事
	var activeStories []model.UserStory
	h.storyRepo.DB().Where("assigned_to = ? AND status IN ?", targetUserID, []string{
		model.StoryStatusReady,
		model.StoryStatusInProgress,
		model.StoryStatusTest,
	}).Preload("Project").Find(&activeStories)

	storyItems := make([]gin.H, 0, len(activeStories))
	var totalStoryPoints int
	for _, story := range activeStories {
		item := gin.H{
			"id":     story.ID,
			"title":  story.Title,
			"status": story.Status,
		}
		if story.Points != nil {
			item["story_points"] = *story.Points
			totalStoryPoints += *story.Points
		}
		if story.Project != nil {
			item["project"] = gin.H{
				"id":   story.Project.ID,
				"name": story.Project.Name,
			}
		}
		storyItems = append(storyItems, item)
	}

	// 获取活跃任务
	var activeTasks []model.Task
	h.storyRepo.DB().Where("assigned_to = ? AND status IN ?", targetUserID, []string{
		model.TaskStatusTodo,
		model.TaskStatusInProgress,
		model.TaskStatusBlocked,
	}).Preload("Story").Find(&activeTasks)

	taskItems := make([]gin.H, 0, len(activeTasks))
	var totalEstimatedHours float64
	for _, task := range activeTasks {
		item := gin.H{
			"id":       task.ID,
			"title":    task.Title,
			"status":   task.Status,
			"progress": task.Progress,
		}
		if task.EstimatedHours > 0 {
			item["estimated_hours"] = task.EstimatedHours
			totalEstimatedHours += task.EstimatedHours
		}
		if task.Story != nil {
			item["story"] = gin.H{
				"id":    task.Story.ID,
				"title": task.Story.Title,
			}
		}
		taskItems = append(taskItems, item)
	}

	// 统计完成情况（最近30天）
	var stats struct {
		StoriesCompleted  int64
		StoriesInProgress int64
		TasksCompleted    int64
		TasksInProgress   int64
	}

	// 计算30天前的时间
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	h.storyRepo.DB().Model(&model.UserStory{}).
		Where("assigned_to = ? AND status = ? AND updated_at >= ?", targetUserID, model.StoryStatusDone, thirtyDaysAgo).
		Count(&stats.StoriesCompleted)

	h.storyRepo.DB().Model(&model.UserStory{}).
		Where("assigned_to = ? AND status = ?", targetUserID, model.StoryStatusInProgress).
		Count(&stats.StoriesInProgress)

	h.taskRepo.DB().Model(&model.Task{}).
		Where("assigned_to = ? AND status = ? AND updated_at >= ?", targetUserID, model.TaskStatusDone, thirtyDaysAgo).
		Count(&stats.TasksCompleted)

	h.taskRepo.DB().Model(&model.Task{}).
		Where("assigned_to = ? AND status = ?", targetUserID, model.TaskStatusInProgress).
		Count(&stats.TasksInProgress)

		// 计算平均完成时间
		avgCompletionDays, err := h.storyRepo.AvgCompletionDaysForUser(targetUserID, thirtyDaysAgo)
		if err != nil {
			api.Internal(c, "统计查询失败")
			return
		}

	api.Success(c, "success", gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"role":     user.Role,
		},
		"active_stories": storyItems,
		"active_tasks":   taskItems,
		"statistics": gin.H{
			"total_story_points":    totalStoryPoints,
			"total_estimated_hours": totalEstimatedHours,
			"stories_completed_30d": stats.StoriesCompleted,
			"stories_in_progress":   stats.StoriesInProgress,
			"tasks_completed_30d":   stats.TasksCompleted,
			"tasks_in_progress":     stats.TasksInProgress,
			"avg_completion_days":   avgCompletionDays,
		},
	})
}

// DeleteUser 删除用户（仅admin）
func (h *UserManagementHandler) DeleteUser(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可删除用户")
		return
	}

	targetUserID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "用户ID无效")
		return
	}

	// 不能删除自己
	if targetUserID == userID {
		api.BadRequest(c, "不能删除当前登录用户")
		return
	}

	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	// 删除用户（软删除或硬删除，这里使用硬删除）
	if err := h.userRepo.HardDelete(targetUser.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	// 记录活动
	logging.LogIfErr(service.WriteActivityLog(h.db, nil, userID, "user", targetUserID, "deleted",
		map[string]any{"email": targetUser.Email, "role": targetUser.Role}, nil),
		"write user activity log", "user_id", targetUserID, "action", "deleted")

	api.Success(c, "用户删除成功", gin.H{"id": targetUserID})
}
