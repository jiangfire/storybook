package handler

import (
	"encoding/json"
	"sort"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
)

// SuggestTagsRequest 标签建议请求
type SuggestTagsRequest struct {
	Title       string `json:"title" binding:"required_without_all=Description"`
	Description string `json:"description" binding:"required_without_all=Title"`
	Content     string `json:"content"` // 可选的额外内容
	Limit       int    `json:"limit"`
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

	if req.Limit == 0 {
		req.Limit = 5
	} else if req.Limit < 1 || req.Limit > 20 {
		api.BadRequest(c, "limit 必须在 1 到 20 之间")
		return
	}

	content := h.buildTagSuggestionContent(req.Title, req.Description, req.Content)

	userID := middleware.MustUserID(c)
	role, _ := middleware.CurrentRole(c)
	accessibleIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "获取项目权限失败")
		return
	}

	stories, err := h.vectorSvc.SearchSimilarStories(
		c.Request.Context(),
		content,
		accessibleIDs,
		req.Limit*3,
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

	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(scores); i++ {
		result = append(result, scores[i].Tag)
	}
	return result
}

// extractTagsFromStory 从故事中提取标签（单一职责）
func (h *SearchHandler) extractTagsFromStory(story model.UserStory) []string {
	if story.Tags == nil {
		return []string{}
	}

	var tags []string
	if err := json.Unmarshal(story.Tags, &tags); err == nil {
		return tags
	}

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
