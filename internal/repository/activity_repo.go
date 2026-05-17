package repository

import (
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type ActivityLogRepository struct {
	BaseRepository[model.ActivityLog]
}

func NewActivityLogRepository(db *gorm.DB) *ActivityLogRepository {
	return &ActivityLogRepository{
		BaseRepository: NewBaseRepository[model.ActivityLog](db),
	}
}

func (r *ActivityLogRepository) ListByEntity(entityType string, entityID uint, page, limit int) ([]model.ActivityLog, int64, error) {
	var total int64
	tx := r.DB().Model(&model.ActivityLog{}).Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.ActivityLog
	offset := (page - 1) * limit
	if err := tx.Preload("User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (r *ActivityLogRepository) ListByProject(projectID uint, page, limit int) ([]model.ActivityLog, int64, error) {
	var total int64
	tx := r.DB().Model(&model.ActivityLog{}).Where("project_id = ?", projectID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.ActivityLog
	offset := (page - 1) * limit
	if err := tx.Preload("User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ListRecentByProject returns the most recent activity logs for a project
// with user preloading, ordered by created_at DESC.
func (r *ActivityLogRepository) ListRecentByProject(projectID uint, limit int) ([]model.ActivityLog, error) {
	var logs []model.ActivityLog
	if err := r.DB().Where("project_id = ?", projectID).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// ListByEntityFiltered returns activity logs for a specific entity with optional
// action filter, pagination and user preloading.
func (r *ActivityLogRepository) ListByEntityFiltered(entityType string, entityID uint, action string, page, limit int) ([]model.ActivityLog, int64, error) {
	var total int64
	tx := r.DB().Model(&model.ActivityLog{}).Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	if action != "" {
		tx = tx.Where("action = ?", action)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.ActivityLog
	offset := (page - 1) * limit
	if err := tx.Preload("User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
