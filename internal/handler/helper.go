package handler

import (
	"strconv"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := c.Param(key)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(value), true
}

func parseUintQuery(c *gin.Context, key string) (uint, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(value), true
}

func parseUint(s string) (uint, bool) {
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(value), true
}

func createActivityLog(db *gorm.DB, projectID *uint, userID uint, entityType string, entityID uint, action string, oldValue, newValue map[string]any) error {
	log := model.ActivityLog{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		ProjectID:  projectID,
	}
	if oldValue != nil {
		log.OldValue = model.MarshalJSON(oldValue)
	}
	if newValue != nil {
		log.NewValue = model.MarshalJSON(newValue)
	}
	return db.Create(&log).Error
}
