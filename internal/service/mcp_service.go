package service

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jiangfire/storybook/internal/fileutil"
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type MCPService struct {
	db            *gorm.DB
	workspaceRoot string
	storySvc      *StoryService
}

type MCPAssignedUser struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

type MCPStory struct {
	ID                 uint                        `json:"id"`
	ProjectID          uint                        `json:"project_id"`
	Title              string                      `json:"title"`
	Description        string                      `json:"description"`
	StoryType          string                      `json:"story_type"`
	Status             string                      `json:"status"`
	Priority           int                         `json:"priority"`
	StoryPoints        *int                        `json:"story_points,omitempty"`
	AssignedTo         *MCPAssignedUser            `json:"assigned_to,omitempty"`
	AcceptanceCriteria []model.AcceptanceCriterion `json:"acceptance_criteria"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

type MCPStoryListItem struct {
	ID         uint             `json:"id"`
	ProjectID  uint             `json:"project_id"`
	Title      string           `json:"title"`
	StoryType  string           `json:"story_type"`
	Status     string           `json:"status"`
	Priority   int              `json:"priority"`
	AssignedTo *MCPAssignedUser `json:"assigned_to,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type MCPStoryListResult struct {
	ProjectID uint               `json:"project_id"`
	Stories   []MCPStoryListItem `json:"stories"`
}

type MCPACMetadata struct {
	Total                int     `json:"total"`
	Passed               int     `json:"passed"`
	Pending              int     `json:"pending"`
	Failed               int     `json:"failed"`
	CompletionPercentage float64 `json:"completion_percentage"`
}

type MCPGetStoryACResult struct {
	StoryID            uint                        `json:"story_id"`
	AcceptanceCriteria []model.AcceptanceCriterion `json:"acceptance_criteria"`
	Metadata           MCPACMetadata               `json:"metadata"`
}

type MCPACCoverageBreakdown struct {
	TotalAC              int     `json:"total_ac"`
	PassedAC             int     `json:"passed_ac"`
	PendingAC            int     `json:"pending_ac"`
	FailedAC             int     `json:"failed_ac"`
	CompletionPercentage float64 `json:"completion_percentage"`
	CanComplete          bool    `json:"can_complete"`
}

type MCPPendingAC struct {
	ACID        string `json:"ac_id"`
	Ref         string `json:"ref"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

type MCPACCoverageResult struct {
	StoryID        uint                   `json:"story_id"`
	Coverage       MCPACCoverageBreakdown `json:"coverage"`
	PendingItems   []MCPPendingAC         `json:"pending_items"`
	Recommendation string                 `json:"recommendation"`
}

type MCPValidateACResult struct {
	ACID           string        `json:"ac_id"`
	Status         string        `json:"status"`
	VerifiedAt     time.Time     `json:"verified_at"`
	CoverageUpdate MCPACMetadata `json:"coverage_update"`
}

type MCPACUpdateInput struct {
	ACID     string `json:"ac_id"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type MCPBatchUpdateACStatusResult struct {
	Success      bool          `json:"success"`
	UpdatedCount int           `json:"updated_count"`
	NewCoverage  MCPACMetadata `json:"new_coverage"`
}

type MCPACTestCase struct {
	ACRef       string `json:"ac_ref"`
	TestName    string `json:"test_name"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

type MCPGenerateACTestsResult struct {
	StoryID   uint            `json:"story_id"`
	TestFile  string          `json:"test_file"`
	TestCases []MCPACTestCase `json:"test_cases"`
}

type MCPACMapItem struct {
	ACRef       string  `json:"ac_ref"`
	Description string  `json:"description"`
	Covered     bool    `json:"covered"`
	Confidence  float64 `json:"confidence"`
	Suggestion  string  `json:"suggestion"`
}

type MCPAnalyzeCodeACResult struct {
	StoryID            uint           `json:"story_id"`
	FilePath           string         `json:"file_path"`
	ACMapping          []MCPACMapItem `json:"ac_mapping"`
	CoveragePercentage float64        `json:"coverage_percentage"`
}

type MCPACCompletionStatsResult struct {
	TotalStories            int     `json:"total_stories"`
	CompletedStories        int     `json:"completed_stories"`
	AvgCompletionPercentage float64 `json:"avg_completion_percentage"`
	PendingHighPriorityACs  int     `json:"pending_high_priority_acs"`
}

type mcpCoverageSummary struct {
	total      int
	passed     int
	pending    int
	failed     int
	completion float64
}

func NewMCPService(db *gorm.DB, workspaceRoot string, storySvc *StoryService) *MCPService {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		root, _ = os.Getwd()
	}
	if storySvc == nil {
		storySvc = NewStoryService(db, nil)
	}
	return &MCPService{
		db:            db,
		workspaceRoot: root,
		storySvc:      storySvc,
	}
}

func (s *MCPService) GetStory(storyID uint) (*MCPStory, error) {
	story, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	return &MCPStory{
		ID:                 story.ID,
		ProjectID:          story.ProjectID,
		Title:              story.Title,
		Description:        story.Description,
		StoryType:          story.StoryType,
		Status:             story.Status,
		Priority:           story.Priority,
		StoryPoints:        story.Points,
		AssignedTo:         assignedUser(story.Assignee),
		AcceptanceCriteria: criteria,
		CreatedAt:          story.CreatedAt,
		UpdatedAt:          story.UpdatedAt,
	}, nil
}

func (s *MCPService) ListStories(projectID uint, status string) (*MCPStoryListResult, error) {
	query := s.db.Model(&model.UserStory{}).Where("project_id = ?", projectID)
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}

	var stories []model.UserStory
	if err := query.Preload("Assignee").Order("priority DESC, id ASC").Find(&stories).Error; err != nil {
		return nil, err
	}

	items := make([]MCPStoryListItem, 0, len(stories))
	for _, story := range stories {
		items = append(items, MCPStoryListItem{
			ID:         story.ID,
			ProjectID:  story.ProjectID,
			Title:      story.Title,
			StoryType:  story.StoryType,
			Status:     story.Status,
			Priority:   story.Priority,
			AssignedTo: assignedUser(story.Assignee),
			CreatedAt:  story.CreatedAt,
			UpdatedAt:  story.UpdatedAt,
		})
	}

	return &MCPStoryListResult{
		ProjectID: projectID,
		Stories:   items,
	}, nil
}

func (s *MCPService) GetStoryAC(storyID uint) (*MCPGetStoryACResult, error) {
	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	summary := calcMCPACCoverage(criteria)
	return &MCPGetStoryACResult{
		StoryID:            storyID,
		AcceptanceCriteria: criteria,
		Metadata:           toMCPACMetadata(summary),
	}, nil
}

func (s *MCPService) GetACCoverage(storyID uint) (*MCPACCoverageResult, error) {
	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	summary := calcMCPACCoverage(criteria)
	pendingItems := make([]MCPPendingAC, 0)
	for _, ac := range criteria {
		if ac.Status == model.ACStatusPending || ac.Status == model.ACStatusFailed {
			pendingItems = append(pendingItems, MCPPendingAC{
				ACID:        ac.ID,
				Ref:         ac.Ref,
				Description: ac.Description,
				Status:      ac.Status,
				Priority:    inferMCPPriority(ac.Ref),
			})
		}
	}

	recommendation := "建议在提交前完成所有高优先级的AC"
	if summary.pending == 0 && summary.failed == 0 {
		recommendation = "所有AC已通过，可以提交"
	}

	return &MCPACCoverageResult{
		StoryID: storyID,
		Coverage: MCPACCoverageBreakdown{
			TotalAC:              summary.total,
			PassedAC:             summary.passed,
			PendingAC:            summary.pending,
			FailedAC:             summary.failed,
			CompletionPercentage: summary.completion,
			CanComplete:          summary.pending == 0 && summary.failed == 0,
		},
		PendingItems:   pendingItems,
		Recommendation: recommendation,
	}, nil
}

func (s *MCPService) ValidateAC(storyID, userID uint, acID, status, evidence string) (*MCPValidateACResult, error) {
	if strings.TrimSpace(acID) == "" {
		return nil, ErrACNotFound
	}
	if !isValidMCPACStatus(status) {
		return nil, fmt.Errorf("status is invalid")
	}

	story, _, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	// 走 StoryService.UpdateACStatus 复用 activity_log + Hub 广播 + VerifiedBy/VerifiedAt 链路,
	// 避免 MCP 路径下的 AC 变更绕过审计与实时事件。
	now, err := s.storySvc.UpdateACStatus(story, userID, strings.TrimSpace(acID), status, evidence, "", mcpActor(userID))
	if err != nil {
		return nil, err
	}

	// 重新拉取最新 criteria 以生成 coverage 摘要,确保返回结果对前端是一致快照。
	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}
	summary := calcMCPACCoverage(criteria)
	return &MCPValidateACResult{
		ACID:           strings.TrimSpace(acID),
		Status:         status,
		VerifiedAt:     now,
		CoverageUpdate: toMCPACMetadata(summary),
	}, nil
}

func (s *MCPService) BatchUpdateACStatus(storyID, userID uint, updates []MCPACUpdateInput) (*MCPBatchUpdateACStatusResult, error) {
	story, _, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	updatedCount := 0
	actor := mcpActor(userID)
	for _, update := range updates {
		acID := strings.TrimSpace(update.ACID)
		if acID == "" || !isValidMCPACStatus(update.Status) {
			continue
		}
		// 逐条走 StoryService.UpdateACStatus,每条单独写 activity_log + 推送 Hub,
		// 这样 MCP 批量更新与单条更新在审计链路上保持一致。
		if _, err := s.storySvc.UpdateACStatus(story, userID, acID, update.Status, update.Evidence, "", actor); err != nil {
			return nil, err
		}
		updatedCount++
	}

	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}
	summary := calcMCPACCoverage(criteria)
	return &MCPBatchUpdateACStatusResult{
		Success:      true,
		UpdatedCount: updatedCount,
		NewCoverage:  toMCPACMetadata(summary),
	}, nil
}

func (s *MCPService) ACCompletionStats(projectIDs []uint) (*MCPACCompletionStatsResult, error) {
	var stories []model.UserStory
	query := s.db
	if len(projectIDs) > 0 {
		query = query.Where("project_id IN ?", projectIDs)
	}
	if err := query.Find(&stories).Error; err != nil {
		return nil, err
	}

	totalStories := len(stories)
	completedStories := 0
	totalCompletion := 0.0
	pendingHighPriorityACs := 0

	for _, story := range stories {
		criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
		if err != nil {
			continue
		}

		summary := calcMCPACCoverage(criteria)
		totalCompletion += summary.completion
		if summary.total > 0 && summary.pending == 0 && summary.failed == 0 {
			completedStories++
		}

		for _, ac := range criteria {
			if (ac.Status == model.ACStatusPending || ac.Status == model.ACStatusFailed) && inferMCPPriority(ac.Ref) == "high" {
				pendingHighPriorityACs++
			}
		}
	}

	avgCompletion := 0.0
	if totalStories > 0 {
		avgCompletion = totalCompletion / float64(totalStories)
	}

	return &MCPACCompletionStatsResult{
		TotalStories:            totalStories,
		CompletedStories:        completedStories,
		AvgCompletionPercentage: avgCompletion,
		PendingHighPriorityACs:  pendingHighPriorityACs,
	}, nil
}

func (s *MCPService) GenerateACTests(storyID uint) (*MCPGenerateACTestsResult, error) {
	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	testCases := make([]MCPACTestCase, 0, len(criteria))
	for i, ac := range criteria {
		name := buildMCPTestName(i+1, ac.Description)
		testCases = append(testCases, MCPACTestCase{
			ACRef:       ac.Ref,
			TestName:    name,
			Description: ac.Description,
			Code: "func " + name + "(t *testing.T) {\n" +
				"\t// TODO: 根据 AC 补充测试\n" +
				"}",
		})
	}

	return &MCPGenerateACTestsResult{
		StoryID:   storyID,
		TestFile:  "internal/story/story_test.go",
		TestCases: testCases,
	}, nil
}

func (s *MCPService) AnalyzeCodeAC(storyID uint, filePath string) (*MCPAnalyzeCodeACResult, error) {
	_, criteria, err := s.loadStoryAndCriteria(storyID)
	if err != nil {
		return nil, err
	}

	cleanPath, content, err := fileutil.ReadWorkspaceFile(s.workspaceRoot, filePath)
	if err != nil {
		return nil, err
	}

	source := string(content)
	acMapping := make([]MCPACMapItem, 0, len(criteria))
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
			for _, word := range tokenizeMCPText(ac.Description) {
				if len(word) < 4 {
					continue
				}
				if strings.Contains(strings.ToLower(source), strings.ToLower(word)) {
					matched = true
					break
				}
			}
		}

		item := MCPACMapItem{
			ACRef:       ac.Ref,
			Description: ac.Description,
			Covered:     matched,
			Confidence:  0.0,
			Suggestion:  "建议补充与该AC对应的实现或测试",
		}
		if matched {
			item.Confidence = 0.7
			item.Suggestion = ""
			covered++
		}
		acMapping = append(acMapping, item)
	}

	coveragePercentage := 0.0
	if len(criteria) > 0 {
		coveragePercentage = float64(covered) / float64(len(criteria)) * 100
	}

	return &MCPAnalyzeCodeACResult{
		StoryID:            storyID,
		FilePath:           cleanPath,
		ACMapping:          acMapping,
		CoveragePercentage: coveragePercentage,
	}, nil
}

func (s *MCPService) loadStoryAndCriteria(storyID uint) (*model.UserStory, []model.AcceptanceCriterion, error) {
	var story model.UserStory
	if err := s.db.Preload("Assignee").First(&story, storyID).Error; err != nil {
		return nil, nil, err
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return nil, nil, ErrACCorrupted
	}
	sort.SliceStable(criteria, func(i, j int) bool {
		return criteria[i].Order < criteria[j].Order
	})
	return &story, criteria, nil
}

func assignedUser(user *model.User) *MCPAssignedUser {
	if user == nil {
		return nil
	}
	return &MCPAssignedUser{
		ID:    user.ID,
		Email: user.Email,
	}
}

func calcMCPACCoverage(criteria []model.AcceptanceCriterion) mcpCoverageSummary {
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

	return mcpCoverageSummary{
		total:      total,
		passed:     passed,
		pending:    pending,
		failed:     failed,
		completion: completion,
	}
}

func toMCPACMetadata(summary mcpCoverageSummary) MCPACMetadata {
	return MCPACMetadata{
		Total:                summary.total,
		Passed:               summary.passed,
		Pending:              summary.pending,
		Failed:               summary.failed,
		CompletionPercentage: summary.completion,
	}
}

func inferMCPPriority(ref string) string {
	if strings.Contains(strings.ToUpper(ref), "P0") {
		return "high"
	}
	return "medium"
}

func buildMCPTestName(index int, description string) string {
	words := tokenizeMCPText(description)
	if len(words) == 0 {
		return "TestAC" + strconv.Itoa(index)
	}

	parts := make([]string, 0, 4)
	for _, word := range words {
		if len(parts) >= 4 || len(word) == 0 {
			break
		}
		parts = append(parts, strings.ToUpper(word[:1])+strings.ToLower(word[1:]))
	}
	return "TestAC" + strings.Join(parts, "")
}

func tokenizeMCPText(text string) []string {
	clean := strings.NewReplacer(
		"，", " ",
		"。", " ",
		",", " ",
		".", " ",
		"：", " ",
		":", " ",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"\n", " ",
	).Replace(text)
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

func isValidMCPACStatus(status string) bool {
	switch status {
	case model.ACStatusPending, model.ACStatusPassed, model.ACStatusFailed:
		return true
	default:
		return false
	}
}

// mcpActor 拼出统一的 actor 标记,塞进 StoryService.UpdateACStatus 的 actor 字段:
// - source=mcp 让前端/订阅者能区分这次变更来自 MCP 协议入口而非常规 REST
// - 当 userID==0(无认证身份)时省略 user_id,避免传出 "user 0" 这种脏值
func mcpActor(userID uint) map[string]any {
	actor := map[string]any{
		"source": "mcp",
	}
	if userID != 0 {
		actor["user_id"] = userID
	}
	return actor
}
