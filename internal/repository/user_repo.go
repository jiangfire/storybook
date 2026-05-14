package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	BaseRepository[model.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository[model.User](db),
		db:             db,
	}
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(userID uint) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) ExistsByUsername(username string, excludeID *uint) (bool, error) {
	var count int64
	tx := r.db.Model(&model.User{}).Where("username = ?", username)
	if excludeID != nil {
		tx = tx.Where("id != ?", *excludeID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) List(search string, page, limit int) ([]model.User, int64, error) {
	var total int64
	tx := r.db.Model(&model.User{})
	if search != "" {
		tx = tx.Where("email LIKE ? OR username LIKE ?", BuildLike(search), BuildLike(search))
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	offset := (page - 1) * limit
	if err := tx.Select("id, email, username, role, avatar_url, created_at, last_login_at").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepository) ListAdmins() ([]model.User, error) {
	var users []model.User
	if err := r.db.Where("role = ?", model.RoleAdmin).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) UpdateLoginState(userID uint, updates map[string]any) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepository) IncrementFailedLoginAttempts(userID uint) (int64, error) {
	if err := r.db.Model(&model.User{}).Where("id = ?", userID).
		UpdateColumn("failed_login_attempts", gorm.Expr("failed_login_attempts + ?", 1)).Error; err != nil {
		return 0, err
	}
	var fresh model.User
	if err := r.db.Select("failed_login_attempts").First(&fresh, userID).Error; err != nil {
		return 0, err
	}
	return int64(fresh.FailedLoginAttempts), nil
}

func (r *UserRepository) UpdateLockedUntil(userID uint, until *time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("locked_until", until).Error
}
