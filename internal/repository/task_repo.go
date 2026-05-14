package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	BaseRepository[model.Task]
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{
		BaseRepository: NewBaseRepository[model.Task](db),
		db:             db,
	}
}

func (r *TaskRepository) FindByIDWithDetails(taskID uint) (*model.Task, error) {
	var task model.Task
	if err := r.db.Preload("Assignee").Preload("Creator").Preload("Story").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) ListByStory(storyID uint) ([]model.Task, error) {
	var tasks []model.Task
	if err := r.db.Where("story_id = ?", storyID).Preload("Assignee").Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) ListByProject(projectID uint, opts ListOptions) ([]model.Task, int64, error) {
	var total int64
	tx := r.db.Model(&model.Task{}).Where("project_id = ?", projectID)
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
	if err := r.db.Model(&model.Task{}).Where("project_id = ? AND status = ?", projectID, status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *TaskRepository) UpdateStatus(taskID uint, status string) error {
	return r.db.Model(&model.Task{}).Where("id = ?", taskID).Update("status", status).Error
}

func (r *TaskRepository) UpdateProgress(taskID uint, progress int) error {
	return r.db.Model(&model.Task{}).Where("id = ?", taskID).Update("progress", progress).Error
}

func (r *TaskRepository) UpdateAssignee(taskID uint, assignedTo *uint) error {
	return r.db.Model(&model.Task{}).Where("id = ?", taskID).Update("assigned_to", assignedTo).Error
}
