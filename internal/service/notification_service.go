package service

import (
	"context"
	"log/slog"

	"git.neolidy.top/neo/storybook/internal/model"
	"git.neolidy.top/neo/storybook/internal/repository"
	"gorm.io/gorm"
)

// NotificationEvent is the wire-level description of a notification before
// it is persisted. ActorID being nil represents a system-initiated event.
type NotificationEvent struct {
	Type       string
	EntityType string
	EntityID   uint
	ProjectID  *uint
	ActorID    *uint
	Title      string
	Body       string
	Metadata   any
}

// Notifier writes notifications into the database and pushes a real-time
// signal to the recipients' WebSocket connections. Implementations should
// never propagate failure to the caller — notifications are best-effort and
// must not abort the originating business transaction.
type Notifier interface {
	Notify(ctx context.Context, userID uint, ev NotificationEvent)
	NotifyMany(ctx context.Context, userIDs []uint, ev NotificationEvent)
	// NotifyProjectMembers fans out to every user with read access to the
	// project (owner, members, tech leads, admins). Used for project-wide
	// announcements such as sprint started / sprint completed.
	NotifyProjectMembers(ctx context.Context, projectID uint, ev NotificationEvent)
}

// NoopNotifier is the safe default when DI hasn't supplied a real notifier
// (e.g. unit tests).
type NoopNotifier struct{}

func (NoopNotifier) Notify(context.Context, uint, NotificationEvent)              {}
func (NoopNotifier) NotifyMany(context.Context, []uint, NotificationEvent)        {}
func (NoopNotifier) NotifyProjectMembers(context.Context, uint, NotificationEvent) {}

type NotificationService struct {
	repo   *repository.NotificationRepository
	db     *gorm.DB
	events EventPublisher
	logger *slog.Logger
}

func NewNotificationService(db *gorm.DB, events EventPublisher, logger *slog.Logger) *NotificationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotificationService{
		repo:   repository.NewNotificationRepository(db),
		db:     db,
		events: events,
		logger: logger,
	}
}

func (s *NotificationService) Notify(ctx context.Context, userID uint, ev NotificationEvent) {
	if userID == 0 {
		return
	}
	s.NotifyMany(ctx, []uint{userID}, ev)
}

func (s *NotificationService) NotifyMany(ctx context.Context, userIDs []uint, ev NotificationEvent) {
	recipients := dedupeNonZero(userIDs)
	// Self-targeted notifications (actor == recipient) are noise; skip them so
	// a developer who claims their own story doesn't notify themselves.
	if ev.ActorID != nil {
		filtered := recipients[:0]
		for _, id := range recipients {
			if id != *ev.ActorID {
				filtered = append(filtered, id)
			}
		}
		recipients = filtered
	}
	if len(recipients) == 0 {
		return
	}

	metadataJSON := model.MarshalJSON(ev.Metadata)
	items := make([]model.Notification, 0, len(recipients))
	for _, uid := range recipients {
		items = append(items, model.Notification{
			UserID:     uid,
			ActorID:    ev.ActorID,
			Type:       ev.Type,
			EntityType: ev.EntityType,
			EntityID:   ev.EntityID,
			ProjectID:  ev.ProjectID,
			Title:      ev.Title,
			Body:       ev.Body,
			Metadata:   metadataJSON,
		})
	}

	if err := s.repo.BulkCreate(items); err != nil {
		s.logger.Warn("notify: bulk create failed", "error", err, "type", ev.Type, "recipients", len(recipients))
		return
	}

	if s.events == nil {
		return
	}
	for i := range items {
		payload := map[string]any{
			"id":          items[i].ID,
			"type":        items[i].Type,
			"entity_type": items[i].EntityType,
			"entity_id":   items[i].EntityID,
			"project_id":  items[i].ProjectID,
			"actor_id":    items[i].ActorID,
			"title":       items[i].Title,
			"body":        items[i].Body,
			"created_at":  items[i].CreatedAt,
		}
		s.events.BroadcastUser(items[i].UserID, "notification.new", payload)
	}
}

func dedupeNonZero(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *NotificationService) NotifyProjectMembers(ctx context.Context, projectID uint, ev NotificationEvent) {
	if s.db == nil || projectID == 0 {
		return
	}

	ids, err := s.projectRecipientIDs(projectID)
	if err != nil {
		s.logger.Warn("notify: collect project members failed", "error", err, "project_id", projectID)
		return
	}
	if len(ids) == 0 {
		return
	}
	if ev.ProjectID == nil {
		pid := projectID
		ev.ProjectID = &pid
	}
	s.NotifyMany(ctx, ids, ev)
}

// projectRecipientIDs returns every user who should be notified about
// project-wide events: project owner, current members, tech leads, and
// platform admins. Mirrors realtime.Hub.projectRecipientIDs so the
// notifier can fan out without taking a Hub dependency.
func (s *NotificationService) projectRecipientIDs(projectID uint) ([]uint, error) {
	ids := make(map[uint]struct{})

	var project model.Project
	if err := s.db.Select("id, owner_id").First(&project, projectID).Error; err != nil {
		return nil, err
	}
	ids[project.OwnerID] = struct{}{}

	var memberIDs []uint
	if err := s.db.Model(&model.ProjectMember{}).Where("project_id = ?", projectID).Pluck("user_id", &memberIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range memberIDs {
		ids[id] = struct{}{}
	}

	var techLeadIDs []uint
	if err := s.db.Model(&model.ProjectTechLead{}).Where("project_id = ?", projectID).Pluck("user_id", &techLeadIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range techLeadIDs {
		ids[id] = struct{}{}
	}

	var adminIDs []uint
	if err := s.db.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Pluck("id", &adminIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range adminIDs {
		ids[id] = struct{}{}
	}

	out := make([]uint, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out, nil
}
