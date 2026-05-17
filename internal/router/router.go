package router

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/metrics"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/webui"
	"github.com/jiangfire/storybook/internal/wiring"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// New 把 wiring.Container 装配成 Gin engine。
// main.go 与所有测试都通过 wiring.Build 拿到 Container 后再传入本函数，
// router 不再持有任何 env 读取逻辑，专心做 HTTP 路由声明。
func New(c *wiring.Container) *gin.Engine {
	// 统一启用严格JSON解码，避免未知字段静默吞掉。
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.New()
	r.Use(middleware.RequestLogger(c.Logger), middleware.Recovery(c.Logger), middleware.CORS())

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// /metrics exposes Prometheus collectors. Skipped entirely when credentials
	// are unset so a misconfigured deployment cannot silently leak histograms
	// over an unauthenticated endpoint.
	if metricsUser, metricsPass := strings.TrimSpace(c.Cfg.MetricsUser), strings.TrimSpace(c.Cfg.MetricsPass); metricsUser != "" && metricsPass != "" {
		r.GET("/metrics",
			middleware.BasicAuth(metricsUser, metricsPass, "metrics"),
			gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})),
		)
		c.Logger.Info("metrics endpoint registered", "path", "/metrics")
	} else {
		c.Logger.Info("metrics endpoint disabled: METRICS_USER/METRICS_PASS unset")
	}

	// /readyz performs a real DB ping with a short timeout. Returns 503 when the
	// underlying connection cannot answer in time so orchestrators (k8s,
	// load balancers) can route traffic away from a degraded instance.
	r.GET("/readyz", func(ctx *gin.Context) {
		reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()

		sqlDB, err := c.DB.DB()
		if err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "db handle: " + err.Error()})
			return
		}
		if err := sqlDB.PingContext(reqCtx); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "db ping: " + err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api := r.Group("/api")

	authLimiter := middleware.NewIPLimiter(3, 1)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authLimiter.Middleware(), c.Auth.Register)
		authGroup.POST("/login", authLimiter.Middleware(), c.Auth.Login)
		authGroup.POST("/refresh", authLimiter.Middleware(), c.Auth.Refresh)
	}

	userLimiter := middleware.NewUserRateLimiter(100)
	aiRate := c.Cfg.AIUserRateLimitPerMin
	if aiRate <= 0 {
		aiRate = 10
	}
	aiLimiter := middleware.NewNamedUserRateLimiter("ai", aiRate)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired(c.TokenManager))
	protected.Use(userLimiter.Middleware())
	{
		protected.POST("/projects", c.Project.CreateProject)
		protected.GET("/projects", c.Project.ListProjects)
		projectAccess := middleware.RequireProjectAccess(c.DB, "id")

		protected.GET("/projects/:id", projectAccess, c.Project.GetProject)
		protected.PUT("/projects/:id", projectAccess, c.Project.UpdateProject)
		protected.GET("/projects/:id/overview", projectAccess, c.Project.GetOverview)
		protected.DELETE("/projects/:id", projectAccess, c.Project.DeleteProject)
		protected.POST("/projects/:id/archive", projectAccess, c.Project.ArchiveProject)
		protected.POST("/projects/:id/unarchive", projectAccess, c.Project.UnarchiveProject)
		protected.GET("/projects/:id/export", projectAccess, c.Project.ExportProject)
		protected.GET("/projects/:id/members", projectAccess, c.Project.ListMembers)
		protected.GET("/projects/:id/member-candidates", projectAccess, c.Project.ListMemberCandidates)
		protected.POST("/projects/:id/members", projectAccess, c.Project.AddMember)
		protected.DELETE("/projects/:id/members/:userID", projectAccess, c.Project.RemoveMember)

		protected.POST("/projects/:id/stories", projectAccess, c.Story.CreateStory)
		protected.GET("/projects/:id/stories", projectAccess, c.Story.ListStories)
		protected.GET("/projects/:id/board", projectAccess, c.Story.GetBoard)
		protected.POST("/projects/:id/sprints", projectAccess, c.Sprint.Create)
		protected.GET("/projects/:id/sprints", projectAccess, c.Sprint.List)
		protected.POST("/projects/:id/bugs", projectAccess, c.Bug.Create)
		protected.GET("/projects/:id/bugs", projectAccess, c.Bug.List)
		protected.GET("/projects/:id/reports/velocity", projectAccess, c.Report.Velocity)
		protected.GET("/projects/:id/reports/quality", projectAccess, c.Report.Quality)
		protected.GET("/projects/:id/reports/burndown", projectAccess, c.Report.Burndown)
		protected.GET("/projects/:id/reports/cumulative-flow", projectAccess, c.Report.CumulativeFlow)
		protected.GET("/projects/:id/reports/cycle-time", projectAccess, c.Report.CycleTime)
		protected.GET("/projects/:id/reports/lead-time", projectAccess, c.Report.LeadTime)
		protected.GET("/projects/:id/reports/throughput", projectAccess, c.Report.Throughput)

		protected.GET("/me/dashboard", c.Me.Dashboard)
		protected.GET("/notifications", c.Notification.List)
		protected.GET("/notifications/unread-count", c.Notification.UnreadCount)
		protected.POST("/notifications/:id/read", c.Notification.MarkRead)
		protected.POST("/notifications/mark-all-read", c.Notification.MarkAllRead)
		protected.GET("/search", c.Search.Search)
		protected.GET("/search/capabilities", c.Search.Capabilities)

		// 语义搜索（向量搜索）
		protected.GET("/search/semantic", c.Search.SearchSemantic)
		protected.GET("/search/projects", c.Search.SearchProjectsSemantic)
		protected.POST("/stories/similar", c.Search.SimilarStories)
		protected.POST("/tags/suggest", c.Search.SuggestTags)

		storyAccess := middleware.RequireStoryAccess(c.DB, "id")

		protected.GET("/stories/:id", storyAccess, c.Story.GetStory)
		protected.PUT("/stories/:id", storyAccess, c.Story.UpdateStory)
		protected.DELETE("/stories/:id", storyAccess, c.Story.DeleteStory)
		protected.PATCH("/stories/:id/archive", storyAccess, c.Story.ArchiveStory)
		protected.PATCH("/stories/:id/restore", storyAccess, c.Story.RestoreStory)
		protected.PATCH("/stories/:id/status", storyAccess, c.Story.UpdateStatus)
		protected.POST("/stories/:id/claim", storyAccess, c.Story.ClaimStory)
		protected.DELETE("/stories/:id/claim", storyAccess, c.Story.ReleaseStory)
		protected.PATCH("/stories/:id/acceptance-criteria/:acID", storyAccess, c.Story.UpdateACStatus)
		protected.POST("/stories/:id/ac", storyAccess, c.Story.AddAC)
		protected.PUT("/stories/:id/ac/:acID", storyAccess, c.Story.UpdateAC)
		protected.DELETE("/stories/:id/ac/:acID", storyAccess, c.Story.DeleteAC)
		protected.POST("/stories/:id/code-refs", storyAccess, c.Story.AddCodeReference)
		protected.GET("/stories/:id/activities", storyAccess, c.Story.GetActivities)
		protected.PATCH("/stories/:id/assignee", storyAccess, c.Story.AssignStory)
		protected.POST("/stories/:id/review", storyAccess, c.Story.ReviewStory)
		protected.PATCH("/stories/:id/sprint", storyAccess, c.Sprint.AssignStory)
		protected.POST("/stories/:id/tasks", storyAccess, c.Task.Create)
		protected.POST("/stories/:id/tasks/split-from-ac", storyAccess, c.Task.SplitFromAC)
		protected.GET("/stories/:id/tasks", storyAccess, c.Task.ListByStory)
		protected.POST("/stories/:id/test-cases", storyAccess, c.TestCase.Create)
		protected.GET("/stories/:id/test-cases", storyAccess, c.TestCase.ListByStory)

		testCaseAccess := middleware.RequireTestCaseAccess(c.DB, "id")
		taskAccess := middleware.RequireTaskAccess(c.DB, "id")
		sprintAccess := middleware.RequireSprintAccess(c.DB, "id")
		bugAccess := middleware.RequireBugAccess(c.DB, "id")

		protected.PATCH("/test-cases/:id/status", testCaseAccess, c.TestCase.UpdateStatus)
		protected.PUT("/test-cases/:id", testCaseAccess, c.TestCase.Update)
		protected.DELETE("/test-cases/:id", testCaseAccess, c.TestCase.Delete)
		protected.GET("/tasks/:id", taskAccess, c.Task.Get)
		protected.PUT("/tasks/:id", taskAccess, c.Task.Update)
		protected.DELETE("/tasks/:id", taskAccess, c.Task.Delete)
		protected.PATCH("/tasks/:id/status", taskAccess, c.Task.UpdateStatus)
		protected.PATCH("/tasks/:id/progress", taskAccess, c.Task.UpdateProgress)
		protected.POST("/tasks/:id/claim", taskAccess, c.Task.Claim)
		protected.DELETE("/tasks/:id/claim", taskAccess, c.Task.Release)
		protected.POST("/tasks/:id/code-refs", taskAccess, c.Task.AddCodeReference)
		protected.PATCH("/sprints/:id/status", sprintAccess, c.Sprint.UpdateStatus)
		protected.POST("/sprints/:id/close", sprintAccess, c.Sprint.Close)
		protected.POST("/sprints/:id/cancel", sprintAccess, c.Sprint.Cancel)
		protected.POST("/sprints/:id/reorder", sprintAccess, c.Sprint.Reorder)
		protected.DELETE("/sprints/:id", sprintAccess, c.Sprint.Delete)
		protected.GET("/bugs/:id", bugAccess, c.Bug.Get)
		protected.PUT("/bugs/:id", bugAccess, c.Bug.Update)
		protected.DELETE("/bugs/:id", bugAccess, c.Bug.Delete)
		protected.PATCH("/bugs/:id/status", bugAccess, c.Bug.UpdateStatus)
		protected.PATCH("/bugs/:id/assign", bugAccess, c.Bug.Assign)
		protected.POST("/bugs/:id/comments", bugAccess, c.BugComment.Create)
		protected.GET("/bugs/:id/comments", bugAccess, c.BugComment.List)
		protected.PUT("/bugs/:id/comments/:commentID", bugAccess, c.BugComment.Update)
		protected.DELETE("/bugs/:id/comments/:commentID", bugAccess, c.BugComment.Delete)

		protected.POST("/ai/generate-story", aiLimiter.Middleware(), c.AI.GenerateStory)
		protected.POST("/ai/stories/:id/split", aiLimiter.Middleware(), storyAccess, c.AI.SplitStory)
		protected.GET("/ai/stories/:id/invest-check", aiLimiter.Middleware(), storyAccess, c.AI.INVESTCheck)
		protected.POST("/ai/stories/:id/refine-ac", aiLimiter.Middleware(), storyAccess, c.AI.RefineAC)
		protected.GET("/ai/stories/:id/summary", aiLimiter.Middleware(), storyAccess, c.AI.SummarizeStory)
		protected.POST("/ai/stories/:id/translate", aiLimiter.Middleware(), storyAccess, c.AI.TranslateStory)
		protected.GET("/ai/stories/:id/dor-check", aiLimiter.Middleware(), storyAccess, c.AI.DoRCheck)

		// 技术负责人专用接口
		techlead := protected.Group("/techlead")
		techlead.Use(middleware.RequireRoles(model.RoleTechLead, model.RoleAdmin))
		{
			techlead.GET("/pending-stories", c.TechLead.ListPendingStories)
			techlead.GET("/workload", c.TechLead.ListWorkload)
			techlead.GET("/projects", c.TechLead.ListMyProjects)
		}

		// 项目技术负责人管理（仅admin）
		protected.POST("/projects/:id/techleads", projectAccess, c.TechLead.AddTechLead)
		protected.DELETE("/projects/:id/techleads/:userID", projectAccess, c.TechLead.RemoveTechLead)
		protected.GET("/projects/:id/techleads", projectAccess, c.TechLead.ListProjectTechLeads)

		// 管理员配置
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRoles(model.RoleTechLead, model.RoleAdmin))
		{
			admin.GET("/ai/config", c.AI.GetConfig)
			admin.PUT("/ai/config", c.AI.UpsertConfig)
			admin.POST("/ai/config/test", c.AI.TestConfig)
		}

		userAdmin := protected.Group("/admin/users")
		userAdmin.Use(middleware.RequireRoles(model.RoleAdmin))
		{
			userAdmin.GET("", c.UserManagement.ListUsers)
			userAdmin.POST("", c.UserManagement.CreateUser)
			userAdmin.PUT("/:id", c.UserManagement.UpdateUser)
			userAdmin.GET("/:id/workload", c.UserManagement.GetUserWorkload)
			userAdmin.DELETE("/:id", c.UserManagement.DeleteUser)
		}
	}

	r.GET("/ws", c.WS.Connect)

	mcpPublic := r.Group("/mcp")
	{
		mcpPublic.GET("/health", c.MCP.Health)
	}

	mcp := r.Group("/mcp")
	mcp.Use(middleware.AuthRequired(c.TokenManager))
	mcp.Use(userLimiter.Middleware())
	{
		mcpStoryAccess := middleware.RequireStoryAccess(c.DB, "id")
		mcpProjectAccess := middleware.RequireProjectAccess(c.DB, "id")

		mcp.GET("/stories/:id/acceptance-criteria", mcpStoryAccess, c.MCP.GetStoryAC)
		mcp.GET("/stories/:id/ac-coverage", mcpStoryAccess, c.MCP.ACCoverage)
		mcp.POST("/stories/:id/acceptance-criteria/:acID/status", mcpStoryAccess, c.MCP.Validate)
		mcp.POST("/v1/stories/:id/validate", mcpStoryAccess, c.MCP.Validate)
		mcp.GET("/v1/stories/:id", mcpStoryAccess, c.MCP.GetStory)
		mcp.GET("/v1/projects/:id/stories", mcpProjectAccess, c.MCP.ListStories)
		mcp.POST("/v1/stories/:id/update-ac-status", mcpStoryAccess, c.MCP.BatchUpdateACStatus)
		mcp.POST("/v1/stories/:id/generate-ac-tests", mcpStoryAccess, c.MCP.GenerateACTests)
		mcp.POST("/v1/stories/:id/analyze-code-ac", mcpStoryAccess, c.MCP.AnalyzeCodeAC)
		mcp.GET("/v1/stats/ac-completion", c.MCP.ACCompletionStats)
	}

	webui.Register(r)

	return r
}
