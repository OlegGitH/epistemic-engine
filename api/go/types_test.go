package epistemic

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventRoundTripAndValidation(t *testing.T) {
	event := Event{SpecVersion: Version, ID: "evt-1", Type: "claim.declared", Source: Source{Name: "test"}, Subject: Subject{Type: "claim", ID: "claim-1"}, Time: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), Data: json.RawMessage(`{"statement":"Build passes"}`)}
	if err := ValidateEvent(event); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Event
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != event.ID || decoded.Type != event.Type {
		t.Fatalf("round trip changed event: %+v", decoded)
	}
}

func TestCanonicalHashIgnoresMapInsertionOrder(t *testing.T) {
	left := map[string]any{"b": 2, "a": map[string]any{"z": 1, "c": true}}
	right := map[string]any{"a": map[string]any{"c": true, "z": 1}, "b": 2}
	l, err := Hash(left)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Hash(right)
	if err != nil {
		t.Fatal(err)
	}
	if l != r {
		t.Fatalf("hashes differ: %s != %s", l, r)
	}
}

func TestEventPreservesUnknownMinorFields(t *testing.T) {
	raw := []byte(`{"spec_version":"0.1","id":"evt-forward","type":"claim.declared","source":{"name":"test"},"subject":{"type":"claim","id":"claim"},"time":"2026-07-16T10:00:00Z","data":{},"future_hint":{"value":1}}`)
	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	if _, exists := event.Unknown["future_hint"]; !exists {
		t.Fatal("unknown field was discarded")
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if _, exists := roundTrip["future_hint"]; !exists {
		t.Fatal("relay round trip discarded unknown field")
	}
}
