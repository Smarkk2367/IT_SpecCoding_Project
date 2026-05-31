package redisx

import "testing"

func TestParseURL(t *testing.T) {
	opts, err := parseURL("redis://user:pass@redis:6380/2")
	if err != nil {
		t.Fatalf("parseURL returned error: %v", err)
	}

	if opts.addr != "redis:6380" {
		t.Fatalf("expected addr redis:6380, got %s", opts.addr)
	}
	if opts.username != "user" {
		t.Fatalf("expected username user, got %s", opts.username)
	}
	if opts.password != "pass" {
		t.Fatalf("expected password pass, got %s", opts.password)
	}
	if opts.db != 2 {
		t.Fatalf("expected db 2, got %d", opts.db)
	}
}

func TestParseURLPasswordOnly(t *testing.T) {
	opts, err := parseURL("redis://secret@redis/0")
	if err != nil {
		t.Fatalf("parseURL returned error: %v", err)
	}

	if opts.addr != "redis:6379" {
		t.Fatalf("expected default port, got %s", opts.addr)
	}
	if opts.username != "" {
		t.Fatalf("expected empty username, got %s", opts.username)
	}
	if opts.password != "secret" {
		t.Fatalf("expected password secret, got %s", opts.password)
	}
}

func TestParseURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := parseURL("http://redis:6379"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}
