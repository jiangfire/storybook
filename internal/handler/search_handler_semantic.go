package handler

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/service"
)

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

// SearchSemantic 语义搜索
// GET /api/search/semantic?q=查询文本&project_ids=1,2,3&limit=10
func (h *SearchHandler) SearchSemantic(c *gin.Context) {
	if h.vectorSvc == nil {
		api.Internal(c, "向量搜索服务未启用")
		return
	}
	if h.db == nil {
		api.Internal(c, "搜索服务配置错误")
		return
	}

	userID := middleware.MustUserID(c)

	var req SemanticSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	} else if req.Limit < 1 || req.Limit > 100 {
		api.BadRequest(c, "limit 必须在 1 到 100 之间")
		return
	}

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

	if len(projectIDs) == 0 {
		projectIDs = accessibleIDs
	} else {
		projectIDs = intersectUintSlices(projectIDs, accessibleIDs)
	}

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

	userID := middleware.MustUserID(c)

	var req SimilarStoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "参数错误: "+err.Error())
		return
	}

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

	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		query,
		[]uint{req.ProjectID},
		req.Limit,
	)
	if err != nil {
		api.Internal(c, "相似故事搜索失败")
		return
	}

	filtered := h.filterAndConvertStories(stories, 0.7)

	api.Success(c, "success", gin.H{
		"similar_stories": filtered,
	})
}

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

	userID := middleware.MustUserID(c)

	query := c.Query("q")
	if query == "" {
		api.BadRequest(c, "q 不能为空")
		return
	}

	limit := parseIntQuery(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}

	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		query,
		accessibleIDs,
		limit*5,
	)
	if err != nil {
		api.Internal(c, "项目语义搜索失败")
		return
	}

	projectMatches := h.groupStoriesByProject(stories, accessibleIDs)

	result := make([]gin.H, 0, len(projectMatches))
	for _, pm := range projectMatches {
		project, err := h.projectRepo.FindByID(pm.ProjectID)
		if err != nil {
			continue
		}

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

// ProjectMatchResult 项目匹配结果
type ProjectMatchResult struct {
	ProjectID     uint
	Stories       []service.SimilarStory
	AvgSimilarity float64
	MaxSimilarity float64
}

// groupStoriesByProject 按项目分组故事（单一职责）
func (h *SearchHandler) groupStoriesByProject(stories []service.SimilarStory, projectIDs []uint) []ProjectMatchResult {
	projectMap := make(map[uint]*ProjectMatchResult)
	for _, pid := range projectIDs {
		projectMap[pid] = &ProjectMatchResult{
			ProjectID:     pid,
			Stories:       []service.SimilarStory{},
			AvgSimilarity: 0,
			MaxSimilarity: 0,
		}
	}

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
