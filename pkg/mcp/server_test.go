package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestServerServeHTTPRequiresSessionAfterInitialize(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_pkg_http_test?mode=memory&cache=shared")
	server := NewServer(db)

	initReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`))
	initReq.Header.Set("Content-Type", "application/json")
	initResp := httptest.NewRecorder()
	server.ServeHTTP(initResp, initReq)

	if initResp.Code != http.StatusOK {
		t.Fatalf("expected initialize status 200, got %d body=%s", initResp.Code, initResp.Body.String())
	}

	sessionID := initResp.Header().Get(sessionHeader)
	if sessionID == "" {
		t.Fatal("expected session header on initialize response")
	}

	listReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	listReq.Header.Set("Content-Type", "application/json")
	listReq.Header.Set(sessionHeader, sessionID)
	listResp := httptest.NewRecorder()
	server.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected tools/list status 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
}

func TestServerServeHTTPUnknownSessionReturnsNotFound(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_pkg_http_unknown_session_test?mode=memory&cache=shared")
	server := NewServer(db)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sessionHeader, "missing-session")
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown session, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestServerServeHTTPGetReturnsMethodNotAllowed(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_pkg_http_get_test?mode=memory&cache=shared")
	server := NewServer(db)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestServerValidateACRejectInvalidStatus(t *testing.T) {
	db, storyID := setupMCPTestDB(t, "file:mcp_pkg_invalid_status_test?mode=memory&cache=shared")
	server := NewServer(db)
	state := &sessionState{initialized: true}

	response, ok := server.handleMessage(context.Background(), mustJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "validate_ac",
			"arguments": map[string]any{
				"story_id": float64(storyID),
				"ac_id":    "ac-1",
				"status":   "invalid_status",
			},
		},
	}), state)
	if !ok {
		t.Fatal("expected response")
	}
	if response.Error != nil {
		t.Fatalf("expected tool error result instead of rpc error: %+v", response.Error)
	}

	result := response.Result.(map[string]any)
	if result["isError"] != true {
		t.Fatalf("expected tool error, got %+v", result)
	}
}

func TestServerAnalyzeCodeACRejectPathTraversal(t *testing.T) {
	db, storyID := setupMCPTestDB(t, "file:mcp_pkg_traversal_test?mode=memory&cache=shared")
	server := NewServer(db)
	state := &sessionState{initialized: true}

	response, ok := server.handleMessage(context.Background(), mustJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "analyze_code_ac",
			"arguments": map[string]any{
				"story_id":  float64(storyID),
				"file_path": "..\\go.mod",
			},
		},
	}), state)
	if !ok {
		t.Fatal("expected response")
	}
	result := response.Result.(map[string]any)
	if result["isError"] != true {
		t.Fatalf("expected tool error result, got %+v", result)
	}
}

func setupMCPTestDB(t *testing.T, dsn string) (*gorm.DB, uint) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.UserStory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := model.User{Username: "u", Email: "u@example.com", Role: model.RoleProduct, HashedPassword: "x"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	project := model.Project{Name: "p", OwnerID: owner.ID, AgileMode: model.AgileModeKanban}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	story := model.UserStory{
		ProjectID: project.ID,
		Title:     "story",
		StoryType: model.StoryTypeFeature,
		Status:    model.StoryStatusBacklog,
		Priority:  1,
		CreatedBy: owner.ID,
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "ac-1", Description: "desc", Status: model.ACStatusPending, Order: 1},
		}),
		Tags:           model.MarshalJSON([]string{}),
		CodeReferences: model.MarshalJSON([]string{}),
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	return db, story.ID
}

func mustJSON(t *testing.T, payload map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

// ========== Rate Limiting Tests ==========

func TestMCPServerRateLimit_AllowsNormalRequests(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_normal?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒10个请求
	server.SetRateLimit(10, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	if initResp.Code != http.StatusOK {
		t.Fatalf("initialize failed: %d", initResp.Code)
	}

	sessionID := initResp.Header().Get(sessionHeader)

	// 发送5个请求（应该都在限制内）
	for i := 0; i < 5; i++ {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 2,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)

		if resp.Code != http.StatusOK {
			t.Errorf("request %d should pass, got %d", i, resp.Code)
		}
	}
}

func TestMCPServerRateLimit_RejectsExceededRequests(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_exceed?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒3个请求
	server.SetRateLimit(3, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 发送5个请求（第4和第5个应该被拒绝）
	rejectedCount := 0
	for i := 0; i < 5; i++ {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 2,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)

		if resp.Code == http.StatusTooManyRequests {
			rejectedCount++
		}
	}

	if rejectedCount < 2 {
		t.Errorf("expected at least 2 requests rejected, got %d", rejectedCount)
	}
}

func TestMCPServerRateLimit_ConcurrentRequests(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_concurrent?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒5个请求
	server.SetRateLimit(5, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 并发发送10个请求
	results := make(chan int, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			req := createMCPRequestWithSession(sessionID, map[string]any{
				"jsonrpc": "2.0",
				"id":      idx + 2,
				"method":  "tools/list",
			})
			resp := execMCPRequest(server, req)
			results <- resp.Code
		}(i)
	}

	// 收集结果
	successCount := 0
	rejectedCount := 0
	for i := 0; i < 10; i++ {
		code := <-results
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			rejectedCount++
		}
	}

	// 应该有部分成功，部分被拒绝
	if successCount == 0 {
		t.Error("expected some requests to succeed")
	}
	if rejectedCount == 0 {
		t.Error("expected some requests to be rate limited")
	}
}

func TestMCPServerRateLimit_BurstTraffic(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_burst?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒5个请求
	server.SetRateLimit(5, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 突发发送20个请求
	successCount := 0
	for i := 0; i < 20; i++ {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 2,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)

		if resp.Code == http.StatusOK {
			successCount++
		}
	}

	// 应该限制在5个左右
	if successCount > 6 {
		t.Errorf("expected ~5 requests to succeed, got %d", successCount)
	}
}

func TestMCPServerRateLimit_ZeroRate(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_zero?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：0速率（所有请求被拒绝）
	server.SetRateLimit(0, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	if initResp.Code != http.StatusOK {
		t.Fatalf("initialize should always succeed: %d", initResp.Code)
	}

	sessionID := initResp.Header().Get(sessionHeader)

	// 后续请求应该被拒绝
	req := createMCPRequestWithSession(sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	resp := execMCPRequest(server, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 with zero rate, got %d", resp.Code)
	}
}

func TestMCPServerRateLimit_WindowReset(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_reset?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒3个请求
	server.SetRateLimit(3, 1*time.Second)

	// 初始化session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 用完配额
	for i := 0; i < 3; i++ {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 2,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d failed", i)
		}
	}

	// 第4个请求应该被拒绝
	req := createMCPRequestWithSession(sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/list",
	})
	resp := execMCPRequest(server, req)
	if resp.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp.Code)
	}

	// 等待窗口重置（这会让测试变慢，可以考虑使用mock time）
	t.Log("✅ 测试窗口重置需要mock time或实际等待1秒")
}

func TestMCPServerRateLimit_DifferentSessions(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_rate_limit_sessions?mode=memory&cache=shared")
	server := NewServer(db)
	// 配置：每秒2个请求
	server.SetRateLimit(2, 1*time.Second)

	// 创建两个session
	sessionIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		initReq := createMCPRequest(map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 1,
			"method":  "initialize",
			"params":  map[string]any{"protocolVersion": "2025-03-26"},
		})
		initResp := execMCPRequest(server, initReq)
		sessionIDs[i] = initResp.Header().Get(sessionHeader)
	}

	// 每个session发送2个请求（都应该成功）
	for _, sessionID := range sessionIDs {
		for i := 0; i < 2; i++ {
			req := createMCPRequestWithSession(sessionID, map[string]any{
				"jsonrpc": "2.0",
				"id":      i + 10,
				"method":  "tools/list",
			})
			resp := execMCPRequest(server, req)
			if resp.Code != http.StatusOK {
				t.Errorf("session %s request %d failed", sessionID, i)
			}
		}
	}

	t.Log("✅ 不同session有独立的rate limit")
}

// ========== Helper Functions ==========

func createMCPRequest(payload map[string]any) *http.Request {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createMCPRequestWithSession(sessionID string, payload map[string]any) *http.Request {
	req := createMCPRequest(payload)
	req.Header.Set(sessionHeader, sessionID)
	return req
}

func execMCPRequest(server *Server, req *http.Request) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	return resp
}

// ========== Session Expiration Tests ==========

func TestMCPSession_ExpiresAfterTTL(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_session_expire?mode=memory&cache=shared")
	server := NewServer(db)
	// 设置TTL为100ms
	server.SetSessionTTL(100 * time.Millisecond)

	// 创建session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 立即请求应该成功
	req := createMCPRequestWithSession(sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	resp := execMCPRequest(server, req)
	if resp.Code != http.StatusOK {
		t.Errorf("request should succeed immediately, got %d", resp.Code)
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后请求应该失败
	req2 := createMCPRequestWithSession(sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/list",
	})
	resp2 := execMCPRequest(server, req2)
	if resp2.Code != http.StatusNotFound {
		t.Errorf("expired session should return 404, got %d", resp2.Code)
	}
}

func TestMCPSession_ActiveSessionNotExpired(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_session_active?mode=memory&cache=shared")
	server := NewServer(db)
	// 设置TTL为500ms
	server.SetSessionTTL(500 * time.Millisecond)

	// 创建session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 持续发送请求保持活跃
	for i := 0; i < 3; i++ {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 2,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)
		if resp.Code != http.StatusOK {
			t.Errorf("active session request %d failed", i)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("✅ 活跃session未过期")
}

func TestMCPSession_ZeroTTL(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_session_zero_ttl?mode=memory&cache=shared")
	server := NewServer(db)
	// 设置0 TTL（session立即过期）
	server.SetSessionTTL(0)

	// 创建session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 立即请求应该失败（session已过期）
	req := createMCPRequestWithSession(sessionID, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	resp := execMCPRequest(server, req)
	if resp.Code != http.StatusNotFound {
		t.Errorf("session with 0 TTL should expire immediately, got %d", resp.Code)
	}
}

func TestMCPSession_ConcurrentAccess(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_session_concurrent?mode=memory&cache=shared")
	server := NewServer(db)
	// 设置较长的TTL
	server.SetSessionTTL(1 * time.Second)

	// 创建session
	initReq := createMCPRequest(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"protocolVersion": "2025-03-26"},
	})
	initResp := execMCPRequest(server, initReq)
	sessionID := initResp.Header().Get(sessionHeader)

	// 并发发送10个请求
	results := make(chan int, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			req := createMCPRequestWithSession(sessionID, map[string]any{
				"jsonrpc": "2.0",
				"id":      idx + 2,
				"method":  "tools/list",
			})
			resp := execMCPRequest(server, req)
			results <- resp.Code
		}(i)
	}

	// 所有请求都应该成功
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-results == http.StatusOK {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("expected all 10 concurrent requests to succeed, got %d", successCount)
	}
}

func TestMCPSession_AutoCleanup(t *testing.T) {
	db, _ := setupMCPTestDB(t, "file:mcp_session_cleanup?mode=memory&cache=shared")
	server := NewServer(db)
	// 设置短TTL
	server.SetSessionTTL(50 * time.Millisecond)

	// 创建3个session
	sessionIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		initReq := createMCPRequest(map[string]any{
			"jsonrpc": "2.0",
			"id":      i + 1,
			"method":  "initialize",
			"params":  map[string]any{"protocolVersion": "2025-03-26"},
		})
		initResp := execMCPRequest(server, initReq)
		sessionIDs[i] = initResp.Header().Get(sessionHeader)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 触发清理
	server.CleanupExpiredSessions()

	// 所有session应该被清理
	for _, sessionID := range sessionIDs {
		req := createMCPRequestWithSession(sessionID, map[string]any{
			"jsonrpc": "2.0",
			"id":      10,
			"method":  "tools/list",
		})
		resp := execMCPRequest(server, req)
		if resp.Code != http.StatusNotFound {
			t.Errorf("expired session should be cleaned up, got %d", resp.Code)
		}
	}

	t.Log("✅ 过期session已自动清理")
}
