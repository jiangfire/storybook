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

	DB() *gorm.DB
}

// TaskRepo 是 handler 包对 TaskRepository 的最小依赖。
//
// handler 层对 task 的写入大多走 service.TaskService,因此此处方法集很薄;
// techlead / user_management 内的统计仍直接走 DB() 起手的 GORM 查询。
type TaskRepo interface {
	ListByStoryFiltered(storyID uint, status, assignee string) ([]model.Task, error)
	DB() *gorm.DB
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
	HasTechLead(projectID, userID uint) (bool, error)
	AddTechLead(lead *model.ProjectTechLead) error
	RemoveTechLead(projectID, userID uint) (int64, error)
	ListTechLeads(projectID uint) ([]model.ProjectTechLead, error)

	DB() *gorm.DB
}
