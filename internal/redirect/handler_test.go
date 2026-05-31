package redirect

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/event"
	"trackflow/internal/link"
)

type fakeLinkRepo struct {
	link link.Link
	err  error
	calls int
}

func (f *fakeLinkRepo) FindRedirectByShortCode(context.Context, string) (link.Link, error) {
	f.calls++
	return f.link, f.err
}

type fakeCache struct {
	link link.Link
	err  error
	sets int
}

type fakePublisher struct {
	payloads chan event.ClickRecordedPayload
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{payloads: make(chan event.ClickRecordedPayload, 1)}
}

func (f *fakePublisher) PublishClickRecorded(_ context.Context, payload event.ClickRecordedPayload) error {
	f.payloads <- payload
	return nil
}

func (f *fakeCache) Get(context.Context, string) (link.Link, error) {
	return f.link, f.err
}

func (f *fakeCache) Set(context.Context, link.Link) error {
	f.sets++
	return nil
}

func TestRedirectUsesCacheHit(t *testing.T) {
	repo := &fakeLinkRepo{}
	cache := &fakeCache{link: link.Link{ID: "link-id", ShortCode: "xK9mP", OriginalURL: "https://example.com"}}
	publisher := newFakePublisher()
	handler := NewHandler(repo, cache, publisher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := httptest.NewRecorder()
	req := requestWithShortCode("xK9mP")

	handler.Redirect(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "https://example.com" {
		t.Fatalf("expected redirect location, got %s", location)
	}
	if repo.calls != 0 {
		t.Fatalf("expected no db fallback, got %d calls", repo.calls)
	}
	assertPublishedClick(t, publisher, "link-id", "xK9mP")
}

func TestRedirectFallsBackToDBAndWritesCache(t *testing.T) {
	repo := &fakeLinkRepo{link: link.Link{
		ID:          "link-id",
		ShortCode:   "xK9mP",
		OriginalURL: "https://example.com",
	}}
	cache := &fakeCache{err: link.ErrNotFound}
	publisher := newFakePublisher()
	handler := NewHandler(repo, cache, publisher, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := httptest.NewRecorder()
	req := requestWithShortCode("xK9mP")

	handler.Redirect(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if repo.calls != 1 {
		t.Fatalf("expected one db fallback, got %d calls", repo.calls)
	}
	if cache.sets != 1 {
		t.Fatalf("expected cache write-through, got %d sets", cache.sets)
	}
	assertPublishedClick(t, publisher, "link-id", "xK9mP")
}

func TestRedirectReturnsNotFound(t *testing.T) {
	repo := &fakeLinkRepo{err: link.ErrNotFound}
	cache := &fakeCache{err: link.ErrNotFound}
	handler := NewHandler(repo, cache, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := httptest.NewRecorder()
	req := requestWithShortCode("missing")

	handler.Redirect(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRedirectTreatsDBErrorAsNotFound(t *testing.T) {
	repo := &fakeLinkRepo{err: errors.New("db down")}
	cache := &fakeCache{err: link.ErrNotFound}
	handler := NewHandler(repo, cache, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := httptest.NewRecorder()
	req := requestWithShortCode("xK9mP")

	handler.Redirect(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestClientIPUsesForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")

	if got := clientIP(req); got != "203.0.113.10" {
		t.Fatalf("expected forwarded client ip, got %s", got)
	}
}

func assertPublishedClick(t *testing.T, publisher *fakePublisher, linkID string, shortCode string) {
	t.Helper()

	select {
	case payload := <-publisher.payloads:
		if payload.LinkID != linkID {
			t.Fatalf("expected link id %s, got %s", linkID, payload.LinkID)
		}
		if payload.ShortCode != shortCode {
			t.Fatalf("expected short code %s, got %s", shortCode, payload.ShortCode)
		}
	case <-time.After(time.Second):
		t.Fatal("expected click event publish")
	}
}

func requestWithShortCode(shortCode string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("short_code", shortCode)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
}
