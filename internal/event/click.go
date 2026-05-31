package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"trackflow/internal/id"
	"trackflow/internal/redisx"
)

const (
	ClickRecordedType = "click.recorded"
	ClicksStream      = "clicks"
)

type ClickRecordedPayload struct {
	LinkID    string  `json:"link_id"`
	ShortCode string  `json:"short_code"`
	ClickedAt time.Time `json:"clicked_at"`
	IPAddress string  `json:"ip_address"`
	UserAgent string  `json:"user_agent"`
	Referrer  *string `json:"referrer"`
}

type Publisher struct {
	redisURL string
	outbox   *Outbox
}

func NewPublisher(redisURL string, outbox *Outbox) Publisher {
	return Publisher{redisURL: redisURL, outbox: outbox}
}

func (p Publisher) PublishClickRecorded(ctx context.Context, payload ClickRecordedPayload) error {
	eventID, err := id.NewUUID()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	envelope := Envelope[ClickRecordedPayload]{
		EventID:   eventID,
		EventType: ClickRecordedType,
		Version:   Version,
		Timestamp: now,
		Payload:   payload,
	}
	if envelope.Payload.ClickedAt.IsZero() {
		envelope.Payload.ClickedAt = now
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal click event: %w", err)
	}

	_, err = redisx.XAdd(ctx, p.redisURL, ClicksStream, map[string]string{
		"event": string(body),
	})
	if err != nil {
		if p.outbox != nil {
			if outboxErr := p.outbox.Save(ctx, ClicksStream, eventID, ClickRecordedType, body, err); outboxErr == nil {
				return nil
			}
		}
		return fmt.Errorf("publish click event: %w", err)
	}

	return nil
}
