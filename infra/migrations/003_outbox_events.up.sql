CREATE TABLE outbox_events (
    id bigserial PRIMARY KEY,
    event_id uuid NOT NULL UNIQUE,
    event_type text NOT NULL,
    stream text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'published', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz NULL,
    last_error text NULL
);

CREATE INDEX idx_outbox_events_status_created_at ON outbox_events(status, created_at);
