package router

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/auth"
	"git.neolidy.top/neo/storybook/internal/handler"
	"git.neolidy.top/neo/storybook/internal/metrics"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/realtime"
	"git.neolidy.top/neo/storybook/internal/service"
	"git.neolidy.top/neo/storybook/internal/webui"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func New(db *gorm.DB, tokenManager *auth.TokenManager) *gin.Engine {
	return NewWithLogger(db, tokenManager, slog.Default())
}

func NewWithLogger(db *gorm.DB, tokenManager *auth.TokenManager, logger *slog.Logger) *gin.Engine {
	// 统一启用严格JSON解码，避免未知字段静默吞掉。
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.New()
	r.Use(middleware.RequestLogger(logger), middleware.Recovery(logger), middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// /metrics exposes Prometheus collectors. Skipped entirely when credentials
	// are unset so a misconfigured deployment cannot silently leak histograms
	// over an unauthenticated endpoint.
	if metricsUser, metricsPass := strings.TrimSpace(os.Getenv("METRICS_USER")), strings.TrimSpace(os.Getenv("METRICS_PASS")); metricsUser != "" && metricsPass != "" {
		r.GET("/metrics",
			middleware.BasicAuth(metricsUser, metricsPass, "metrics"),
			gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})),
		)
		logger.Info("metrics endpoint registered", "path", "/metrics")
	} else {
		logger.Info("metrics endpoint disabled: METRICS_USER/METRICS_PASS unset")
	}

	// /readyz performs a real DB ping with a short timeout. Returns 503 when the
	// underlying connection cannot answer in time so orchestrators (k8s,
	// load balancers) can route traffic away from a degraded instance.
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "db handle: " + err.Error()})
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "db ping: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	authHandler := handler.NewAuthHandler(db, tokenManager)
	projectHandler := handler.NewProjectHandler(db)
	hub := realtime.NewHub(db)
	vectorSvc := buildVectorService(db, logger)
	storyHandler := handler.NewStoryHandlerWithVector(db, hub, vectorSvc)
	meHandler := handler.NewMeHandler(db)
	aiHandler := handler.NewAIHandler(db)
	testCaseHandler := handler.NewTestCaseHandler(db)
	taskHandler := handler.NewTaskHandler(db, hub)
	sprintHandler := handler.NewSprintHandler(db)
	bugHandler := handler.NewBugHandler(db)
	bugCommentHandler := handler.NewBugCommentHandler(db)
	reportHandler := handler.NewReportHandler(db)
	searchHandler := handler.NewSearchHandlerWithVector(db, vectorSvc)
	if vectorSvc == nil {
		searchHandler = handler.NewSearchHandler(db)
	}
	mcpHandler := handler.NewMCPHandler(db)
	wsHandler := handler.NewWSHandler(tokenManager, hub)
	techLeadHandler := handler.NewTechLeadHandler(db)
	userManagementHandler := handler.NewUserManagementHandler(db)
	notificationHandler := handler.NewNotificationHandler(db)

	// Wire the in-app notifier into every handler that emits user-facing events.
	// Constructed after the handlers so we can keep their constructor signatures
	// stable (existing tests pass *gorm.DB + Hub only); WithNotifier mutates the
	// already-built instances in place.
	notifier := service.NewNotificationService(db, hub, logger)
	storyHandler.WithNotifier(notifier)
	taskHandler.WithNotifier(notifier)
	sprintHandler.WithNotifier(notifier)
	bugHandler.WithNotifier(notifier)

	// Bug & Sprint emit project-scope WebSocket events (bug.updated/deleted,
	// sprint.closed/cancelled/deleted) — wire the Hub the same way as Notifier
	// so the constructors stay db-only for tests.
	bugHandler.WithEvents(hub)
	sprintHandler.WithEvents(hub)
	bugCommentHandler.WithEvents(hub)

	api := r.Group("/api")

	authLimiter := middleware.NewIPLimiter(3, 1)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authLimiter.Middleware(), authHandler.Register)
		authGroup.POST("/login", authLimiter.Middleware(), authHandler.Login)
		authGroup.POST("/refresh", authLimiter.Middleware(), authHandler.Refresh)
	}

	userLimiter := middleware.NewUserRateLimiter(100)
	aiLimiter := middleware.NewNamedUserRateLimiter("ai", aiUserRateLimitPerMin())
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
		protected.POST("/projects/:id/archive", projectHandler.ArchiveProject)
		protected.POST("/projects/:id/unarchive", projectHandler.UnarchiveProject)
		protected.GET("/projects/:id/export", projectHandler.ExportProject)
		protected.GET("/projects/:id/members", projectHandler.ListMembers)
		protected.GET("/projects/:id/member-candidates", projectHandler.ListMemberCandidates)
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
		protected.GET("/projects/:id/reports/cumulative-flow", reportHandler.CumulativeFlow)
		protected.GET("/projects/:id/reports/cycle-time", reportHandler.CycleTime)
		protected.GET("/projects/:id/reports/lead-time", reportHandler.LeadTime)
		protected.GET("/projects/:id/reports/throughput", reportHandler.Throughput)

		protected.GET("/me/dashboard", meHandler.Dashboard)
		protected.GET("/notifications", notificationHandler.List)
		protected.GET("/notifications/unread-count", notificationHandler.UnreadCount)
		protected.POST("/notifications/:id/read", notificationHandler.MarkRead)
		protected.POST("/notifications/mark-all-read", notificationHandler.MarkAllRead)
		protected.GET("/search", searchHandler.Search)
		protected.GET("/search/capabilities", searchHandler.Capabilities)

		// 语义搜索（向量搜索）
		protected.GET("/search/semantic", searchHandler.SearchSemantic)
		protected.GET("/search/projects", searchHandler.SearchProjectsSemantic)
		protected.POST("/stories/similar", searchHandler.SimilarStories)
		protected.POST("/tags/suggest", searchHandler.SuggestTags)

		protected.GET("/stories/:id", storyHandler.GetStory)
		protected.PUT("/stories/:id", storyHandler.UpdateStory)
		protected.DELETE("/stories/:id", storyHandler.DeleteStory)
		protected.PATCH("/stories/:id/archive", storyHandler.ArchiveStory)
		protected.PATCH("/stories/:id/restore", storyHandler.RestoreStory)
		protected.PATCH("/stories/:id/status", storyHandler.UpdateStatus)
		protected.POST("/stories/:id/claim", storyHandler.ClaimStory)
		protected.DELETE("/stories/:id/claim", storyHandler.ReleaseStory)
		protected.PATCH("/stories/:id/acceptance-criteria/:acID", storyHandler.UpdateACStatus)
		protected.POST("/stories/:id/ac", storyHandler.AddAC)
		protected.PUT("/stories/:id/ac/:acID", storyHandler.UpdateAC)
		protected.DELETE("/stories/:id/ac/:acID", storyHandler.DeleteAC)
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
		protected.PUT("/test-cases/:id", testCaseHandler.Update)
		protected.DELETE("/test-cases/:id", testCaseHandler.Delete)
		protected.GET("/tasks/:id", taskHandler.Get)
		protected.PUT("/tasks/:id", taskHandler.Update)
		protected.DELETE("/tasks/:id", taskHandler.Delete)
		protected.PATCH("/tasks/:id/status", taskHandler.UpdateStatus)
		protected.PATCH("/tasks/:id/progress", taskHandler.UpdateProgress)
		protected.POST("/tasks/:id/claim", taskHandler.Claim)
		protected.DELETE("/tasks/:id/claim", taskHandler.Release)
		protected.POST("/tasks/:id/code-refs", taskHandler.AddCodeReference)
		protected.PATCH("/sprints/:id/status", sprintHandler.UpdateStatus)
		protected.POST("/sprints/:id/close", sprintHandler.Close)
		protected.POST("/sprints/:id/cancel", sprintHandler.Cancel)
		protected.POST("/sprints/:id/reorder", sprintHandler.Reorder)
		protected.DELETE("/sprints/:id", sprintHandler.Delete)
		protected.GET("/bugs/:id", bugHandler.Get)
		protected.PUT("/bugs/:id", bugHandler.Update)
		protected.DELETE("/bugs/:id", bugHandler.Delete)
		protected.PATCH("/bugs/:id/status", bugHandler.UpdateStatus)
		protected.PATCH("/bugs/:id/assign", bugHandler.Assign)
		protected.POST("/bugs/:id/comments", bugCommentHandler.Create)
		protected.GET("/bugs/:id/comments", bugCommentHandler.List)
		protected.PUT("/bugs/:id/comments/:commentID", bugCommentHandler.Update)
		protected.DELETE("/bugs/:id/comments/:commentID", bugCommentHandler.Delete)

		protected.POST("/ai/generate-story", aiLimiter.Middleware(), aiHandler.GenerateStory)
		protected.POST("/ai/stories/:id/split", aiLimiter.Middleware(), aiHandler.SplitStory)
		protected.GET("/ai/stories/:id/invest-check", aiLimiter.Middleware(), aiHandler.INVESTCheck)
		protected.POST("/ai/stories/:id/refine-ac", aiLimiter.Middleware(), aiHandler.RefineAC)
		protected.GET("/ai/stories/:id/summary", aiLimiter.Middleware(), aiHandler.SummarizeStory)
		protected.POST("/ai/stories/:id/translate", aiLimiter.Middleware(), aiHandler.TranslateStory)
		protected.GET("/ai/stories/:id/dor-check", aiLimiter.Middleware(), aiHandler.DoRCheck)

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

func buildVectorService(db *gorm.DB, logger *slog.Logger) service.VectorService {
	provider := strings.TrimSpace(os.Getenv("EMBEDDING_PROVIDER"))
	if provider == "" {
		return nil
	}

	if db == nil || db.Dialector.Name() != "postgres" {
		logger.Warn("vector search disabled: postgres + pgvector is required", "provider", provider)
		return nil
	}

	embeddingSvc, err := service.NewEmbeddingServiceFromConfig(service.EmbeddingConfig{
		Provider:        provider,
		OpenAIKey:       strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OllamaURL:       strings.TrimSpace(os.Getenv("OLLAMA_URL")),
		OllamaModel:     strings.TrimSpace(os.Getenv("OLLAMA_MODEL")),
		OllamaDimension: service.ReadPositiveIntEnv("OLLAMA_DIMENSION"),
	})
	if err != nil {
		logger.Warn("vector search disabled: failed to init embedding service", "provider", provider, "error", err)
		return nil
	}

	if err := service.ValidateStoryEmbeddingDimension(db, embeddingSvc); err != nil {
		logger.Warn("vector search disabled: invalid vector schema", "provider", provider, "error", err, "service_dimension", embeddingSvc.GetDimension())
		return nil
	}

	logger.Info("vector search enabled", "provider", provider, "dimension", embeddingSvc.GetDimension())
	return service.NewVectorService(db, embeddingSvc)
}

// aiUserRateLimitPerMin reads AI_USER_RATE_LIMIT_PER_MIN with a 10 req/min/user
// default. router is the single consumer of this env var (Config does not hold
// it), keeping router.New / NewWithLogger signature unchanged for tests.
func aiUserRateLimitPerMin() int {
	v := strings.TrimSpace(os.Getenv("AI_USER_RATE_LIMIT_PER_MIN"))
	if v == "" {
		return 10
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 10
	}
	return n
}
