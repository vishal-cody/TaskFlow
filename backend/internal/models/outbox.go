package models

import "time"

// outboxevent represents a pending message to be published to the message broker.
// it is created in the same database transaction as the business operation,
// solving the dual-write problem between postgresql and rabbitmq.
type OutboxEvent struct {
	ID            int64      `json:"id"`
	AggregateType string     `json:"aggregate_type"` // e.g. "job"
	AggregateID   string     `json:"aggregate_id"`   // e.g. job UUID
	EventType     string     `json:"event_type"`     // e.g. "job.created", "job.retry"
	Payload       []byte     `json:"payload"`
	Published     bool       `json:"published"`
	CreatedAt     time.Time  `json:"created_at"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
}

// outbox event types.
const (
	OutboxEventJobCreated = "job.created"
	OutboxEventJobRetry   = "job.retry"
)
