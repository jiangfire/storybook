// Package wiring 将 router 之外的依赖装配集中到一处：
// 接收来自 main / 测试的 cfg/db/logger/tokenManager，组装出所有 handler、
// 共享服务（Hub / Notifier / VectorService）以及它们之间的关联。
//
// 引入它的目的：
//  1. main.go 只关心 cfg → 容器 → 路由的装配，单一入口；
//  2. router.go 不再读取环境变量，专心做 HTTP 路由声明；
//  3. 后续若需要替换某个依赖（mock notifier、关闭 vector），只改本包。
package wiring

import (
	"log/slog"

	"github.com/jiangfire/storybook/internal/auth"
	"github.com/jiangfire/storybook/internal/config"
	"github.com/jiangfire/storybook/internal/handler"
	"github.com/jiangfire/storybook/internal/realtime"
	"github.com/jiangfire/storybook/internal/repository"
	"github.com/jiangfire/storybook/internal/service"
	"gorm.io/gorm"
)

// Container 持有一次启动所需的全部依赖。
// 字段为 exported 以便 router 直接读取。
type Container struct {
	Cfg          *config.Config
	DB           *gorm.DB
	Logger       *slog.Logger
	TokenManager *auth.TokenManager

	Hub      *realtime.Hub
	Notifier service.Notifier
	Vector   service.VectorService

	Auth           *handler.AuthHandler
	Project        *handler.ProjectHandler
	Story          *handler.StoryHandler
	Me             *handler.MeHandler
	AI             *handler.AIHandler
	TestCase       *handler.TestCaseHandler
	Task           *handler.TaskHandler
	Sprint         *handler.SprintHandler
	Bug            *handler.BugHandler
	BugComment     *handler.BugCommentHandler
	Report         *handler.ReportHandler
	Search         *handler.SearchHandler
	MCP            *handler.MCPHandler
	WS             *handler.WSHandler
	TechLead       *handler.TechLeadHandler
	UserManagement *handler.UserManagementHandler
	Notification   *handler.NotificationHandler
}

// Build 按 main.go 的预期构造完整 Container：tokenManager 从外部传入
// 以兼容现存测试（它们用自有 secret 构造 token），而非由 cfg 重建。
// 调用顺序保持与原 router.NewWithLogger 完全一致，避免行为漂移。
func Build(cfg *config.Config, db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager) (*Container, error) {
	c := &Container{
		Cfg:          cfg,
		DB:           db,
		Logger:       logger,
		TokenManager: tokenManager,
	}

	c.Hub = realtime.NewHub(db)
	c.Vector = buildVectorService(cfg, db, logger)

	// Repositories — 供不需要直接操作 db 的 handler 注入使用。
	userRepo := repository.NewUserRepository(db)
	storyRepo := repository.NewStoryRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	sprintRepo := repository.NewSprintRepository(db)
	bugRepo := repository.NewBugRepository(db)
	activityRepo := repository.NewActivityLogRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	// Services — 在 handler 之前构造，避免 handler 持有 db。
	taskSvc := service.NewTaskService(db, c.Hub)

	c.Auth = handler.NewAuthHandler(userRepo, tokenManager)
	c.Project = handler.NewProjectHandler(db)
	c.Story = handler.NewStoryHandlerWithVector(db, c.Hub, c.Vector)
	c.Me = handler.NewMeHandler(userRepo, storyRepo, taskRepo)
	c.AI = handler.NewAIHandler(db)
	c.TestCase = handler.NewTestCaseHandler(db)
	c.Task = handler.NewTaskHandler(taskSvc, taskRepo)
	c.Sprint = handler.NewSprintHandler(db)
	c.Bug = handler.NewBugHandler(db)
	c.BugComment = handler.NewBugCommentHandler(db)
	c.Report = handler.NewReportHandler(storyRepo, sprintRepo, taskRepo, bugRepo, activityRepo)
	// SearchHandler 有两种构造函数：vectorSvc 为 nil 时退化为纯 SQL 搜索，
	// 保留 router 原本的等价分支。
	if c.Vector != nil {
		c.Search = handler.NewSearchHandlerWithVector(db, c.Vector)
	} else {
		c.Search = handler.NewSearchHandler(db)
	}
	c.MCP = handler.NewMCPHandler(db)
	c.WS = handler.NewWSHandler(tokenManager, c.Hub)
	c.TechLead = handler.NewTechLeadHandler(db)
	c.UserManagement = handler.NewUserManagementHandler(db)
	c.Notification = handler.NewNotificationHandler(notificationRepo)

	// 通知/事件需要在 handler 构造之后注入：
	// - Notifier 让 story/task/sprint/bug 在状态变更时下发站内通知；
	// - Hub 让 bug/sprint/bug-comment 推送项目维度的实时事件。
	// 这种"先构造，再注入"的方式保护了 11+ 个仅传 db/hub 的 handler 单测。
	c.Notifier = service.NewNotificationService(db, c.Hub, logger)
	c.Story.WithNotifier(c.Notifier)
	c.Task.WithNotifier(c.Notifier)
	c.Sprint.WithNotifier(c.Notifier)
	c.Bug.WithNotifier(c.Notifier)
	c.Bug.WithEvents(c.Hub)
	c.Sprint.WithEvents(c.Hub)
	c.BugComment.WithEvents(c.Hub)
	// MCP AC 变更复用 StoryService.UpdateACStatus 的活动日志 + 实时事件链路,
	// 这里把 Hub 注进去,让 mcp 调用的 ac 更新也能广播 story.ac_updated。
	c.MCP.WithEvents(c.Hub)

	return c, nil
}

// buildVectorService 在 cfg.EmbeddingProvider 设置且 db 为 postgres 时
// 初始化向量服务。失败仅记日志、返回 nil，让搜索退化为纯 SQL。
// 行为与原 router.buildVectorService 完全等价，只把 os.Getenv 替换为 cfg 字段。
func buildVectorService(cfg *config.Config, db *gorm.DB, logger *slog.Logger) service.VectorService {
	provider := cfg.EmbeddingProvider
	if provider == "" {
		return nil
	}

	if db == nil || db.Dialector.Name() != "postgres" {
		logger.Warn("vector search disabled: postgres + pgvector is required", "provider", provider)
		return nil
	}

	embeddingSvc, err := service.NewEmbeddingServiceFromConfig(service.EmbeddingConfig{
		Provider:        provider,
		OpenAIKey:       cfg.OpenAIAPIKey,
		OllamaURL:       cfg.OllamaURL,
		OllamaModel:     cfg.OllamaModel,
		OllamaDimension: cfg.OllamaDimension,
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
