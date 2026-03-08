package service

import (
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"gorm.io/gorm"
)

type StoryService struct {
	db        *gorm.DB
	events    EventPublisher
	workflow  *WorkflowService
	storyRepo *repository.StoryRepository
}

type CreateStoryInput struct {
	ProjectID          uint
	UserID             uint
	Title              string
	Description        string
	StoryType          string
	Priority           int
	StoryPoints        *int
	AcceptanceCriteria []model.AcceptanceCriterion
	Tags               []string
	Position           float64
}

type UpdateStoryInput struct {
	Title              *string
	Description        *string
	StoryType          *string
	Priority           *int
	StoryPoints        *int
	AcceptanceCriteria *[]model.AcceptanceCriterion
	Tags               *[]string
}

func NewStoryService(db *gorm.DB, events EventPublisher) *StoryService {
	return &StoryService{
		db:        db,
		events:    events,
		workflow:  Workflow,
		storyRepo: repository.NewStoryRepository(db),
	}
}

func (s *StoryService) EnsureProjectMember(projectID, userID uint) error {
	_, _, err := EnsureProjectAccess(s.db, projectID, userID)
	return err
}

func (s *StoryService) GetWithAccess(storyID, userID uint) (*model.UserStory, error) {
	story, err := s.storyRepo.FindByIDWithDetails(storyID)
	if err != nil {
		return nil, err
	}

	if _, _, err := EnsureProjectAccess(s.db, story.ProjectID, userID); err != nil {
		return nil, err
	}
	return story, nil
}

func (s *StoryService) Create(input CreateStoryInput) (*model.UserStory, error) {
	storyType := strings.TrimSpace(input.StoryType)
	if storyType != model.StoryTypeFeature && storyType != model.StoryTypeBug && storyType != model.StoryTypeChore {
		return nil, NewValidationError(ValidationIssue{Field: "story_type", Message: "story_type仅支持feature/bug/chore"})
	}

	title := strings.TrimSpace(input.Title)
	if len(title) < 2 || len(title) > 200 {
		return nil, NewValidationError(ValidationIssue{Field: "title", Message: "标题长度需在2-200之间"})
	}
	description := strings.TrimSpace(input.Description)
	if len(description) > 2000 {
		return nil, NewValidationError(ValidationIssue{Field: "description", Message: "描述最多2000字符"})
	}
	if input.Priority < 0 || input.Priority > 4 {
		return nil, NewValidationError(ValidationIssue{Field: "priority", Message: "priority仅支持0-4"})
	}
	if input.StoryPoints != nil {
		if _, ok := validStoryPoints[*input.StoryPoints]; !ok {
			return nil, NewValidationError(ValidationIssue{Field: "story_points", Message: "故事点仅支持 1,2,3,5,8,13"})
		}
	}

	position := input.Position
	if position <= 0 {
		position = float64(time.Now().UnixNano())
	}

	story := model.UserStory{
		ProjectID:          input.ProjectID,
		Title:              title,
		Description:        description,
		StoryType:          storyType,
		Status:             model.StoryStatusPending, // 新创建的故事进入待审批状态
		Archived:           false,
		Priority:           input.Priority,
		Points:             input.StoryPoints,
		CreatedBy:          input.UserID,
		Position:           position,
		AcceptanceCriteria: model.MarshalJSON(input.AcceptanceCriteria),
		Tags:               model.MarshalJSON(input.Tags),
		CodeReferences:     model.MarshalJSON([]string{}),
	}

	if err := s.db.Create(&story).Error; err != nil {
		return nil, err
	}

	_ = createActivityLog(s.db, story.ProjectID, input.UserID, "story", story.ID, "created", nil, map[string]any{
		"title":      story.Title,
		"status":     story.Status,
		"story_type": story.StoryType,
	})

	return &story, nil
}

func (s *StoryService) Update(story *model.UserStory, userID uint, input UpdateStoryInput) (bool, error) {
	oldFields := map[string]any{}
	newFields := map[string]any{}
	changed := false

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if len(title) < 2 || len(title) > 200 {
			return false, NewValidationError(ValidationIssue{Field: "title", Message: "标题长度需在2-200之间"})
		}
		if story.Title != title {
			oldFields["title"] = story.Title
			newFields["title"] = title
			story.Title = title
			changed = true
		}
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return false, NewValidationError(ValidationIssue{Field: "description", Message: "描述最多2000字符"})
		}
		if story.Description != desc {
			oldFields["description"] = story.Description
			newFields["description"] = desc
			story.Description = desc
			changed = true
		}
	}

	if input.StoryType != nil {
		storyType := strings.TrimSpace(*input.StoryType)
		if storyType != model.StoryTypeFeature && storyType != model.StoryTypeBug && storyType != model.StoryTypeChore {
			return false, NewValidationError(ValidationIssue{Field: "story_type", Message: "story_type仅支持feature/bug/chore"})
		}
		if story.StoryType != storyType {
			oldFields["story_type"] = story.StoryType
			newFields["story_type"] = storyType
			story.StoryType = storyType
			changed = true
		}
	}

	if input.Priority != nil {
		if *input.Priority < 0 || *input.Priority > 4 {
			return false, NewValidationError(ValidationIssue{Field: "priority", Message: "priority仅支持0-4"})
		}
		if story.Priority != *input.Priority {
			oldFields["priority"] = story.Priority
			newFields["priority"] = *input.Priority
			story.Priority = *input.Priority
			changed = true
		}
	}

	if input.StoryPoints != nil {
		if _, ok := validStoryPoints[*input.StoryPoints]; !ok {
			return false, NewValidationError(ValidationIssue{Field: "story_points", Message: "故事点仅支持 1,2,3,5,8,13"})
		}
		if story.Points == nil || *story.Points != *input.StoryPoints {
			oldFields["story_points"] = func() any {
				if story.Points == nil {
					return nil
				}
				return *story.Points
			}()
			newFields["story_points"] = *input.StoryPoints
			p := *input.StoryPoints
			story.Points = &p
			changed = true
		}
	}

	if input.Tags != nil {
		oldFields["tags"] = story.Tags
		newFields["tags"] = *input.Tags
		story.Tags = model.MarshalJSON(*input.Tags)
		changed = true
	}

	if input.AcceptanceCriteria != nil {
		oldFields["acceptance_criteria"] = story.AcceptanceCriteria
		newFields["acceptance_criteria"] = *input.AcceptanceCriteria
		story.AcceptanceCriteria = model.MarshalJSON(*input.AcceptanceCriteria)
		changed = true
	}

	if !changed {
		return false, nil
	}

	if err := s.db.Save(story).Error; err != nil {
		return false, err
	}
	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "updated", oldFields, newFields)
	return true, nil
}

func (s *StoryService) UpdateStatus(story *model.UserStory, userID uint, newStatus string, position float64, actor any) error {
	if !s.workflow.CanStoryTransit(story.Status, newStatus) {
		return ErrInvalidTransition
	}

	oldStatus := story.Status
	oldPosition := story.Position

	story.Status = newStatus
	if position > 0 {
		story.Position = position
	}

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "status_changed", map[string]any{
		"status":   oldStatus,
		"position": oldPosition,
	}, map[string]any{
		"status":   story.Status,
		"position": story.Position,
	})

	if s.events != nil {
		s.events.Broadcast("story.status_changed", map[string]any{
			"story_id":   story.ID,
			"project_id": story.ProjectID,
			"old_status": oldStatus,
			"new_status": story.Status,
			"actor":      actor,
		})
	}

	return nil
}

func (s *StoryService) UpdateACStatus(story *model.UserStory, userID uint, acID, status, evidence, notes string, actor any) (time.Time, error) {
	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return time.Time{}, ErrACCorrupted
	}

	found := false
	var oldValue map[string]any
	now := time.Now()
	for i := range criteria {
		if criteria[i].ID == acID {
			found = true
			oldValue = map[string]any{
				"status":   criteria[i].Status,
				"evidence": criteria[i].Evidence,
				"notes":    criteria[i].Notes,
			}
			criteria[i].Status = status
			criteria[i].Evidence = strings.TrimSpace(evidence)
			criteria[i].Notes = strings.TrimSpace(notes)
			criteria[i].VerifiedBy = &userID
			criteria[i].VerifiedAt = &now
			break
		}
	}
	if !found {
		return time.Time{}, ErrACNotFound
	}

	story.AcceptanceCriteria = model.MarshalJSON(criteria)
	if err := s.db.Save(story).Error; err != nil {
		return time.Time{}, err
	}

	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "ac_updated", oldValue, map[string]any{
		"ac_id":    acID,
		"status":   status,
		"evidence": evidence,
		"notes":    notes,
	})

	if s.events != nil {
		s.events.Broadcast("story.ac_updated", map[string]any{
			"story_id":  story.ID,
			"ac_id":     acID,
			"ac_status": status,
			"actor":     actor,
		})
	}

	return now, nil
}

func (s *StoryService) Claim(story *model.UserStory, userID uint) error {
	if story.AssignedTo != nil && *story.AssignedTo != userID {
		return ErrAlreadyClaimed
	}
	if story.AssignedTo == nil && story.Status != model.StoryStatusBacklog && story.Status != model.StoryStatusReady {
		return ErrClaimNotAllowed
	}

	oldStatus := story.Status
	oldAssigned := story.AssignedTo
	story.AssignedTo = &userID
	story.Status = model.StoryStatusInProgress

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "claimed", map[string]any{
		"status":      oldStatus,
		"assigned_to": oldAssigned,
	}, map[string]any{
		"status":      story.Status,
		"assigned_to": userID,
	})

	return nil
}

func (s *StoryService) AddCodeReference(story *model.UserStory, userID uint, reference string) ([]string, error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return nil, NewValidationError(ValidationIssue{Field: "reference", Message: "reference不能为空"})
	}

	refs := parseStringArrayJSON(story.CodeReferences)
	for _, existing := range refs {
		if existing == ref {
			return refs, nil
		}
	}
	refs = append(refs, ref)
	story.CodeReferences = model.MarshalJSON(refs)

	if err := s.db.Save(story).Error; err != nil {
		return nil, err
	}
	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "code_ref_added", nil, map[string]any{"reference": ref})
	return refs, nil
}

func (s *StoryService) Release(story *model.UserStory, userID uint, role string) error {
	if story.AssignedTo == nil {
		return ErrNotClaimed
	}
	if *story.AssignedTo != userID && role != model.RoleProduct && role != model.RoleAdmin {
		return ErrNoReleasePermission
	}

	oldAssigned := *story.AssignedTo
	oldStatus := story.Status
	story.AssignedTo = nil
	story.Status = model.StoryStatusReady

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "released", map[string]any{
		"status":      oldStatus,
		"assigned_to": oldAssigned,
	}, map[string]any{
		"status":      story.Status,
		"assigned_to": nil,
	})

	return nil
}

func (s *StoryService) Delete(story *model.UserStory, userID uint) error {
	if err := s.db.Delete(&model.UserStory{}, story.ID).Error; err != nil {
		return err
	}
	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "deleted", map[string]any{
		"title":  story.Title,
		"status": story.Status,
	}, nil)
	return nil
}

func (s *StoryService) Archive(story *model.UserStory, userID uint) (bool, error) {
	if story.Archived {
		return false, nil
	}
	story.Archived = true
	if err := s.db.Save(story).Error; err != nil {
		return false, err
	}
	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "archived", map[string]any{"archived": false}, map[string]any{"archived": true})
	return true, nil
}

func (s *StoryService) Restore(story *model.UserStory, userID uint) (bool, error) {
	if !story.Archived {
		return false, nil
	}
	story.Archived = false
	if err := s.db.Save(story).Error; err != nil {
		return false, err
	}
	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "restored", map[string]any{"archived": true}, map[string]any{"archived": false})
	return true, nil
}

// Review 审批故事（技术负责人）
func (s *StoryService) Review(story *model.UserStory, userID uint, approved bool, comment string) error {
	if story.Status != model.StoryStatusPending {
		return NewValidationError(ValidationIssue{Field: "status", Message: "只有待审批状态的故事可以审批"})
	}

	oldStatus := story.Status
	now := time.Now()

	if approved {
		story.Status = model.StoryStatusBacklog
	} else {
		// 拒绝审批可以设置为特殊状态或直接删除，这里选择标记为backlog但记录拒绝
		story.Status = model.StoryStatusBacklog
	}

	story.ReviewedBy = &userID
	story.ReviewedAt = &now

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	_ = createActivityLog(s.db, story.ProjectID, userID, "story", story.ID, "reviewed", map[string]any{
		"status":   oldStatus,
		"approved": false,
	}, map[string]any{
		"status":   story.Status,
		"approved": approved,
		"comment":  comment,
	})

	return nil
}

// Assign 分配故事（技术负责人或产品经理）
func (s *StoryService) Assign(story *model.UserStory, assignerID uint, assigneeID *uint) error {
	oldAssigned := story.AssignedTo
	story.AssignedTo = assigneeID

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	_ = createActivityLog(s.db, story.ProjectID, assignerID, "story", story.ID, "assigned", map[string]any{
		"assigned_to": oldAssigned,
	}, map[string]any{
		"assigned_to": assigneeID,
	})

	return nil
}

var validStoryPoints = map[int]struct{}{
	1: {}, 2: {}, 3: {}, 5: {}, 8: {}, 13: {},
}
