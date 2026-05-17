package repository

import (
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
