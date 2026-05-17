package repository

import (
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type TestCaseRepository struct {
	BaseRepository[model.TestCase]
}

func NewTestCaseRepository(db *gorm.DB) *TestCaseRepository {
	return &TestCaseRepository{
		BaseRepository: NewBaseRepository[model.TestCase](db),
	}
}

func (r *TestCaseRepository) ListByStory(storyID uint) ([]model.TestCase, error) {
	var tcs []model.TestCase
	if err := r.DB().Where("story_id = ?", storyID).Preload("Creator").Order("id DESC").Find(&tcs).Error; err != nil {
		return nil, err
	}
	return tcs, nil
}

func (r *TestCaseRepository) ListByStoryIDs(storyIDs []uint) ([]model.TestCase, error) {
	if len(storyIDs) == 0 {
		return []model.TestCase{}, nil
	}
	var tcs []model.TestCase
	if err := r.DB().Where("story_id IN ?", storyIDs).Find(&tcs).Error; err != nil {
		return nil, err
	}
	return tcs, nil
}

func (r *TestCaseRepository) UpdateStatus(tcID uint, status string) error {
	return r.DB().Model(&model.TestCase{}).Where("id = ?", tcID).Update("status", status).Error
}
