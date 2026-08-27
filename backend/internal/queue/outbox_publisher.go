package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/vishalyadav/jobplatform/internal/metrics"
	"github.com/vishalyadav/jobplatform/internal/repository"
)

// outboxpublisher polls the outbox_events table and publishes pending events to rabbitmq.
type OutboxPublisher struct {
	repo   *repository.OutboxRepository
	amqp   *Connection
	logger *slog.Logger
	ticker *time.Ticker
	done   chan struct{}
}

func NewOutboxPublisher(repo *repository.OutboxRepository, amqp *Connection, logger *slog.Logger, pollInterval time.Duration) *OutboxPublisher {
	return &OutboxPublisher{
		repo:   repo,
		amqp:   amqp,
		logger: logger.With("component", "outbox_publisher"),
		ticker: time.NewTicker(pollInterval),
		done:   make(chan struct{}),
	}
}

// start runs the polling loop in a new goroutine.
func (p *OutboxPublisher) Start(ctx context.Context) {
	p.logger.Info("starting outbox publisher")
	go p.poll(ctx)
}

// stop gracefully shuts down the publisher.
func (p *OutboxPublisher) Stop() {
	p.ticker.Stop()
	close(p.done)
	p.logger.Info("stopped outbox publisher")
}

func (p *OutboxPublisher) poll(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case <-p.ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *OutboxPublisher) processBatch(ctx context.Context) {
	// fetch up to 100 unpublished events.
	events, err := p.repo.FetchUnpublished(ctx, 100)
	if err != nil {
		p.logger.Error("failed to fetch outbox events", "error", err)
		return
	}

	for _, evt := range events {
		// determine routing key based on event type.
		var routingKey string
		switch evt.EventType {
		case "job.created":
			routingKey = RoutingKeyCreated
		case "job.retry":
			routingKey = RoutingKeyRetry
		default:
			p.logger.Warn("unknown event type in outbox", "event_type", evt.EventType, "event_id", evt.ID)
			continue
		}

		// wrap payload in an envelope for the queue if needed,
		// or just send the raw payload (which contains the job id).
		// we'll send the raw payload as it's already structured by the service.

		// in order to decouple the outbox event from the rabbitmq message,
		// we publish the evt.payload directly as the json body.
		if err := p.amqp.Publish(ctx, routingKey, evt.Payload); err != nil {
			p.logger.Error("failed to publish outbox event", "error", err, "event_id", evt.ID)
			metrics.OutboxPublishErrors.Inc()
			// break on publish error to avoid out-of-order delivery.
			// next tick will retry.
			break
		}

		metrics.OutboxPublishedTotal.Inc()

		// mark as published in the database.
		if err := p.repo.MarkPublished(ctx, evt.ID); err != nil {
			p.logger.Error("failed to mark outbox event as published", "error", err, "event_id", evt.ID)
			// continue to next event even if mark fails — duplicate delivery is handled
			// by the idempotency of the worker.
		} else {
			p.logger.Debug("published outbox event", "event_id", evt.ID, "type", evt.EventType)
		}
	}
}

// messagepayload is the standard contract for messages sent over rabbitmq.
type MessagePayload struct {
	JobID string `json:"job_id"`
}

// buildmessagepayload creates the json payload for a job queue message.
func BuildMessagePayload(jobID string) ([]byte, error) {
	payload := MessagePayload{
		JobID: jobID,
	}
	return json.Marshal(payload)
}
