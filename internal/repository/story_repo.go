package repository

import (
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

type StoryRepository struct {
	BaseRepository[model.UserStory]
}

func NewStoryRepository(db *gorm.DB) *StoryRepository {
	return &StoryRepository{
		BaseRepository: NewBaseRepository[model.UserStory](db),
	}
}

func (r *StoryRepository) FindByIDWithDetails(storyID uint) (*model.UserStory, error) {
	var story model.UserStory
	if err := r.DB().
		Preload("Project").
		Preload("Assignee").
		Preload("Creator").
		Preload("Sprint").
		Preload("Reviewer").
		First(&story, storyID).Error; err != nil {
		return nil, err
	}
	return &story, nil
}

func (r *StoryRepository) ListByProject(projectID uint, opts ListOptions) ([]model.UserStory, int64, error) {
	var total int64
	tx := r.DB().Model(&model.UserStory{}).Where("project_id = ? AND archived = ?", projectID, false)

	if opts.Status != "" {
		tx = tx.Where("status = ?", opts.Status)
	}
	if opts.SprintID != nil {
		tx = tx.Where("sprint_id = ?", *opts.SprintID)
	}
	if opts.AssignedTo != nil {
		tx = tx.Where("assigned_to = ?", *opts.AssignedTo)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var stories []model.UserStory
	offset := (opts.Page - 1) * opts.Limit
	if err := tx.Offset(offset).Limit(opts.Limit).Order("position ASC").Find(&stories).Error; err != nil {
		return nil, 0, err
	}
	return stories, total, nil
}

func (r *StoryRepository) ListBoardByProject(projectID uint) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().Where("project_id = ? AND archived = ?", projectID, false).Order("position ASC").Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) ListBoardByProjectWithAssignee(projectID uint) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().Where("project_id = ? AND archived = ?", projectID, false).Preload("Assignee").Order("position ASC, priority DESC").Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) ListByIDs(storyIDs []uint) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().Where("id IN ?", storyIDs).Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) CountBySprint(sprintID uint) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).Where("sprint_id = ? AND archived = ?", sprintID, false).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *StoryRepository) CountBySprintAndStatus(sprintID uint, status string) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).Where("sprint_id = ? AND status = ? AND archived = ?", sprintID, status, false).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *StoryRepository) CountByProjectAndStatus(projectID uint, status string) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).Where("project_id = ? AND status = ? AND archived = ?", projectID, status, false).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *StoryRepository) ListBySprint(sprintID uint) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().Where("sprint_id = ? AND archived = ?", sprintID, false).Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) ListByAssignee(userID uint, limit int) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().
		Preload("Project").
		Where("assigned_to = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) ListByCreator(userID uint, limit int) ([]model.UserStory, error) {
	var stories []model.UserStory
	if err := r.DB().
		Preload("Project").
		Where("created_by = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) CountByAssignee(userID uint) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).Where("assigned_to = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *StoryRepository) CountByAssigneeAndStatus(userID uint, status string) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).Where("assigned_to = ? AND status = ?", userID, status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySprintAndIDs counts how many of the given story IDs are attached to the sprint.
func (r *StoryRepository) CountBySprintAndIDs(sprintID uint, storyIDs []uint) (int64, error) {
	var count int64
	if err := r.DB().Model(&model.UserStory{}).
		Where("id IN ? AND sprint_id = ?", storyIDs, sprintID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdatePositionsBatch updates positions for multiple stories within a sprint in a transaction.
func (r *StoryRepository) UpdatePositionsBatch(sprintID uint, positions map[uint]float64) error {
	return r.DB().Transaction(func(tx *gorm.DB) error {
		for storyID, position := range positions {
			if err := tx.Model(&model.UserStory{}).
				Where("id = ? AND sprint_id = ?", storyID, sprintID).
				Update("position", position).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *StoryRepository) ListPendingReview(projectIDs []uint, search string) ([]model.UserStory, error) {
	if len(projectIDs) == 0 {
		return []model.UserStory{}, nil
	}
	tx := r.DB().Where("project_id IN ? AND review_status = ? AND archived = ?", projectIDs, model.ReviewStatusPending, false)
	if search != "" {
		tx = tx.Where("title LIKE ?", BuildLike(search))
	}
	var stories []model.UserStory
	if err := tx.Order("created_at DESC").Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *StoryRepository) UpdateStatus(storyID uint, status string, position float64) error {
	return r.DB().Model(&model.UserStory{}).Where("id = ?", storyID).Updates(map[string]any{
		"status":   status,
		"position": position,
	}).Error
}

func (r *StoryRepository) UpdateAssignee(storyID uint, assignedTo *uint) error {
	return r.DB().Model(&model.UserStory{}).Where("id = ?", storyID).Update("assigned_to", assignedTo).Error
}

// AvgCompletionDaysForUser returns the average number of days between creation
// and completion for stories done by the given user since the provided time.
// It handles SQLite (JULIANDAY) and Postgres (EXTRACT EPOCH) dialects internally.
func (r *StoryRepository) AvgCompletionDaysForUser(userID uint, since time.Time) (float64, error) {
	var avg float64
	dateDiffExpr := "JULIANDAY(updated_at) - JULIANDAY(created_at)"
	if r.DB().Dialector.Name() == "postgres" {
		dateDiffExpr = "EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400.0"
	}
	err := r.DB().Raw("SELECT COALESCE(AVG("+dateDiffExpr+"), 0) FROM user_stories WHERE assigned_to = ? AND status = ? AND updated_at >= ?", userID, model.StoryStatusDone, since).Scan(&avg).Error
	return avg, err
}

type ListOptions struct {
	Page       int
	Limit      int
	Status     string
	SprintID   *uint
	AssignedTo *uint
}
