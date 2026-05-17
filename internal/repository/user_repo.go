package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	BaseRepository[model.User]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository[model.User](db),
	}
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.DB().Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	if err := r.DB().Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) ExistsByUsername(username string, excludeID *uint) (bool, error) {
	var count int64
	tx := r.DB().Model(&model.User{}).Where("username = ?", username)
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
	tx := r.DB().Model(&model.User{})
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

// ListFiltered 是 user_management 列表查询的具名版本:支持可选角色 + 模糊搜索 + 分页,
// 返回精简字段(id/email/username/role/avatar/created/last_login)与总数。空 role 表示不过滤。
func (r *UserRepository) ListFiltered(role, search string, page, limit int) ([]model.User, int64, error) {
	tx := r.DB().Model(&model.User{})
	if role != "" {
		tx = tx.Where("role = ?", role)
	}
	if search != "" {
		like := BuildLike(search)
		tx = tx.Where("email LIKE ? OR username LIKE ?", like, like)
	}
	var total int64
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

// ListIDsByRole 返回指定角色的所有用户 ID;techlead 工作负载视图按 admin 视角拉全量开发者列表用。
func (r *UserRepository) ListIDsByRole(role string) ([]uint, error) {
	var ids []uint
	if err := r.DB().Model(&model.User{}).Where("role = ?", role).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *UserRepository) ListAdmins() ([]model.User, error) {
	var users []model.User
	if err := r.DB().Where("role = ?", model.RoleAdmin).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) UpdateLoginState(userID uint, updates map[string]any) error {
	return r.DB().Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *UserRepository) IncrementFailedLoginAttempts(userID uint) (int64, error) {
	if err := r.DB().Model(&model.User{}).Where("id = ?", userID).
		UpdateColumn("failed_login_attempts", gorm.Expr("failed_login_attempts + ?", 1)).Error; err != nil {
		return 0, err
	}
	var fresh model.User
	if err := r.DB().Select("failed_login_attempts").First(&fresh, userID).Error; err != nil {
		return 0, err
	}
	return int64(fresh.FailedLoginAttempts), nil
}

func (r *UserRepository) UpdateLockedUntil(userID uint, until *time.Time) error {
	return r.DB().Model(&model.User{}).Where("id = ?", userID).Update("locked_until", until).Error
}
