package router

import (
	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/handler"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/realtime"
	"git.neolidy.top/neo/storybook/internal/webui"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(db *gorm.DB, tokenManager *auth.TokenManager) *gin.Engine {
	// 统一启用严格JSON解码，避免未知字段静默吞掉。
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	authHandler := handler.NewAuthHandler(db, tokenManager)
	projectHandler := handler.NewProjectHandler(db)
	hub := realtime.NewHub(db)
	storyHandler := handler.NewStoryHandler(db, hub)
	meHandler := handler.NewMeHandler(db)
	aiHandler := handler.NewAIHandler(db)
	testCaseHandler := handler.NewTestCaseHandler(db)
	taskHandler := handler.NewTaskHandler(db, hub)
	sprintHandler := handler.NewSprintHandler(db)
	bugHandler := handler.NewBugHandler(db)
	reportHandler := handler.NewReportHandler(db)
	searchHandler := handler.NewSearchHandler(db)
	mcpHandler := handler.NewMCPHandler(db)
	wsHandler := handler.NewWSHandler(tokenManager, hub)
	techLeadHandler := handler.NewTechLeadHandler(db)
	userManagementHandler := handler.NewUserManagementHandler(db)

	api := r.Group("/api")

	authLimiter := middleware.NewIPLimiter(3, 1)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authLimiter.Middleware(), authHandler.Register)
		authGroup.POST("/login", authLimiter.Middleware(), authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	userLimiter := middleware.NewUserRateLimiter(100)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired(tokenManager))
	protected.Use(userLimiter.Middleware())
	{
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/projects", projectHandler.ListProjects)
		protected.GET("/projects/:id", projectHandler.GetProject)
		protected.PUT("/projects/:id", projectHandler.UpdateProject)
		protected.GET("/projects/:id/overview", projectHandler.GetOverview)
		protected.DELETE("/projects/:id", projectHandler.DeleteProject)
		protected.GET("/projects/:id/members", projectHandler.ListMembers)
		protected.POST("/projects/:id/members", projectHandler.AddMember)
		protected.DELETE("/projects/:id/members/:userID", projectHandler.RemoveMember)

		protected.POST("/projects/:id/stories", storyHandler.CreateStory)
		protected.GET("/projects/:id/stories", storyHandler.ListStories)
		protected.GET("/projects/:id/board", storyHandler.GetBoard)
		protected.POST("/projects/:id/sprints", sprintHandler.Create)
		protected.GET("/projects/:id/sprints", sprintHandler.List)
		protected.POST("/projects/:id/bugs", bugHandler.Create)
		protected.GET("/projects/:id/bugs", bugHandler.List)
		protected.GET("/projects/:id/reports/velocity", reportHandler.Velocity)
		protected.GET("/projects/:id/reports/quality", reportHandler.Quality)
		protected.GET("/projects/:id/reports/burndown", reportHandler.Burndown)

		protected.GET("/me/dashboard", meHandler.Dashboard)
		protected.GET("/search", searchHandler.Search)

		protected.GET("/stories/:id", storyHandler.GetStory)
		protected.PUT("/stories/:id", storyHandler.UpdateStory)
		protected.DELETE("/stories/:id", storyHandler.DeleteStory)
		protected.PATCH("/stories/:id/archive", storyHandler.ArchiveStory)
		protected.PATCH("/stories/:id/restore", storyHandler.RestoreStory)
		protected.PATCH("/stories/:id/status", storyHandler.UpdateStatus)
		protected.POST("/stories/:id/claim", storyHandler.ClaimStory)
		protected.DELETE("/stories/:id/claim", storyHandler.ReleaseStory)
		protected.PATCH("/stories/:id/acceptance-criteria/:acID", storyHandler.UpdateACStatus)
		protected.POST("/stories/:id/code-refs", storyHandler.AddCodeReference)
		protected.GET("/stories/:id/activities", storyHandler.GetActivities)
		protected.PATCH("/stories/:id/assignee", storyHandler.AssignStory)
		protected.POST("/stories/:id/review", storyHandler.ReviewStory)
		protected.PATCH("/stories/:id/sprint", sprintHandler.AssignStory)
		protected.POST("/stories/:id/tasks", taskHandler.Create)
		protected.POST("/stories/:id/tasks/split-from-ac", taskHandler.SplitFromAC)
		protected.GET("/stories/:id/tasks", taskHandler.ListByStory)
		protected.POST("/stories/:id/test-cases", testCaseHandler.Create)
		protected.GET("/stories/:id/test-cases", testCaseHandler.ListByStory)
		protected.PATCH("/test-cases/:id/status", testCaseHandler.UpdateStatus)
		protected.GET("/tasks/:id", taskHandler.Get)
		protected.PUT("/tasks/:id", taskHandler.Update)
		protected.DELETE("/tasks/:id", taskHandler.Delete)
		protected.PATCH("/tasks/:id/status", taskHandler.UpdateStatus)
		protected.PATCH("/tasks/:id/progress", taskHandler.UpdateProgress)
		protected.POST("/tasks/:id/claim", taskHandler.Claim)
		protected.DELETE("/tasks/:id/claim", taskHandler.Release)
		protected.POST("/tasks/:id/code-refs", taskHandler.AddCodeReference)
		protected.PATCH("/sprints/:id/status", sprintHandler.UpdateStatus)
		protected.GET("/bugs/:id", bugHandler.Get)
		protected.PATCH("/bugs/:id/status", bugHandler.UpdateStatus)
		protected.PATCH("/bugs/:id/assign", bugHandler.Assign)

		protected.POST("/ai/generate-story", aiHandler.GenerateStory)
		protected.POST("/ai/stories/:id/split", aiHandler.SplitStory)
		protected.GET("/ai/stories/:id/invest-check", aiHandler.INVESTCheck)

		// 技术负责人专用接口
		techlead := protected.Group("/techlead")
		techlead.Use(middleware.RequireRoles(model.RoleTechLead, model.RoleAdmin))
		{
			techlead.GET("/pending-stories", techLeadHandler.ListPendingStories)
			techlead.GET("/workload", techLeadHandler.ListWorkload)
			techlead.GET("/projects", techLeadHandler.ListMyProjects)
		}

		// 项目技术负责人管理（仅admin）
		protected.POST("/projects/:id/techleads", techLeadHandler.AddTechLead)
		protected.DELETE("/projects/:id/techleads/:userID", techLeadHandler.RemoveTechLead)
		protected.GET("/projects/:id/techleads", techLeadHandler.ListProjectTechLeads)

		// 管理员配置
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRoles(model.RoleTechLead, model.RoleAdmin))
		{
			admin.GET("/ai/config", aiHandler.GetConfig)
			admin.PUT("/ai/config", aiHandler.UpsertConfig)
			admin.POST("/ai/config/test", aiHandler.TestConfig)
		}

		userAdmin := protected.Group("/admin/users")
		userAdmin.Use(middleware.RequireRoles(model.RoleAdmin))
		{
			userAdmin.GET("", userManagementHandler.ListUsers)
			userAdmin.POST("", userManagementHandler.CreateUser)
			userAdmin.PUT("/:id", userManagementHandler.UpdateUser)
			userAdmin.GET("/:id/workload", userManagementHandler.GetUserWorkload)
			userAdmin.DELETE("/:id", userManagementHandler.DeleteUser)
		}
	}

	r.GET("/ws", wsHandler.Connect)

	mcpPublic := r.Group("/mcp")
	{
		mcpPublic.GET("/health", mcpHandler.Health)
	}

	mcp := r.Group("/mcp")
	mcp.Use(middleware.AuthRequired(tokenManager))
	mcp.Use(userLimiter.Middleware())
	{
		mcp.GET("/stories/:id/acceptance-criteria", mcpHandler.GetStoryAC)
		mcp.GET("/stories/:id/ac-coverage", mcpHandler.ACCoverage)
		mcp.POST("/stories/:id/acceptance-criteria/:acID/status", mcpHandler.Validate)
		mcp.POST("/v1/stories/:id/validate", mcpHandler.Validate)
		mcp.GET("/v1/stories/:id", mcpHandler.GetStory)
		mcp.GET("/v1/projects/:id/stories", mcpHandler.ListStories)
		mcp.POST("/v1/stories/:id/update-ac-status", mcpHandler.BatchUpdateACStatus)
		mcp.POST("/v1/stories/:id/generate-ac-tests", mcpHandler.GenerateACTests)
		mcp.POST("/v1/stories/:id/analyze-code-ac", mcpHandler.AnalyzeCodeAC)
		mcp.GET("/v1/stats/ac-completion", mcpHandler.ACCompletionStats)
	}

	webui.Register(r)

	return r
}
