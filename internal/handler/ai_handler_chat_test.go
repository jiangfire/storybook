package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/service"
	"gorm.io/gorm"
)

type stubChatCompleter struct {
	reply      string
	err        error
	configured bool
}

func (s stubChatCompleter) Chat(_ context.Context, _, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.reply, nil
}

func (s stubChatCompleter) IsConfigured() bool {
	return s.configured
}

func setupStoryChatRouter(t *testing.T, role string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:ai_handler_story_chat_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	h := NewAIHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, role)
		c.Next()
	})
	r.POST("/api/ai/story-chat", h.StoryChat)
	return r, db
}

func TestStoryChatRequiresProductOrAdminRole(t *testing.T) {
	r, _ := setupStoryChatRouter(t, model.RoleDeveloper)

	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "帮我写一个登录故事"}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ai/story-chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestStoryChatRejectsEmptyUserMessage(t *testing.T) {
	r, _ := setupStoryChatRouter(t, model.RoleProduct)

	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "assistant", "content": "请描述你的需求"}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ai/story-chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestStoryChatRequiresAIConfig(t *testing.T) {
	t.Cleanup(service.InvalidateAIServiceCache)
	service.InvalidateAIServiceCache()

	r, _ := setupStoryChatRouter(t, model.RoleProduct)

	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "帮我写一个登录故事"}},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ai/story-chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRunStoryChatParsesAndNormalizesReply(t *testing.T) {
	reply := "下面是当前草稿，还需要补充异常场景吗？" + "\n" +
		`{"reply":"已经生成草稿，还需要补充异常场景吗？","form_draft":{"title":"支持手机号验证码登录","description":"作为注册用户，我想要用手机号和验证码登录，以便快速进入系统","story_type":"epic","priority":9,"story_points":7,"acceptance_criteria":[{"description":"Given 未登录用户 When 输入正确手机号和验证码 Then 登录成功","order":1},{"description":"   ","order":2}],"tags":["auth","  ","登录"]}}`

	current := &storyChatDraft{
		Title:       "旧标题",
		StoryType:   model.StoryTypeFeature,
		Priority:    2,
		StoryPoints: 5,
		AcceptanceCriteria: []storyChatAC{
			{Description: "已有 AC", Order: 1},
		},
		Tags: []string{"legacy"},
	}

	result, err := runStoryChat(context.Background(), stubChatCompleter{reply: reply, configured: true}, storyChatRequest{
		Messages: []storyChatMessage{
			{Role: "user", Content: "做一个手机号登录"},
			{Role: "assistant", Content: "好的，优先级是多少？"},
			{Role: "user", Content: "高优先级"},
		},
		CurrentDraft: current,
	})
	if err != nil {
		t.Fatalf("runStoryChat: %v", err)
	}

	if result.Reply != "已经生成草稿，还需要补充异常场景吗？" {
		t.Fatalf("unexpected reply: %q", result.Reply)
	}
	draft := result.Draft
	if draft.Title != "支持手机号验证码登录" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
	// 非法 story_type 回退当前草稿值
	if draft.StoryType != model.StoryTypeFeature {
		t.Fatalf("expected feature fallback, got %q", draft.StoryType)
	}
	// 越界 priority/story_points 回退当前草稿值
	if draft.Priority != 2 {
		t.Fatalf("expected priority fallback 2, got %d", draft.Priority)
	}
	if draft.StoryPoints != 5 {
		t.Fatalf("expected points fallback 5, got %d", draft.StoryPoints)
	}
	// 空 AC 被剔除
	if len(draft.AcceptanceCriteria) != 1 || draft.AcceptanceCriteria[0].Description == "" {
		t.Fatalf("unexpected AC list: %#v", draft.AcceptanceCriteria)
	}
	if len(draft.Tags) != 2 {
		t.Fatalf("unexpected tags: %#v", draft.Tags)
	}
}

func TestRunStoryChatFallsBackToCurrentDraftWhenFieldsMissing(t *testing.T) {
	reply := `{"reply":"先给出初稿","form_draft":{}}`

	result, err := runStoryChat(context.Background(), stubChatCompleter{reply: reply, configured: true}, storyChatRequest{
		Messages: []storyChatMessage{{Role: "user", Content: "做一个导出报表的功能"}},
		CurrentDraft: &storyChatDraft{
			Title:       "导出报表",
			Description: "作为项目经理，我想要导出报表，以便离线汇报",
			StoryType:   model.StoryTypeChore,
			Priority:    3,
			StoryPoints: 8,
			AcceptanceCriteria: []storyChatAC{
				{Description: "Given 已完成冲刺 When 点击导出 Then 下载 JSON 文件", Order: 1},
			},
			Tags: []string{"report"},
		},
	})
	if err != nil {
		t.Fatalf("runStoryChat: %v", err)
	}

	draft := result.Draft
	if draft.Title != "导出报表" || draft.Description == "" {
		t.Fatalf("expected fallback to current draft, got %#v", draft)
	}
	if draft.StoryType != model.StoryTypeChore || draft.Priority != 3 || draft.StoryPoints != 8 {
		t.Fatalf("expected enum fallback, got %#v", draft)
	}
	if len(draft.AcceptanceCriteria) != 1 || len(draft.Tags) != 1 {
		t.Fatalf("expected AC/tags fallback, got %#v", draft)
	}
}

func TestRunStoryChatRejectsReplyWithoutJSON(t *testing.T) {
	_, err := runStoryChat(context.Background(), stubChatCompleter{reply: "好的，我明白了", configured: true}, storyChatRequest{
		Messages: []storyChatMessage{{Role: "user", Content: "做一个登录功能"}},
	})
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
}

func TestRunStoryChatPropagatesChatError(t *testing.T) {
	_, err := runStoryChat(context.Background(), stubChatCompleter{err: errors.New("timeout")}, storyChatRequest{
		Messages: []storyChatMessage{{Role: "user", Content: "做一个登录功能"}},
	})
	if err == nil {
		t.Fatalf("expected chat error, got nil")
	}
}
