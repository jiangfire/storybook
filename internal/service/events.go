package service

type EventPublisher interface {
	BroadcastProject(projectID uint, eventType string, data any)
}
