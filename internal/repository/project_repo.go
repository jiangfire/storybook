package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) FindByID(projectID uint) (*model.Project, error) {
	var project model.Project
	if err := r.db.First(&project, projectID).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) IsMember(projectID, userID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
