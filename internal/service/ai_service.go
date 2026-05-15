package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/metrics"
	"git.neolidy.top/neo/storybook/internal/model"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// StoryResult holds the AI-generated user story decomposition.
type StoryResult struct {
	Title       string   `json:"title"`
	UserStory   string   `json:"user_story"`
	Actor       string   `json:"actor"`
	Action      string   `json:"action"`
	Value       string   `json:"value"`
	StoryType   string   `json:"story_type"`
	Priority    int      `json:"priority"`
	SuggestedAC []string `json:"suggested_ac"`
	StoryPoints int      `json:"story_points"`
	Tags        []string `json:"tags"`
	Warnings    []string `json:"warnings,omitempty"`
	Source      string   `json:"-"`
}

const (
	AIResponseSourceOpenAI    = "openai"
	AIResponseSourceHeuristic = "heuristic"
)

// StreamCallback receives incremental content chunks during streaming.
// Return non-nil error to abort the stream.
type StreamCallback func(chunk string) error

type RuntimeAIConfig struct {
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	Enabled     bool
}

// AIService defines the contract for AI-powered story generation.
type AIService interface {
	// GenerateStory decomposes a requirement into a user story.
	GenerateStory(ctx context.Context, requirement string) (*StoryResult, error)

	// StreamGenerateStory streams the story generation, calling cb for each chunk.
	StreamGenerateStory(ctx context.Context, requirement string, cb StreamCallback) (*StoryResult, error)

	// ChatRefine refines an existing story based on feedback.
	ChatRefine(ctx context.Context, original *StoryResult, feedback string) (*StoryResult, error)

	// BatchGenerate generates multiple story variants (max 5).
	BatchGenerate(ctx context.Context, requirement string, count int) ([]*StoryResult, error)

	// Chat issues a plain chat completion with the given system+user prompts and
	// returns the assistant message content. Used by §8.6 helpers (AC refine,
	// summary, translate) that need free-form text rather than a StoryResult.
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)

	// IsConfigured returns true when the service can make real API calls.
	IsConfigured() bool
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

// NewAIService queries ai_configs and returns the appropriate implementation.
// Caches the constructed service keyed by (config_id, updated_at) so the hot
// path skips AES-256-GCM decryption and OpenAI client construction when the
// config has not changed since the last call.
func NewAIService(db *gorm.DB) AIService {
	cfg, err := loadActiveAIConfig(db)
	if err != nil || cfg == nil {
		return &heuristicAIService{}
	}

	if svc := cachedAIServiceFor(cfg.ID, cfg.UpdatedAt); svc != nil {
		return svc
	}

	apiKey, err := DecryptAPIKey(cfg.APIKeyEncrypted)
	if err != nil || apiKey == "" {
		return &heuristicAIService{}
	}

	svc := NewAIServiceFromConfig(RuntimeAIConfig{
		APIKey:      apiKey,
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		Enabled:     cfg.Enabled,
	})
	return storeAIServiceCache(svc, cfg.ID, cfg.UpdatedAt)
}

func NewAIServiceFromConfig(cfg RuntimeAIConfig) AIService {
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" {
		return &heuristicAIService{}
	}

	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1200
	}

	client := openai.NewClient(cfg.APIKey)
	return &openAIService{
		client:      client,
		model:       modelName,
		temperature: float32(cfg.Temperature),
		maxTokens:   maxTokens,
	}
}

// ---------------------------------------------------------------------------
// OpenAI implementation
// ---------------------------------------------------------------------------

type openAIService struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

const systemPrompt = `你是一位资深的产品经理和用户故事专家。根据用户提供的需求描述，将其拆解为标准的用户故事格式。

输出要求（JSON格式）：
{
  "title": "故事标题（简洁，不超过200字）",
  "actor": "角色（如：用户、管理员、买家等）",
  "action": "想要做的动作",
  "value": "获得的价值",
  "user_story": "完整的用户故事，格式：作为<角色>，我想要<动作>，以便<价值>",
  "story_type": "feature | bug | chore",
  "priority": "0-4 的整数，4最高",
  "suggested_ac": ["验收标准1", "验收标准2", ...],
  "story_points": "故事点数(1,2,3,5,8,13)",
  "tags": ["标签1", "标签2"],
  "warnings": ["如果需求过大或信息不足，输出提醒"]
}

注意：
- 验收标准应具体、可测试
- 故事点数基于复杂度合理估算
- story_type 只能是 feature、bug、chore
- priority 只能是 0、1、2、3、4
- tags 最多 5 个
- 只输出 JSON，不要输出 Markdown 代码块`

func (s *openAIService) IsConfigured() bool { return true }

func (s *openAIService) GenerateStory(ctx context.Context, requirement string) (*StoryResult, error) {
	ctx, cancel := withDefaultAIDeadline(ctx)
	defer cancel()

	start := time.Now()
	var resp openai.ChatCompletionResponse
	err := retryAPI(ctx, func() error {
		var apiErr error
		resp, apiErr = s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       s.model,
			Temperature: s.temperature,
			MaxTokens:   s.maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: requirement},
			},
		})
		return apiErr
	})
	metrics.AICallDuration.WithLabelValues("generate_story", s.model).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("generate_story", s.model, "error").Inc()
		return nil, fmt.Errorf("openai completion: %w", err)
	}
	metrics.AICallsTotal.WithLabelValues("generate_story", s.model, "success").Inc()
	metrics.AITokensTotal.WithLabelValues("generate_story", s.model, "prompt").Add(float64(resp.Usage.PromptTokens))
	metrics.AITokensTotal.WithLabelValues("generate_story", s.model, "completion").Add(float64(resp.Usage.CompletionTokens))

	if len(resp.Choices) == 0 {
		return nil, errors.New("openai: empty response")
	}

	result, err := parseStoryFromContent(resp.Choices[0].Message.Content, requirement)
	if err != nil {
		return nil, err
	}
	return withOpenAIFallbackWarning(result), nil
}

func (s *openAIService) StreamGenerateStory(ctx context.Context, requirement string, cb StreamCallback) (_ *StoryResult, err error) {
	start := time.Now()
	stream, err := s.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       s.model,
		Temperature: s.temperature,
		MaxTokens:   s.maxTokens,
		Stream:      true,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: requirement},
		},
	})
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("stream_generate_story", s.model, "error").Inc()
		metrics.AICallDuration.WithLabelValues("stream_generate_story", s.model).Observe(time.Since(start).Seconds())
		return nil, fmt.Errorf("openai stream create: %w", err)
	}
	defer func() {
		closeErr := stream.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("openai stream close: %w", closeErr)
		}
		metrics.AICallDuration.WithLabelValues("stream_generate_story", s.model).Observe(time.Since(start).Seconds())
		if err == nil {
			metrics.AICallsTotal.WithLabelValues("stream_generate_story", s.model, "success").Inc()
		} else {
			metrics.AICallsTotal.WithLabelValues("stream_generate_story", s.model, "error").Inc()
		}
	}()

	var fullContent strings.Builder
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("openai stream recv: %w", err)
		}

		chunk := response.Choices[0].Delta.Content
		if chunk == "" {
			continue
		}

		fullContent.WriteString(chunk)
		if cb != nil {
			if cbErr := cb(chunk); cbErr != nil {
				return nil, cbErr
			}
		}
	}

	result, err := parseStoryFromContent(fullContent.String(), requirement)
	if err != nil {
		return nil, err
	}
	return withOpenAIFallbackWarning(result), nil
}

func (s *openAIService) ChatRefine(ctx context.Context, original *StoryResult, feedback string) (*StoryResult, error) {
	ctx, cancel := withDefaultAIDeadline(ctx)
	defer cancel()

	refinePrompt := fmt.Sprintf(`原始用户故事：
- 角色：%s
- 动作：%s
- 价值：%s
- 验收标准：%s
- 故事点：%d

用户反馈：%s

请根据反馈重新生成改进后的用户故事（保持相同JSON格式）。`,
		original.Actor,
		original.Action,
		original.Value,
		strings.Join(original.SuggestedAC, "；"),
		original.StoryPoints,
		feedback,
	)

	var resp openai.ChatCompletionResponse
	start := time.Now()
	err := retryAPI(ctx, func() error {
		var apiErr error
		resp, apiErr = s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       s.model,
			Temperature: s.temperature,
			MaxTokens:   s.maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: refinePrompt},
			},
		})
		return apiErr
	})
	metrics.AICallDuration.WithLabelValues("chat_refine", s.model).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("chat_refine", s.model, "error").Inc()
		return nil, fmt.Errorf("openai refine: %w", err)
	}
	metrics.AICallsTotal.WithLabelValues("chat_refine", s.model, "success").Inc()
	metrics.AITokensTotal.WithLabelValues("chat_refine", s.model, "prompt").Add(float64(resp.Usage.PromptTokens))
	metrics.AITokensTotal.WithLabelValues("chat_refine", s.model, "completion").Add(float64(resp.Usage.CompletionTokens))

	if len(resp.Choices) == 0 {
		return nil, errors.New("openai: empty refine response")
	}

	result, err := parseStoryFromContent(resp.Choices[0].Message.Content, original.Action)
	if err != nil {
		return nil, err
	}
	return withOpenAIFallbackWarning(result), nil
}

func (s *openAIService) BatchGenerate(ctx context.Context, requirement string, count int) ([]*StoryResult, error) {
	if count <= 0 {
		count = 1
	}
	if count > 5 {
		count = 5
	}

	results := make([]*StoryResult, count)
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 2) // bound concurrency to protect the connection pool
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			r, err := s.GenerateStory(gctx, requirement)
			if err != nil {
				return fmt.Errorf("batch item %d: %w", i, err)
			}
			results[i] = r
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// Chat issues a single chat completion with a custom system+user pair and
// returns the assistant message text. Used by §8.6 helpers that need free-form
// text (AC refinement, summarization, translation) rather than parsed StoryResult.
func (s *openAIService) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	ctx, cancel := withDefaultAIDeadline(ctx)
	defer cancel()

	var resp openai.ChatCompletionResponse
	start := time.Now()
	err := retryAPI(ctx, func() error {
		var apiErr error
		resp, apiErr = s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       s.model,
			Temperature: s.temperature,
			MaxTokens:   s.maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
		})
		return apiErr
	})
	metrics.AICallDuration.WithLabelValues("chat", s.model).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("chat", s.model, "error").Inc()
		return "", fmt.Errorf("openai chat: %w", err)
	}
	metrics.AICallsTotal.WithLabelValues("chat", s.model, "success").Inc()
	metrics.AITokensTotal.WithLabelValues("chat", s.model, "prompt").Add(float64(resp.Usage.PromptTokens))
	metrics.AITokensTotal.WithLabelValues("chat", s.model, "completion").Add(float64(resp.Usage.CompletionTokens))

	if len(resp.Choices) == 0 {
		return "", errors.New("openai: empty chat response")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// ---------------------------------------------------------------------------
// Heuristic fallback
// ---------------------------------------------------------------------------

type heuristicAIService struct{}

func (s *heuristicAIService) IsConfigured() bool { return false }

func (s *heuristicAIService) GenerateStory(_ context.Context, requirement string) (*StoryResult, error) {
	reqText := strings.TrimSpace(requirement)
	actor := inferActor(reqText)
	action := inferAction(reqText)
	value := inferValue(reqText)
	title := inferTitle(reqText, action)
	storyType := inferStoryType(reqText)
	priority := inferPriority(reqText)
	tags := inferTags(reqText)
	warnings := inferWarnings(reqText)

	story := fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", actor, action, value)
	ac := []string{
		"支持核心输入与校验流程",
		"操作成功后返回明确反馈",
		"异常场景有清晰错误提示",
		"关键操作记录活动日志",
	}

	return &StoryResult{
		Title:       title,
		UserStory:   story,
		Actor:       actor,
		Action:      action,
		Value:       value,
		StoryType:   storyType,
		Priority:    priority,
		SuggestedAC: ac,
		StoryPoints: estimatePoints(reqText, len(ac)),
		Tags:        tags,
		Warnings:    warnings,
		Source:      AIResponseSourceHeuristic,
	}, nil
}

func (s *heuristicAIService) StreamGenerateStory(ctx context.Context, requirement string, cb StreamCallback) (*StoryResult, error) {
	// Heuristic has no streaming; simulate with single chunk
	result, err := s.GenerateStory(ctx, requirement)
	if err != nil {
		return nil, err
	}
	if cb != nil {
		if cbErr := cb(result.UserStory); cbErr != nil {
			return nil, cbErr
		}
	}
	return result, nil
}

func (s *heuristicAIService) ChatRefine(_ context.Context, original *StoryResult, _ string) (*StoryResult, error) {
	// Heuristic cannot refine; return original unchanged
	return original, nil
}

func (s *heuristicAIService) BatchGenerate(ctx context.Context, requirement string, count int) ([]*StoryResult, error) {
	if count <= 0 {
		count = 1
	}
	if count > 5 {
		count = 5
	}

	result, err := s.GenerateStory(ctx, requirement)
	if err != nil {
		return nil, err
	}

	// Heuristic produces identical results; return copies
	results := make([]*StoryResult, count)
	for i := range results {
		cp := *result
		results[i] = &cp
	}
	return results, nil
}

// Chat returns a sentinel error so handlers can degrade gracefully when AI is
// not configured. Callers should branch on IsConfigured() before calling.
func (s *heuristicAIService) Chat(_ context.Context, _, _ string) (string, error) {
	return "", errors.New("ai_not_configured: 仅在配置 OpenAI 后可用")
}

// ---------------------------------------------------------------------------
// Shared helpers (extracted from handler layer)
// ---------------------------------------------------------------------------

func inferActor(text string) string {
	candidates := []string{"产品经理", "开发人员", "测试人员", "用户", "访客", "买家", "卖家", "管理员"}
	for _, c := range candidates {
		if strings.Contains(text, c) {
			return c
		}
	}
	return "用户"
}

func inferAction(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "用户") || strings.HasPrefix(text, "作为") {
		return text
	}
	if len([]rune(text)) > 40 {
		return string([]rune(text)[:40]) + "..."
	}
	return text
}

func inferValue(text string) string {
	if strings.Contains(text, "以便") {
		parts := strings.SplitN(text, "以便", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return "提升任务交付效率"
}

func estimatePoints(text string, acCount int) int {
	l := len([]rune(text))
	score := l/120 + acCount/2
	switch {
	case score <= 1:
		return 1
	case score <= 2:
		return 2
	case score <= 3:
		return 3
	case score <= 5:
		return 5
	case score <= 8:
		return 8
	default:
		return 13
	}
}

// parseStoryFromContent attempts to extract a StoryResult from raw LLM output.
// Falls back to heuristic decomposition if JSON parsing fails.
func parseStoryFromContent(content, fallbackRequirement string) (*StoryResult, error) {
	// Try to find JSON in the response
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		// No JSON found; use heuristic fallback
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	var result StoryResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	// Validate minimum fields
	if strings.TrimSpace(result.Actor) == "" || strings.TrimSpace(result.Action) == "" {
		h := &heuristicAIService{}
		return h.GenerateStory(context.Background(), fallbackRequirement)
	}

	result.Actor = strings.TrimSpace(result.Actor)
	result.Action = strings.TrimSpace(result.Action)
	result.Value = strings.TrimSpace(result.Value)
	result.Title = normalizeTitle(result.Title, fallbackRequirement, result.Action)
	result.StoryType = normalizeStoryType(result.StoryType, fallbackRequirement)
	result.Priority = normalizePriority(result.Priority, fallbackRequirement)
	result.SuggestedAC = normalizeStringList(result.SuggestedAC, 8)
	result.Tags = normalizeTags(result.Tags, fallbackRequirement)
	result.Warnings = normalizeStringList(result.Warnings, 3)

	if strings.TrimSpace(result.UserStory) == "" {
		result.UserStory = fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", result.Actor, result.Action, normalizeValue(result.Value))
	}
	if strings.TrimSpace(result.Value) == "" {
		result.Value = inferValue(fallbackRequirement)
	}
	if result.UserStory == "" {
		result.UserStory = fmt.Sprintf("作为 %s，\n我想要 %s，\n以便 %s", result.Actor, result.Action, result.Value)
	}
	if result.StoryPoints <= 0 {
		result.StoryPoints = estimatePoints(result.Action, len(result.SuggestedAC))
	}
	result.StoryPoints = normalizeStoryPoints(result.StoryPoints)
	result.Source = AIResponseSourceOpenAI

	return &result, nil
}

// extractJSON finds the first {...} block in content.
func extractJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}
	// Find matching closing brace
	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return ""
}

func inferTitle(text, action string) string {
	title := strings.TrimSpace(action)
	if title == "" {
		title = strings.TrimSpace(text)
	}
	if title == "" {
		return "未命名故事"
	}
	title = strings.TrimPrefix(title, "作为")
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}
	return title
}

func inferStoryType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "bug"), strings.Contains(lower, "缺陷"), strings.Contains(lower, "报错"), strings.Contains(lower, "修复"):
		return model.StoryTypeBug
	case strings.Contains(lower, "清理"), strings.Contains(lower, "重构"), strings.Contains(lower, "迁移"), strings.Contains(lower, "优化脚本"):
		return model.StoryTypeChore
	default:
		return model.StoryTypeFeature
	}
}

func inferPriority(text string) int {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "紧急"), strings.Contains(lower, "阻塞"), strings.Contains(lower, "critical"), strings.Contains(lower, "p0"):
		return 4
	case strings.Contains(lower, "高优"), strings.Contains(lower, "high"), strings.Contains(lower, "p1"):
		return 3
	case strings.Contains(lower, "低优"), strings.Contains(lower, "low"), strings.Contains(lower, "p3"):
		return 1
	default:
		return 2
	}
}

func inferTags(text string) []string {
	candidates := []struct {
		keyword string
		tag     string
	}{
		{keyword: "登录", tag: "登录"},
		{keyword: "认证", tag: "认证"},
		{keyword: "支付", tag: "支付"},
		{keyword: "订单", tag: "订单"},
		{keyword: "搜索", tag: "搜索"},
		{keyword: "消息", tag: "消息"},
		{keyword: "权限", tag: "权限"},
		{keyword: "报表", tag: "报表"},
		{keyword: "导出", tag: "导出"},
		{keyword: "后台", tag: "后台"},
		{keyword: "移动端", tag: "移动端"},
	}

	tags := make([]string, 0, 5)
	for _, candidate := range candidates {
		if strings.Contains(text, candidate.keyword) {
			tags = append(tags, candidate.tag)
		}
		if len(tags) == 5 {
			break
		}
	}
	return tags
}

func inferWarnings(text string) []string {
	warnings := make([]string, 0, 2)
	if len([]rune(text)) > 300 {
		warnings = append(warnings, "需求描述较长，建议人工确认是否需要拆分为多个故事")
	}
	if !strings.Contains(text, "用户") && !strings.Contains(text, "管理员") && !strings.Contains(text, "角色") {
		warnings = append(warnings, "需求中未明确角色，AI 已按通用用户角色推断")
	}
	return warnings
}

func normalizeTitle(title, requirement, action string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = inferTitle(requirement, action)
	}
	if len([]rune(title)) > 200 {
		title = string([]rune(title)[:200])
	}
	return title
}

func normalizeStoryType(storyType, requirement string) string {
	switch strings.TrimSpace(strings.ToLower(storyType)) {
	case model.StoryTypeFeature, model.StoryTypeBug, model.StoryTypeChore:
		return strings.TrimSpace(strings.ToLower(storyType))
	default:
		return inferStoryType(requirement)
	}
}

func normalizePriority(priority int, requirement string) int {
	if priority < 0 || priority > 4 {
		return inferPriority(requirement)
	}
	return priority
}

func normalizeStoryPoints(points int) int {
	allowed := []int{1, 2, 3, 5, 8, 13}
	for _, item := range allowed {
		if points == item {
			return points
		}
	}

	if points <= 1 {
		return 1
	}
	if points <= 2 {
		return 2
	}
	if points <= 3 {
		return 3
	}
	if points <= 5 {
		return 5
	}
	if points <= 8 {
		return 8
	}
	return 13
}

func normalizeStringList(items []string, maxCount int) []string {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]string, 0, min(len(items), maxCount))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
		if len(normalized) == maxCount {
			break
		}
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeTags(tags []string, requirement string) []string {
	normalized := normalizeStringList(tags, 5)
	if len(normalized) > 0 {
		return normalized
	}
	return inferTags(requirement)
}

func normalizeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "提升任务交付效率"
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SanitizeRequirement(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	spaceRegexp := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(spaceRegexp.ReplaceAllString(trimmed, " "))
}

func IsOpenAIConfigured(db *gorm.DB) bool {
	var cfg model.AIConfig
	if err := db.Where("enabled = ?", true).Order("id DESC").Limit(1).Find(&cfg).Error; err != nil {
		return false
	}
	if cfg.ID == 0 {
		return false
	}
	return cfg.Enabled && strings.TrimSpace(cfg.APIKeyEncrypted) != ""
}

func ResolveAIResponseSource(svc AIService) string {
	if svc != nil && svc.IsConfigured() {
		return AIResponseSourceOpenAI
	}
	return AIResponseSourceHeuristic
}

func ResolveStoryResultSource(result *StoryResult, fallbackSvc AIService) string {
	if result != nil && strings.TrimSpace(result.Source) != "" {
		return result.Source
	}
	return ResolveAIResponseSource(fallbackSvc)
}

func withOpenAIFallbackWarning(result *StoryResult) *StoryResult {
	if result == nil || result.Source == AIResponseSourceOpenAI {
		return result
	}
	result.Warnings = prependStoryWarning(
		result.Warnings,
		"OpenAI 返回结果不可解析，已自动回退到规则草稿，请人工确认后再保存",
	)
	return result
}

func prependStoryWarning(warnings []string, warning string) []string {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return warnings
	}
	for _, item := range warnings {
		if item == warning {
			return warnings
		}
	}
	return append([]string{warning}, warnings...)
}
