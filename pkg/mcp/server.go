package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/fileutil"
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

type Server struct {
	name          string
	version       string
	db            *gorm.DB
	workspaceRoot string
	tools         map[string]ToolHandler
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
		name:          "storybook-mcp",
		version:       "1.0.0",
		db:            db,
		workspaceRoot: root,
		tools:         map[string]ToolHandler{},
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
	h, ok := s.tools[req.Tool]
	if !ok {
		return Response{ID: req.ID, Success: false, Error: "unknown tool: " + req.Tool}
	}
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}

	res, err := h(ctx, req.Arguments)
	if err != nil {
		return Response{ID: req.ID, Success: false, Error: err.Error()}
	}
	return Response{ID: req.ID, Success: true, Result: res}
}

func (s *Server) handleGetStory(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	story, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"id":                  story.ID,
		"project_id":          story.ProjectID,
		"title":               story.Title,
		"description":         story.Description,
		"story_type":          story.StoryType,
		"status":              story.Status,
		"priority":            story.Priority,
		"acceptance_criteria": criteria,
		"created_at":          story.CreatedAt,
		"updated_at":          story.UpdatedAt,
	}
	if story.Points != nil {
		payload["story_points"] = *story.Points
	}
	if story.Assignee != nil {
		payload["assigned_to"] = map[string]any{"id": story.Assignee.ID, "email": story.Assignee.Email}
	}
	return payload, nil
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

	query := s.db.Model(&model.UserStory{}).Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var stories []model.UserStory
	if err := query.Preload("Assignee").Order("priority DESC, id ASC").Find(&stories).Error; err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(stories))
	for _, story := range stories {
		item := map[string]any{
			"id":         story.ID,
			"project_id": story.ProjectID,
			"title":      story.Title,
			"story_type": story.StoryType,
			"status":     story.Status,
			"priority":   story.Priority,
			"created_at": story.CreatedAt,
			"updated_at": story.UpdatedAt,
		}
		if story.Assignee != nil {
			item["assigned_to"] = map[string]any{"id": story.Assignee.ID, "email": story.Assignee.Email}
		}
		out = append(out, item)
	}

	return map[string]any{
		"project_id": projectID,
		"stories":    out,
	}, nil
}

func (s *Server) handleGetAcceptanceCriteria(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}
	cov := coverage(criteria)

	return map[string]any{
		"story_id":            storyID,
		"acceptance_criteria": criteria,
		"metadata": map[string]any{
			"total":                 cov.Total,
			"passed":                cov.Passed,
			"pending":               cov.Pending,
			"failed":                cov.Failed,
			"completion_percentage": cov.Completion,
		},
	}, nil
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
	if !isValidACStatus(status) {
		return nil, fmt.Errorf("status is invalid")
	}
	evidence := strings.TrimSpace(getOptionalString(args, "evidence"))

	story, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	found := false
	now := time.Now()
	for i := range criteria {
		if criteria[i].ID == acID {
			criteria[i].Status = status
			criteria[i].Evidence = evidence
			criteria[i].VerifiedAt = &now
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("ac not found")
	}

	story.AcceptanceCriteria = model.MarshalJSON(criteria)
	if err := s.db.Save(story).Error; err != nil {
		return nil, err
	}

	cov := coverage(criteria)
	return map[string]any{
		"success": true,
		"message": "AC状态已更新",
		"data": map[string]any{
			"ac_id":       acID,
			"status":      status,
			"verified_at": now,
			"coverage_update": map[string]any{
				"total":                 cov.Total,
				"passed":                cov.Passed,
				"pending":               cov.Pending,
				"failed":                cov.Failed,
				"completion_percentage": cov.Completion,
			},
		},
	}, nil
}

func (s *Server) handleCheckACCoverage(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}
	cov := coverage(criteria)

	pendingItems := make([]map[string]any, 0)
	for _, ac := range criteria {
		if ac.Status == model.ACStatusPending || ac.Status == model.ACStatusFailed {
			pendingItems = append(pendingItems, map[string]any{
				"ac_id":       ac.ID,
				"ref":         ac.Ref,
				"description": ac.Description,
				"status":      ac.Status,
				"priority":    inferPriority(ac.Ref),
			})
		}
	}

	return map[string]any{
		"story_id": storyID,
		"coverage": map[string]any{
			"total_ac":              cov.Total,
			"passed_ac":             cov.Passed,
			"pending_ac":            cov.Pending,
			"failed_ac":             cov.Failed,
			"completion_percentage": cov.Completion,
			"can_complete":          cov.Pending == 0 && cov.Failed == 0,
		},
		"pending_items": pendingItems,
		"recommendation": func() string {
			if cov.Pending == 0 && cov.Failed == 0 {
				return "所有AC已通过，可以提交"
			}
			return "建议在提交前完成所有高优先级的AC"
		}(),
	}, nil
}

func (s *Server) handleGenerateACTests(_ context.Context, args map[string]any) (any, error) {
	storyID, err := getUintArg(args, "story_id")
	if err != nil {
		return nil, err
	}

	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	cases := make([]map[string]any, 0, len(criteria))
	for i, ac := range criteria {
		name := buildTestName(i+1, ac.Description)
		cases = append(cases, map[string]any{
			"ac_ref":      ac.Ref,
			"test_name":   name,
			"description": ac.Description,
			"code": "func " + name + "(t *testing.T) {\n" +
				"\t// TODO: 根据 AC 补充测试\n" +
				"}",
		})
	}

	return map[string]any{
		"story_id":   storyID,
		"test_file":  "internal/story/story_test.go",
		"test_cases": cases,
	}, nil
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

	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	cleanPath, content, err := fileutil.ReadWorkspaceFile(s.workspaceRoot, filePath)
	if err != nil {
		if errors.Is(err, fileutil.ErrPathOutsideWorkspace) || errors.Is(err, fileutil.ErrPathEmpty) {
			return nil, fmt.Errorf("file_path outside workspace")
		}
		return nil, fmt.Errorf("code file not found")
	}
	source := string(content)

	acMapping := make([]map[string]any, 0, len(criteria))
	covered := 0
	for _, ac := range criteria {
		matched := false
		if ref := strings.TrimSpace(ac.Ref); ref != "" && strings.Contains(strings.ToLower(source), strings.ToLower(ref)) {
			matched = true
		}
		if !matched && strings.TrimSpace(ac.ID) != "" && strings.Contains(strings.ToLower(source), strings.ToLower(ac.ID)) {
			matched = true
		}
		if !matched && strings.TrimSpace(ac.Description) != "" {
			for _, w := range tokenize(ac.Description) {
				if len(w) < 4 {
					continue
				}
				if strings.Contains(strings.ToLower(source), strings.ToLower(w)) {
					matched = true
					break
				}
			}
		}

		confidence := 0.0
		suggestion := "建议补充与该AC对应的实现或测试"
		if matched {
			confidence = 0.7
			suggestion = ""
			covered++
		}
		acMapping = append(acMapping, map[string]any{
			"ac_ref":      ac.Ref,
			"description": ac.Description,
			"covered":     matched,
			"confidence":  confidence,
			"suggestion":  suggestion,
		})
	}

	coveragePercentage := 0.0
	if len(criteria) > 0 {
		coveragePercentage = float64(covered) / float64(len(criteria)) * 100
	}

	return map[string]any{
		"story_id":            storyID,
		"file_path":           cleanPath,
		"ac_mapping":          acMapping,
		"coverage_percentage": coveragePercentage,
	}, nil
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

	story, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	updatedCount := 0
	now := time.Now()
	for _, up := range updates {
		for i := range criteria {
			if criteria[i].ID == up.ACID {
				criteria[i].Status = up.Status
				criteria[i].Evidence = up.Evidence
				criteria[i].VerifiedAt = &now
				updatedCount++
				break
			}
		}
	}

	story.AcceptanceCriteria = model.MarshalJSON(criteria)
	if err := s.db.Save(story).Error; err != nil {
		return nil, err
	}

	cov := coverage(criteria)
	return map[string]any{
		"success":       true,
		"updated_count": updatedCount,
		"new_coverage": map[string]any{
			"total":                 cov.Total,
			"passed":                cov.Passed,
			"pending":               cov.Pending,
			"failed":                cov.Failed,
			"completion_percentage": cov.Completion,
		},
	}, nil
}

type acUpdate struct {
	ACID     string
	Status   string
	Evidence string
}

type acCoverage struct {
	Total      int
	Passed     int
	Pending    int
	Failed     int
	Completion float64
}

func coverage(criteria []model.AcceptanceCriterion) acCoverage {
	total := len(criteria)
	passed := 0
	pending := 0
	failed := 0
	for _, ac := range criteria {
		switch ac.Status {
		case model.ACStatusPassed:
			passed++
		case model.ACStatusFailed:
			failed++
		default:
			pending++
		}
	}
	completion := 0.0
	if total > 0 {
		completion = float64(passed) / float64(total) * 100
	}
	return acCoverage{Total: total, Passed: passed, Pending: pending, Failed: failed, Completion: completion}
}

func (s *Server) loadStoryAndCriteria(storyID uint) (*model.UserStory, []model.AcceptanceCriterion, error) {
	var story model.UserStory
	if err := s.db.Preload("Assignee").First(&story, storyID).Error; err != nil {
		return nil, nil, err
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return nil, nil, err
	}
	sort.SliceStable(criteria, func(i, j int) bool { return criteria[i].Order < criteria[j].Order })
	return &story, criteria, nil
}

func getUintArg(args map[string]any, key string) (uint, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	switch val := v.(type) {
	case float64:
		return uint(val), nil
	case int:
		return uint(val), nil
	case int64:
		return uint(val), nil
	case uint:
		return val, nil
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s is invalid", key)
		}
		return uint(n), nil
	default:
		return 0, fmt.Errorf("%s is invalid", key)
	}
}

func getOptionalString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func parseUpdates(raw any) ([]acUpdate, error) {
	rows, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("updates is invalid")
	}

	out := make([]acUpdate, 0, len(rows))
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		acID := strings.TrimSpace(getOptionalString(m, "ac_id"))
		status := strings.TrimSpace(getOptionalString(m, "status"))
		if acID == "" || status == "" || !isValidACStatus(status) {
			continue
		}
		out = append(out, acUpdate{
			ACID:     acID,
			Status:   status,
			Evidence: strings.TrimSpace(getOptionalString(m, "evidence")),
		})
	}
	return out, nil
}

func inferPriority(ref string) string {
	if strings.Contains(strings.ToUpper(ref), "P0") {
		return "high"
	}
	return "medium"
}

func buildTestName(index int, desc string) string {
	words := tokenize(desc)
	if len(words) == 0 {
		return "TestAC" + strconv.Itoa(index)
	}

	parts := make([]string, 0, 4)
	for _, w := range words {
		if len(parts) >= 4 || len(w) == 0 {
			break
		}
		parts = append(parts, strings.ToUpper(w[:1])+strings.ToLower(w[1:]))
	}
	return "TestAC" + strings.Join(parts, "")
}

func tokenize(text string) []string {
	clean := strings.NewReplacer("，", " ", "。", " ", ",", " ", ".", " ", "：", " ", ":", " ", "（", " ", "）", " ", "(", " ", ")", " ", "\n", " ").Replace(text)
	items := strings.Fields(clean)
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func isValidACStatus(status string) bool {
	switch status {
	case model.ACStatusPending, model.ACStatusPassed, model.ACStatusFailed:
		return true
	default:
		return false
	}
}
