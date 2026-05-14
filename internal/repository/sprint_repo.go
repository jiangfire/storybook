package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type SprintRepository struct {
	BaseRepository[model.Sprint]
	db *gorm.DB
}

func NewSprintRepository(db *gorm.DB) *SprintRepository {
	return &SprintRepository{
		BaseRepository: NewBaseRepository[model.Sprint](db),
		db:             db,
	}
}

func (r *SprintRepository) FindByID(sprintID uint) (*model.Sprint, error) {
	var sprint model.Sprint
	if err := r.db.First(&sprint, sprintID).Error; err != nil {
		return nil, err
	}
	return &sprint, nil
}

func (r *SprintRepository) ListByProject(projectID uint) ([]model.Sprint, error) {
	var sprints []model.Sprint
	if err := r.db.Where("project_id = ?", projectID).Order("start_date ASC").Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

func (r *SprintRepository) ListByProjectDesc(projectID uint) ([]model.Sprint, error) {
	var sprints []model.Sprint
	if err := r.db.Where("project_id = ?", projectID).Order("start_date DESC, id DESC").Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

func (r *SprintRepository) UpdateStatus(sprintID uint, status string) error {
	return r.db.Model(&model.Sprint{}).Where("id = ?", sprintID).Update("status", status).Error
}

func (r *SprintRepository) AssignStory(sprintID uint, storyID uint) error {
	return r.db.Model(&model.UserStory{}).Where("id = ?", storyID).Update("sprint_id", sprintID).Error
}
