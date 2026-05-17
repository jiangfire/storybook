package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

// 本文件集中放置 handler 包依赖的 repository 接口契约。
//
// 设计约定:
//   - 接口名称使用 `XxxRepo` 后缀,避免与具体 struct(`XxxRepository`)重名。
//   - 接口方法集 = 当前 handler 调用方法的并集(含从 BaseRepository[T] 继承的方法),
//     不要把整个 BaseRepository 都搬进来,避免接口膨胀。
//   - 部分接口含 `DB() *gorm.DB`,这是过渡态,允许 report/search 等使用复杂查询的
//     handler 继续直接走 GORM。后续应当把这些复杂查询下沉到 repository 的具名方法,
//     届时即可从接口移除 `DB()`。

// ActivityRepo 是 handler 包对 ActivityLogRepository 的最小依赖。
type ActivityRepo interface {
	ListRecentByProject(projectID uint, limit int) ([]model.ActivityLog, error)
	ListByEntityFiltered(entityType string, entityID uint, action string, page, limit int) ([]model.ActivityLog, int64, error)
	DB() *gorm.DB
}

// UserRepo 是 handler 包对 UserRepository 的最小依赖。
type UserRepo interface {
	// 来自 BaseRepository[model.User]
	FindByID(id uint) (*model.User, error)
	Create(item *model.User) error
	Save(item *model.User) error
	HardDelete(id uint) error

	// user_repo.go 自有方法
	FindByEmail(email string) (*model.User, error)
	ExistsByEmail(email string) (bool, error)
	ExistsByUsername(username string, excludeID *uint) (bool, error)
	UpdateLoginState(userID uint, updates map[string]any) error
	IncrementFailedLoginAttempts(userID uint) (int64, error)
	UpdateLockedUntil(userID uint, until *time.Time) error
	ListFiltered(role, search string, page, limit int) ([]model.User, int64, error)
	ListIDsByRole(role string) ([]uint, error)
}

// TaskRepo 是 handler 包对 TaskRepository 的最小依赖。
//
// handler 层对 task 的写入大多走 service.TaskService;读取和统计经 P1.2 下沉具名
// 方法后,此处不再暴露 DB() 逃逸口。
type TaskRepo interface {
	ListByStoryFiltered(storyID uint, status, assignee string) ([]model.Task, error)
	CountTasksByAssignee(userID uint, statuses []string, projectIDs []uint, since time.Time) (int64, error)
	SumEstimatedHoursByAssignee(userID uint, excludeStatus string, projectIDs []uint) (float64, error)
}

// ProjectRepo 是 handler 包对 ProjectRepository 的最小依赖。
//
// 覆盖项目本体、成员关系、技术负责人三类调用面;Search/User-management 的
// 复杂 join 仍走 DB() 起手。
type ProjectRepo interface {
	// 来自 BaseRepository[model.Project]
	FindByID(id uint) (*model.Project, error)
	Save(item *model.Project) error
	Delete(id uint) error

	// project_repo.go 自有方法
	ExistsByOwnerAndName(ownerID uint, name string, excludeID ...uint) (bool, error)
	CreateWithTransaction(project *model.Project, member *model.ProjectMember, columns []model.BoardColumn) error
	CountMembers(projectID uint) (int64, error)
	ListMembers(projectID uint) ([]model.ProjectMember, error)
	IsMember(projectID, userID uint) (bool, error)
	AddMember(member *model.ProjectMember) error
	RemoveMember(projectID, userID uint) (int64, error)
	GetMember(projectID, userID uint) (*model.ProjectMember, error)
	ListMemberCandidates(projectID, ownerID uint) ([]model.User, error)
	HasTechLead(projectID, userID uint) (bool, error)
	AddTechLead(lead *model.ProjectTechLead) error
	RemoveTechLead(projectID, userID uint) (int64, error)
	ListTechLeads(projectID uint) ([]model.ProjectTechLead, error)

	DB() *gorm.DB
}

// StoryRepo 是 handler 包对 StoryRepository 的最小依赖。
//
// 覆盖 bug/me/sprint/story/report/user_management/story_assignment 调用面;
// 大量 list 视图聚合 / report / techlead 报表仍走 DB() 起手的 GORM 查询。
type StoryRepo interface {
	// 来自 BaseRepository[model.UserStory]
	FindByID(id uint) (*model.UserStory, error)
	Save(item *model.UserStory) error

	// story_repo.go 自有方法
	ListByAssignee(userID uint, limit int) ([]model.UserStory, error)
	ListByCreator(userID uint, limit int) ([]model.UserStory, error)
	CountByAssignee(userID uint) (int64, error)
	CountByAssigneeAndStatus(userID uint, status string) (int64, error)
	CountBySprint(sprintID uint) (int64, error)
	CountBySprintAndStatus(sprintID uint, status string) (int64, error)
	CountBySprintAndIDs(sprintID uint, storyIDs []uint) (int64, error)
	UpdatePositionsBatch(sprintID uint, positions map[uint]float64) error
	ListBoardByProjectWithAssignee(projectID uint) ([]model.UserStory, error)
	ListBySprint(sprintID uint) ([]model.UserStory, error)
	AvgCompletionDaysForUser(userID uint, since time.Time) (float64, error)

	DB() *gorm.DB
}

// BugRepo 是 handler 包对 BugRepository 的最小依赖。
//
// 覆盖 bug / bug_comment / report / search 调用面;P1.2 将 report 的状态/严重度
// 计数与 search 的高级过滤下沉成具名方法后,此处不再暴露 DB() 逃逸口。
type BugRepo interface {
	// 来自 BaseRepository[model.BugReport]
	FindByID(id uint) (*model.BugReport, error)
	Create(item *model.BugReport) error
	Save(item *model.BugReport) error
	Delete(id uint) error
	UpdateWithVersion(id uint, version int, fields map[string]any) error

	// bug_repo.go 自有方法
	FindByIDWithDetails(bugID uint) (*model.BugReport, error)
	ListByProjectUnpaged(projectID uint, opts BugListOptions) ([]model.BugReport, error)
	CountByProjectAndStatus(projectID uint, status string) (int64, error)
	CountByProjectAndSeverity(projectID uint, severity string) (int64, error)
	SearchByProjects(projectIDs []uint, like string, limit int, f BugSearchFilter) ([]model.BugReport, error)
}

// SprintRepo 是 handler 包对 SprintRepository 的最小依赖。
//
// 覆盖 sprint / report 调用面;无 handler 走 DB() 起手,故不含 DB()。
type SprintRepo interface {
	// 来自 BaseRepository[model.Sprint]
	FindByID(id uint) (*model.Sprint, error)
	Create(item *model.Sprint) error
	Save(item *model.Sprint) error

	// sprint_repo.go 自有方法
	ListByProject(projectID uint) ([]model.Sprint, error)
	ListByProjectDesc(projectID uint) ([]model.Sprint, error)
	DeleteWithClearStories(sprintID uint) error
	CloseOrCancel(sprintID uint, newStatus string, excludeDone bool) error
}

// BugCommentRepo 是 handler 包对 BugCommentRepository 的最小依赖。
//
// 仅 bug_comment_handler 使用,接口化主要为方便测试桩与未来 cache 注入。
type BugCommentRepo interface {
	// 来自 BaseRepository[model.BugComment]
	FindByID(id uint) (*model.BugComment, error)
	Create(item *model.BugComment) error
	Delete(id uint) error
	UpdateWithVersion(id uint, version int, fields map[string]any) error

	// bug_comment_repo.go 自有方法
	FindByIDWithAuthor(id uint) (*model.BugComment, error)
	ListByBug(bugID uint) ([]model.BugComment, error)
}

// AIConfigRepo 是 handler 包对 AIConfigRepository 的最小依赖。
//
// ai_handler 通过此接口读写 AI 配置,后续 InvalidateAIServiceCache 失效
// 注入也走该接口,便于 service 层共享同一抽象。
type AIConfigRepo interface {
	// 来自 BaseRepository[model.AIConfig]
	Save(item *model.AIConfig) error

	// ai_config_repo.go 自有方法
	FindLatestEnabled() (*model.AIConfig, error)
}
