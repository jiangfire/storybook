package service

import (
	"context"

	"git.neolidy.top/neo/storybook/internal/model"
)

// 本文件集中放置 service 包对外暴露的接口契约。
//
// 设计约定:
//   - 接口声明集中到此处,便于 handler / wiring / 测试桩在同一处对账。
//   - 具体实现仍保留在各业务文件(ai_service.go / notification_service.go /
//     vector_service.go 等),只是从中删除了 interface 声明本身。
//   - 与 interface 紧耦合的数据载体(如 NotificationEvent / StoryResult)留在
//     原文件,避免本文件膨胀成"包内类型字典"。

// StoryGenerator covers user-story-flavored AI operations: decomposition,
// streaming, refine, and batch variants. Handlers that only need to produce
// stories should depend on this narrower interface instead of AIService.
type StoryGenerator interface {
	GenerateStory(ctx context.Context, requirement string) (*StoryResult, error)
	StreamGenerateStory(ctx context.Context, requirement string, cb StreamCallback) (*StoryResult, error)
	ChatRefine(ctx context.Context, original *StoryResult, feedback string) (*StoryResult, error)
	BatchGenerate(ctx context.Context, requirement string, count int) ([]*StoryResult, error)
	IsConfigured() bool
}

// ChatCompleter is the free-form chat contract used by §8.6 helpers (AC
// refine, summary, translate). Handlers that only call Chat should depend on
// this rather than the full AIService.
type ChatCompleter interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	IsConfigured() bool
}

// AIService is the aggregate contract preserved for back-compat. NewAIService
// keeps returning it so the construction-time cache and existing callers do
// not need to choose between the two narrower interfaces.
type AIService interface {
	StoryGenerator
	ChatCompleter
}

// Notifier writes notifications into the database and pushes a real-time
// signal to the recipients' WebSocket connections. Implementations should
// never propagate failure to the caller — notifications are best-effort and
// must not abort the originating business transaction.
type Notifier interface {
	Notify(ctx context.Context, userID uint, ev NotificationEvent)
	NotifyMany(ctx context.Context, userIDs []uint, ev NotificationEvent)
	// NotifyProjectMembers fans out to every user with read access to the
	// project (owner, members, tech leads, admins). Used for project-wide
	// announcements such as sprint started / sprint completed.
	NotifyProjectMembers(ctx context.Context, projectID uint, ev NotificationEvent)
}

// VectorService 向量搜索服务接口。
type VectorService interface {
	// SearchSimilarStories 搜索相似故事
	SearchSimilarStories(ctx context.Context, query string, projectIDs []uint, limit int) ([]SimilarStory, error)

	// IndexStory 为单个故事生成并存储向量
	IndexStory(ctx context.Context, story *model.UserStory) error

	// BatchIndexStories 批量为故事生成并存储向量
	BatchIndexStories(ctx context.Context, stories []model.UserStory) error

	// PrepareStoryContent 准备用于向量化的文本内容
	PrepareStoryContent(story *model.UserStory) string
}

// EventPublisher 把 service 层产生的实时事件投递给 realtime.Hub 或等价物。
type EventPublisher interface {
	BroadcastProject(projectID uint, eventType string, data any)
	BroadcastUser(userID uint, eventType string, data any)
}
