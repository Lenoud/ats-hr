package events

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ResumeEvent represents a resume event from Redis Stream
type ResumeEvent struct {
	ResumeID string `json:"resume_id"`
	Action   string `json:"action"` // created, updated, deleted
	Payload  string `json:"payload,omitempty"`
}

// EventHandler handles events from Redis Stream
type EventHandler func(ctx context.Context, event ResumeEvent) error

// ErrConsumerStopped is returned when the consumer is stopped
var ErrConsumerStopped = errors.New("consumer stopped")

// StreamConsumer consumes events from Redis Stream
type StreamConsumer struct {
	client   *redis.Client
	stream   string
	group    string
	consumer string
	handler  EventHandler
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(client *redis.Client, stream, group, consumer string, handler EventHandler) *StreamConsumer {
	return &StreamConsumer{
		client:   client,
		stream:   stream,
		group:    group,
		consumer: consumer,
		handler:  handler,
	}
}

// Start starts consuming events
func (c *StreamConsumer) Start(ctx context.Context) error {
	// Create consumer group if not exists
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil && !isGroupExistsError(err) {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read from stream with shorter block time for faster shutdown response
			streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.group,
				Consumer: c.consumer,
				Streams:  []string{c.stream, ">"},
				Count:    10,
				Block:    time.Second, // Reduced from 5s to 1s for faster shutdown
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				// Check context on error
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					time.Sleep(time.Second)
					continue
				}
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					// Check context before processing each message
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						c.processMessage(ctx, message)
					}
				}
			}
		}
	}
}

// processMessage processes a single message
func (c *StreamConsumer) processMessage(ctx context.Context, message redis.XMessage) {
	event := ResumeEvent{}
	if resumeID, ok := message.Values["resume_id"].(string); ok {
		event.ResumeID = resumeID
	}
	if action, ok := message.Values["action"].(string); ok {
		event.Action = action
	}
	if payload, ok := message.Values["payload"].(string); ok {
		event.Payload = payload
	}

	if err := c.handler(ctx, event); err != nil {
		// Don't acknowledge message
		return
	}

	// Acknowledge message
	_ = c.client.XAck(ctx, c.stream, c.group, message.ID).Err()
}

// isGroupExistsError checks if error is BUSYGROUP (group already exists)
func isGroupExistsError(err error) bool {
	return strings.Contains(err.Error(), "BUSYGROUP")
}
