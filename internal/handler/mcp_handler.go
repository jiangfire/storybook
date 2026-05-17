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
	svc         *service.MCPService
	db          *gorm.DB
	storyRepo   repository.StoryRepo
	projectRepo repository.ProjectRepo
	storySvc    *service.StoryService
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
	// 默认 StoryService 不携带 events,这样 NewMCPHandler 在测试里仍然零配置;
	// wiring 阶段会通过 WithEvents 重新装配出带 Hub 的版本,保证生产环境的
	// MCP AC 变更能广播到项目订阅者。
	storySvc := service.NewStoryService(db, nil)
	return &MCPHandler{
		svc:         service.NewMCPService(db, "", storySvc),
		db:          db,
		storyRepo:   repository.NewStoryRepository(db),
		projectRepo: repository.NewProjectRepository(db),
		storySvc:    storySvc,
	}
}

// WithEvents 在 wiring 阶段把 EventPublisher(通常是 *realtime.Hub)注入进
// 内部的 StoryService,让 MCP 路径下的 AC 变更也能触发 story.ac_updated 广播。
// 调用顺序与 bug.WithEvents / sprint.WithEvents 等保持一致,避免破坏单测。
func (h *MCPHandler) WithEvents(events service.EventPublisher) *MCPHandler {
	h.storySvc = service.NewStoryService(h.db, events)
	h.svc = service.NewMCPService(h.db, "", h.storySvc)
	return h
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
	// MCP REST 路径已经过 AuthRequired,这里取到的 userID 用于 activity_log 与 VerifiedBy。
	userID := middleware.MustUserID(c)

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

	data, err := h.svc.ValidateAC(story.ID, userID, req.ACID, req.Status, req.Evidence)
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
	userID := middleware.MustUserID(c)

	var req mcpBatchUpdateReq
	if !middleware.BindJSON(c, &req) {
		return
	}
	if len(req.Updates) == 0 {
		api.BadRequest(c, "参数验证失败", api.ErrorItem{Field: "updates", Message: "至少包含一条更新项"})
		return
	}

	data, err := h.svc.BatchUpdateACStatus(story.ID, userID, req.Updates)
	if err != nil {
		h.respondMCPError(c, err, "用户故事不存在")
		return
	}

	api.Success(c, "success", data)
}

func (h *MCPHandler) ACCompletionStats(c *gin.Context) {
	userID := middleware.MustUserID(c)
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
