package service

import (
	"context"
	"strings"

	"github.com/jiangfire/storybook/internal/logging"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/jiangfire/storybook/internal/repository"
	"gorm.io/gorm"
)

type TaskService struct {
	db       *gorm.DB
	events   EventPublisher
	workflow *WorkflowService
	taskRepo *repository.TaskRepository
	notifier Notifier
}

type CreateTaskInput struct {
	ProjectID      uint
	StoryID        uint
	CreatedBy      uint
	Title          string
	Description    string
	Priority       int
	EstimatedHours float64
}

type UpdateTaskInput struct {
	Title          *string
	Description    *string
	Priority       *int
	EstimatedHours *float64
}

func NewTaskService(db *gorm.DB, events EventPublisher) *TaskService {
	return &TaskService{
		db:       db,
		events:   events,
		workflow: Workflow,
		taskRepo: repository.NewTaskRepository(db),
		notifier: NoopNotifier{},
	}
}

func (s *TaskService) WithNotifier(n Notifier) *TaskService {
	if n != nil {
		s.notifier = n
	}
	return s
}

func (s *TaskService) Create(input CreateTaskInput) (*model.Task, error) {
	title := strings.TrimSpace(input.Title)
	if len(title) < 2 || len(title) > 255 {
		return nil, NewValidationError(ValidationIssue{Field: "title", Message: "title长度需在2-255之间"})
	}
	description := strings.TrimSpace(input.Description)
	if len(description) > 2000 {
		return nil, NewValidationError(ValidationIssue{Field: "description", Message: "description最多2000字符"})
	}
	if input.Priority < 0 || input.Priority > 4 {
		return nil, NewValidationError(ValidationIssue{Field: "priority", Message: "priority仅支持0-4"})
	}
	if input.EstimatedHours < 0 || input.EstimatedHours > 500 {
		return nil, NewValidationError(ValidationIssue{Field: "estimated_hours", Message: "estimated_hours需在0-500之间"})
	}

	task := model.Task{
		ProjectID:      input.ProjectID,
		StoryID:        input.StoryID,
		Title:          title,
		Description:    description,
		Status:         model.TaskStatusTodo,
		Priority:       input.Priority,
		Progress:       0,
		EstimatedHours: input.EstimatedHours,
		CreatedBy:      input.CreatedBy,
		CodeReferences: model.MarshalJSON([]string{}),
	}

	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &task.ProjectID, input.CreatedBy, "task", task.ID, "created", nil, map[string]any{
		"title":    task.Title,
		"status":   task.Status,
		"priority": task.Priority,
	}), "write task activity log", "task_id", task.ID, "action", "created")
	return &task, nil
}

func (s *TaskService) Update(task *model.Task, projectID, userID uint, input UpdateTaskInput) (bool, error) {
	oldValue := map[string]any{}
	newValue := map[string]any{}
	changed := false

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if len(title) < 2 || len(title) > 255 {
			return false, NewValidationError(ValidationIssue{Field: "title", Message: "title长度需在2-255之间"})
		}
		if task.Title != title {
			oldValue["title"] = task.Title
			newValue["title"] = title
			task.Title = title
			changed = true
		}
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return false, NewValidationError(ValidationIssue{Field: "description", Message: "description最多2000字符"})
		}
		if task.Description != desc {
			oldValue["description"] = task.Description
			newValue["description"] = desc
			task.Description = desc
			changed = true
		}
	}

	if input.Priority != nil {
		if *input.Priority < 0 || *input.Priority > 4 {
			return false, NewValidationError(ValidationIssue{Field: "priority", Message: "priority仅支持0-4"})
		}
		if task.Priority != *input.Priority {
			oldValue["priority"] = task.Priority
			newValue["priority"] = *input.Priority
			task.Priority = *input.Priority
			changed = true
		}
	}

	if input.EstimatedHours != nil {
		if *input.EstimatedHours < 0 || *input.EstimatedHours > 500 {
			return false, NewValidationError(ValidationIssue{Field: "estimated_hours", Message: "estimated_hours需在0-500之间"})
		}
		if task.EstimatedHours != *input.EstimatedHours {
			oldValue["estimated_hours"] = task.EstimatedHours
			newValue["estimated_hours"] = *input.EstimatedHours
			task.EstimatedHours = *input.EstimatedHours
			changed = true
		}
	}

	if !changed {
		return false, nil
	}
	if err := s.db.Save(task).Error; err != nil {
		return false, err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "updated", oldValue, newValue), "write task activity log", "task_id", task.ID, "action", "updated")
	return true, nil
}

func (s *TaskService) UpdateStatus(task *model.Task, projectID, userID uint, role, newStatus string, actor any) error {
	if task.AssignedTo != nil && *task.AssignedTo != userID && role != model.RoleProduct && role != model.RoleAdmin {
		return ErrNoStatusPermission
	}
	if !s.workflow.CanTaskTransit(task.Status, newStatus) {
		return ErrInvalidTransition
	}

	oldStatus := task.Status
	oldProgress := task.Progress
	task.Status = newStatus
	if task.Status == model.TaskStatusDone {
		task.Progress = 100
	} else if task.Progress == 100 {
		task.Progress = 90
	}

	if err := s.db.Save(task).Error; err != nil {
		return err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "status_changed", map[string]any{
		"status":   oldStatus,
		"progress": oldProgress,
	}, map[string]any{
		"status":   task.Status,
		"progress": task.Progress,
	}), "write task activity log", "task_id", task.ID, "action", "status_changed")

	if s.events != nil {
		s.events.BroadcastProject(task.ProjectID, "task.status_changed", map[string]any{
			"task_id":    task.ID,
			"project_id": task.ProjectID,
			"old_status": oldStatus,
			"new_status": task.Status,
			"actor":      actor,
		})
	}

	return nil
}

func (s *TaskService) AddCodeReference(task *model.Task, projectID, userID uint, reference string) ([]string, error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return nil, NewValidationError(ValidationIssue{Field: "reference", Message: "reference不能为空"})
	}

	refs := ParseStringArrayJSON(task.CodeReferences)
	for _, ex := range refs {
		if ex == ref {
			return refs, nil
		}
	}
	refs = append(refs, ref)
	task.CodeReferences = model.MarshalJSON(refs)

	if err := s.db.Save(task).Error; err != nil {
		return nil, err
	}
	logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "code_ref_added", nil, map[string]any{"reference": ref}), "write task activity log", "task_id", task.ID, "action", "code_ref_added")
	return refs, nil
}

func (s *TaskService) UpdateProgress(task *model.Task, projectID, userID uint, role string, progress int, actor any) error {
	if task.AssignedTo != nil && *task.AssignedTo != userID && role != model.RoleProduct && role != model.RoleAdmin {
		return ErrNoProgressPermission
	}

	oldProgress := task.Progress
	oldStatus := task.Status
	task.Progress = progress
	if task.Progress >= 100 {
		task.Progress = 100
		task.Status = model.TaskStatusDone
	} else if task.Progress > 0 && task.Status == model.TaskStatusTodo {
		task.Status = model.TaskStatusInProgress
	}

	if err := s.db.Save(task).Error; err != nil {
		return err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "progress_changed", map[string]any{
		"progress": oldProgress,
		"status":   oldStatus,
	}, map[string]any{
		"progress": task.Progress,
		"status":   task.Status,
	}), "write task activity log", "task_id", task.ID, "action", "progress_changed")

	if s.events != nil {
		s.events.BroadcastProject(task.ProjectID, "task.progress_changed", map[string]any{
			"task_id":      task.ID,
			"project_id":   task.ProjectID,
			"old_progress": oldProgress,
			"new_progress": task.Progress,
			"actor":        actor,
		})
	}

	return nil
}

func (s *TaskService) Claim(task *model.Task, projectID, userID uint) error {
	claimed, err := executeClaim(s.db, task.ID, userID, "task",
		nil,
		func(current *model.Task) {
			current.AssignedTo = &userID
			if current.Status == model.TaskStatusTodo {
				current.Status = model.TaskStatusInProgress
			}
		},
	)
	if err != nil {
		return err
	}
	if claimed.CreatedBy != 0 && claimed.CreatedBy != userID {
		pid := projectID
		s.notifier.Notify(context.Background(), claimed.CreatedBy, NotificationEvent{
			Type:       model.NotificationTaskAssigned,
			EntityType: model.NotificationEntityTask,
			EntityID:   claimed.ID,
			ProjectID:  &pid,
			ActorID:    &userID,
			Title:      "任务被领取",
			Body:       claimed.Title,
			Metadata: map[string]any{
				"task_id":  claimed.ID,
				"title":    claimed.Title,
				"status":   claimed.Status,
				"story_id": claimed.StoryID,
			},
		})
	}
	return nil
}

func (s *TaskService) Release(task *model.Task, projectID, userID uint, role string) error {
	_, err := executeRelease(s.db, task.ID, userID, role, "task",
		nil,
		func(current *model.Task) {
			current.AssignedTo = nil
			if current.Progress == 0 {
				current.Status = model.TaskStatusTodo
			} else if current.Status == model.TaskStatusInProgress {
				current.Status = model.TaskStatusBlocked
			}
		},
	)
	return err
}

func (s *TaskService) Delete(task *model.Task, projectID, userID uint) error {
	if err := s.db.Delete(&model.Task{}, task.ID).Error; err != nil {
		return err
	}
	logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "deleted", map[string]any{"title": task.Title}, nil), "write task activity log", "task_id", task.ID, "action", "deleted")
	return nil
}

func (s *TaskService) SplitFromAC(story *model.UserStory, projectID, userID uint, criteria []model.AcceptanceCriterion) ([]model.Task, error) {
	if len(criteria) == 0 {
		return nil, ErrNoSplittableAC
	}

	created := make([]model.Task, 0)
	for i, ac := range criteria {
		title := strings.TrimSpace(ac.Description)
		if title == "" {
			title = "子任务 " + ac.ID
		}

		var exists int64
		if err := s.db.Model(&model.Task{}).Where("story_id = ? AND title = ?", story.ID, title).Count(&exists).Error; err != nil {
			continue
		}
		if exists > 0 {
			continue
		}

		task := model.Task{
			ProjectID:      projectID,
			StoryID:        story.ID,
			Title:          title,
			Description:    "由AC拆分自动生成",
			Status:         model.TaskStatusTodo,
			Priority:       story.Priority,
			Progress:       0,
			EstimatedHours: float64(i + 1),
			CreatedBy:      userID,
			CodeReferences: model.MarshalJSON([]string{}),
		}

		if err := s.db.Create(&task).Error; err == nil {
			created = append(created, task)
			logging.LogIfErr(WriteActivityLog(s.db, &projectID, userID, "task", task.ID, "created_from_ac", nil, map[string]any{
				"title": task.Title,
				"ac_id": ac.ID,
			}), "write task activity log", "task_id", task.ID, "action", "created_from_ac")
		}
	}

	return created, nil
}
