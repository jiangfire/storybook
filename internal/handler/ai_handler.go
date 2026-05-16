package handler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIHandler struct {
	db           *gorm.DB
	userRepo     *repository.UserRepository
	storyRepo    *repository.StoryRepository
	aiConfigRepo *repository.AIConfigRepository
}

func NewAIHandler(db *gorm.DB) *AIHandler {
	return &AIHandler{
		db:           db,
		userRepo:     repository.NewUserRepository(db),
		storyRepo:    repository.NewStoryRepository(db),
		aiConfigRepo: repository.NewAIConfigRepository(db),
	}
}

type generateStoryRequest struct {
	Requirement string `json:"requirement" binding:"required,min=5,max=2000"`
}

func (h *AIHandler) GenerateStory(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用AI拆解")
		return
	}

	var req generateStoryRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	requirement := service.SanitizeRequirement(req.Requirement)
	if len([]rune(requirement)) < 5 {
		api.BadRequest(c, "需求描述过短")
		return
	}

	ctx, cancel := contextWithTimeout(c, 45*time.Second)
	defer cancel()

	aiService := service.NewAIService(h.db)
	result, resolvedService, err := generateStoryWithFallback(ctx, requirement, aiService)
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, fmt.Sprintf("AI生成失败: %v", err))
		return
	}

	acList := make([]gin.H, 0, len(result.SuggestedAC))
	for index, description := range result.SuggestedAC {
		acList = append(acList, gin.H{
			"description": description,
			"order":       index + 1,
		})
	}

	api.Success(c, "success", gin.H{
		"title":         result.Title,
		"user_story":    result.UserStory,
		"actor":         result.Actor,
		"action":        result.Action,
		"value":         result.Value,
		"story_type":    result.StoryType,
		"priority":      result.Priority,
		"suggested_ac":  result.SuggestedAC,
		"story_points":  result.StoryPoints,
		"tags":          result.Tags,
		"warnings":      result.Warnings,
		"source":        service.ResolveStoryResultSource(result, resolvedService),
		"is_configured": aiService.IsConfigured(),
		"form_draft": gin.H{
			"title":               result.Title,
			"description":         result.UserStory,
			"story_type":          result.StoryType,
			"priority":            result.Priority,
			"story_points":        result.StoryPoints,
			"acceptance_criteria": acList,
			"tags":                result.Tags,
		},
	})
}

func generateStoryWithFallback(ctx context.Context, requirement string, primary service.AIService) (*service.StoryResult, service.AIService, error) {
	if primary == nil {
		primary = service.NewAIServiceFromConfig(service.RuntimeAIConfig{})
	}

	result, err := primary.GenerateStory(ctx, requirement)
	if err == nil {
		return result, primary, nil
	}
	if !primary.IsConfigured() {
		return nil, primary, err
	}

	fallback := service.NewAIServiceFromConfig(service.RuntimeAIConfig{})
	fallbackResult, fallbackErr := fallback.GenerateStory(ctx, requirement)
	if fallbackErr != nil {
		return nil, primary, fmt.Errorf("openai 调用失败: %w; 规则降级也失败: %v", err, fallbackErr)
	}
	fallbackResult.Warnings = prependAIWarning(
		fallbackResult.Warnings,
		"OpenAI 调用失败，已自动回退到规则草稿，请检查 AI 配置或稍后重试",
	)
	return fallbackResult, fallback, nil
}

func prependAIWarning(warnings []string, warning string) []string {
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

type aiConfigRequest struct {
	APIKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	Temperature *float64 `json:"temperature"`
	MaxTokens   *int     `json:"max_tokens"`
	Enabled     *bool    `json:"enabled"`
}

func (h *AIHandler) GetConfig(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可查看AI配置")
		return
	}

	config, err := h.loadLatestConfig()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Success(c, "success", gin.H{
				"config": gin.H{
					"provider":       "openai",
					"model":          "gpt-4o-mini",
					"temperature":    0.2,
					"max_tokens":     1200,
					"enabled":        false,
					"api_key_masked": "",
					"is_configured":  false,
				},
			})
			return
		}
		api.Internal(c, "读取AI配置失败")
		return
	}

	api.Success(c, "success", gin.H{
		"config": h.serializeConfig(config),
	})
}

func (h *AIHandler) UpsertConfig(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可修改AI配置")
		return
	}

	var req aiConfigRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	existing, err := h.loadLatestConfig()
	if err != nil && err != gorm.ErrRecordNotFound {
		api.Internal(c, "读取AI配置失败")
		return
	}

	config, err := h.mergeConfig(existing, req, userID)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	if err := h.aiConfigRepo.Save(config); err != nil {
		api.Internal(c, "保存AI配置失败")
		return
	}

	api.Success(c, "success", gin.H{
		"config": h.serializeConfig(*config),
	})
}

func (h *AIHandler) TestConfig(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleAdmin {
		api.Forbidden(c, "仅管理员可测试AI配置")
		return
	}

	var req aiConfigRequest
	if c.Request.ContentLength > 0 {
		if !middleware.BindJSON(c, &req) {
			return
		}
	}

	existing, err := h.loadLatestConfig()
	if err != nil && err != gorm.ErrRecordNotFound {
		api.Internal(c, "读取AI配置失败")
		return
	}

	runtimeConfig, err := h.resolveTestRuntimeConfig(existing, req)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	ctx, cancel := contextWithTimeout(c, 30*time.Second)
	defer cancel()

	aiService := service.NewAIServiceFromConfig(runtimeConfig)
	result, err := aiService.GenerateStory(ctx, "需求：用户可以通过邮箱和密码登录系统，并在失败时看到明确提示。")
	if err != nil {
		api.Error(c, http.StatusBadGateway, api.CodeInternal, fmt.Sprintf("AI测试失败: %v", err))
		return
	}

	api.Success(c, "success", gin.H{
		"provider": "openai",
		"model":    runtimeConfig.Model,
		"preview": gin.H{
			"title":        result.Title,
			"user_story":   result.UserStory,
			"story_type":   result.StoryType,
			"priority":     result.Priority,
			"story_points": result.StoryPoints,
		},
	})
}

type splitStoryReq struct {
	TargetCount int `json:"target_count"`
}

func (h *AIHandler) SplitStory(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用AI拆分")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	var req splitStoryReq
	if c.Request.ContentLength > 0 {
		if !middleware.BindJSON(c, &req) {
			return
		}
	}
	if req.TargetCount <= 0 || req.TargetCount > 8 {
		req.TargetCount = 3
	}

	story, err := h.storyRepo.FindByID(storyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		api.Internal(c, "AC数据损坏")
		return
	}

	if len(criteria) == 0 {
		criteria = []model.AcceptanceCriterion{{ID: "ac-1", Description: story.Title, Status: model.ACStatusPending, Order: 1}}
	}

	count := req.TargetCount
	if count > len(criteria) {
		count = len(criteria)
	}
	if count <= 0 {
		count = 1
	}

	chunks := chunkCriteria(criteria, count)
	subStories := make([]gin.H, 0, len(chunks))
	for i, chunk := range chunks {
		title := fmt.Sprintf("%s - 子故事%d", story.Title, i+1)
		subStories = append(subStories, gin.H{
			"title":               title,
			"description":         fmt.Sprintf("由原故事 #%d 拆分", story.ID),
			"acceptance_criteria": chunk,
			"story_points":        estimatePoints(title, len(chunk)),
		})
	}

	api.Success(c, "success", gin.H{
		"story_id":     story.ID,
		"origin_title": story.Title,
		"sub_stories":  subStories,
	})
}

func (h *AIHandler) INVESTCheck(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "仅产品经理可使用INVEST检查")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.storyRepo.FindByID(storyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	criteria, _ := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	contentLen := len([]rune(strings.TrimSpace(story.Description + " " + story.Title)))
	acCount := len(criteria)

	independent := scoreIndependent(story.Title)
	negotiable := scoreNegotiable(story.Description)
	valuable := scoreValuable(story.Description)
	estimable := scoreEstimable(acCount, story.Points)
	small := scoreSmall(contentLen, acCount, story.Points)
	testable := scoreTestable(acCount)

	total := (independent + negotiable + valuable + estimable + small + testable) / 6
	total = math.Round(total*10) / 10

	suggestions := make([]string, 0)
	if small < 0.6 {
		suggestions = append(suggestions, "故事过大，建议拆分为2-4个子故事")
	}
	if testable < 0.7 {
		suggestions = append(suggestions, "建议补充可执行的验收标准（AC）")
	}
	if negotiable < 0.7 {
		suggestions = append(suggestions, "描述过于刚性，建议减少实现细节以保留协商空间")
	}

	api.Success(c, "success", gin.H{
		"story_id":     story.ID,
		"invest_score": total,
		"checks": gin.H{
			"independent": checkPayload(independent, "故事独立性", "评估故事是否依赖其他故事"),
			"negotiable":  checkPayload(negotiable, "可协商性", "评估描述是否过于限制实现方式"),
			"valuable":    checkPayload(valuable, "价值性", "评估是否体现用户价值"),
			"estimable":   checkPayload(estimable, "可估算性", "评估是否可进行工作量估算"),
			"small":       checkPayload(small, "小粒度", "评估是否需要拆分"),
			"testable":    checkPayload(testable, "可测试性", "评估验收标准是否充分"),
		},
		"suggestions": suggestions,
	})
}

func (h *AIHandler) loadLatestConfig() (model.AIConfig, error) {
	cfg, err := h.aiConfigRepo.FindLatestEnabled()
	if err != nil {
		return model.AIConfig{}, err
	}
	if cfg == nil {
		return model.AIConfig{}, gorm.ErrRecordNotFound
	}
	return *cfg, nil
}

func (h *AIHandler) serializeConfig(cfg model.AIConfig) gin.H {
	apiKeyMasked := ""
	if cfg.APIKeyEncrypted != "" {
		if decrypted, err := service.DecryptAPIKey(cfg.APIKeyEncrypted); err == nil {
			apiKeyMasked = service.MaskAPIKey(decrypted)
		}
	}

	return gin.H{
		"id":             cfg.ID,
		"provider":       "openai",
		"model":          cfg.Model,
		"temperature":    cfg.Temperature,
		"max_tokens":     cfg.MaxTokens,
		"enabled":        cfg.Enabled,
		"api_key_masked": apiKeyMasked,
		"updated_by":     cfg.UpdatedBy,
		"created_at":     cfg.CreatedAt,
		"updated_at":     cfg.UpdatedAt,
		"is_configured":  cfg.APIKeyEncrypted != "",
	}
}

func (h *AIHandler) mergeConfig(existing model.AIConfig, req aiConfigRequest, userID uint) (*model.AIConfig, error) {
	hasExisting := existing.ID != 0
	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		if hasExisting {
			modelName = existing.Model
		} else {
			modelName = "gpt-4o-mini"
		}
	}

	temperature := 0.2
	if hasExisting {
		temperature = existing.Temperature
	}
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	if temperature < 0 || temperature > 2 {
		return nil, fmt.Errorf("temperature 必须在 0 到 2 之间")
	}

	maxTokens := 1200
	if hasExisting && existing.MaxTokens > 0 {
		maxTokens = existing.MaxTokens
	}
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if maxTokens < 256 || maxTokens > 8192 {
		return nil, fmt.Errorf("max_tokens 必须在 256 到 8192 之间")
	}

	enabled := hasExisting && existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	encryptedKey := existing.APIKeyEncrypted
	if strings.TrimSpace(req.APIKey) != "" {
		var err error
		encryptedKey, err = service.EncryptAPIKey(strings.TrimSpace(req.APIKey))
		if err != nil {
			return nil, fmt.Errorf("加密API Key失败: %w", err)
		}
	}
	if strings.TrimSpace(encryptedKey) == "" {
		return nil, fmt.Errorf("请填写 OpenAI API Key")
	}

	config := &model.AIConfig{
		ID:              existing.ID,
		APIKeyEncrypted: encryptedKey,
		Model:           modelName,
		Temperature:     temperature,
		MaxTokens:       maxTokens,
		Enabled:         enabled,
		UpdatedBy:       userID,
	}
	if !hasExisting {
		config.CreatedAt = time.Now()
	}
	return config, nil
}

func (h *AIHandler) resolveRuntimeConfig(existing model.AIConfig, req aiConfigRequest) (service.RuntimeAIConfig, error) {
	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = existing.Model
	}
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	temperature := existing.Temperature
	if temperature == 0 {
		temperature = 0.2
	}
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	if temperature < 0 || temperature > 2 {
		return service.RuntimeAIConfig{}, fmt.Errorf("temperature 必须在 0 到 2 之间")
	}

	maxTokens := existing.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1200
	}
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if maxTokens < 256 || maxTokens > 8192 {
		return service.RuntimeAIConfig{}, fmt.Errorf("max_tokens 必须在 256 到 8192 之间")
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" && strings.TrimSpace(existing.APIKeyEncrypted) != "" {
		decrypted, err := service.DecryptAPIKey(existing.APIKeyEncrypted)
		if err != nil {
			return service.RuntimeAIConfig{}, fmt.Errorf("读取现有API Key失败: %w", err)
		}
		apiKey = decrypted
	}
	if apiKey == "" {
		return service.RuntimeAIConfig{}, fmt.Errorf("请先保存 OpenAI API Key")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else if existing.ID != 0 {
		enabled = existing.Enabled
	}

	return service.RuntimeAIConfig{
		APIKey:      apiKey,
		Model:       modelName,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Enabled:     enabled,
	}, nil
}

func (h *AIHandler) resolveTestRuntimeConfig(existing model.AIConfig, req aiConfigRequest) (service.RuntimeAIConfig, error) {
	runtimeConfig, err := h.resolveRuntimeConfig(existing, req)
	if err != nil {
		return service.RuntimeAIConfig{}, err
	}
	// “测试连接”应始终验证真实调用能力，而不是受启用开关影响回退到规则草稿。
	runtimeConfig.Enabled = true
	return runtimeConfig, nil
}

func contextWithTimeout(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
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

func chunkCriteria(criteria []model.AcceptanceCriterion, chunkCount int) [][]model.AcceptanceCriterion {
	if chunkCount <= 1 {
		return [][]model.AcceptanceCriterion{criteria}
	}
	chunks := make([][]model.AcceptanceCriterion, chunkCount)
	for i, ac := range criteria {
		idx := i % chunkCount
		chunks[idx] = append(chunks[idx], ac)
	}
	out := make([][]model.AcceptanceCriterion, 0, chunkCount)
	for _, ch := range chunks {
		if len(ch) > 0 {
			out = append(out, ch)
		}
	}
	return out
}

func scoreIndependent(title string) float64 {
	if strings.Contains(title, "和") || strings.Contains(strings.ToLower(title), "and") {
		return 0.6
	}
	return 0.8
}

func scoreNegotiable(desc string) float64 {
	if len([]rune(desc)) > 500 {
		return 0.6
	}
	return 0.8
}

func scoreValuable(desc string) float64 {
	keywords := []string{"价值", "效率", "转化", "体验", "质量", "收益"}
	for _, k := range keywords {
		if strings.Contains(desc, k) {
			return 0.9
		}
	}
	if strings.TrimSpace(desc) == "" {
		return 0.5
	}
	return 0.7
}

func scoreEstimable(acCount int, points *int) float64 {
	if points != nil {
		return 0.85
	}
	if acCount >= 2 {
		return 0.75
	}
	return 0.55
}

func scoreSmall(contentLen, acCount int, points *int) float64 {
	if points != nil && *points >= 8 {
		return 0.45
	}
	if contentLen > 500 || acCount > 7 {
		return 0.5
	}
	if contentLen > 300 || acCount > 5 {
		return 0.65
	}
	return 0.85
}

func scoreTestable(acCount int) float64 {
	if acCount >= 5 {
		return 0.9
	}
	if acCount >= 3 {
		return 0.75
	}
	if acCount >= 1 {
		return 0.6
	}
	return 0.3
}

func checkPayload(score float64, title, desc string) gin.H {
	status := "pass"
	if score < 0.6 {
		status = "fail"
	} else if score < 0.75 {
		status = "warning"
	}
	return gin.H{
		"score":       score,
		"status":      status,
		"title":       title,
		"description": desc,
	}
}
