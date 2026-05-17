package handler

import (
	"encoding/json"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TestCaseHandler struct {
	db         *gorm.DB
	tcRepo     *repository.TestCaseRepository
	activityRepo repository.ActivityRepo
}

func NewTestCaseHandler(db *gorm.DB) *TestCaseHandler {
	return &TestCaseHandler{
		db:         db,
		tcRepo:     repository.NewTestCaseRepository(db),
		activityRepo: repository.NewActivityLogRepository(db),
	}
}

type createTestCaseRequest struct {
	Title          string   `json:"title" binding:"required,min=2,max=255"`
	Description    string   `json:"description"`
	Steps          []string `json:"steps" binding:"required,min=1"`
	ExpectedResult string   `json:"expected_result"`
}

type updateTestCaseStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending passed failed"`
}

type updateTestCaseRequest struct {
	Title          *string   `json:"title" binding:"omitempty,min=2,max=255"`
	Description    *string   `json:"description" binding:"omitempty,max=5000"`
	Steps          *[]string `json:"steps" binding:"omitempty,min=1,dive,min=1"`
	ExpectedResult *string   `json:"expected_result" binding:"omitempty,max=2000"`
	Version        int       `json:"version"`
}

func (h *TestCaseHandler) Create(c *gin.Context) {
	story := middleware.MustStory(c)
	userID := middleware.MustUserID(c)
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可创建测试用例")
		return
	}

	var req createTestCaseRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	steps := make([]string, 0, len(req.Steps))
	for _, s := range req.Steps {
		s = strings.TrimSpace(s)
		if s != "" {
			steps = append(steps, s)
		}
	}
	if len(steps) == 0 {
		api.BadRequest(c, "参数验证失败", api.ErrorItem{Field: "steps", Message: "steps 至少包含一条有效步骤"})
		return
	}

	tc := model.TestCase{
		StoryID:        story.ID,
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		Steps:          model.MarshalJSON(steps),
		ExpectedResult: strings.TrimSpace(req.ExpectedResult),
		Status:         "pending",
		CreatedBy:      userID,
	}
	if err := h.tcRepo.Create(&tc); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &story.ProjectID, userID, "story", story.ID, "test_case_created",
		nil, map[string]any{"test_case_id": tc.ID, "title": tc.Title}),
		"write testcase activity log", "test_case_id", tc.ID, "action", "test_case_created")

	api.Success(c, "测试用例创建成功", gin.H{
		"id":              tc.ID,
		"story_id":        tc.StoryID,
		"title":           tc.Title,
		"description":     tc.Description,
		"steps":           steps,
		"expected_result": tc.ExpectedResult,
		"status":          tc.Status,
		"created_at":      tc.CreatedAt,
	})
}

func (h *TestCaseHandler) ListByStory(c *gin.Context) {
	story := middleware.MustStory(c)

	tcs, err := h.tcRepo.ListByStory(story.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	items := make([]gin.H, 0, len(tcs))
	for _, tc := range tcs {
		var steps []string
		_ = jsonUnmarshalSteps(tc.Steps, &steps)

		row := gin.H{
			"id":              tc.ID,
			"story_id":        tc.StoryID,
			"title":           tc.Title,
			"description":     tc.Description,
			"steps":           steps,
			"expected_result": tc.ExpectedResult,
			"status":          tc.Status,
			"created_at":      tc.CreatedAt,
			"updated_at":      tc.UpdatedAt,
		}
		if tc.Creator != nil {
			row["created_by"] = gin.H{"id": tc.Creator.ID, "email": tc.Creator.Email}
		}
		items = append(items, row)
	}

	api.Success(c, "success", gin.H{"test_cases": items})
}

func (h *TestCaseHandler) UpdateStatus(c *gin.Context) {
	userID := middleware.MustUserID(c)
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可更新测试用例状态")
		return
	}

	tc := middleware.MustTestCase(c)
	story := middleware.MustStory(c)

	var req updateTestCaseStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	oldStatus := tc.Status
	tc.Status = req.Status
	if err := h.tcRepo.Save(tc); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &story.ProjectID, userID, "story", story.ID, "test_case_status_changed",
		map[string]any{"test_case_id": tc.ID, "status": oldStatus},
		map[string]any{"test_case_id": tc.ID, "status": tc.Status}),
		"write testcase activity log", "test_case_id", tc.ID, "action", "test_case_status_changed")

	api.Success(c, "测试用例状态更新成功", gin.H{
		"id":         tc.ID,
		"status":     tc.Status,
		"updated_at": tc.UpdatedAt,
	})
}

// Update edits a test case's title/description/steps/expected_result with
// optimistic locking. Status changes still go through UpdateStatus so verify
// audit fields stay aligned.
func (h *TestCaseHandler) Update(c *gin.Context) {
	userID := middleware.MustUserID(c)

	tc := middleware.MustTestCase(c)
	story := middleware.MustStory(c)

	role, _ := middleware.CurrentRole(c)
	if tc.CreatedBy != userID && role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试用例创建人或测试人员可编辑")
		return
	}

	var req updateTestCaseRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	fields := map[string]any{}
	oldValue := gin.H{}
	newValue := gin.H{}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			api.BadRequest(c, "title 不能为空")
			return
		}
		oldValue["title"] = tc.Title
		newValue["title"] = trimmed
		fields["title"] = trimmed
	}
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		oldValue["description"] = tc.Description
		newValue["description"] = trimmed
		fields["description"] = trimmed
	}
	if req.Steps != nil {
		steps := make([]string, 0, len(*req.Steps))
		for _, s := range *req.Steps {
			s = strings.TrimSpace(s)
			if s != "" {
				steps = append(steps, s)
			}
		}
		if len(steps) == 0 {
			api.BadRequest(c, "steps 至少包含一条有效步骤")
			return
		}
		stepsJSON := model.MarshalJSON(steps)
		fields["steps"] = stepsJSON
		newValue["steps"] = steps
	}
	if req.ExpectedResult != nil {
		trimmed := strings.TrimSpace(*req.ExpectedResult)
		oldValue["expected_result"] = tc.ExpectedResult
		newValue["expected_result"] = trimmed
		fields["expected_result"] = trimmed
	}
	if len(fields) == 0 {
		api.BadRequest(c, "未提供需要更新的字段")
		return
	}

	if err := h.tcRepo.UpdateWithVersion(tc.ID, req.Version, fields); err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			api.Conflict(c, "测试用例已被他人修改，请刷新后重试")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	updated, err := h.tcRepo.FindByID(tc.ID)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &story.ProjectID, userID, "story", story.ID, "test_case_updated",
		oldValue, newValue),
		"write testcase activity log", "test_case_id", updated.ID, "action", "test_case_updated")

	var steps []string
	_ = jsonUnmarshalSteps(updated.Steps, &steps)
	api.Success(c, "测试用例更新成功", gin.H{
		"id":              updated.ID,
		"story_id":        updated.StoryID,
		"title":           updated.Title,
		"description":     updated.Description,
		"steps":           steps,
		"expected_result": updated.ExpectedResult,
		"status":          updated.Status,
		"version":         updated.Version,
		"updated_at":      updated.UpdatedAt,
	})
}

// Delete soft-deletes a test case. Creator or admin only.
func (h *TestCaseHandler) Delete(c *gin.Context) {
	userID := middleware.MustUserID(c)

	tc := middleware.MustTestCase(c)
	story := middleware.MustStory(c)

	role, _ := middleware.CurrentRole(c)
	if tc.CreatedBy != userID && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试用例创建人或管理员可删除")
		return
	}

	if err := h.tcRepo.Delete(tc.ID); err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	logging.LogIfErr(service.WriteActivityLog(h.db, &story.ProjectID, userID, "story", story.ID, "test_case_deleted",
		map[string]any{"test_case_id": tc.ID, "title": tc.Title, "status": tc.Status}, nil),
		"write testcase activity log", "test_case_id", tc.ID, "action", "test_case_deleted")

	api.Success(c, "测试用例删除成功", gin.H{"id": tc.ID})
}

func jsonUnmarshalSteps(raw []byte, out *[]string) error {
	if len(raw) == 0 {
		*out = []string{}
		return nil
	}
	return json.Unmarshal(raw, out)
}
