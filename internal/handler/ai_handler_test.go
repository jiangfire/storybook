package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type stubAIService struct {
	result     *service.StoryResult
	err        error
	configured bool
}

func (s stubAIService) GenerateStory(_ context.Context, _ string) (*service.StoryResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, nil
	}
	result := *s.result
	return &result, nil
}

func (s stubAIService) StreamGenerateStory(_ context.Context, _ string, _ service.StreamCallback) (*service.StoryResult, error) {
	return s.GenerateStory(context.Background(), "")
}

func (s stubAIService) ChatRefine(_ context.Context, original *service.StoryResult, _ string) (*service.StoryResult, error) {
	return original, s.err
}

func (s stubAIService) BatchGenerate(_ context.Context, _ string, _ int) ([]*service.StoryResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, nil
	}
	result := *s.result
	return []*service.StoryResult{&result}, nil
}

func (s stubAIService) IsConfigured() bool {
	return s.configured
}

func (s stubAIService) Chat(_ context.Context, _, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "stub chat reply", nil
}

func TestGenerateStoryFallsBackToHeuristicWhenNoConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:ai_handler_generate_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	h := NewAIHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, model.RoleProduct)
		c.Next()
	})
	r.POST("/api/ai/generate-story", h.GenerateStory)

	body, _ := json.Marshal(map[string]any{
		"requirement": "管理员希望用户可以通过邮箱密码登录后台，并在失败时看到清晰提示。",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ai/generate-story", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data := payload["data"].(map[string]any)
	if data["source"] != "heuristic" {
		t.Fatalf("expected heuristic source, got %#v", data["source"])
	}
	if data["form_draft"] == nil {
		t.Fatalf("expected form_draft in response")
	}
}

func TestAdminCanSaveAIConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.Setenv("AI_CONFIG_ENCRYPTION_KEY", key); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("AI_CONFIG_ENCRYPTION_KEY")
	})

	db, err := gorm.Open(sqlite.Open("file:ai_handler_config_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	h := NewAIHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(7))
		c.Set(middleware.CtxRoleKey, model.RoleAdmin)
		c.Next()
	})
	r.PUT("/api/admin/ai/config", h.UpsertConfig)

	body, _ := json.Marshal(map[string]any{
		"api_key":     "sk-test-1234567890",
		"model":       "gpt-4o-mini",
		"temperature": 0.3,
		"max_tokens":  1024,
		"enabled":     true,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var cfg model.AIConfig
	if err := db.First(&cfg).Error; err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", cfg.Model)
	}
	decrypted, err := service.DecryptAPIKey(cfg.APIKeyEncrypted)
	if err != nil {
		t.Fatalf("decrypt api key: %v", err)
	}
	if decrypted != "sk-test-1234567890" {
		t.Fatalf("unexpected api key: %s", decrypted)
	}
}

// Upsert 写入 AI 配置后,handler 显式调用 InvalidateAIServiceCache;
// 紧随其后的 NewAIService 应当立即反映新配置,而不是命中旧缓存继续返回 heuristic。
func TestUpsertConfigInvalidatesAIServiceCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.Setenv("AI_CONFIG_ENCRYPTION_KEY", key); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("AI_CONFIG_ENCRYPTION_KEY")
	})
	t.Cleanup(service.InvalidateAIServiceCache)
	service.InvalidateAIServiceCache()

	db, err := gorm.Open(sqlite.Open("file:ai_handler_invalidate_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// 起点:无配置 → heuristic,缓存层第一次冷查询。
	if svc := service.NewAIService(db); svc.IsConfigured() {
		t.Fatalf("expected heuristic before upsert, got configured")
	}

	h := NewAIHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(7))
		c.Set(middleware.CtxRoleKey, model.RoleAdmin)
		c.Next()
	})
	r.PUT("/api/admin/ai/config", h.UpsertConfig)

	body, _ := json.Marshal(map[string]any{
		"api_key":     "sk-test-1234567890",
		"model":       "gpt-4o-mini",
		"temperature": 0.3,
		"max_tokens":  1024,
		"enabled":     true,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert failed: %d body=%s", w.Code, w.Body.String())
	}

	// upsert 完成之后立即查询应当看到 openAI 实现,而不是上一轮缓存里的 heuristic。
	if svc := service.NewAIService(db); !svc.IsConfigured() {
		t.Fatalf("expected configured AI service after upsert, got heuristic")
	}
}

func TestGenerateStoryWithFallbackOnConfiguredServiceError(t *testing.T) {
	result, resolvedService, err := generateStoryWithFallback(
		context.Background(),
		"用户需要登录系统",
		stubAIService{
			err:        errors.New("openai unavailable"),
			configured: true,
		},
	)
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if resolvedService == nil || resolvedService.IsConfigured() {
		t.Fatalf("expected heuristic fallback service")
	}
	if result == nil {
		t.Fatalf("expected fallback result")
	}
	if len(result.Warnings) == 0 || result.Warnings[0] != "OpenAI 调用失败，已自动回退到规则草稿，请检查 AI 配置或稍后重试" {
		t.Fatalf("expected fallback warning, got %#v", result.Warnings)
	}
}

func TestResolveTestRuntimeConfigForcesEnabled(t *testing.T) {
	h := NewAIHandler(nil)

	runtimeConfig, err := h.resolveTestRuntimeConfig(model.AIConfig{}, aiConfigRequest{
		APIKey:    "sk-test-1234567890",
		Model:     "gpt-4o-mini",
		MaxTokens: ptrInt(1024),
		Enabled:   ptrBool(false),
	})
	if err != nil {
		t.Fatalf("resolve test runtime config: %v", err)
	}
	if !runtimeConfig.Enabled {
		t.Fatalf("expected test runtime config to force enabled=true")
	}
}

func ptrBool(value bool) *bool {
	return &value
}

func ptrInt(value int) *int {
	return &value
}
