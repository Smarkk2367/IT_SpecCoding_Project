package redirect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"trackflow/internal/link"
	"trackflow/internal/redisx"
)

const defaultCacheTTL = 24 * time.Hour

type Cache struct {
	redisURL string
	ttl      time.Duration
}

type cacheEntry struct {
	ID          string `json:"id"`
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func NewCache(redisURL string) Cache {
	return Cache{
		redisURL: redisURL,
		ttl:      defaultCacheTTL,
	}
}

func (c Cache) Get(ctx context.Context, shortCode string) (link.Link, error) {
	value, err := redisx.Get(ctx, c.redisURL, cacheKey(shortCode))
	if errors.Is(err, redisx.ErrNil) {
		return link.Link{}, link.ErrNotFound
	}
	if err != nil {
		return link.Link{}, err
	}

	var entry cacheEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil {
		return link.Link{}, fmt.Errorf("decode redirect cache: %w", err)
	}

	var expiresAt *time.Time
	if entry.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, entry.ExpiresAt)
		if err != nil {
			return link.Link{}, fmt.Errorf("decode redirect cache expiry: %w", err)
		}
		if !parsed.After(time.Now().UTC()) {
			return link.Link{}, link.ErrNotFound
		}
		expiresAt = &parsed
	}

	return link.Link{
		ID:          entry.ID,
		ShortCode:   entry.ShortCode,
		OriginalURL: entry.OriginalURL,
		ExpiresAt:   expiresAt,
	}, nil
}

func (c Cache) Set(ctx context.Context, target link.Link) error {
	ttl := c.ttl
	var expiresAt string
	if target.ExpiresAt != nil {
		expiresAt = target.ExpiresAt.UTC().Format(time.RFC3339Nano)
		untilExpiry := time.Until(target.ExpiresAt.UTC())
		if untilExpiry <= 0 {
			return nil
		}
		if untilExpiry < ttl {
			ttl = untilExpiry
		}
	}

	payload, err := json.Marshal(cacheEntry{
		ID:          target.ID,
		ShortCode:   target.ShortCode,
		OriginalURL: target.OriginalURL,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return fmt.Errorf("encode redirect cache: %w", err)
	}

	return redisx.SetEX(ctx, c.redisURL, cacheKey(target.ShortCode), string(payload), ttl)
}

func (c Cache) Delete(ctx context.Context, shortCode string) error {
	return redisx.Del(ctx, c.redisURL, cacheKey(shortCode))
}

func cacheKey(shortCode string) string {
	return "redirect:" + shortCode
}
