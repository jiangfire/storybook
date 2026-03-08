package service

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

func createActivityLog(db *gorm.DB, projectID, userID uint, entityType string, entityID uint, action string, oldValue, newValue map[string]any) error {
	log := model.ActivityLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		ProjectID:  &projectID,
	}
	if oldValue != nil {
		log.OldValue = model.MarshalJSON(oldValue)
	}
	if newValue != nil {
		log.NewValue = model.MarshalJSON(newValue)
	}
	return db.Create(&log).Error
}
