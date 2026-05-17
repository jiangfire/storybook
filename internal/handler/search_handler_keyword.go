package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/repository"
	"github.com/jiangfire/storybook/internal/service"
	"github.com/jiangfire/storybook/internal/util/dateparse"
)

func (h *SearchHandler) Search(c *gin.Context) {
	userID := middleware.MustUserID(c)

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

	filters, err := parseSearchFilters(c)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	like := fmt.Sprintf("%%%s%%", q)
	projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	data := gin.H{}

	if searchType == "all" || searchType == "project" {
		data["projects"] = h.searchProjects(projectIDs, like, limit, filters)
	}
	if searchType == "all" || searchType == "story" {
		data["stories"] = h.searchStories(projectIDs, like, limit, filters)
	}
	if searchType == "all" || searchType == "bug" {
		data["bugs"] = h.searchBugs(projectIDs, like, limit, filters)
	}

	api.Success(c, "success", data)
}

// searchFilters captures the optional advanced filters supported by GET /api/search.
// All fields are independent: date_range narrows by created_at, statuses narrows by
// the entity's status column (multi-value), and assignee narrows by assigned_to.
type searchFilters struct {
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Statuses    []string
	AssigneeID  *uint
}

// parseSearchFilters extracts created_from / created_to (RFC3339 or YYYY-MM-DD),
// status (repeatable or comma-separated), and assignee (user ID) from the query
// string. Returns a user-facing error if any value is malformed.
func parseSearchFilters(c *gin.Context) (searchFilters, error) {
	f := searchFilters{}

	if v := strings.TrimSpace(c.Query("created_from")); v != "" {
		t, err := dateparse.Parse(v)
		if err != nil {
			return f, fmt.Errorf("created_from 格式无效，需为 RFC3339 或 YYYY-MM-DD")
		}
		f.CreatedFrom = &t
	}
	if v := strings.TrimSpace(c.Query("created_to")); v != "" {
		t, err := dateparse.Parse(v)
		if err != nil {
			return f, fmt.Errorf("created_to 格式无效，需为 RFC3339 或 YYYY-MM-DD")
		}
		// When only a date is supplied, treat the upper bound as end-of-day so
		// "created_to=2026-05-15" matches anything on that day.
		if len(v) == 10 {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		f.CreatedTo = &t
	}

	// status accepts both repeated (?status=open&status=closed) and comma-separated
	// (?status=open,closed) forms so callers can pick whichever the client lib makes easy.
	raw := append([]string(nil), c.QueryArray("status")...)
	if csv := strings.TrimSpace(c.Query("statuses")); csv != "" {
		raw = append(raw, strings.Split(csv, ",")...)
	}
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			f.Statuses = append(f.Statuses, s)
		}
	}

	if v := strings.TrimSpace(c.Query("assignee")); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || id == 0 {
			return f, fmt.Errorf("assignee 必须为有效用户 ID")
		}
		uid := uint(id)
		f.AssigneeID = &uid
	}

	return f, nil
}

func (h *SearchHandler) searchProjects(projectIDs []uint, like string, limit int, f searchFilters) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	q := h.projectRepo.DB().Model(&model.Project{}).
		Where("projects.id IN ? AND projects.name LIKE ?", projectIDs, like)
	if f.CreatedFrom != nil {
		q = q.Where("projects.created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		q = q.Where("projects.created_at <= ?", *f.CreatedTo)
	}
	var rows []model.Project
	if err := q.Order("projects.updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
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

func (h *SearchHandler) searchStories(projectIDs []uint, like string, limit int, f searchFilters) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	q := h.storyRepo.DB().Model(&model.UserStory{}).
		Where("project_id IN ? AND archived = false AND (title LIKE ? OR description LIKE ?)", projectIDs, like, like)
	if f.CreatedFrom != nil {
		q = q.Where("created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		q = q.Where("created_at <= ?", *f.CreatedTo)
	}
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	}
	if f.AssigneeID != nil {
		q = q.Where("assigned_to = ?", *f.AssigneeID)
	}
	var rows []model.UserStory
	if err := q.Order("user_stories.updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
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

func (h *SearchHandler) searchBugs(projectIDs []uint, like string, limit int, f searchFilters) []gin.H {
	if len(projectIDs) == 0 {
		return []gin.H{}
	}
	rows, err := h.bugRepo.SearchByProjects(projectIDs, like, limit, repository.BugSearchFilter{
		CreatedFrom: f.CreatedFrom,
		CreatedTo:   f.CreatedTo,
		Statuses:    f.Statuses,
		AssigneeID:  f.AssigneeID,
	})
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
