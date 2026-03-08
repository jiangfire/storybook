package service

type EventPublisher interface {
	Broadcast(eventType string, data any)
}
