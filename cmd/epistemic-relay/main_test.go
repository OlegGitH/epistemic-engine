package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type captureProvider struct{ events []epistemic.Event }

func (p *captureProvider) Emit(_ context.Context, event epistemic.Event) error {
	p.events = append(p.events, event)
	return nil
}
func (p *captureProvider) Evaluate(context.Context, epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	return epistemic.DecisionResult{}, nil
}
func (p *captureProvider) Flush(context.Context) error    { return nil }
func (p *captureProvider) Shutdown(context.Context) error { return nil }

func TestRelayValidatesAndRedacts(t *testing.T) {
	provider := &captureProvider{}
	r := &relay{provider: provider, redact: map[string]bool{"token": true}, maxBatch: 10}
	event := epistemic.Event{SpecVersion: epistemic.Version, ID: "relay-event", Type: "evidence.discovered", Source: epistemic.Source{Name: "test"}, Subject: epistemic.Subject{Type: "evidence", ID: "one"}, Time: time.Now().UTC(), Data: json.RawMessage(`{"token":"sensitive","nested":{"token":"also-sensitive","safe":"visible"}}`)}
	if err := r.process(context.Background(), &event); err != nil {
		t.Fatal(err)
	}
	if len(provider.events) != 1 {
		t.Fatal("event was not exported")
	}
	var data map[string]any
	if err := json.Unmarshal(provider.events[0].Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["token"] != "[REDACTED]" || data["nested"].(map[string]any)["token"] != "[REDACTED]" {
		t.Fatalf("redaction failed: %+v", data)
	}
}

func TestRelayConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relay.yaml")
	data := []byte("api_version: epistemic.dev/relay/v1alpha1\nreceivers:\n  http:\n    listen: ':9999'\nprocessors:\n  batch:\n    max_size: 42\nexporters:\n  archive:\n    path: out.jsonl\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := loadRelayConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Receivers.HTTP.Listen != ":9999" || configuration.Processors.Batch.MaxSize != 42 {
		t.Fatalf("unexpected config: %+v", configuration)
	}
}
