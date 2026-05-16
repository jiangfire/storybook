package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type BugCommentRepository struct {
	BaseRepository[model.BugComment]
}

func NewBugCommentRepository(db *gorm.DB) *BugCommentRepository {
	return &BugCommentRepository{
		BaseRepository: NewBaseRepository[model.BugComment](db),
	}
}

// ListByBug returns comments oldest-first so the UI can render a chronological
// thread without re-sorting. Author is preloaded for avatar/name rendering.
func (r *BugCommentRepository) ListByBug(bugID uint) ([]model.BugComment, error) {
	var items []model.BugComment
	if err := r.DB().Preload("Author").
		Where("bug_id = ?", bugID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BugCommentRepository) FindByIDWithAuthor(id uint) (*model.BugComment, error) {
	var comment model.BugComment
	if err := r.DB().Preload("Author").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *BugCommentRepository) CountByBug(bugID uint) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.BugComment{}).Where("bug_id = ?", bugID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
