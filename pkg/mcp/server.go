package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"git.neolidy.top/neo/storybook/internal/fileutil"
	"git.neolidy.top/neo/storybook/internal/service"
	"gorm.io/gorm"
)

type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

type Server struct {
	name    string
	version string
	service *service.MCPService
	tools   map[string]ToolHandler
}

type Request struct {
	ID        string         `json:"id"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type Response struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewServer(db *gorm.DB) *Server {
	root, _ := os.Getwd()
	s := &Server{
		name:    "storybook-mcp",
		version: "1.0.0",
		service: service.NewMCPService(db, root),
		tools:   map[string]ToolHandler{},
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	s.tools["get_story"] = s.handleGetStory
	s.tools["list_stories"] = s.handleListStories
	s.tools["get_acceptance_criteria"] = s.handleGetAcceptanceCriteria
	s.tools["validate_ac"] = s.handleValidateAC
	s.tools["check_ac_coverage"] = s.handleCheckACCoverage
	s.tools["generate_ac_tests"] = s.handleGenerateACTests
	s.tools["analyze_code_ac"] = s.handleAnalyzeCodeAC
	s.tools["update_ac_status"] = s.handleUpdateACStatus
}

func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req Request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		resp := s.handleRequest(ctx, req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

func (s *Server) Start() error {
	return s.ServeStdio(context.Background(), os.Stdin, os.Stdout)
}

func (s *Server) handleRequest(ctx context.Context, req Request) Response {
	handler, ok := s.tools[req.Tool]
	if !ok {
		return Response{ID: req.ID, Success: false, Error: "unknown tool: " + req.Tool}
	}
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}

	result, err := handler(ctx, req.Arguments)
	if err != nil {
		return Response{ID: req.ID, Success: false, Error: err.Error()}
	}
	return Response{ID: req.ID, Success: true, Result: result}
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

	data, err := s.service.ValidateAC(storyID, acID, status, evidence)
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

	return s.service.BatchUpdateACStatus(storyID, updates)
}

func getUintArg(args map[string]any, key string) (uint, error) {
	value, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch typed := value.(type) {
	case float64:
		if typed < 0 || math.Trunc(typed) != typed || typed > float64(^uint(0)) {
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
