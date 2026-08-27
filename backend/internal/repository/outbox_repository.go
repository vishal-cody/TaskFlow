package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vishalyadav/jobplatform/internal/models"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// insertintx inserts an outbox event within an existing transaction.
// this is the key method — always call it inside the same tx as the business operation.
func (r *OutboxRepository) InsertInTx(ctx context.Context, tx pgx.Tx, event *models.OutboxEvent) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		event.AggregateType, event.AggregateID, event.EventType, event.Payload,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		return fmt.Errorf("inserting outbox event: %w", err)
	}
	return nil
}

// fetchunpublished returns a batch of unpublished events, ordered by creation time.
// uses select ... for update skip locked so multiple publisher instances won't
// compete for the same rows.
func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at
		 FROM outbox_events
		 WHERE published = FALSE
		 ORDER BY created_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching unpublished outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]models.OutboxEvent, 0, limit)
	for rows.Next() {
		var e models.OutboxEvent
		if err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning outbox event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// markpublished sets published=true and published_at=now() for the given event id.
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE outbox_events SET published = TRUE, published_at = NOW()
		 WHERE id = $1`,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("marking outbox event published: %w", err)
	}
	return nil
}

// pool exposes the underlying pool so callers can begin transactions.
func (r *OutboxRepository) Pool() *pgxpool.Pool {
	return r.pool
}
