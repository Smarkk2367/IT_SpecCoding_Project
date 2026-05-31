package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"trackflow/internal/click"
	"trackflow/internal/event"
)

type ClickConsumer struct {
	redis   *redis.Client
	clicks  *click.Repository
	group   string
	name    string
	logger  *slog.Logger
}

func NewClickConsumer(redisClient *redis.Client, clicks *click.Repository, group string, name string, logger *slog.Logger) *ClickConsumer {
	return &ClickConsumer{
		redis:  redisClient,
		clicks: clicks,
		group:  group,
		name:   name,
		logger: logger,
	}
}

func (c *ClickConsumer) Run(ctx context.Context) error {
	if err := c.redis.XGroupCreateMkStream(ctx, event.ClicksStream, c.group, "0").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create click consumer group: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.name,
			Streams:  []string{event.ClicksStream, ">"},
			Count:    10,
			Block:    time.Second,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			c.logger.Warn("read click stream failed", "error", err)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := c.processMessage(ctx, message); err != nil {
					c.logger.Warn("process click message failed", "message_id", message.ID, "error", err)
					continue
				}
				if err := c.redis.XAck(ctx, event.ClicksStream, c.group, message.ID).Err(); err != nil {
					c.logger.Warn("ack click message failed", "message_id", message.ID, "error", err)
				}
			}
		}
	}
}

func (c *ClickConsumer) processMessage(ctx context.Context, message redis.XMessage) error {
	raw, ok := message.Values["event"].(string)
	if !ok || raw == "" {
		return errors.New("missing event field")
	}

	var envelope event.Envelope[event.ClickRecordedPayload]
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return fmt.Errorf("decode click envelope: %w", err)
	}
	if envelope.EventType != event.ClickRecordedType || envelope.Version != event.Version {
		return fmt.Errorf("unsupported event %s version %s", envelope.EventType, envelope.Version)
	}

	exists, err := c.clicks.ExistsEvent(ctx, envelope.EventID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	enriched := enrichClick(envelope)
	if err := c.clicks.Insert(ctx, enriched); err != nil && !errors.Is(err, click.ErrDuplicate) {
		return err
	}

	return c.updateCounters(ctx, envelope.Payload)
}

func (c *ClickConsumer) updateCounters(ctx context.Context, payload event.ClickRecordedPayload) error {
	pipe := c.redis.Pipeline()
	pipe.Incr(ctx, "stats:link:"+payload.LinkID+":total")
	pipe.Set(ctx, "stats:link:"+payload.LinkID+":last_click_at", payload.ClickedAt.UTC().Format(time.RFC3339Nano), 0)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("update click counters: %w", err)
	}
	return nil
}

func enrichClick(envelope event.Envelope[event.ClickRecordedPayload]) click.Click {
	payload := envelope.Payload
	clickedAt := payload.ClickedAt
	if clickedAt.IsZero() {
		clickedAt = envelope.Timestamp
	}
	if clickedAt.IsZero() {
		clickedAt = time.Now().UTC()
	}

	device, browser, osName := parseUserAgent(payload.UserAgent)

	return click.Click{
		LinkID:     payload.LinkID,
		ClickedAt:  clickedAt,
		DeviceType: nullable(device),
		Browser:    nullable(browser),
		OS:         nullable(osName),
		Referrer:   payload.Referrer,
		IPHash:     hashIP(payload.IPAddress),
		EventID:    envelope.EventID,
		UserAgent:  payload.UserAgent,
	}
}

func parseUserAgent(userAgent string) (string, string, string) {
	lower := strings.ToLower(userAgent)

	device := "desktop"
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "iphone") || strings.Contains(lower, "android") {
		device = "mobile"
	}
	if strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet") {
		device = "tablet"
	}

	browser := ""
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome/"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(lower, "safari/"):
		browser = "Safari"
	}

	osName := ""
	switch {
	case strings.Contains(lower, "windows"):
		osName = "Windows"
	case strings.Contains(lower, "mac os"):
		osName = "macOS"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		osName = "iOS"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	}

	return device, browser, osName
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
