package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestClickRecordedEnvelopeJSON(t *testing.T) {
	clickedAt := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	referrer := "https://example.org"
	envelope := Envelope[ClickRecordedPayload]{
		EventID:   "event-id",
		EventType: ClickRecordedType,
		Version:   Version,
		Timestamp: clickedAt,
		Payload: ClickRecordedPayload{
			LinkID:    "link-id",
			ShortCode: "xK9mP",
			ClickedAt: clickedAt,
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
			Referrer:  &referrer,
		},
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded["event_type"] != ClickRecordedType {
		t.Fatalf("expected event_type %s, got %v", ClickRecordedType, decoded["event_type"])
	}
	if decoded["version"] != Version {
		t.Fatalf("expected version %s, got %v", Version, decoded["version"])
	}
}
