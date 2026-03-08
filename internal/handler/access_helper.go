package handler

import (
	"errors"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/service"
	"gorm.io/gorm"
)

func ensureProjectAccess(db *gorm.DB, projectID, userID uint) (*model.Project, bool, error) {
	project, isOwner, err := service.EnsureProjectAccess(db, projectID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return nil, false, errForbidden
		}
		return nil, false, err
	}
	return project, isOwner, nil
}

func ensureStoryAccess(db *gorm.DB, storyID, userID uint) (*model.UserStory, *model.Project, bool, error) {
	story, project, isOwner, err := service.EnsureStoryAccess(db, storyID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return nil, nil, false, errForbidden
		}
		return nil, nil, false, err
	}
	return story, project, isOwner, nil
}
