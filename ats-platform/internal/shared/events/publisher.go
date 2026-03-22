package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// EventPublisher publishes events to Redis Streams
type EventPublisher struct {
	client *redis.Client
	stream string
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(client *redis.Client, stream string) *EventPublisher {
	return &EventPublisher{
		client: client,
		stream: stream,
	}
}

// Publish publishes a resume event to the stream
func (p *EventPublisher) Publish(ctx context.Context, event ResumeEvent) error {
	values := map[string]interface{}{
		"resume_id": event.ResumeID,
		"action":    event.Action,
	}

	if event.Payload != "" {
		values["payload"] = event.Payload
	}

	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: values,
	}).Err()
}

// PublishWithPayload publishes an event with JSON payload
func (p *EventPublisher) PublishWithPayload(ctx context.Context, resumeID, action string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return p.Publish(ctx, ResumeEvent{
		ResumeID: resumeID,
		Action:   action,
		Payload:  string(payloadBytes),
	})
}

// PublishCreated publishes a resume.created event
func (p *EventPublisher) PublishCreated(ctx context.Context, resumeID string, payload interface{}) error {
	return p.PublishWithPayload(ctx, resumeID, "created", payload)
}

// PublishUpdated publishes a resume.updated event
func (p *EventPublisher) PublishUpdated(ctx context.Context, resumeID string, payload interface{}) error {
	return p.PublishWithPayload(ctx, resumeID, "updated", payload)
}

// PublishDeleted publishes a resume.deleted event
func (p *EventPublisher) PublishDeleted(ctx context.Context, resumeID string) error {
	return p.Publish(ctx, ResumeEvent{
		ResumeID: resumeID,
		Action:   "deleted",
	})
}

// PublishStatusChanged publishes a resume.status_changed event
func (p *EventPublisher) PublishStatusChanged(ctx context.Context, resumeID, oldStatus, newStatus string) error {
	return p.PublishWithPayload(ctx, resumeID, "status_changed", map[string]string{
		"old_status": oldStatus,
		"new_status": newStatus,
	})
}

// PublishParsed publishes a resume.parsed event
func (p *EventPublisher) PublishParsed(ctx context.Context, resumeID string, payload interface{}) error {
	return p.PublishWithPayload(ctx, resumeID, "parsed", payload)
}
