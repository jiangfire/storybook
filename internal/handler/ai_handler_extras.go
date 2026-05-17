package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/api"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/service"
)

type refineACRequest struct {
	Feedback string `json:"feedback" binding:"required,min=2,max=2000"`
}

type translateStoryRequest struct {
	Language string `json:"language" binding:"required,oneof=en zh"`
}

// RefineAC asks the LLM to revise the story's acceptance criteria based on
// feedback. Returns the suggested AC list; the client is expected to review and
// apply individual entries via existing AC CRUD endpoints — this endpoint does
// not mutate the story.
func (h *AIHandler) RefineAC(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理或管理员可使用 AI 优化 AC")
		return
	}

	story := middleware.MustStory(c)

	var req refineACRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	existing := make([]string, 0, len(criteria))
	for _, ac := range criteria {
		existing = append(existing, ac.Description)
	}

	systemPrompt := `你是一位资深敏捷教练。请根据用户的反馈和现有验收标准,生成一份改进版的验收标准列表。

输出要求:
- 每行一条 AC,不超过 8 条
- 不输出任何标题或解释,只输出 AC 列表
- 每条 AC 必须可验证 (Given/When/Then 或类似明确措辞)`

	userPrompt := fmt.Sprintf("故事标题: %s\n故事描述: %s\n\n现有 AC:\n- %s\n\n用户反馈:\n%s",
		story.Title,
		story.Description,
		strings.Join(existing, "\n- "),
		strings.TrimSpace(req.Feedback),
	)

	ctx, cancel := contextWithTimeout(c, 45*time.Second)
	defer cancel()

	aiSvc := service.NewAIService(h.db)
	if !aiSvc.IsConfigured() {
		api.Error(c, http.StatusServiceUnavailable, api.CodeInternal, "AI 未配置,无法使用此功能")
		return
	}

	reply, err := aiSvc.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, fmt.Sprintf("AI 调用失败: %v", err))
		return
	}

	suggestions := parseACList(reply)
	api.Success(c, "success", gin.H{
		"story_id":    story.ID,
		"original_ac": existing,
		"suggested":   suggestions,
		"raw":         reply,
	})
}

// SummarizeStory returns a 2-3 sentence stakeholder summary of the story so
// product can drop it into a status update without reading the full description.
func (h *AIHandler) SummarizeStory(c *gin.Context) {
	story := middleware.MustStory(c)

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	acText := make([]string, 0, len(criteria))
	for _, ac := range criteria {
		acText = append(acText, ac.Description)
	}

	systemPrompt := `你是项目经理助手。请用 2-3 句中文为以下用户故事写一段面向干系人的简明摘要,
不要使用 Markdown,不要复述 AC 全文,只输出摘要正文。`

	userPrompt := fmt.Sprintf("标题: %s\n描述: %s\n验收标准:\n- %s\n状态: %s",
		story.Title,
		story.Description,
		strings.Join(acText, "\n- "),
		story.Status,
	)

	ctx, cancel := contextWithTimeout(c, 30*time.Second)
	defer cancel()

	aiSvc := service.NewAIService(h.db)
	if !aiSvc.IsConfigured() {
		api.Error(c, http.StatusServiceUnavailable, api.CodeInternal, "AI 未配置,无法使用此功能")
		return
	}
	summary, err := aiSvc.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, fmt.Sprintf("AI 调用失败: %v", err))
		return
	}

	api.Success(c, "success", gin.H{
		"story_id": story.ID,
		"summary":  summary,
	})
}

// TranslateStory returns title+description+AC translated into the requested
// language. Source is whichever language the story is currently in.
func (h *AIHandler) TranslateStory(c *gin.Context) {
	story := middleware.MustStory(c)

	var req translateStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	acLines := make([]string, 0, len(criteria))
	for i, ac := range criteria {
		acLines = append(acLines, fmt.Sprintf("AC-%d: %s", i+1, ac.Description))
	}

	target := "English"
	if req.Language == "zh" {
		target = "中文 (简体)"
	}

	systemPrompt := fmt.Sprintf(`You are a professional product translator. Translate the following user story into %s.

Output requirements:
- Return strictly the format:
  Title: <translated title>
  Description: <translated description>
  AC-1: <translated AC>
  AC-2: <translated AC>
  ...
- Preserve any acronyms / product names verbatim.
- Do not add commentary or markdown.`, target)

	userPrompt := fmt.Sprintf("Title: %s\nDescription: %s\n%s",
		story.Title,
		story.Description,
		strings.Join(acLines, "\n"),
	)

	ctx, cancel := contextWithTimeout(c, 45*time.Second)
	defer cancel()

	aiSvc := service.NewAIService(h.db)
	if !aiSvc.IsConfigured() {
		api.Error(c, http.StatusServiceUnavailable, api.CodeInternal, "AI 未配置,无法使用此功能")
		return
	}
	translated, err := aiSvc.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, fmt.Sprintf("AI 调用失败: %v", err))
		return
	}

	parsed := parseTranslationOutput(translated)
	api.Success(c, "success", gin.H{
		"story_id":   story.ID,
		"language":   req.Language,
		"translated": parsed,
		"raw":        translated,
	})
}

// DoRCheck (Definition of Ready) returns a deterministic readiness scorecard.
// It runs without invoking the LLM so it's always available; the rate-limited
// AI endpoint group still fronts it for consistency with other AI helpers.
func (h *AIHandler) DoRCheck(c *gin.Context) {
	story := middleware.MustStory(c)

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	titleLen := len([]rune(strings.TrimSpace(story.Title)))
	descLen := len([]rune(strings.TrimSpace(story.Description)))

	type rule struct {
		Key     string
		Title   string
		Pass    bool
		Message string
	}
	rules := []rule{
		{
			Key:     "title",
			Title:   "标题清晰",
			Pass:    titleLen >= 5,
			Message: "标题需≥5个字符,简明描述目标",
		},
		{
			Key:     "description",
			Title:   "描述完整",
			Pass:    descLen >= 30,
			Message: "描述需≥30个字符,包含背景、目标和影响范围",
		},
		{
			Key:     "acceptance_criteria",
			Title:   "验收标准数量",
			Pass:    len(criteria) >= 2,
			Message: "至少 2 条可验证的 AC",
		},
		{
			Key:     "story_points",
			Title:   "故事点已估",
			Pass:    story.Points != nil && *story.Points > 0,
			Message: "需在排入冲刺前完成估算",
		},
		{
			Key:     "assignee",
			Title:   "已指派负责人",
			Pass:    story.AssignedTo != nil,
			Message: "请通过认领或指派分配负责人",
		},
		{
			Key:     "review_status",
			Title:   "评审已通过",
			Pass:    story.ReviewStatus == model.ReviewStatusApproved,
			Message: "故事必须通过评审才能进入冲刺",
		},
	}

	passed := 0
	checks := make([]gin.H, 0, len(rules))
	suggestions := make([]string, 0)
	for _, r := range rules {
		if r.Pass {
			passed++
		} else {
			suggestions = append(suggestions, r.Message)
		}
		checks = append(checks, gin.H{
			"key":     r.Key,
			"title":   r.Title,
			"pass":    r.Pass,
			"message": r.Message,
		})
	}

	score := 0
	if len(rules) > 0 {
		score = int(float64(passed) / float64(len(rules)) * 100)
	}

	api.Success(c, "success", gin.H{
		"story_id":    story.ID,
		"ready":       passed == len(rules),
		"score":       score,
		"passed":      passed,
		"total":       len(rules),
		"checks":      checks,
		"suggestions": suggestions,
	})
}

// parseACList breaks an LLM reply into individual AC strings. Accepts numbered
// (1. /  1) ), bulleted (- / *), and plain-line formats.
func parseACList(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Strip leading bullets / numbering.
		for _, prefix := range []string{"- ", "* ", "• "} {
			trimmed = strings.TrimPrefix(trimmed, prefix)
		}
		// 1. 2) — strip up to the first space.
		if idx := strings.IndexFunc(trimmed, func(r rune) bool { return r == ' ' }); idx > 0 && idx < 4 {
			head := trimmed[:idx]
			if len(head) > 0 && (head[len(head)-1] == '.' || head[len(head)-1] == ')') {
				digits := head[:len(head)-1]
				allDigits := digits != ""
				for _, r := range digits {
					if r < '0' || r > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					trimmed = strings.TrimSpace(trimmed[idx+1:])
				}
			}
		}
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// parseTranslationOutput extracts the structured "Title: / Description: / AC-N:"
// block back into a struct the client can apply piece-meal.
func parseTranslationOutput(raw string) gin.H {
	result := gin.H{"title": "", "description": "", "ac": []string{}}
	acList := make([]string, 0)
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(strings.ToLower(line), "title:"):
			result["title"] = strings.TrimSpace(line[len("title:"):])
		case strings.HasPrefix(strings.ToLower(line), "description:"):
			result["description"] = strings.TrimSpace(line[len("description:"):])
		case strings.HasPrefix(strings.ToUpper(line), "AC-"):
			if idx := strings.Index(line, ":"); idx > 0 {
				acList = append(acList, strings.TrimSpace(line[idx+1:]))
			}
		}
	}
	if len(acList) > 0 {
		result["ac"] = acList
	}
	return result
}
