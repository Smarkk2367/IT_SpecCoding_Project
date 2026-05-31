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
	NotificationSendType = "notification.send"
	NotificationsStream  = "notifications"
)

type NotificationTemplateData struct {
	ReportID *string `json:"report_id,omitempty"`
	LinkID   *string `json:"link_id,omitempty"`
	Message  *string `json:"message,omitempty"`
}

type NotificationSendPayload struct {
	Type           string                   `json:"type"` // report_ready | alert_no_clicks | weekly_report
	RecipientEmail string                   `json:"recipient_email"`
	Subject        string                   `json:"subject"`
	TemplateData   NotificationTemplateData `json:"template_data"`
}

func (p Publisher) PublishNotificationSend(ctx context.Context, payload NotificationSendPayload) error {
	eventID, err := id.NewUUID()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	envelope := Envelope[NotificationSendPayload]{
		EventID:   eventID,
		EventType: NotificationSendType,
		Version:   Version,
		Timestamp: now,
		Payload:   payload,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}

	_, err = redisx.XAdd(ctx, p.redisURL, NotificationsStream, map[string]string{
		"event": string(body),
	})
	if err != nil {
		if p.outbox != nil {
			if outboxErr := p.outbox.Save(ctx, NotificationsStream, eventID, NotificationSendType, body, err); outboxErr == nil {
				return nil
			}
		}
		return fmt.Errorf("publish notification event: %w", err)
	}

	return nil
}
