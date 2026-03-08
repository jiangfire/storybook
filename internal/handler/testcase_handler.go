package handler

import (
	"encoding/json"
	"errors"
	"strings"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TestCaseHandler struct {
	db *gorm.DB
}

func NewTestCaseHandler(db *gorm.DB) *TestCaseHandler {
	return &TestCaseHandler{db: db}
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

func (h *TestCaseHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可创建测试用例")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.loadStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
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
	if err := h.db.Create(&tc).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	log := model.ActivityLog{
		EntityType: "story",
		EntityID:   story.ID,
		Action:     "test_case_created",
		UserID:     userID,
		ProjectID:  &story.ProjectID,
		NewValue:   model.MarshalJSON(gin.H{"test_case_id": tc.ID, "title": tc.Title}),
	}
	_ = h.db.Create(&log).Error

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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}

	storyID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "故事ID无效")
		return
	}

	story, err := h.loadStoryWithAccess(storyID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	var tcs []model.TestCase
	if err := h.db.Where("story_id = ?", story.ID).Preload("Creator").Order("id DESC").Find(&tcs).Error; err != nil {
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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleTester && role != model.RoleAdmin {
		api.Forbidden(c, "仅测试人员可更新测试用例状态")
		return
	}

	testCaseID, ok := parseUintParam(c, "id")
	if !ok {
		api.BadRequest(c, "测试用例ID无效")
		return
	}

	var req updateTestCaseStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var tc model.TestCase
	if err := h.db.First(&tc, testCaseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "测试用例不存在")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	story, err := h.loadStoryWithAccess(tc.StoryID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			api.NotFound(c, "用户故事不存在")
			return
		}
		if errors.Is(err, errForbidden) {
			api.Forbidden(c, "非项目成员无法访问")
			return
		}
		api.Internal(c, "服务器内部错误")
		return
	}

	oldStatus := tc.Status
	tc.Status = req.Status
	if err := h.db.Save(&tc).Error; err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	log := model.ActivityLog{
		EntityType: "story",
		EntityID:   story.ID,
		Action:     "test_case_status_changed",
		UserID:     userID,
		ProjectID:  &story.ProjectID,
		OldValue:   model.MarshalJSON(gin.H{"test_case_id": tc.ID, "status": oldStatus}),
		NewValue:   model.MarshalJSON(gin.H{"test_case_id": tc.ID, "status": tc.Status}),
	}
	_ = h.db.Create(&log).Error

	api.Success(c, "测试用例状态更新成功", gin.H{
		"id":         tc.ID,
		"status":     tc.Status,
		"updated_at": tc.UpdatedAt,
	})
}

func (h *TestCaseHandler) loadStoryWithAccess(storyID, userID uint) (*model.UserStory, error) {
	var story model.UserStory
	if err := h.db.First(&story, storyID).Error; err != nil {
		return nil, err
	}

	var project model.Project
	if err := h.db.First(&project, story.ProjectID).Error; err != nil {
		return nil, err
	}
	if project.OwnerID == userID {
		return &story, nil
	}

	var count int64
	if err := h.db.Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", project.ID, userID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errForbidden
	}
	return &story, nil
}

func jsonUnmarshalSteps(raw []byte, out *[]string) error {
	if len(raw) == 0 {
		*out = []string{}
		return nil
	}
	return json.Unmarshal(raw, out)
}
