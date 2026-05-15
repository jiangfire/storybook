package service

type EventPublisher interface {
	BroadcastProject(projectID uint, eventType string, data any)
	BroadcastUser(userID uint, eventType string, data any)
}
