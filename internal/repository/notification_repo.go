package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	BaseRepository[model.Notification]
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{
		BaseRepository: NewBaseRepository[model.Notification](db),
		db:             db,
	}
}

// BulkCreate inserts a batch of notifications in a single statement so events
// targeting many recipients (e.g. sprint started) don't hammer the DB with N
// round-trips.
func (r *NotificationRepository) BulkCreate(items []model.Notification) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *NotificationRepository) ListByUser(userID uint, unreadOnly bool, page, limit int) ([]model.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tx := r.db.Model(&model.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		tx = tx.Where("read_at IS NULL")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Notification
	offset := (page - 1) * limit
	if err := tx.Preload("Actor").Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *NotificationRepository) UnreadCount(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// MarkRead sets read_at to now if-and-only-if the notification belongs to the
// caller and is still unread. The userID guard prevents a user from marking
// another user's notification.
func (r *NotificationRepository) MarkRead(id, userID uint) (bool, error) {
	now := time.Now()
	result := r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", now)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *NotificationRepository) MarkAllRead(userID uint) (int64, error) {
	now := time.Now()
	result := r.db.Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
