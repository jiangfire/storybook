package handler

import (
	"errors"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/fileutil"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"git.neolidy.top/neo/storybook/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MCPHandler struct {
	svc       *service.MCPService
	db        *gorm.DB
	storyRepo *repository.StoryRepository
	projectRepo *repository.ProjectRepository
}

type mcpValidateReq struct {
	ACID     string `json:"ac_id"`
	Status   string `json:"status" binding:"required,oneof=pending passed failed"`
	Evidence string `json:"evidence"`
}

type mcpBatchUpdateReq struct {
	Updates []service.MCPACUpdateInput `json:"updates" binding:"required"`
}

type mcpAnalyzeReq struct {
	FilePath string `json:"file_path" binding:"required"`
}

func NewMCPHandler(db *gorm.DB) *MCPHandler {
	return &MCPHandler{
		svc:       service.NewMCPService(db, ""),
		db:        db,
		storyRepo: repository.NewStoryRepository(db),
		projectRepo: repository.NewProjectRepository(db),
	}
}

func (h *MCPHandler) Health(c *gin.Context) {
	api.Success(c, "success", gin.H{
		"status":  "ok",
		"version": "1.0.0",
	})
}

func (h *MCPHandler) GetStory(c *gin.Context) {
	story := middleware.MustStory(c)

	data, err := h.svc.GetStory(story.ID)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) ListStories(c *gin.Context) {
	project := middleware.MustProject(c)

	data, err := h.svc.ListStories(project.ID, c.Query("status"))
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) GetStoryAC(c *gin.Context) {
	story := middleware.MustStory(c)

	data, err := h.svc.GetStoryAC(story.ID)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) ACCoverage(c *gin.Context) {
	story := middleware.MustStory(c)

	data, err := h.svc.GetACCoverage(story.ID)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) Validate(c *gin.Context) {
	story := middleware.MustStory(c)

	var req mcpValidateReq
	if !middleware.BindJSON(c, &req) {
		return
	}

	if req.ACID == "" {
		req.ACID = c.Param("acID")
	}
	if req.ACID == "" {
		api.BadRequest(c, "ac_id不能为空")
		return
	}

	data, err := h.svc.ValidateAC(story.ID, req.ACID, req.Status, req.Evidence)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrACNotFound):
			api.NotFound(c, "AC不存在")
		case errors.Is(err, service.ErrACCorrupted):
			api.Internal(c, "AC数据损坏")
		case errors.Is(err, gorm.ErrRecordNotFound):
			api.NotFound(c, "用户故事不存在")
		default:
			api.Internal(c, "服务器内部错误")
		}
		return
	}

	api.Success(c, "AC状态已更新", data)
}

func (h *MCPHandler) BatchUpdateACStatus(c *gin.Context) {
	story := middleware.MustStory(c)

	var req mcpBatchUpdateReq
	if !middleware.BindJSON(c, &req) {
		return
	}
	if len(req.Updates) == 0 {
		api.BadRequest(c, "参数验证失败", api.ErrorItem{Field: "updates", Message: "至少包含一条更新项"})
		return
	}

	data, err := h.svc.BatchUpdateACStatus(story.ID, req.Updates)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) ACCompletionStats(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		api.Unauthorized(c, "未登录")
		return
	}
	role, _ := middleware.CurrentRole(c)
	if role != model.RoleProduct && role != model.RoleAdmin {
		api.Forbidden(c, "权限不足")
		return
	}

	projectIDs, err := service.AccessibleProjectIDs(h.db, userID, role)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	data, err := h.svc.ACCompletionStats(projectIDs)
	if err != nil {
		api.Internal(c, "服务器内部错误")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) GenerateACTests(c *gin.Context) {
	story := middleware.MustStory(c)

	data, err := h.svc.GenerateACTests(story.ID)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) AnalyzeCodeAC(c *gin.Context) {
	story := middleware.MustStory(c)

	var req mcpAnalyzeReq
	if !middleware.BindJSON(c, &req) {
		return
	}

	data, err := h.svc.AnalyzeCodeAC(story.ID, req.FilePath)
	if err != nil {
		switch {
		case errors.Is(err, fileutil.ErrPathOutsideWorkspace), errors.Is(err, fileutil.ErrPathEmpty):
			api.BadRequest(c, "file_path超出工作区范围")
		case errors.Is(err, service.ErrACCorrupted):
			api.Internal(c, "AC数据损坏")
		case errors.Is(err, gorm.ErrRecordNotFound):
			api.NotFound(c, "用户故事不存在")
		default:
			api.NotFound(c, "代码文件不存在")
		}
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) respondMCPError(c *gin.Context, err error, notFoundMessage string) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		api.NotFound(c, notFoundMessage)
	case errors.Is(err, service.ErrACCorrupted):
		api.Internal(c, "AC数据损坏")
	default:
		api.Internal(c, "服务器内部错误")
	}
}
