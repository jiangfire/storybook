package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	BaseRepository[model.Task]
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{
		BaseRepository: NewBaseRepository[model.Task](db),
	}
}

func (r *TaskRepository) FindByIDWithDetails(taskID uint) (*model.Task, error) {
	var task model.Task
	if err := r.DB().Preload("Assignee").Preload("Creator").Preload("Story").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) ListByStory(storyID uint) ([]model.Task, error) {
	var tasks []model.Task
	if err := r.DB().Where("story_id = ?", storyID).Preload("Assignee").Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListByStoryFiltered returns tasks for a story with optional status / assignee
// filters and the same ordering used by the task list endpoint.
// status / assignee empty string means no filter applied.
func (r *TaskRepository) ListByStoryFiltered(storyID uint, status, assignee string) ([]model.Task, error) {
	tx := r.DB().Model(&model.Task{}).Where("story_id = ?", storyID)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if assignee != "" {
		tx = tx.Where("assigned_to = ?", assignee)
	}
	var tasks []model.Task
	if err := tx.Preload("Assignee").Order("priority DESC, id ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) ListByProject(projectID uint, opts ListOptions) ([]model.Task, int64, error) {
	var total int64
	tx := r.DB().Model(&model.Task{}).Where("project_id = ?", projectID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.AssignedTo != nil {
		tx = tx.Where("assigned_to = ?", *opts.AssignedTo)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []model.Task
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Offset(offset).Limit(opts.Limit).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *TaskRepository) CountByProjectAndStatus(projectID uint, status string) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.Task{}).Where("project_id = ? AND status = ?", projectID, status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *TaskRepository) UpdateStatus(taskID uint, status string) error {
	return r.DB().Model(&model.Task{}).Where("id = ?", taskID).Update("status", status).Error
}

func (r *TaskRepository) UpdateProgress(taskID uint, progress int) error {
	return r.DB().Model(&model.Task{}).Where("id = ?", taskID).Update("progress", progress).Error
}

func (r *TaskRepository) UpdateAssignee(taskID uint, assignedTo *uint) error {
	return r.DB().Model(&model.Task{}).Where("id = ?", taskID).Update("assigned_to", assignedTo).Error
}

// CountTasksByAssignee 统计某经办人的任务数,可叠加状态多选 / 项目范围 / 起始更新时间。
// 空切片表示不过滤,since 零值表示不过滤时间。techlead 工作负载视图 + user_management
// 个人统计共用此方法,避免 handler 手攒一段 GORM 链。
func (r *TaskRepository) CountTasksByAssignee(userID uint, statuses []string, projectIDs []uint, since time.Time) (int64, error) {
	tx := r.DB().Model(&model.Task{}).Where("assigned_to = ?", userID)
	if len(statuses) > 0 {
		tx = tx.Where("status IN ?", statuses)
	}
	if len(projectIDs) > 0 {
		tx = tx.Where("project_id IN ?", projectIDs)
	}
	if !since.IsZero() {
		tx = tx.Where("updated_at >= ?", since)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// SumEstimatedHoursByAssignee 汇总某经办人在指定项目范围内、排除指定状态的任务 estimated_hours。
// projectIDs / excludeStatus 都允许零值表示不过滤,供 techlead 工作负载视图的预估工时汇总使用。
func (r *TaskRepository) SumEstimatedHoursByAssignee(userID uint, excludeStatus string, projectIDs []uint) (float64, error) {
	tx := r.DB().Model(&model.Task{}).Where("assigned_to = ?", userID)
	if excludeStatus != "" {
		tx = tx.Where("status != ?", excludeStatus)
	}
	if len(projectIDs) > 0 {
		tx = tx.Where("project_id IN ?", projectIDs)
	}
	var hours float64
	if err := tx.Select("COALESCE(SUM(estimated_hours), 0)").Scan(&hours).Error; err != nil {
		return 0, err
	}
	return hours, nil
}
