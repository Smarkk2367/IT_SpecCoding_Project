package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenManagerIssueAndParse(t *testing.T) {
	manager := NewTokenManager("secret", time.Hour)

	token, err := manager.Issue(Claims{
		Subject: "user-id",
		Email:   "marketer@example.com",
		Role:    "marketer",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if claims.Subject != "user-id" {
		t.Fatalf("expected subject user-id, got %s", claims.Subject)
	}
	if claims.Email != "marketer@example.com" {
		t.Fatalf("expected email marketer@example.com, got %s", claims.Email)
	}
	if claims.Role != "marketer" {
		t.Fatalf("expected role marketer, got %s", claims.Role)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager := NewTokenManager("secret", -time.Hour)

	token, err := manager.Issue(Claims{
		Subject: "user-id",
		Email:   "marketer@example.com",
		Role:    "marketer",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	_, err = manager.Parse(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenManagerRejectsTamperedToken(t *testing.T) {
	manager := NewTokenManager("secret", time.Hour)

	token, err := manager.Issue(Claims{
		Subject: "user-id",
		Email:   "marketer@example.com",
		Role:    "marketer",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	_, err = manager.Parse(token + "x")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
