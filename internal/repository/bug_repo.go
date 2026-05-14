package repository

import (
	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type BugRepository struct {
	BaseRepository[model.BugReport]
	db *gorm.DB
}

func NewBugRepository(db *gorm.DB) *BugRepository {
	return &BugRepository{
		BaseRepository: NewBaseRepository[model.BugReport](db),
		db:             db,
	}
}

func (r *BugRepository) FindByIDWithDetails(bugID uint) (*model.BugReport, error) {
	var bug model.BugReport
	if err := r.db.Preload("Reporter").Preload("Assignee").Preload("Story").First(&bug, bugID).Error; err != nil {
		return nil, err
	}
	return &bug, nil
}

func (r *BugRepository) ListByProject(projectID uint, opts BugListOptions) ([]model.BugReport, int64, error) {
	var total int64
	tx := r.db.Model(&model.BugReport{}).Where("project_id = ?", projectID)
	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.Severity != "" {
		tx = tx.Where("severity = ?", opts.Severity)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var bugs []model.BugReport
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Offset(offset).Limit(opts.Limit).Order("created_at DESC").Find(&bugs).Error; err != nil {
		return nil, 0, err
	}
	return bugs, total, nil
}

func (r *BugRepository) UpdateStatus(bugID uint, status string) error {
	return r.db.Model(&model.BugReport{}).Where("id = ?", bugID).Update("status", status).Error
}

func (r *BugRepository) UpdateAssignee(bugID uint, assignedTo *uint) error {
	return r.db.Model(&model.BugReport{}).Where("id = ?", bugID).Update("assigned_to", assignedTo).Error
}

type BugListOptions struct {
	ListOptions
	Severity string
}
