package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) FindByIDWithDetails(taskID uint) (*model.Task, error) {
	var task model.Task
	if err := r.db.Preload("Assignee").Preload("Creator").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
