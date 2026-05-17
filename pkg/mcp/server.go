package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiangfire/storybook/internal/fileutil"
	"github.com/jiangfire/storybook/internal/service"
	"gorm.io/gorm"
)

const (
	jsonRPCVersion      = "2.0"
	protocolVersion2025 = "2025-03-26"
	sessionHeader       = "Mcp-Session-Id"
)

type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     ToolHandler    `json:"-"`
}

type Server struct {
	name            string
	version         string
	protocolVersion string
	service         *service.MCPService
	tools           map[string]ToolDefinition

	mu         sync.Mutex
	sessions   map[string]*sessionState
	rateLimit  *rateLimiter
	sessionTTL time.Duration // session过期时间
}

type sessionState struct {
	initialized bool
	createdAt   time.Time   // session创建时间
	requests    []time.Time // 请求时间戳队列
}

type rateLimiter struct {
	maxRequests int
	window      time.Duration
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func NewServer(db *gorm.DB) *Server {
	root, _ := os.Getwd()
	s := &Server{
		name:            "storybook-mcp",
		version:         "1.1.0",
		protocolVersion: protocolVersion2025,
		service:         service.NewMCPService(db, root, nil),
		tools:           map[string]ToolDefinition{},
		sessions:        map[string]*sessionState{},
		rateLimit:       nil,            // 默认无限制
		sessionTTL:      -1 * time.Hour, // 默认永不过期（-1表示禁用）
	}
	s.registerTools()
	return s
}

// SetRateLimit 设置速率限制
// maxRequests: 时间窗口内允许的最大请求数（0表示禁止所有请求）
// window: 时间窗口大小
func (s *Server) SetRateLimit(maxRequests int, window time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxRequests <= 0 {
		s.rateLimit = &rateLimiter{maxRequests: 0, window: window}
		return
	}

	s.rateLimit = &rateLimiter{
		maxRequests: maxRequests,
		window:      window,
	}
}

// checkRateLimit 检查是否超过速率限制
// 返回 true 表示允许请求，false 表示超过限制
func (s *Server) checkRateLimit(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 没有配置限制
	if s.rateLimit == nil {
		return true
	}

	// maxRequests=0 表示禁止所有请求
	if s.rateLimit.maxRequests == 0 {
		return false
	}

	session := s.sessions[sessionID]
	if session == nil {
		return true
	}

	now := time.Now()
	windowStart := now.Add(-s.rateLimit.window)

	// 清理窗口外的旧请求
	validRequests := []time.Time{}
	for _, reqTime := range session.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= s.rateLimit.maxRequests {
		return false
	}

	// 记录当前请求
	session.requests = append(validRequests, now)
	return true
}

// SetSessionTTL 设置session过期时间
// ttl: session存活时间（0表示立即过期，负数表示永不过期）
func (s *Server) SetSessionTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionTTL = ttl
}

// CleanupExpiredSessions 清理过期的session
func (s *Server) CleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 负数表示永不过期
	if s.sessionTTL < 0 {
		return
	}

	now := time.Now()
	for sessionID, session := range s.sessions {
		// 检查是否过期
		if s.sessionTTL == 0 || now.Sub(session.createdAt) > s.sessionTTL {
			delete(s.sessions, sessionID)
		}
	}
}

// isSessionExpired 检查session是否过期
func (s *Server) isSessionExpired(session *sessionState) bool {
	// 负数表示永不过期
	if s.sessionTTL < 0 {
		return false
	}

	// 0表示立即过期
	if s.sessionTTL == 0 {
		return true
	}

	// 检查是否超过TTL
	return time.Since(session.createdAt) > s.sessionTTL
}

func (s *Server) registerTools() {
	s.tools["analyze_code_ac"] = ToolDefinition{
		Name:        "analyze_code_ac",
		Description: "Analyze a code file against a story's acceptance criteria and report likely coverage.",
		InputSchema: objectSchema(
			requiredStringProperty("file_path", "Workspace-relative code file path."),
			requiredIntegerProperty("story_id", "Story ID."),
		),
		Handler: s.handleAnalyzeCodeAC,
	}
	s.tools["check_ac_coverage"] = ToolDefinition{
		Name:        "check_ac_coverage",
		Description: "Summarize acceptance criteria coverage for a story.",
		InputSchema: objectSchema(
			requiredIntegerProperty("story_id", "Story ID."),
		),
		Handler: s.handleCheckACCoverage,
	}
	s.tools["generate_ac_tests"] = ToolDefinition{
		Name:        "generate_ac_tests",
		Description: "Generate test case scaffolding from a story's acceptance criteria.",
		InputSchema: objectSchema(
			requiredIntegerProperty("story_id", "Story ID."),
		),
		Handler: s.handleGenerateACTests,
	}
	s.tools["get_acceptance_criteria"] = ToolDefinition{
		Name:        "get_acceptance_criteria",
		Description: "Get acceptance criteria for a story with coverage metadata.",
		InputSchema: objectSchema(
			requiredIntegerProperty("story_id", "Story ID."),
		),
		Handler: s.handleGetAcceptanceCriteria,
	}
	s.tools["get_story"] = ToolDefinition{
		Name:        "get_story",
		Description: "Get a story with assignee and acceptance criteria.",
		InputSchema: objectSchema(
			requiredIntegerProperty("story_id", "Story ID."),
		),
		Handler: s.handleGetStory,
	}
	s.tools["list_stories"] = ToolDefinition{
		Name:        "list_stories",
		Description: "List stories in a project, optionally filtered by status.",
		InputSchema: objectSchema(
			requiredIntegerProperty("project_id", "Project ID."),
			optionalStringProperty("status", "Optional story status filter."),
		),
		Handler: s.handleListStories,
	}
	s.tools["update_ac_status"] = ToolDefinition{
		Name:        "update_ac_status",
		Description: "Batch update acceptance criteria statuses for a story.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"story_id": integerProperty("Story ID."),
				"updates": map[string]any{
					"type":        "array",
					"description": "AC status updates.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"ac_id":    stringProperty("Acceptance criteria ID."),
							"status":   stringEnumProperty("AC status.", "pending", "passed", "failed"),
							"evidence": optionalStringProperty("evidence", "Optional evidence text.")["properties"].(map[string]any)["evidence"],
						},
						"required": []string{"ac_id", "status"},
					},
				},
			},
			"required": []string{"story_id", "updates"},
		},
		Handler: s.handleUpdateACStatus,
	}
	s.tools["validate_ac"] = ToolDefinition{
		Name:        "validate_ac",
		Description: "Update a single acceptance criterion status for a story.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"story_id": integerProperty("Story ID."),
				"ac_id":    stringProperty("Acceptance criteria ID."),
				"status":   stringEnumProperty("AC status.", "pending", "passed", "failed"),
				"evidence": optionalStringProperty("evidence", "Optional evidence text.")["properties"].(map[string]any)["evidence"],
			},
			"required": []string{"story_id", "ac_id", "status"},
		},
		Handler: s.handleValidateAC,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAllowedOrigin(r.Header.Get("Origin")) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleHTTPPost(w, r)
	case http.MethodGet:
		w.Header().Set("Allow", "POST, GET, DELETE")
		http.Error(w, "streaming GET is not supported by this server", http.StatusMethodNotAllowed)
	case http.MethodDelete:
		s.handleHTTPDelete(w, r)
	default:
		w.Header().Set("Allow", "POST, GET, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHTTPPost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body failed", http.StatusBadRequest)
		return
	}

	sessionID := strings.TrimSpace(r.Header.Get(sessionHeader))
	state, resolvedSessionID, created, err := s.resolveHTTPSession(sessionID, body)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "unknown session") {
			statusCode = http.StatusNotFound
		}
		http.Error(w, err.Error(), statusCode)
		return
	}
	if created {
		w.Header().Set(sessionHeader, resolvedSessionID)
	}

	// 检查rate limiting（初始化请求除外）
	if !created && !s.checkRateLimit(resolvedSessionID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": jsonRPCVersion,
			"id":      nil,
			"error": map[string]any{
				"code":    -32000,
				"message": "rate limit exceeded",
			},
		})
		return
	}

	responses, isBatch, err := s.handlePayload(r.Context(), body, state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(responses) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if isBatch {
		_ = json.NewEncoder(w).Encode(responses)
		return
	}
	_ = json.NewEncoder(w).Encode(responses[0])
}

func (s *Server) handleHTTPDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.Header.Get(sessionHeader))
	if sessionID == "" {
		http.Error(w, "missing session header", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	_, ok := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	if !ok {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resolveHTTPSession(sessionID string, body []byte) (*sessionState, string, bool, error) {
	if sessionID == "" {
		if !payloadContainsInitialize(body) {
			return nil, "", false, fmt.Errorf("missing %s header", sessionHeader)
		}
		sessionID = newSessionID()
		s.mu.Lock()
		defer s.mu.Unlock()
		state := &sessionState{
			createdAt: time.Now(), // 设置创建时间
		}
		s.sessions[sessionID] = state
		return state, sessionID, true, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.sessions[sessionID]
	if !ok {
		return nil, "", false, fmt.Errorf("unknown session")
	}

	// 检查session是否过期
	if s.isSessionExpired(state) {
		delete(s.sessions, sessionID)
		return nil, "", false, fmt.Errorf("unknown session") // 保持与missing session一致
	}

	return state, sessionID, false, nil
}

func (s *Server) handlePayload(ctx context.Context, payload []byte, state *sessionState) ([]jsonRPCResponse, bool, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return []jsonRPCResponse{newErrorResponse(nil, -32700, "empty payload")}, false, nil
	}

	if trimmed[0] == '[' {
		var raws []json.RawMessage
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return []jsonRPCResponse{newErrorResponse(nil, -32700, "parse error")}, true, nil
		}
		if len(raws) == 0 {
			return []jsonRPCResponse{newErrorResponse(nil, -32600, "invalid request")}, true, nil
		}

		responses := make([]jsonRPCResponse, 0, len(raws))
		for _, raw := range raws {
			resp, ok := s.handleMessage(ctx, raw, state)
			if ok {
				responses = append(responses, resp)
			}
		}
		return responses, true, nil
	}

	resp, ok := s.handleMessage(ctx, trimmed, state)
	if !ok {
		return nil, false, nil
	}
	return []jsonRPCResponse{resp}, false, nil
}

func (s *Server) handleMessage(ctx context.Context, raw json.RawMessage, state *sessionState) (jsonRPCResponse, bool) {
	var req jsonRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return newErrorResponse(nil, -32700, "parse error"), true
	}
	if req.JSONRPC != jsonRPCVersion || strings.TrimSpace(req.Method) == "" {
		return newErrorResponse(req.ID, -32600, "invalid request"), true
	}

	isNotification := len(req.ID) == 0
	if !state.initialized && req.Method != "initialize" && req.Method != "notifications/initialized" {
		if isNotification {
			return jsonRPCResponse{}, false
		}
		return newErrorResponse(req.ID, -32002, "server not initialized"), true
	}

	switch req.Method {
	case "initialize":
		return newResultResponse(req.ID, s.handleInitialize(req.Params, state)), !isNotification
	case "notifications/initialized":
		state.initialized = true
		return jsonRPCResponse{}, false
	case "notifications/cancelled":
		return jsonRPCResponse{}, false
	case "ping":
		if isNotification {
			return jsonRPCResponse{}, false
		}
		return newResultResponse(req.ID, map[string]any{}), true
	case "tools/list":
		if isNotification {
			return jsonRPCResponse{}, false
		}
		return newResultResponse(req.ID, s.handleToolsList()), true
	case "tools/call":
		if isNotification {
			return jsonRPCResponse{}, false
		}
		result, rpcErr := s.handleToolsCall(ctx, req.Params)
		if rpcErr != nil {
			return newErrorResponse(req.ID, rpcErr.Code, rpcErr.Message), true
		}
		return newResultResponse(req.ID, result), true
	default:
		if isNotification {
			return jsonRPCResponse{}, false
		}
		return newErrorResponse(req.ID, -32601, "method not found"), true
	}
}

func (s *Server) handleInitialize(raw json.RawMessage, state *sessionState) map[string]any {
	state.initialized = true

	requestedVersion := protocolVersion2025
	if len(raw) > 0 {
		var params initializeParams
		if err := json.Unmarshal(raw, &params); err == nil && strings.TrimSpace(params.ProtocolVersion) != "" {
			requestedVersion = strings.TrimSpace(params.ProtocolVersion)
		}
	}

	return map[string]any{
		"protocolVersion": minSupportedProtocolVersion(requestedVersion, s.protocolVersion),
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
		"instructions": "Storybook MCP server exposes story, acceptance-criteria, and code-analysis tools for local project workflows.",
	}
}

func (s *Server) handleToolsList() map[string]any {
	tools := make([]ToolDefinition, 0, len(s.tools))
	for _, name := range sortedToolNames(s.tools) {
		tools = append(tools, s.tools[name])
	}
	return map[string]any{
		"tools": tools,
	}
}

func (s *Server) handleToolsCall(ctx context.Context, raw json.RawMessage) (map[string]any, *jsonRPCError) {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, &jsonRPCError{Code: -32602, Message: "invalid params"}
	}

	definition, ok := s.tools[params.Name]
	if !ok {
		return nil, &jsonRPCError{Code: -32602, Message: "unknown tool"}
	}

	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}

	result, err := definition.Handler(ctx, params.Arguments)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": err.Error(),
				},
			},
			"isError": true,
		}, nil
	}

	text := marshalToolResult(result)
	response := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	}
	if structured := structuredResult(result); structured != nil {
		response["structuredContent"] = structured
	}
	return response, nil
}

func (s *Server) handleGetStory(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}
	return s.service.GetStory(storyID)
}

func (s *Server) handleListStories(_ context.Context, args map[string]any) (any, error) {
	projectID, err := getUintArg(args, "project_id")
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(getOptionalString(args, "status"))
	if status == "" {
		if filters, ok := args["filters"].(map[string]any); ok {
			status = strings.TrimSpace(getOptionalString(filters, "status"))
		}
	}

	return s.service.ListStories(projectID, status)
}

func (s *Server) handleGetAcceptanceCriteria(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}
	return s.service.GetStoryAC(storyID)
}

func (s *Server) handleValidateAC(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	acID := strings.TrimSpace(getOptionalString(args, "ac_id"))
	if acID == "" {
		return nil, fmt.Errorf("ac_id is required")
	}
	status := strings.TrimSpace(getOptionalString(args, "status"))
	if status == "" {
		return nil, fmt.Errorf("status is required")
	}
	evidence := strings.TrimSpace(getOptionalString(args, "evidence"))

	// MCP 协议入口没有认证身份,userID 传 0 让 service 层走系统调用语义:
	// VerifiedBy 保持 nil,activity_log 记录 user_id=0 + actor.source=mcp。
	data, err := s.service.ValidateAC(storyID, 0, acID, status, evidence)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"success": true,
		"message": "AC状态已更新",
		"data":    data,
	}, nil
}

func (s *Server) handleCheckACCoverage(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}
	return s.service.GetACCoverage(storyID)
}

func (s *Server) handleGenerateACTests(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}
	return s.service.GenerateACTests(storyID)
}

func (s *Server) handleAnalyzeCodeAC(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	filePath := strings.TrimSpace(getOptionalString(args, "file_path"))
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	data, err := s.service.AnalyzeCodeAC(storyID, filePath)
	if err != nil {
		if errors.Is(err, fileutil.ErrPathOutsideWorkspace) || errors.Is(err, fileutil.ErrPathEmpty) {
			return nil, fmt.Errorf("file_path outside workspace")
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("code file not found")
		}
		return nil, err
	}
	return data, nil
}

func (s *Server) handleUpdateACStatus(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	rawUpdates, ok := args["updates"]
	if !ok {
		return nil, fmt.Errorf("updates is required")
	}

	updates, err := parseUpdates(rawUpdates)
	if err != nil {
		return nil, err
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("updates is empty")
	}

	// 协议入口同样以 userID=0 调用,服务层会跳过 VerifiedBy 写入。
	return s.service.BatchUpdateACStatus(storyID, 0, updates)
}

func newResultResponse(id json.RawMessage, result any) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  result,
	}
}

func newErrorResponse(id json.RawMessage, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
}

func marshalToolResult(result any) string {
	if result == nil {
		return "null"
	}

	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", result)
	}
	return string(raw)
}

func structuredResult(result any) any {
	if result == nil {
		return nil
	}
	switch result.(type) {
	case map[string]any, []any, []map[string]any:
		return result
	default:
		return result
	}
}

func payloadContainsInitialize(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}

	if trimmed[0] == '[' {
		var raws []json.RawMessage
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return false
		}
		for _, raw := range raws {
			if payloadContainsInitialize(raw) {
				return true
			}
		}
		return false
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(trimmed, &req); err != nil {
		return false
	}
	return req.Method == "initialize"
}

func getUintArg(args map[string]any, key string) (uint, error) {
	value, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch typed := value.(type) {
	case float64:
		if typed < 0 || float64(uint64(typed)) != typed {
			return 0, fmt.Errorf("%s is invalid", key)
		}
		return uint(typed), nil
	case int:
		if typed < 0 {
			return 0, fmt.Errorf("%s is invalid", key)
		}
		return uint(typed), nil
	case int64:
		if typed < 0 {
			return 0, fmt.Errorf("%s is invalid", key)
		}
		return uint(typed), nil
	case uint:
		return typed, nil
	case string:
		n, err := strconvParseUint(strings.TrimSpace(typed))
		if err != nil {
			return 0, fmt.Errorf("%s is invalid", key)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s is invalid", key)
	}
}

func getOptionalString(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func parseUpdates(raw any) ([]service.MCPACUpdateInput, error) {
	rows, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("updates is invalid")
	}

	updates := make([]service.MCPACUpdateInput, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}

		acID := strings.TrimSpace(getOptionalString(item, "ac_id"))
		status := strings.TrimSpace(getOptionalString(item, "status"))
		if acID == "" || status == "" {
			continue
		}

		updates = append(updates, service.MCPACUpdateInput{
			ACID:     acID,
			Status:   status,
			Evidence: strings.TrimSpace(getOptionalString(item, "evidence")),
		})
	}

	return updates, nil
}

func strconvParseUint(value string) (uint, error) {
	out, err := strconv.ParseUint(value, 10, strconv.IntSize)
	return uint(out), err
}

func objectSchema(properties ...map[string]any) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	required := make([]string, 0, len(properties))

	props := schema["properties"].(map[string]any)
	for _, property := range properties {
		for key, value := range property["properties"].(map[string]any) {
			props[key] = value
		}
		if names, ok := property["required"].([]string); ok {
			required = append(required, names...)
		}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func requiredIntegerProperty(name, description string) map[string]any {
	return map[string]any{
		"properties": map[string]any{
			name: integerProperty(description),
		},
		"required": []string{name},
	}
}

func requiredStringProperty(name, description string) map[string]any {
	return map[string]any{
		"properties": map[string]any{
			name: stringProperty(description),
		},
		"required": []string{name},
	}
}

func optionalStringProperty(name, description string) map[string]any {
	return map[string]any{
		"properties": map[string]any{
			name: stringProperty(description),
		},
	}
}

func integerProperty(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
	}
}

func stringProperty(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func stringEnumProperty(description string, values ...string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
}

func sortedToolNames(items map[string]ToolDefinition) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

func minSupportedProtocolVersion(requested, supported string) string {
	if strings.TrimSpace(requested) == "" {
		return supported
	}
	// 目前仅实现单一协议版本；对兼容客户端总是返回当前支持版本。
	return supported
}

func newSessionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "storybook-session"
	}
	return hex.EncodeToString(raw[:])
}

func isAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
