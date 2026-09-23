package handler

import (
	"context"
	"encoding/json"
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

// storyChatMessage 是对话式录入里的一条消息；role 限 user/assistant，
// content 上限与需求描述保持一致（2000 字符）。
type storyChatMessage struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required,max=2000"`
}

type storyChatAC struct {
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// storyChatDraft 与 generate-story 返回的 form_draft 字段一致，方便前端复用。
type storyChatDraft struct {
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	StoryType          string        `json:"story_type"`
	Priority           int           `json:"priority"`
	StoryPoints        int           `json:"story_points"`
	AcceptanceCriteria []storyChatAC `json:"acceptance_criteria"`
	Tags               []string      `json:"tags"`
}

// storyChatDraftPayload 是模型返回的原始 form_draft；数值字段用指针以区分
// “字段缺失”与“显式 0”。
type storyChatDraftPayload struct {
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	StoryType          string        `json:"story_type"`
	Priority           *int          `json:"priority"`
	StoryPoints        *int          `json:"story_points"`
	AcceptanceCriteria []storyChatAC `json:"acceptance_criteria"`
	Tags               []string      `json:"tags"`
}

type storyChatRequest struct {
	Messages     []storyChatMessage `json:"messages" binding:"required,min=1,max=20"`
	CurrentDraft *storyChatDraft    `json:"current_draft,omitempty"`
}

type storyChatResult struct {
	Reply string
	Draft *storyChatDraft
}

type storyChatPayloadResult struct {
	Reply string
	Draft *storyChatDraftPayload
}

const storyChatSystemPrompt = `你是一位资深敏捷需求分析师，正在和产品经理多轮对话，帮助他把一句需求逐步完善成一个用户故事。

每次回复必须只输出一个 JSON 对象（不要 Markdown 代码块、不要解释），结构如下:
{"reply": "给用户的简短中文回复；如果信息不足，在这里提出下一个澄清问题", "form_draft": {"title": "...", "description": "作为...，我想要...，以便...", "story_type": "feature|bug|chore", "priority": 0-4的整数(4最高), "story_points": 1|2|3|5|8|13, "acceptance_criteria": [{"description": "可验证的验收标准", "order": 1}], "tags": ["标签"]}}

要求:
- title 不超过 60 字；description 使用“作为...我想要...以便...”句式
- acceptance_criteria 不超过 8 条，每条必须可验证
- 信息不足时也要给出当前最佳草稿，并在 reply 中追问
- 需求已经完整时，在 reply 中说明可以直接保存`

// runStoryChat 组装提示词、调用 ChatCompleter 并解析回复。
// 独立成函数便于用 stub 注入做单元测试。
func runStoryChat(ctx context.Context, completer service.ChatCompleter, req storyChatRequest) (*storyChatResult, error) {
	var transcript strings.Builder
	for _, msg := range req.Messages {
		label := "用户"
		if msg.Role == "assistant" {
			label = "助手"
		}
		fmt.Fprintf(&transcript, "%s: %s\n", label, strings.TrimSpace(msg.Content))
	}

	var draftPrompt strings.Builder
	if req.CurrentDraft != nil {
		draft, err := json.Marshal(req.CurrentDraft)
		if err == nil {
			draftPrompt.WriteString("\n【当前表单草稿】\n")
			draftPrompt.Write(draft)
		}
	}

	userPrompt := fmt.Sprintf("【对话历史】\n%s%s", transcript.String(), draftPrompt.String())

	reply, err := completer.Chat(ctx, storyChatSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	parsed, parseErr := parseStoryChatReply(reply)
	if parseErr != nil {
		return nil, fmt.Errorf("AI 返回内容无法解析，请重试: %w", parseErr)
	}
	draft := normalizeStoryChatDraft(parsed.Draft, req.CurrentDraft)
	return &storyChatResult{Reply: parsed.Reply, Draft: draft}, nil
}

// parseStoryChatReply 从模型回复中提取 JSON 并解析出 reply 与 form_draft。
func parseStoryChatReply(reply string) (*storyChatPayloadResult, error) {
	jsonStr := extractHandlerJSON(reply)
	if jsonStr == "" {
		return nil, fmt.Errorf("回复中未找到 JSON")
	}

	var payload struct {
		Reply     string                `json:"reply"`
		FormDraft storyChatDraftPayload `json:"form_draft"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		return nil, err
	}

	return &storyChatPayloadResult{
		Reply: strings.TrimSpace(payload.Reply),
		Draft: &payload.FormDraft,
	}, nil
}

// extractHandlerJSON 找到回复中第一个配平的 {...} 块。
func extractHandlerJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if escaped {
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			if inString {
				escaped = true
			}
		case '"':
			inString = !inString
		case '{':
			if !inString {
				depth++
			}
		case '}':
			if !inString {
				depth--
				if depth == 0 {
					return content[start : i+1]
				}
			}
		}
	}
	return ""
}

var validStoryPoints = map[int]bool{1: true, 2: true, 3: true, 5: true, 8: true, 13: true}

// normalizeStoryChatDraft 兜底修正模型输出：非法枚举回退默认值，
// 缺失字段回退到当前表单草稿的值。
func normalizeStoryChatDraft(payload *storyChatDraftPayload, current *storyChatDraft) *storyChatDraft {
	if payload == nil {
		payload = &storyChatDraftPayload{}
	}
	fallback := current
	if fallback == nil {
		fallback = &storyChatDraft{}
	}

	draft := &storyChatDraft{}

	draft.Title = strings.TrimSpace(payload.Title)
	if draft.Title == "" {
		draft.Title = fallback.Title
	}
	draft.Description = strings.TrimSpace(payload.Description)
	if draft.Description == "" {
		draft.Description = fallback.Description
	}

	draft.StoryType = strings.TrimSpace(strings.ToLower(payload.StoryType))
	if draft.StoryType != model.StoryTypeFeature && draft.StoryType != model.StoryTypeBug && draft.StoryType != model.StoryTypeChore {
		if fallback.StoryType == model.StoryTypeBug || fallback.StoryType == model.StoryTypeChore {
			draft.StoryType = fallback.StoryType
		} else {
			draft.StoryType = model.StoryTypeFeature
		}
	}

	if payload.Priority != nil && *payload.Priority >= 0 && *payload.Priority <= 4 {
		draft.Priority = *payload.Priority
	} else if fallback.Priority >= 0 && fallback.Priority <= 4 {
		draft.Priority = fallback.Priority
	} else {
		draft.Priority = 2
	}

	if payload.StoryPoints != nil && validStoryPoints[*payload.StoryPoints] {
		draft.StoryPoints = *payload.StoryPoints
	} else if validStoryPoints[fallback.StoryPoints] {
		draft.StoryPoints = fallback.StoryPoints
	} else {
		draft.StoryPoints = 3
	}

	cleanedAC := make([]storyChatAC, 0, len(payload.AcceptanceCriteria))
	for index, ac := range payload.AcceptanceCriteria {
		description := strings.TrimSpace(ac.Description)
		if description == "" {
			continue
		}
		order := ac.Order
		if order <= 0 {
			order = index + 1
		}
		cleanedAC = append(cleanedAC, storyChatAC{Description: description, Order: order})
		if len(cleanedAC) >= 8 {
			break
		}
	}
	if len(cleanedAC) == 0 && len(fallback.AcceptanceCriteria) > 0 {
		cleanedAC = fallback.AcceptanceCriteria
	}
	draft.AcceptanceCriteria = cleanedAC

	cleanedTags := make([]string, 0, len(payload.Tags))
	for _, tag := range payload.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanedTags = append(cleanedTags, tag)
		}
	}
	if len(cleanedTags) == 0 {
		cleanedTags = fallback.Tags
	}
	draft.Tags = cleanedTags

	return draft
}

// StoryChat 多轮对话式录入：产品经理与 AI 聊需求，AI 每轮返回回复与最新表单草稿。
// 端点不修改任何故事，只产出草稿，由前端显式应用到表单。
func (h *AIHandler) StoryChat(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理或管理员可使用 AI 对话录入")
		return
	}

	var req storyChatRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	hasUserMessage := false
	for _, msg := range req.Messages {
		if msg.Role == "user" && strings.TrimSpace(msg.Content) != "" {
			hasUserMessage = true
			break
		}
	}
	if !hasUserMessage {
		api.BadRequest(c, "对话内容不能为空")
		return
	}

	ctx, cancel := contextWithTimeout(c, 45*time.Second)
	defer cancel()

	aiSvc := service.NewAIService(h.db)
	if !aiSvc.IsConfigured() {
		api.Error(c, http.StatusServiceUnavailable, api.CodeInternal,
			"对话式录入需要先配置 AI：请前往 管理后台 → AI 配置 完成设置")
		return
	}

	result, err := runStoryChat(ctx, aiSvc, req)
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, err.Error())
		return
	}

	acList := make([]gin.H, 0, len(result.Draft.AcceptanceCriteria))
	for index, ac := range result.Draft.AcceptanceCriteria {
		order := ac.Order
		if order <= 0 {
			order = index + 1
		}
		acList = append(acList, gin.H{
			"description": ac.Description,
			"order":       order,
		})
	}

	api.Success(c, "success", gin.H{
		"reply": result.Reply,
		"form_draft": gin.H{
			"title":               result.Draft.Title,
			"description":         result.Draft.Description,
			"story_type":          result.Draft.StoryType,
			"priority":            result.Draft.Priority,
			"story_points":        result.Draft.StoryPoints,
			"acceptance_criteria": acList,
			"tags":                result.Draft.Tags,
		},
		"source": "openai",
	})
}
