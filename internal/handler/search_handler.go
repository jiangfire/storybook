package handler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"git.neolidy.top/neo/storybook/internal/util/dateparse"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchHandler struct {
	db        *gorm.DB
	vectorSvc service.VectorService
	storyRepo repository.StoryRepo
	bugRepo   repository.BugRepo
	projectRepo repository.ProjectRepo
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return NewSearchHandlerWithVector(db, nil)
}

// NewSearchHandlerWithVector 创建带向量搜索的 SearchHandler
func NewSearchHandlerWithVector(db *gorm.DB, vectorSvc service.VectorService) *SearchHandler {
	return &SearchHandler{
		db:        db,
		vectorSvc: vectorSvc,
		storyRepo: repository.NewStoryRepository(db),
		bugRepo:   repository.NewBugRepository(db),
		projectRepo: repository.NewProjectRepository(db),
	}
}

func (h *SearchHandler) Capabilities(c *gin.Context) {
	api.Success(c, "success", gin.H{
		"semantic_enabled": h.vectorSvc != nil,
		"filters": gin.H{
			"date_range":   []string{"created_from", "created_to"},
			"status_array": true,
			"assignee":     true,
		},
	})
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

// ========== 语义搜索功能 ==========

// SemanticSearchRequest 语义搜索请求
type SemanticSearchRequest struct {
	Query      string `form:"q" binding:"required"`
	ProjectIDs string `form:"project_ids"` // 逗号分隔的项目 ID
	Limit      int    `form:"limit"`
}

// SimilarStoriesRequest 相似故事推荐请求
type SimilarStoriesRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	ProjectID   uint   `json:"project_id" binding:"required"`
	Limit       int    `json:"limit"`
}

// SuggestTagsRequest 标签建议请求
type SuggestTagsRequest struct {
	Title       string `json:"title" binding:"required_without_all=Description"`
	Description string `json:"description" binding:"required_without_all=Title"`
	Content     string `json:"content"` // 可选的额外内容
	Limit       int    `json:"limit"`
}

// SearchSemantic 语义搜索
// GET /api/search/semantic?q=查询文本&project_ids=1,2,3&limit=10
func (h *SearchHandler) SearchSemantic(c *gin.Context) {
	// 检查向量服务是否可用
	if h.vectorSvc == nil {
		api.Internal(c, "向量搜索服务未启用")
		return
	}
	if h.db == nil {
		api.Internal(c, "搜索服务配置错误")
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	var req SemanticSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认 limit
	if req.Limit == 0 {
		req.Limit = 10
	} else if req.Limit < 1 || req.Limit > 100 {
		api.BadRequest(c, "limit 必须在 1 到 100 之间")
		return
	}

	// 解析项目 ID
	var projectIDs []uint
	if req.ProjectIDs != "" {
		for _, idStr := range strings.Split(req.ProjectIDs, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32)
			if err == nil {
				projectIDs = append(projectIDs, uint(id))
			}
		}
	}

	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}

	// 如果显式指定项目，则只保留用户有权限的项目，避免越权搜索。
	if len(projectIDs) == 0 {
		projectIDs = accessibleIDs
	} else {
		projectIDs = intersectUintSlices(projectIDs, accessibleIDs)
	}

	// 执行语义搜索
	stories, err := h.vectorSvc.SearchSimilarStories(c.Request.Context(), req.Query, projectIDs, req.Limit)
	if err != nil {
		api.Internal(c, "语义搜索失败")
		return
	}

	response := h.convertSimilarStoriesToResponse(stories)

	api.Success(c, "success", gin.H{
		"stories": response,
	})
}

// SimilarStories 相似故事推荐
// POST /api/stories/similar
func (h *SearchHandler) SimilarStories(c *gin.Context) {
	if h.vectorSvc == nil {
		api.Internal(c, "向量搜索服务未启用")
		return
	}
	if h.db == nil {
		api.Internal(c, "搜索服务配置错误")
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	var req SimilarStoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 检查项目权限
	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}
	if !contains(accessibleIDs, req.ProjectID) {
		api.Forbidden(c, "无权限访问该项目")
		return
	}

	// 设置默认 limit
	if req.Limit == 0 {
		req.Limit = 5
	} else if req.Limit < 1 || req.Limit > 50 {
		api.BadRequest(c, "limit 必须在 1 到 50 之间")
		return
	}

	query := req.Title
	if req.Description != "" {
		query += "\n" + req.Description
	}

	// 搜索相似故事（同项目内）
	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		query,
		[]uint{req.ProjectID}, // 已验证权限，直接使用
		req.Limit,
	)
	if err != nil {
		api.Internal(c, "相似故事搜索失败")
		return
	}

	// 过滤低相似度结果（< 0.7）并转换响应
	filtered := h.filterAndConvertStories(stories, 0.7)

	api.Success(c, "success", gin.H{
		"similar_stories": filtered,
	})
}

// SuggestTags 智能标签建议
// POST /api/tags/suggest
func (h *SearchHandler) SuggestTags(c *gin.Context) {
	if h.vectorSvc == nil {
		api.Internal(c, "向量搜索服务未启用")
		return
	}
	if h.db == nil {
		api.Internal(c, "搜索服务配置错误")
		return
	}

	var req SuggestTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认 limit
	if req.Limit == 0 {
		req.Limit = 5
	} else if req.Limit < 1 || req.Limit > 20 {
		api.BadRequest(c, "limit 必须在 1 到 20 之间")
		return
	}

	// 构建内容（单一职责）
	content := h.buildTagSuggestionContent(req.Title, req.Description, req.Content)

	// 获取用户有权限的项目
	userID, _ := middleware.CurrentUserID(c)
	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}

	// 搜索相似故事（从所有项目）
	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		content,
		accessibleIDs,
		req.Limit*3, // 获取更多结果用于统计
	)
	if err != nil {
		api.Internal(c, "标签建议失败")
		return
	}

	tags := h.extractTopTags(stories, req.Limit)

	api.Success(c, "success", gin.H{
		"tags": tags,
	})
}

// ========== 辅助方法 ==========

// convertSimilarStoriesToResponse 转换相似故事为响应格式
func (h *SearchHandler) convertSimilarStoriesToResponse(stories []service.SimilarStory) []gin.H {
	response := make([]gin.H, len(stories))
	for i, story := range stories {
		if us, ok := story.Story.(model.UserStory); ok {
			response[i] = gin.H{
				"id":         us.ID,
				"project_id": us.ProjectID,
				"title":      us.Title,
				"story_type": us.StoryType,
				"status":     us.Status,
				"priority":   us.Priority,
				"similarity": story.Similarity,
			}
		}
	}
	return response
}

// filterAndConvertStories 过滤并转换故事（单一职责）
func (h *SearchHandler) filterAndConvertStories(stories []service.SimilarStory, minSimilarity float64) []gin.H {
	var filtered []gin.H
	for _, story := range stories {
		if story.Similarity >= minSimilarity {
			if us, ok := story.Story.(model.UserStory); ok {
				filtered = append(filtered, gin.H{
					"id":         us.ID,
					"project_id": us.ProjectID,
					"title":      us.Title,
					"story_type": us.StoryType,
					"status":     us.Status,
					"priority":   us.Priority,
					"similarity": story.Similarity,
				})
			}
		}
	}
	return filtered
}

// buildTagSuggestionContent 构建标签建议内容（单一职责）
func (h *SearchHandler) buildTagSuggestionContent(title, description, content string) string {
	var parts []string
	if title != "" {
		parts = append(parts, title)
	}
	if description != "" {
		parts = append(parts, description)
	}
	if content != "" {
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n")
}

// extractTopTags 提取 top N 标签（单一职责）
func (h *SearchHandler) extractTopTags(stories []service.SimilarStory, limit int) []string {
	// 统计标签频率
	tagCounts := make(map[string]int)
	for _, story := range stories {
		if us, ok := story.Story.(model.UserStory); ok {
			tags := h.extractTagsFromStory(us)
			for _, tag := range tags {
				tagCounts[tag]++
			}
		}
	}

	type tagScore struct {
		Tag   string
		Count int
	}
	var scores []tagScore
	for tag, count := range tagCounts {
		scores = append(scores, tagScore{Tag: tag, Count: count})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].Count > scores[j].Count })

	// 返回 top N
	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(scores); i++ {
		result = append(result, scores[i].Tag)
	}
	return result
}

// extractTagsFromStory 从故事中提取标签（单一职责）
func (h *SearchHandler) extractTagsFromStory(story model.UserStory) []string {
	// 从 story.Tags JSONB 字段解析标签
	if story.Tags == nil {
		return []string{}
	}

	// 尝试解析为 []string
	var tags []string
	if err := json.Unmarshal(story.Tags, &tags); err == nil {
		return tags
	}

	// 尝试解析为 []interface{}
	var interfaceTags []interface{}
	if err := json.Unmarshal(story.Tags, &interfaceTags); err == nil {
		tags = make([]string, 0, len(interfaceTags))
		for _, item := range interfaceTags {
			if str, ok := item.(string); ok {
				tags = append(tags, str)
			}
		}
		return tags
	}

	return []string{}
}

// contains 检查 slice 是否包含元素（工具函数）
func contains(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func intersectUintSlices(left, right []uint) []uint {
	if len(left) == 0 || len(right) == 0 {
		return []uint{}
	}

	allowed := make(map[uint]struct{}, len(right))
	for _, value := range right {
		allowed[value] = struct{}{}
	}

	result := make([]uint, 0, len(left))
	seen := make(map[uint]struct{}, len(left))
	for _, value := range left {
		if _, ok := allowed[value]; !ok {
			continue
		}
		if _, duplicated := seen[value]; duplicated {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// SearchProjectsSemantic 项目级语义搜索
// GET /api/search/projects?q=查询文本&limit=10
func (h *SearchHandler) SearchProjectsSemantic(c *gin.Context) {
	if h.vectorSvc == nil {
		api.Internal(c, "向量搜索服务未启用")
		return
	}
	if h.db == nil {
		api.Internal(c, "搜索服务配置错误")
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	query := c.Query("q")
	if query == "" {
		api.BadRequest(c, "q 不能为空")
		return
	}

	limit := parseIntQuery(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// 获取用户有权限的项目
	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}

	// 搜索所有项目中的相关故事（跨项目搜索）
	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		query,
		accessibleIDs,
		limit*5, // 获取更多结果
	)
	if err != nil {
		api.Internal(c, "项目语义搜索失败")
		return
	}

	// 按项目分组统计
	projectMatches := h.groupStoriesByProject(stories, accessibleIDs)

	// 转换为响应格式
	result := make([]gin.H, 0, len(projectMatches))
	for _, pm := range projectMatches {
		// 获取项目详情
		project, err := h.projectRepo.FindByID(pm.ProjectID)
		if err != nil {
			continue
		}

		// 获取该项目下最相关的故事
		topStories := h.getTopStoriesInProject(pm.Stories, 3)

		result = append(result, gin.H{
			"project": gin.H{
				"id":          project.ID,
				"name":        project.Name,
				"description": project.Description,
				"agile_mode":  project.AgileMode,
			},
			"similarity":  pm.AvgSimilarity,
			"story_count": len(pm.Stories),
			"top_stories": topStories,
		})
	}

	api.Success(c, "success", gin.H{
		"projects": result,
	})
}

// ========== 项目级搜索辅助方法 ==========

// ProjectMatchResult 项目匹配结果
type ProjectMatchResult struct {
	ProjectID     uint
	Stories       []service.SimilarStory
	AvgSimilarity float64
	MaxSimilarity float64
}

// groupStoriesByProject 按项目分组故事（单一职责）
func (h *SearchHandler) groupStoriesByProject(stories []service.SimilarStory, projectIDs []uint) []ProjectMatchResult {
	// 创建项目映射
	projectMap := make(map[uint]*ProjectMatchResult)
	for _, pid := range projectIDs {
		projectMap[pid] = &ProjectMatchResult{
			ProjectID:     pid,
			Stories:       []service.SimilarStory{},
			AvgSimilarity: 0,
			MaxSimilarity: 0,
		}
	}

	// 分组故事
	for _, story := range stories {
		if us, ok := story.Story.(model.UserStory); ok {
			if pm, exists := projectMap[us.ProjectID]; exists {
				pm.Stories = append(pm.Stories, story)
				if story.Similarity > pm.MaxSimilarity {
					pm.MaxSimilarity = story.Similarity
				}
			}
		}
	}

	// 计算平均相似度并转换为切片
	var results []ProjectMatchResult
	for _, pm := range projectMap {
		if len(pm.Stories) > 0 {
			sum := 0.0
			for _, s := range pm.Stories {
				sum += s.Similarity
			}
			pm.AvgSimilarity = sum / float64(len(pm.Stories))
			results = append(results, *pm)
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].AvgSimilarity > results[j].AvgSimilarity })

	return results
}

// getTopStoriesInProject 获取项目中最相关的 N 个故事（单一职责）
func (h *SearchHandler) getTopStoriesInProject(stories []service.SimilarStory, limit int) []gin.H {
	sort.Slice(stories, func(i, j int) bool { return stories[i].Similarity > stories[j].Similarity })

	// 返回 top N
	result := make([]gin.H, 0, limit)
	for i := 0; i < limit && i < len(stories); i++ {
		if us, ok := stories[i].Story.(model.UserStory); ok {
			result = append(result, gin.H{
				"id":         us.ID,
				"title":      us.Title,
				"story_type": us.StoryType,
				"status":     us.Status,
				"similarity": stories[i].Similarity,
			})
		}
	}
	return result
}
