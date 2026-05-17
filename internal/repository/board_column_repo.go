package repository

import (
	"github.com/jiangfire/storybook/internal/model"
	"gorm.io/gorm"
)

type BoardColumnRepository struct {
	BaseRepository[model.BoardColumn]
}

func NewBoardColumnRepository(db *gorm.DB) *BoardColumnRepository {
	return &BoardColumnRepository{
		BaseRepository: NewBaseRepository[model.BoardColumn](db),
	}
}

func (r *BoardColumnRepository) ListByProject(projectID uint) ([]model.BoardColumn, error) {
	var columns []model.BoardColumn
	if err := r.DB().Where("project_id = ?", projectID).Order("position ASC").Find(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}
