package event

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Outbox struct {
	db *pgxpool.Pool
}

func NewOutbox(db *pgxpool.Pool) Outbox {
	return Outbox{db: db}
}

func (o Outbox) Save(ctx context.Context, stream string, eventID string, eventType string, payload []byte, publishErr error) error {
	const query = `
		INSERT INTO outbox_events (event_id, event_type, stream, payload, status, last_error)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		ON CONFLICT (event_id) DO NOTHING
	`

	var lastError *string
	if publishErr != nil {
		value := publishErr.Error()
		lastError = &value
	}

	if _, err := o.db.Exec(ctx, query, eventID, eventType, stream, payload, lastError); err != nil {
		return fmt.Errorf("save outbox event: %w", err)
	}

	return nil
}
