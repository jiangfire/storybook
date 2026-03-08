package handler

import (
	"fmt"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
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

	like := fmt.Sprintf("%%%s%%", q)
	data := gin.H{}

	if searchType == "all" || searchType == "project" {
		data["projects"] = h.searchProjects(userID, like, limit)
	}
	if searchType == "all" || searchType == "story" {
		data["stories"] = h.searchStories(userID, like, limit)
	}
	if searchType == "all" || searchType == "bug" {
		data["bugs"] = h.searchBugs(userID, like, limit)
	}

	api.Success(c, "success", data)
}

func (h *SearchHandler) searchProjects(userID uint, like string, limit int) []gin.H {
	var rows []model.Project
	err := h.db.Model(&model.Project{}).
		Joins("LEFT JOIN project_members pm ON pm.project_id = projects.id").
		Where("(projects.owner_id = ? OR pm.user_id = ?) AND projects.name LIKE ?", userID, userID, like).
		Group("projects.id").
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

func (h *SearchHandler) searchStories(userID uint, like string, limit int) []gin.H {
	var rows []model.UserStory
	err := h.db.Model(&model.UserStory{}).
		Joins("JOIN projects p ON p.id = user_stories.project_id").
		Joins("LEFT JOIN project_members pm ON pm.project_id = p.id").
		Where("(p.owner_id = ? OR pm.user_id = ?) AND user_stories.archived = false AND (user_stories.title LIKE ? OR user_stories.description LIKE ?)", userID, userID, like, like).
		Group("user_stories.id").
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

func (h *SearchHandler) searchBugs(userID uint, like string, limit int) []gin.H {
	var rows []model.BugReport
	err := h.db.Model(&model.BugReport{}).
		Joins("JOIN projects p ON p.id = bug_reports.project_id").
		Joins("LEFT JOIN project_members pm ON pm.project_id = p.id").
		Where("(p.owner_id = ? OR pm.user_id = ?) AND (bug_reports.title LIKE ? OR bug_reports.description LIKE ?)", userID, userID, like, like).
		Group("bug_reports.id").
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
