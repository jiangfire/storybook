package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// BaseRepository 提供通用 CRUD 操作，各具体 repository 可组合使用。
type BaseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) BaseRepository[T] {
	return BaseRepository[T]{db: db}
}

func (r *BaseRepository[T]) FindByID(id uint) (*T, error) {
	var item T
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *BaseRepository[T]) Create(item *T) error {
	return r.db.Create(item).Error
}

func (r *BaseRepository[T]) Save(item *T) error {
	return r.db.Save(item).Error
}

func (r *BaseRepository[T]) Delete(id uint) error {
	return r.db.Delete(new(T), id).Error
}

func (r *BaseRepository[T]) HardDelete(id uint) error {
	return r.db.Unscoped().Delete(new(T), id).Error
}

// UpdateWithVersion 执行乐观锁更新：WHERE id = ? AND version = ?
func (r *BaseRepository[T]) UpdateWithVersion(id uint, version int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	fields["version"] = gorm.Expr("version + 1")
	result := r.db.Model(new(T)).Where("id = ? AND version = ?", id, version).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("version conflict: resource was modified by another request")
	}
	return nil
}

// Paginate 返回分页查询构建器
func (r *BaseRepository[T]) Paginate(page, limit int) *gorm.DB {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	return r.db.Offset(offset).Limit(limit)
}

// ErrNotFound 包装 gorm.ErrRecordNotFound，方便上层判断
var ErrNotFound = gorm.ErrRecordNotFound

// DB 暴露底层 *gorm.DB，供复杂查询使用（应尽量避免在 handler 中直接使用）
func (r *BaseRepository[T]) DB() *gorm.DB {
	return r.db
}

// ScopeProject 按 project_id 过滤的快捷方法（仅适用于有 ProjectID 字段的模型）
func ScopeProject(projectID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("project_id = ?", projectID)
	}
}

// ScopeStatus 按 status 过滤的快捷方法
func ScopeStatus(status string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", status)
	}
}

// ScopeAssignedTo 按 assigned_to 过滤的快捷方法
func ScopeAssignedTo(userID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("assigned_to = ?", userID)
	}
}

// BuildLike 构建 LIKE 查询（两边加 %）
func BuildLike(s string) string {
	return fmt.Sprintf("%%%s%%", s)
}
