package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"git.neolidy.top/neo/storybook/internal/logging"
	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StoryService struct {
	db        *gorm.DB
	events    EventPublisher
	workflow  *WorkflowService
	storyRepo *repository.StoryRepository
	vectorSvc VectorService
	notifier  Notifier
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
	return NewStoryServiceWithVector(db, events, nil)
}

func NewStoryServiceWithVector(db *gorm.DB, events EventPublisher, vectorSvc VectorService) *StoryService {
	return &StoryService{
		db:        db,
		events:    events,
		workflow:  Workflow,
		storyRepo: repository.NewStoryRepository(db),
		vectorSvc: vectorSvc,
		notifier:  NoopNotifier{},
	}
}

// WithNotifier replaces the default no-op notifier so claim/release/review
// events can fan out as in-app notifications.
func (s *StoryService) WithNotifier(n Notifier) *StoryService {
	if n != nil {
		s.notifier = n
	}
	return s
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
		ReviewStatus:       model.ReviewStatusPending,
		ReviewComment:      "",
		Archived:           false,
		Priority:           input.Priority,
		Points:             input.StoryPoints,
		CreatedBy:          input.UserID,
		Position:           position,
		AcceptanceCriteria: model.MarshalJSON(input.AcceptanceCriteria),
		Tags:               model.MarshalJSON(input.Tags),
		CodeReferences:     model.MarshalJSON([]string{}),
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&story).Error; err != nil {
			return err
		}
		logging.LogIfErr(WriteActivityLog(tx, &story.ProjectID, input.UserID, "story", story.ID, "created", nil, map[string]any{
			"title":      story.Title,
			"status":     story.Status,
			"story_type": story.StoryType,
		}), "write story activity log", "story_id", story.ID, "action", "created")
		return nil
	}); err != nil {
		return nil, err
	}

	s.indexStoryIfEnabled(&story)

	return &story, nil
}

func (s *StoryService) Update(story *model.UserStory, userID uint, input UpdateStoryInput) (bool, error) {
	oldFields := map[string]any{}
	newFields := map[string]any{}
	changed := false
	contentChanged := false

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
			contentChanged = true
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
			contentChanged = true
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
			contentChanged = true
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
		contentChanged = true
	}

	if input.AcceptanceCriteria != nil {
		oldFields["acceptance_criteria"] = story.AcceptanceCriteria
		newFields["acceptance_criteria"] = *input.AcceptanceCriteria
		story.AcceptanceCriteria = model.MarshalJSON(*input.AcceptanceCriteria)
		changed = true
		contentChanged = true
	}

	if !changed {
		return false, nil
	}

	if err := s.db.Save(story).Error; err != nil {
		return false, err
	}
	if contentChanged {
		s.indexStoryIfEnabled(story)
	}
	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "updated", oldFields, newFields), "write story activity log", "story_id", story.ID, "action", "updated")
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

	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "status_changed", map[string]any{
		"status":   oldStatus,
		"position": oldPosition,
	}, map[string]any{
		"status":   story.Status,
		"position": story.Position,
	}), "write story activity log", "story_id", story.ID, "action", "status_changed")

	if s.events != nil {
		s.events.BroadcastProject(story.ProjectID, "story.status_changed", map[string]any{
			"story_id":   story.ID,
			"project_id": story.ProjectID,
			"old_status": oldStatus,
			"new_status": story.Status,
			"actor":      actor,
		})
	}

	return nil
}

// mutateAC executes the common AC mutation skeleton: parse → apply fn → marshal →
// save → activity log. The caller supplies a closure that performs the actual
// business logic and returns old/new values for auditing.
func (s *StoryService) mutateAC(
	story *model.UserStory, userID uint, action string,
	fn func(criteria []model.AcceptanceCriterion) (newCriteria []model.AcceptanceCriterion, oldValue, newValue map[string]any, err error),
) error {
	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return ErrACCorrupted
	}

	newCriteria, oldValue, newValue, err := fn(criteria)
	if err != nil {
		return err
	}

	story.AcceptanceCriteria = model.MarshalJSON(newCriteria)
	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, action, oldValue, newValue),
		"write story activity log", "story_id", story.ID, "action", action)
	return nil
}

// broadcastAC sends a project-scoped WebSocket event when AC data changes.
func (s *StoryService) broadcastAC(projectID uint, event string, storyID uint, acID string, actor any, extra map[string]any) {
	if s.events == nil {
		return
	}
	payload := map[string]any{"story_id": storyID, "ac_id": acID, "actor": actor}
	for k, v := range extra {
		payload[k] = v
	}
	s.events.BroadcastProject(projectID, event, payload)
}

func (s *StoryService) UpdateACStatus(story *model.UserStory, userID uint, acID, status, evidence, notes string, actor any) (time.Time, error) {
	var verifiedAt time.Time
	err := s.mutateAC(story, userID, "ac_updated", func(criteria []model.AcceptanceCriterion) ([]model.AcceptanceCriterion, map[string]any, map[string]any, error) {
		for i := range criteria {
			if criteria[i].ID != acID {
				continue
			}
			oldValue := map[string]any{"status": criteria[i].Status, "evidence": criteria[i].Evidence, "notes": criteria[i].Notes}
			criteria[i].Status = status
			criteria[i].Evidence = strings.TrimSpace(evidence)
			criteria[i].Notes = strings.TrimSpace(notes)
			// userID==0 表示无认证身份的系统调用(例如 MCP 协议入口),
			// 此时保持 VerifiedBy 不变,避免出现 "user 0 验证" 的脏数据。
			if userID != 0 {
				criteria[i].VerifiedBy = &userID
			}
			now := time.Now()
			criteria[i].VerifiedAt = &now
			verifiedAt = now
			return criteria, oldValue, map[string]any{"ac_id": acID, "status": status, "evidence": evidence, "notes": notes}, nil
		}
		return nil, nil, nil, ErrACNotFound
	})
	if err != nil {
		return verifiedAt, err
	}
	s.broadcastAC(story.ProjectID, "story.ac_updated", story.ID, acID, actor, map[string]any{"ac_status": status})
	return verifiedAt, nil
}

// AddAC appends a new acceptance criterion. ID is server-generated as "ac-<N>"
// using max(existing N)+1 so it stays compatible with the existing convention
// produced by normalizeAC.
func (s *StoryService) AddAC(story *model.UserStory, userID uint, description, ref, notes string, actor any) (model.AcceptanceCriterion, error) {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return model.AcceptanceCriterion{}, NewValidationError(ValidationIssue{Field: "description", Message: "description不能为空"})
	}
	if len(desc) > 500 {
		return model.AcceptanceCriterion{}, NewValidationError(ValidationIssue{Field: "description", Message: "description最多500字符"})
	}

	var ac model.AcceptanceCriterion
	err := s.mutateAC(story, userID, "ac_added", func(criteria []model.AcceptanceCriterion) ([]model.AcceptanceCriterion, map[string]any, map[string]any, error) {
		ac = model.AcceptanceCriterion{
			ID: nextACID(criteria), Ref: strings.TrimSpace(ref), Description: desc,
			Status: model.ACStatusPending, Notes: strings.TrimSpace(notes), Order: len(criteria) + 1,
		}
		return append(criteria, ac), nil, map[string]any{"ac_id": ac.ID, "description": ac.Description, "ref": ac.Ref, "order": ac.Order}, nil
	})
	if err != nil {
		return model.AcceptanceCriterion{}, err
	}
	s.broadcastAC(story.ProjectID, "story.ac_added", story.ID, ac.ID, actor, nil)
	return ac, nil
}

// UpdateAC selectively edits an existing AC's content fields (description, ref,
// notes, order). Status changes go through UpdateACStatus to keep the verified
// audit trail intact.
func (s *StoryService) UpdateAC(story *model.UserStory, userID uint, acID string, description, ref, notes *string, order *int, actor any) (model.AcceptanceCriterion, error) {
	criteria, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		return model.AcceptanceCriterion{}, ErrACCorrupted
	}
	idx := -1
	for i := range criteria {
		if criteria[i].ID == acID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return model.AcceptanceCriterion{}, ErrACNotFound
	}

	oldValue := map[string]any{"description": criteria[idx].Description, "ref": criteria[idx].Ref, "notes": criteria[idx].Notes, "order": criteria[idx].Order}
	changed := false

	if description != nil {
		desc := strings.TrimSpace(*description)
		if desc == "" {
			return model.AcceptanceCriterion{}, NewValidationError(ValidationIssue{Field: "description", Message: "description不能为空"})
		}
		if len(desc) > 500 {
			return model.AcceptanceCriterion{}, NewValidationError(ValidationIssue{Field: "description", Message: "description最多500字符"})
		}
		if criteria[idx].Description != desc {
			criteria[idx].Description = desc
			changed = true
		}
	}
	if ref != nil {
		if v := strings.TrimSpace(*ref); criteria[idx].Ref != v {
			criteria[idx].Ref = v
			changed = true
		}
	}
	if notes != nil {
		if v := strings.TrimSpace(*notes); criteria[idx].Notes != v {
			criteria[idx].Notes = v
			changed = true
		}
	}
	if order != nil && *order > 0 && criteria[idx].Order != *order {
		criteria[idx].Order = *order
		changed = true
	}

	if !changed {
		return criteria[idx], nil
	}

	if err = s.mutateAC(story, userID, "ac_edited", func(_ []model.AcceptanceCriterion) ([]model.AcceptanceCriterion, map[string]any, map[string]any, error) {
		return criteria, oldValue, map[string]any{"ac_id": acID, "description": criteria[idx].Description, "ref": criteria[idx].Ref, "notes": criteria[idx].Notes, "order": criteria[idx].Order}, nil
	}); err != nil {
		return model.AcceptanceCriterion{}, err
	}
	s.broadcastAC(story.ProjectID, "story.ac_edited", story.ID, acID, actor, nil)
	return criteria[idx], nil
}

// RemoveAC drops a single criterion by ID. Order of the remaining items is
// preserved as-is; clients that want a compact 1..N sequence can re-issue
// UpdateAC calls.
func (s *StoryService) RemoveAC(story *model.UserStory, userID uint, acID string, actor any) error {
	err := s.mutateAC(story, userID, "ac_removed", func(criteria []model.AcceptanceCriterion) ([]model.AcceptanceCriterion, map[string]any, map[string]any, error) {
		for i := range criteria {
			if criteria[i].ID == acID {
				removed := criteria[i]
				return append(criteria[:i], criteria[i+1:]...), map[string]any{"ac_id": removed.ID, "description": removed.Description, "ref": removed.Ref, "order": removed.Order}, nil, nil
			}
		}
		return nil, nil, nil, ErrACNotFound
	})
	if err != nil {
		return err
	}
	s.broadcastAC(story.ProjectID, "story.ac_removed", story.ID, acID, actor, nil)
	return nil
}

// nextACID picks the smallest "ac-<N>" identifier not already taken so adding
// new ACs stays predictable and aligned with normalizeAC's seed format. IDs
// that don't match the pattern are ignored (no scheme collision with custom
// IDs that may have been imported from elsewhere).
func nextACID(items []model.AcceptanceCriterion) string {
	max := 0
	for _, ac := range items {
		var n int
		if _, err := fmt.Sscanf(ac.ID, "ac-%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("ac-%d", max+1)
}

func (s *StoryService) Claim(story *model.UserStory, userID uint) error {
	var claimedStory model.UserStory
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var current model.UserStory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, story.ID).Error; err != nil {
			return err
		}
		if current.AssignedTo != nil && *current.AssignedTo != userID {
			return ErrAlreadyClaimed
		}
		if current.AssignedTo == nil && current.Status != model.StoryStatusBacklog && current.Status != model.StoryStatusReady {
			return ErrClaimNotAllowed
		}

		oldStatus := current.Status
		oldAssigned := current.AssignedTo
		current.AssignedTo = &userID
		current.Status = model.StoryStatusInProgress

		if err := tx.Save(&current).Error; err != nil {
			return err
		}

		logging.LogIfErr(WriteActivityLog(tx, &current.ProjectID, userID, "story", current.ID, "claimed", map[string]any{
			"status":      oldStatus,
			"assigned_to": oldAssigned,
		}, map[string]any{
			"status":      current.Status,
			"assigned_to": userID,
		}), "write story activity log", "story_id", current.ID, "action", "claimed")

		claimedStory = current
		return nil
	})
	if err != nil {
		return err
	}

	pid := claimedStory.ProjectID
	s.notifier.Notify(context.Background(), claimedStory.CreatedBy, NotificationEvent{
		Type:       model.NotificationStoryClaimed,
		EntityType: model.NotificationEntityStory,
		EntityID:   claimedStory.ID,
		ProjectID:  &pid,
		ActorID:    &userID,
		Title:      "故事已被领取",
		Body:       claimedStory.Title,
		Metadata: map[string]any{
			"story_id":   claimedStory.ID,
			"title":      claimedStory.Title,
			"status":     claimedStory.Status,
			"claimed_by": userID,
		},
	})
	return nil
}

func (s *StoryService) AddCodeReference(story *model.UserStory, userID uint, reference string) ([]string, error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return nil, NewValidationError(ValidationIssue{Field: "reference", Message: "reference不能为空"})
	}

	refs := ParseStringArrayJSON(story.CodeReferences)
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
	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "code_ref_added", nil, map[string]any{"reference": ref}), "write story activity log", "story_id", story.ID, "action", "code_ref_added")
	return refs, nil
}

func (s *StoryService) Release(story *model.UserStory, userID uint, role string) error {
	var releasedStory model.UserStory
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var current model.UserStory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, story.ID).Error; err != nil {
			return err
		}
		if current.AssignedTo == nil {
			return ErrNotClaimed
		}
		if *current.AssignedTo != userID && role != model.RoleProduct && role != model.RoleAdmin {
			return ErrNoReleasePermission
		}

		oldAssigned := *current.AssignedTo
		oldStatus := current.Status
		current.AssignedTo = nil
		current.Status = model.StoryStatusReady

		if err := tx.Save(&current).Error; err != nil {
			return err
		}

		logging.LogIfErr(WriteActivityLog(tx, &current.ProjectID, userID, "story", current.ID, "released", map[string]any{
			"status":      oldStatus,
			"assigned_to": oldAssigned,
		}, map[string]any{
			"status":      current.Status,
			"assigned_to": nil,
		}), "write story activity log", "story_id", current.ID, "action", "released")

		releasedStory = current
		return nil
	})
	if err != nil {
		return err
	}

	pid := releasedStory.ProjectID
	s.notifier.Notify(context.Background(), releasedStory.CreatedBy, NotificationEvent{
		Type:       model.NotificationStoryReleased,
		EntityType: model.NotificationEntityStory,
		EntityID:   releasedStory.ID,
		ProjectID:  &pid,
		ActorID:    &userID,
		Title:      "故事已被释放",
		Body:       releasedStory.Title,
		Metadata: map[string]any{
			"story_id":    releasedStory.ID,
			"title":       releasedStory.Title,
			"status":      releasedStory.Status,
			"released_by": userID,
		},
	})
	return nil
}

func (s *StoryService) Delete(story *model.UserStory, userID uint) error {
	if err := s.db.Delete(&model.UserStory{}, story.ID).Error; err != nil {
		return err
	}
	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "deleted", map[string]any{
		"title":  story.Title,
		"status": story.Status,
	}, nil), "write story activity log", "story_id", story.ID, "action", "deleted")
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
	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "archived", map[string]any{"archived": false}, map[string]any{"archived": true}), "write story activity log", "story_id", story.ID, "action", "archived")
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
	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "restored", map[string]any{"archived": true}, map[string]any{"archived": false}), "write story activity log", "story_id", story.ID, "action", "restored")
	return true, nil
}

// Review 审批故事（技术负责人）
func (s *StoryService) Review(story *model.UserStory, userID uint, approved bool, comment string) error {
	if story.Status != model.StoryStatusPending {
		return NewValidationError(ValidationIssue{Field: "status", Message: "只有待审批状态的故事可以审批"})
	}
	reviewComment := strings.TrimSpace(comment)
	if !approved && reviewComment == "" {
		return NewValidationError(ValidationIssue{Field: "comment", Message: "拒绝审批必须填写原因"})
	}

	oldStatus := story.Status
	oldReviewStatus := story.ReviewStatus
	oldReviewComment := story.ReviewComment
	now := time.Now()

	if approved {
		story.Status = model.StoryStatusBacklog
		story.ReviewStatus = model.ReviewStatusApproved
		story.ReviewComment = ""
	} else {
		// 拒绝后回到待审批，并留下拒绝标记与原因
		story.Status = model.StoryStatusPending
		story.ReviewStatus = model.ReviewStatusRejected
		story.ReviewComment = reviewComment
	}

	story.ReviewedBy = &userID
	story.ReviewedAt = &now

	if err := s.db.Save(story).Error; err != nil {
		return err
	}

	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, userID, "story", story.ID, "reviewed", map[string]any{
		"status":         oldStatus,
		"review_status":  oldReviewStatus,
		"review_comment": oldReviewComment,
	}, map[string]any{
		"status":         story.Status,
		"review_status":  story.ReviewStatus,
		"review_comment": story.ReviewComment,
		"approved":       approved,
	}), "write story activity log", "story_id", story.ID, "action", "reviewed", "approved", approved)

	pid := story.ProjectID
	reviewTitle := "故事审批通过"
	if !approved {
		reviewTitle = "故事审批被拒绝"
	}
	s.notifier.Notify(context.Background(), story.CreatedBy, NotificationEvent{
		Type:       model.NotificationStoryReviewed,
		EntityType: model.NotificationEntityStory,
		EntityID:   story.ID,
		ProjectID:  &pid,
		ActorID:    &userID,
		Title:      reviewTitle,
		Body:       story.Title,
		Metadata: map[string]any{
			"story_id":       story.ID,
			"title":          story.Title,
			"review_status":  story.ReviewStatus,
			"review_comment": story.ReviewComment,
			"approved":       approved,
		},
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

	logging.LogIfErr(WriteActivityLog(s.db, &story.ProjectID, assignerID, "story", story.ID, "assigned", map[string]any{
		"assigned_to": oldAssigned,
	}, map[string]any{
		"assigned_to": assigneeID,
	}), "write story activity log", "story_id", story.ID, "action", "assigned")

	return nil
}

func (s *StoryService) indexStoryIfEnabled(story *model.UserStory) {
	if s.vectorSvc == nil || story == nil || story.Archived {
		return
	}

	err := s.vectorSvc.IndexStory(context.Background(), story)
	if err != nil {
		// 记录索引错误，但不影响故事创建/更新的成功
		slog.Warn("failed to index story",
			"story_id", story.ID,
			"project_id", story.ProjectID,
			"title", story.Title,
			"error", err,
		)
	}
}

var validStoryPoints = map[int]struct{}{
	1: {}, 2: {}, 3: {}, 5: {}, 8: {}, 13: {},
}
