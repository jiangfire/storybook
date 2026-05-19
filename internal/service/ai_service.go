package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jiangfire/storybook/internal/metrics"
	"github.com/jiangfire/storybook/internal/repository"
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
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Enabled     bool
}

// NewAIService queries ai_configs and returns the appropriate implementation.
// Caches the constructed service keyed by (config_id, updated_at) so the hot
// path skips AES-256-GCM decryption and OpenAI client construction when the
// config has not changed since the last call.
func NewAIService(db *gorm.DB) AIService {
	cfg, err := repository.NewAIConfigRepository(db).FindLatestEnabled()
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
		BaseURL:     cfg.BaseURL,
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

	var client *openai.Client
	if baseURL := strings.TrimSpace(cfg.BaseURL); baseURL != "" {
		oc := openai.DefaultConfig(cfg.APIKey)
		oc.BaseURL = baseURL
		client = openai.NewClientWithConfig(oc)
	} else {
		client = openai.NewClient(cfg.APIKey)
	}
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
	recordChatMetrics("generate_story", s.model, &resp, err, start)
	if err != nil {
		return nil, fmt.Errorf("openai completion: %w", err)
	}

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
		recordChatMetrics("stream_generate_story", s.model, nil, err, start)
		return nil, fmt.Errorf("openai stream create: %w", err)
	}
	defer func() {
		closeErr := stream.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("openai stream close: %w", closeErr)
		}
		recordChatMetrics("stream_generate_story", s.model, nil, err, start)
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
	recordChatMetrics("chat_refine", s.model, &resp, err, start)
	if err != nil {
		return nil, fmt.Errorf("openai refine: %w", err)
	}

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
	recordChatMetrics("chat", s.model, &resp, err, start)
	if err != nil {
		return "", fmt.Errorf("openai chat: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("openai: empty chat response")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// recordChatMetrics records duration, call count, and token usage for a single
// OpenAI chat completion. Pass nil resp when token metrics are unavailable
// (e.g. streaming or pre-call error).
func recordChatMetrics(op, model string, resp *openai.ChatCompletionResponse, err error, start time.Time) {
	metrics.AICallDuration.WithLabelValues(op, model).Observe(time.Since(start).Seconds())
	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.AICallsTotal.WithLabelValues(op, model, status).Inc()
	if resp != nil && err == nil {
		metrics.AITokensTotal.WithLabelValues(op, model, "prompt").Add(float64(resp.Usage.PromptTokens))
		metrics.AITokensTotal.WithLabelValues(op, model, "completion").Add(float64(resp.Usage.CompletionTokens))
	}
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
		StoryPoints: EstimatePoints(reqText, len(ac)),
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
