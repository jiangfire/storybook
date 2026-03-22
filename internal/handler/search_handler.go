package handler

import (
	"fmt"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

func (h *SearchHandler) Search(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		api.BadRequest(c, "q不能为空")
		return
	}
	searchType := strings.ToLower(strings.TrimSpace(c.DefaultQuery("type", "all")))
	limit := parseIntQuery(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	role, _ := middleware.CurrentRole(c)

	like := fmt.Sprintf("%%%s%%", q)
	projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	data := gin.H{}

	if searchType == "all" || searchType == "project" {
		data["projects"] = h.searchProjects(projectIDs, like, limit)
	}
	if searchType == "all" || searchType == "story" {
		data["stories"] = h.searchStories(projectIDs, like, limit)
	}
	if searchType == "all" || searchType == "bug" {
		data["bugs"] = h.searchBugs(projectIDs, like, limit)
	}

	api.Success(c, "success", data)
}

func (h *SearchHandler) searchProjects(projectIDs []uint, like string, limit int) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	var rows []model.Project
	err := h.db.Model(&model.Project{}).
		Where("projects.id IN ? AND projects.name LIKE ?", projectIDs, like).
		Order("projects.updated_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return []gin.H{}
	}

	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		out = append(out, gin.H{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"agile_mode":  p.AgileMode,
			"created_at":  p.CreatedAt,
		})
	}
	return out
}

func (h *SearchHandler) searchStories(projectIDs []uint, like string, limit int) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	var rows []model.UserStory
	err := h.db.Model(&model.UserStory{}).
		Where("project_id IN ? AND archived = false AND (title LIKE ? OR description LIKE ?)", projectIDs, like, like).
		Order("user_stories.updated_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return []gin.H{}
	}

	out := make([]gin.H, 0, len(rows))
	for _, s := range rows {
		out = append(out, gin.H{
			"id":         s.ID,
			"project_id": s.ProjectID,
			"title":      s.Title,
			"story_type": s.StoryType,
			"status":     s.Status,
			"priority":   s.Priority,
			"updated_at": s.UpdatedAt,
		})
	}
	return out
}

func (h *SearchHandler) searchBugs(projectIDs []uint, like string, limit int) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	var rows []model.BugReport
	err := h.db.Model(&model.BugReport{}).
		Where("project_id IN ? AND (title LIKE ? OR description LIKE ?)", projectIDs, like, like).
		Order("bug_reports.updated_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return []gin.H{}
	}

	out := make([]gin.H, 0, len(rows))
	for _, b := range rows {
		out = append(out, gin.H{
			"id":         b.ID,
			"project_id": b.ProjectID,
			"story_id":   b.StoryID,
			"title":      b.Title,
			"severity":   b.Severity,
			"status":     b.Status,
			"updated_at": b.UpdatedAt,
		})
	}
	return out
}
