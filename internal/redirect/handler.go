package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/httpx"
	"trackflow/internal/event"
	"trackflow/internal/link"
)

type LinkRepository interface {
	FindRedirectByShortCode(ctx context.Context, shortCode string) (link.Link, error)
}

type LinkCache interface {
	Get(ctx context.Context, shortCode string) (link.Link, error)
	Set(ctx context.Context, target link.Link) error
}

type ClickPublisher interface {
	PublishClickRecorded(ctx context.Context, payload event.ClickRecordedPayload) error
}

type Handler struct {
	links     LinkRepository
	cache     LinkCache
	publisher ClickPublisher
	logger    *slog.Logger
}

func NewHandler(links LinkRepository, cache LinkCache, publisher ClickPublisher, logger *slog.Logger) Handler {
	return Handler{
		links:     links,
		cache:     cache,
		publisher: publisher,
		logger:    logger,
	}
}

func (h Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "short_code")
	if shortCode == "" {
		httpx.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "link not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Millisecond)
	defer cancel()

	target, err := h.cache.Get(ctx, shortCode)
	if err == nil {
		http.Redirect(w, r, target.OriginalURL, http.StatusFound)
		h.publishClick(r, target)
		return
	}
	if err != nil && !errors.Is(err, link.ErrNotFound) {
		h.logger.Warn("redirect cache lookup failed", "short_code", shortCode, "error", err)
	}

	target, err = h.links.FindRedirectByShortCode(ctx, shortCode)
	if errors.Is(err, link.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "link not found")
		return
	}
	if err != nil {
		h.logger.Error("redirect db lookup failed", "short_code", shortCode, "error", err)
		httpx.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "link not found")
		return
	}

	if err := h.cache.Set(ctx, target); err != nil {
		h.logger.Warn("redirect cache write-through failed", "short_code", shortCode, "error", err)
	}

	http.Redirect(w, r, target.OriginalURL, http.StatusFound)
	h.publishClick(r, target)
}

func (h Handler) publishClick(r *http.Request, target link.Link) {
	if h.publisher == nil {
		return
	}

	payload := event.ClickRecordedPayload{
		LinkID:    target.ID,
		ShortCode: target.ShortCode,
		ClickedAt: time.Now().UTC(),
		IPAddress: clientIP(r),
		UserAgent: r.UserAgent(),
		Referrer:  referrer(r),
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		if err := h.publisher.PublishClickRecorded(ctx, payload); err != nil {
			h.logger.Warn("click event publish failed", "short_code", target.ShortCode, "error", err)
		}
	}()
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func referrer(r *http.Request) *string {
	value := r.Referer()
	if value == "" {
		return nil
	}
	return &value
}
