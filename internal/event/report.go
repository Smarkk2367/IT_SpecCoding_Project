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
	ReportRequestedType = "report.requested"
	ReportsStream       = "reports"
)

type ReportRequestedPayload struct {
	ReportID    string    `json:"report_id"`
	RequestedBy string    `json:"requested_by"`
	DateFrom    time.Time `json:"date_from"`
	DateTo      time.Time `json:"date_to"`
	ClientID    *string   `json:"client_id"`
	LinkIDs     []string  `json:"link_ids"`
}

func (p Publisher) PublishReportRequested(ctx context.Context, payload ReportRequestedPayload) error {
	eventID, err := id.NewUUID()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	envelope := Envelope[ReportRequestedPayload]{
		EventID:   eventID,
		EventType: ReportRequestedType,
		Version:   Version,
		Timestamp: now,
		Payload:   payload,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal report event: %w", err)
	}

	_, err = redisx.XAdd(ctx, p.redisURL, ReportsStream, map[string]string{
		"event": string(body),
	})
	if err != nil {
		if p.outbox != nil {
			if outboxErr := p.outbox.Save(ctx, ReportsStream, eventID, ReportRequestedType, body, err); outboxErr == nil {
				return nil
			}
		}
		return fmt.Errorf("publish report event: %w", err)
	}

	return nil
}
