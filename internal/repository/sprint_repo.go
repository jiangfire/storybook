package repository

import (
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type SprintRepository struct {
	BaseRepository[model.Sprint]
}

func NewSprintRepository(db *gorm.DB) *SprintRepository {
	return &SprintRepository{
		BaseRepository: NewBaseRepository[model.Sprint](db),
	}
}

func (r *SprintRepository) ListByProject(projectID uint) ([]model.Sprint, error) {
	var sprints []model.Sprint
	if err := r.DB().Where("project_id = ?", projectID).Order("start_date ASC").Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

func (r *SprintRepository) ListByProjectDesc(projectID uint) ([]model.Sprint, error) {
	var sprints []model.Sprint
	if err := r.DB().Where("project_id = ?", projectID).Order("start_date DESC, id DESC").Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

func (r *SprintRepository) UpdateStatus(sprintID uint, status string) error {
	return r.DB().Model(&model.Sprint{}).Where("id = ?", sprintID).Update("status", status).Error
}

func (r *SprintRepository) AssignStory(sprintID uint, storyID uint) error {
	return r.DB().Model(&model.UserStory{}).Where("id = ?", storyID).Update("sprint_id", sprintID).Error
}

// DeleteWithClearStories removes a sprint and detaches all associated stories.
func (r *SprintRepository) DeleteWithClearStories(sprintID uint) error {
	return r.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserStory{}).
			Where("sprint_id = ?", sprintID).
			Update("sprint_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Sprint{}, sprintID).Error
	})
}

// CloseOrCancel updates sprint status and detaches associated stories.
// When excludeDone is true, only stories with status != done are detached.
func (r *SprintRepository) CloseOrCancel(sprintID uint, newStatus string, excludeDone bool) error {
	return r.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Sprint{}).Where("id = ?", sprintID).Update("status", newStatus).Error; err != nil {
			return err
		}
		query := tx.Model(&model.UserStory{}).Where("sprint_id = ?", sprintID)
		if excludeDone {
			query = query.Where("status <> ?", model.StoryStatusDone)
		}
		return query.Update("sprint_id", nil).Error
	})
}
