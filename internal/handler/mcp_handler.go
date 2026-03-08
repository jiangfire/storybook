package handler

import (
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/fileutil"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MCPHandler struct {
	db            *gorm.DB
	workspaceRoot string
}

type acCoverageSummary struct {
	Total      int
	Passed     int
	Pending    int
	Failed     int
	Completion float64
}

func NewMCPHandler(db *gorm.DB) *MCPHandler {
	root, _ := os.Getwd()
	return &MCPHandler{
		db:            db,
		workspaceRoot: root,
	}
}

func (h *MCPHandler) Health(c *gin.Context) {
	api.Success(c, "success", gin.H{
		"status":  "ok",
		"version": "1.0.0",
	})
}

func (h *MCPHandler) GetStory(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	story, criteria, err := h.loadStoryAndCriteria(storyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	data := gin.H{
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
		data["story_points"] = *story.Points
	}
	if story.Assignee != nil {
		data["assigned_to"] = gin.H{"id": story.Assignee.ID, "email": story.Assignee.Email}
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) ListStories(c *gin.Context) {
	projectID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "项目ID无效")
		return
	}
	if !h.requireProjectAccess(c, projectID) {
		return
	}

	query := h.db.Model(&model.UserStory{}).Where("project_id = ?", projectID)
	status := strings.TrimSpace(c.Query("status"))
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var stories []model.UserStory
	if err := query.Preload("Assignee").Order("priority DESC, id ASC").Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	result := make([]gin.H, 0, len(stories))
	for _, story := range stories {
		item := gin.H{
			"id":         story.ID,
			"project_id": story.ProjectID,
			"title":      story.Title,
			"story_type": story.StoryType,
			"status":     story.Status,
			"priority":   story.Priority,
			"updated_at": story.UpdatedAt,
			"created_at": story.CreatedAt,
		}
		if story.Assignee != nil {
			item["assigned_to"] = gin.H{"id": story.Assignee.ID, "email": story.Assignee.Email}
		}
		result = append(result, item)
	}

	api.Success(c, "success", gin.H{
		"project_id": projectID,
		"stories":    result,
	})
}

func (h *MCPHandler) GetStoryAC(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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

	cov := calcACCoverage(criteria)

	api.Success(c, "success", gin.H{
		"story_id":            story.ID,
		"acceptance_criteria": criteria,
		"metadata": gin.H{
			"total":                 cov.Total,
			"passed":                cov.Passed,
			"pending":               cov.Pending,
			"failed":                cov.Failed,
			"completion_percentage": cov.Completion,
		},
	})
}

func (h *MCPHandler) ACCoverage(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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

	cov := calcACCoverage(criteria)

	pendingItems := make([]gin.H, 0)
	for _, ac := range criteria {
		if ac.Status == model.ACStatusPending || ac.Status == model.ACStatusFailed {
			pendingItems = append(pendingItems, gin.H{
				"ac_id":       ac.ID,
				"ref":         ac.Ref,
				"description": ac.Description,
				"status":      ac.Status,
				"priority":    inferPriority(ac.Ref),
			})
		}
	}

	api.Success(c, "success", gin.H{
		"story_id": story.ID,
		"coverage": gin.H{
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
			return "建议在提交前完成所有高优先级AC"
		}(),
	})
}

type mcpValidateReq struct {
	ACID     string `json:"ac_id"`
	Status   string `json:"status" binding:"required,oneof=pending passed failed"`
	Evidence string `json:"evidence"`
}

func (h *MCPHandler) Validate(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	var req mcpValidateReq
	if !middleware.BindJSON(c, &req) {
		return
	}

	if req.ACID == "" {
		req.ACID = strings.TrimSpace(c.Param("acID"))
	}
	if strings.TrimSpace(req.ACID) == "" {
		api.BadRequest(c, "ac_id不能为空")
		return
	}

	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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

	found := false
	now := time.Now()
	for i := range criteria {
		if criteria[i].ID == req.ACID {
			criteria[i].Status = req.Status
			criteria[i].Evidence = req.Evidence
			criteria[i].VerifiedAt = &now
			found = true
			break
		}
	}
	if !found {
		api.NotFound(c, "AC不存在")
		return
	}

	story.AcceptanceCriteria = model.MarshalJSON(criteria)
	if err := h.db.Save(&story).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	cov := calcACCoverage(criteria)

	api.Success(c, "AC状态已更新", gin.H{
		"ac_id":       req.ACID,
		"status":      req.Status,
		"verified_at": now,
		"coverage_update": gin.H{
			"total":                 cov.Total,
			"passed":                cov.Passed,
			"pending":               cov.Pending,
			"failed":                cov.Failed,
			"completion_percentage": cov.Completion,
		},
	})
}

type mcpBatchUpdateReq struct {
	Updates []mcpValidateReq `json:"updates" binding:"required"`
}

func (h *MCPHandler) BatchUpdateACStatus(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	var req mcpBatchUpdateReq
	if !middleware.BindJSON(c, &req) {
		return
	}
	if len(req.Updates) == 0 {
		api.BadRequest(c, "参数验证失败", api.ErrorItem{Field: "updates", Message: "至少包含一条更新项"})
		return
	}

	story, criteria, err := h.loadStoryAndCriteria(storyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	updatedCount := 0
	now := time.Now()
	for _, up := range req.Updates {
		acID := strings.TrimSpace(up.ACID)
		if acID == "" {
			continue
		}
		for i := range criteria {
			if criteria[i].ID == acID {
				criteria[i].Status = up.Status
				criteria[i].Evidence = up.Evidence
				criteria[i].VerifiedAt = &now
				updatedCount++
				break
			}
		}
	}

	story.AcceptanceCriteria = model.MarshalJSON(criteria)
	if err := h.db.Save(story).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	cov := calcACCoverage(criteria)

	api.Success(c, "success", gin.H{
		"success":       true,
		"updated_count": updatedCount,
		"new_coverage": gin.H{
			"total":                 cov.Total,
			"passed":                cov.Passed,
			"pending":               cov.Pending,
			"failed":                cov.Failed,
			"completion_percentage": cov.Completion,
		},
	})
}

func (h *MCPHandler) ACCompletionStats(c *gin.Context) {
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	var stories []model.UserStory
	if err := h.db.Find(&stories).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	totalStories := len(stories)
	completedStories := 0
	totalCompletion := 0.0
	pendingHighPriority := 0

	for _, story := range stories {
		criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
		if err != nil {
			continue
		}
		cov := calcACCoverage(criteria)
		totalCompletion += cov.Completion
		if cov.Total > 0 && cov.Pending == 0 && cov.Failed == 0 {
			completedStories++
		}
		for _, ac := range criteria {
			if (ac.Status == model.ACStatusPending || ac.Status == model.ACStatusFailed) && inferPriority(ac.Ref) == "high" {
				pendingHighPriority++
			}
		}
	}

	avgCompletion := 0.0
	if totalStories > 0 {
		avgCompletion = totalCompletion / float64(totalStories)
	}

	api.Success(c, "success", gin.H{
		"total_stories":             totalStories,
		"completed_stories":         completedStories,
		"avg_completion_percentage": avgCompletion,
		"pending_high_priority_acs": pendingHighPriority,
	})
}

func (h *MCPHandler) GenerateACTests(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	_, criteria, err := h.loadStoryAndCriteria(storyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	testCases := make([]gin.H, 0, len(criteria))
	for i, ac := range criteria {
		name := buildTestName(i+1, ac.Description)
		testCases = append(testCases, gin.H{
			"ac_ref":      ac.Ref,
			"test_name":   name,
			"description": ac.Description,
			"code": "func " + name + "(t *testing.T) {\n" +
				"\t// TODO: 根据 AC 补充测试\n" +
				"}",
		})
	}

	api.Success(c, "success", gin.H{
		"story_id":   storyID,
		"test_file":  "internal/story/story_test.go",
		"test_cases": testCases,
	})
}

type mcpAnalyzeReq struct {
	FilePath string `json:"file_path" binding:"required"`
}

func (h *MCPHandler) AnalyzeCodeAC(c *gin.Context) {
	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}
	if !h.requireStoryAccess(c, storyID) {
		return
	}

	var req mcpAnalyzeReq
	if !middleware.BindJSON(c, &req) {
		return
	}

	_, criteria, err := h.loadStoryAndCriteria(storyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	cleanPath, content, err := fileutil.ReadWorkspaceFile(h.workspaceRoot, req.FilePath)
	if err != nil {
		if errors.Is(err, fileutil.ErrPathOutsideWorkspace) || errors.Is(err, fileutil.ErrPathEmpty) {
			api.BadRequest(c, "file_path超出工作区范围")
			return
		}
		api.NotFound(c, "代码文件不存在")
		return
	}

	source := string(content)
	acMapping := make([]gin.H, 0, len(criteria))
	covered := 0
	for _, ac := range criteria {
		matched := false
		matchKey := strings.TrimSpace(ac.Ref)
		if matchKey != "" && strings.Contains(strings.ToLower(source), strings.ToLower(matchKey)) {
			matched = true
		}
		if !matched && strings.TrimSpace(ac.ID) != "" && strings.Contains(strings.ToLower(source), strings.ToLower(ac.ID)) {
			matched = true
		}
		if !matched && strings.TrimSpace(ac.Description) != "" {
			words := tokenize(ac.Description)
			for _, w := range words {
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
		if matched {
			confidence = 0.7
			covered++
		}

		acMapping = append(acMapping, gin.H{
			"ac_ref":      ac.Ref,
			"description": ac.Description,
			"covered":     matched,
			"confidence":  confidence,
			"suggestion": func() string {
				if matched {
					return ""
				}
				return "建议补充与该AC对应的实现或测试"
			}(),
		})
	}

	coverage := 0.0
	if len(criteria) > 0 {
		coverage = float64(covered) / float64(len(criteria)) * 100
	}

	api.Success(c, "success", gin.H{
		"story_id":            storyID,
		"file_path":           cleanPath,
		"ac_mapping":          acMapping,
		"coverage_percentage": coverage,
	})
}

func inferPriority(ref string) string {
	if strings.Contains(strings.ToUpper(ref), "P0") {
		return "high"
	}
	return "medium"
}

func (h *MCPHandler) loadStoryAndCriteria(storyID uint) (*model.UserStory, []model.AcceptanceCriterion, error) {
	var story model.UserStory
	if err := h.db.Preload("Assignee").First(&story, storyID).Error; err != nil {
		return nil, nil, err
	}

	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return nil, nil, err
	}

	sort.SliceStable(criteria, func(i, j int) bool {
		return criteria[i].Order < criteria[j].Order
	})
	return &story, criteria, nil
}

func buildTestName(index int, desc string) string {
	words := tokenize(desc)
	if len(words) == 0 {
		return "TestAC" + strconv.Itoa(index)
	}

	pieces := make([]string, 0, 4)
	for _, w := range words {
		if len(pieces) >= 4 {
			break
		}
		if len(w) == 0 {
			continue
		}
		pieces = append(pieces, strings.ToUpper(w[:1])+strings.ToLower(w[1:]))
	}
	return "TestAC" + strings.Join(pieces, "")
}

func tokenize(text string) []string {
	clean := strings.NewReplacer("，", " ", "。", " ", ",", " ", ".", " ", "：", " ", ":", " ", "（", " ", "）", " ", "(", " ", ")", " ", "\n", " ").Replace(text)
	parts := strings.Fields(clean)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func calcACCoverage(criteria []model.AcceptanceCriterion) acCoverageSummary {
	total, passed, pending, failed := summarizeAC(criteria)
	completion := 0.0
	if total > 0 {
		completion = float64(passed) / float64(total) * 100
	}
	return acCoverageSummary{
		Total:      total,
		Passed:     passed,
		Pending:    pending,
		Failed:     failed,
		Completion: completion,
	}
}

func (h *MCPHandler) requireStoryAccess(c *gin.Context, storyID uint) bool {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return false
	}

	if _, _, _, err := ensureStoryAccess(h.db, storyID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return false
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return false
		}
		api.Internal(c, "服务器内部错误")
		return false
	}

	return true
}

func (h *MCPHandler) requireProjectAccess(c *gin.Context, projectID uint) bool {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return false
	}

	if _, _, err := ensureProjectAccess(h.db, projectID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "项目不存在")
			return false
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return false
		}
		api.Internal(c, "服务器内部错误")
		return false
	}

	return true
}
