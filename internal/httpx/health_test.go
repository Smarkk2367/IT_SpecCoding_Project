package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	HealthHandler("api").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"service":"api"`) {
		t.Fatalf("expected service name in response, got %s", body)
	}
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("expected ok status in response, got %s", body)
	}
}
