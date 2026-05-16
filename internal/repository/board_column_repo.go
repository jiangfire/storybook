package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type BoardColumnRepository struct {
	BaseRepository[model.BoardColumn]
	db *gorm.DB
}

func NewBoardColumnRepository(db *gorm.DB) *BoardColumnRepository {
	return &BoardColumnRepository{
		BaseRepository: NewBaseRepository[model.BoardColumn](db),
		db:             db,
	}
}

func (r *BoardColumnRepository) ListByProject(projectID uint) ([]model.BoardColumn, error) {
	var columns []model.BoardColumn
	if err := r.db.Where("project_id = ?", projectID).Order("position ASC").Find(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}
