package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type StoryRepository struct {
	db *gorm.DB
}

func NewStoryRepository(db *gorm.DB) *StoryRepository {
	return &StoryRepository{db: db}
}

func (r *StoryRepository) FindByID(storyID uint) (*model.UserStory, error) {
	var story model.UserStory
	if err := r.db.First(&story, storyID).Error; err != nil {
		return nil, err
	}
	return &story, nil
}

func (r *StoryRepository) FindByIDWithDetails(storyID uint) (*model.UserStory, error) {
	var story model.UserStory
	if err := r.db.
		Preload("Project").
		Preload("Assignee").
		Preload("Creator").
		First(&story, storyID).Error; err != nil {
		return nil, err
	}
	return &story, nil
}
