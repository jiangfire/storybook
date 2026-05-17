package repository

import (
	"time"

	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type BugRepository struct {
	BaseRepository[model.BugReport]
}

func NewBugRepository(db *gorm.DB) *BugRepository {
	return &BugRepository{
		BaseRepository: NewBaseRepository[model.BugReport](db),
	}
}

func (r *BugRepository) FindByIDWithDetails(bugID uint) (*model.BugReport, error) {
	var bug model.BugReport
	if err := r.DB().Preload("Reporter").Preload("Assignee").Preload("Story").First(&bug, bugID).Error; err != nil {
		return nil, err
	}
	return &bug, nil
}

func (r *BugRepository) ListByProject(projectID uint, opts BugListOptions) ([]model.BugReport, int64, error) {
	var total int64
	tx := r.DB().Model(&model.BugReport{}).Where("project_id = ?", projectID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.Severity != "" {
		tx = tx.Where("severity = ?", opts.Severity)
	}
	if opts.Assignee != "" {
		tx = tx.Where("assigned_to = ?", opts.Assignee)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var bugs []model.BugReport
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Preload("Reporter").Preload("Assignee").Offset(offset).Limit(opts.Limit).Order("created_at DESC").Find(&bugs).Error; err != nil {
		return nil, 0, err
	}
	return bugs, total, nil
}

// ListByProjectUnpaged returns all matching bugs for a project without
// pagination. Used by endpoints that render a complete inline list.
func (r *BugRepository) ListByProjectUnpaged(projectID uint, opts BugListOptions) ([]model.BugReport, error) {
	tx := r.DB().Model(&model.BugReport{}).Where("project_id = ?", projectID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.Severity != "" {
		tx = tx.Where("severity = ?", opts.Severity)
	}
	if opts.Assignee != "" {
		tx = tx.Where("assigned_to = ?", opts.Assignee)
	}
	var bugs []model.BugReport
	if err := tx.Preload("Reporter").Preload("Assignee").Order("created_at DESC").Find(&bugs).Error; err != nil {
		return nil, err
	}
	return bugs, nil
}

func (r *BugRepository) UpdateStatus(bugID uint, status string) error {
	return r.DB().Model(&model.BugReport{}).Where("id = ?", bugID).Update("status", status).Error
}

func (r *BugRepository) UpdateAssignee(bugID uint, assignedTo *uint) error {
	return r.DB().Model(&model.BugReport{}).Where("id = ?", bugID).Update("assigned_to", assignedTo).Error
}

type BugListOptions struct {
	ListOptions
	Severity string
	Assignee string
}

// BugSearchFilter 是 search_handler 的高级过滤条件。放到 repository 层后,
// handler 不再需要直接持有 *gorm.DB,过滤组合也能在 repository 单元测试覆盖。
type BugSearchFilter struct {
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Statuses    []string
	AssigneeID  *uint
}

// CountByProjectAndStatus 统计某项目下指定状态的缺陷数;status 为空串表示不带状态过滤。
func (r *BugRepository) CountByProjectAndStatus(projectID uint, status string) (int64, error) {
	tx := r.DB().Model(&model.BugReport{}).Where("project_id = ?", projectID)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// CountByProjectAndSeverity 统计某项目下指定严重度的缺陷数;severity 为空串则不过滤。
func (r *BugRepository) CountByProjectAndSeverity(projectID uint, severity string) (int64, error) {
	tx := r.DB().Model(&model.BugReport{}).Where("project_id = ?", projectID)
	if severity != "" {
		tx = tx.Where("severity = ?", severity)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// SearchByProjects 在指定项目集合内做模糊搜索 + 可选过滤(创建时间区间 / 状态多选 / 经办人),
// 按 updated_at 倒序截取前 limit 条返回。专供 search_handler 跨项目检索使用。
func (r *BugRepository) SearchByProjects(projectIDs []uint, like string, limit int, f BugSearchFilter) ([]model.BugReport, error) {
	if len(projectIDs) == 0 {
		return []model.BugReport{}, nil
	}
	tx := r.DB().Model(&model.BugReport{}).
		Where("project_id IN ? AND (title LIKE ? OR description LIKE ?)", projectIDs, like, like)
	if f.CreatedFrom != nil {
		tx = tx.Where("created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		tx = tx.Where("created_at <= ?", *f.CreatedTo)
	}
	if len(f.Statuses) > 0 {
		tx = tx.Where("status IN ?", f.Statuses)
	}
	if f.AssigneeID != nil {
		tx = tx.Where("assigned_to = ?", *f.AssigneeID)
	}
	var rows []model.BugReport
	if err := tx.Order("bug_reports.updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
