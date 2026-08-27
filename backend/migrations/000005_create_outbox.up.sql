CREATE TABLE outbox_events (
    id         BIGSERIAL    PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,   -- e.g. 'job'
    aggregate_id   UUID         NOT NULL,   -- e.g. job ID
    event_type     VARCHAR(100) NOT NULL,   -- e.g. 'job.created', 'job.retry'
    payload        JSONB        NOT NULL,
    published      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at   TIMESTAMPTZ
);

-- The publisher polls for unpublished events ordered by creation time.
CREATE INDEX idx_outbox_unpublished ON outbox_events (created_at ASC) WHERE published = FALSE;
