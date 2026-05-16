package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type AIConfigRepository struct {
	BaseRepository[model.AIConfig]
	db *gorm.DB
}

func NewAIConfigRepository(db *gorm.DB) *AIConfigRepository {
	return &AIConfigRepository{
		BaseRepository: NewBaseRepository[model.AIConfig](db),
		db:             db,
	}
}

func (r *AIConfigRepository) FindLatestEnabled() (*model.AIConfig, error) {
	var cfg model.AIConfig
	if err := r.db.Where("enabled = ?", true).Order("id DESC").Limit(1).Find(&cfg).Error; err != nil {
		return nil, err
	}
	if cfg.ID == 0 {
		return nil, nil
	}
	return &cfg, nil
}
