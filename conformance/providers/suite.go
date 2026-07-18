package providers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Factory func(t *testing.T) epistemic.Provider

func Run(t *testing.T, factory Factory) {
	t.Helper()
	provider := factory(t)
	ctx := context.Background()
	event := epistemic.Event{SpecVersion: epistemic.Version, ID: "provider-conformance-event", Type: "claim.declared", Source: epistemic.Source{Name: "conformance"}, Subject: epistemic.Subject{Type: "claim", ID: "provider-claim"}, Time: time.Now().UTC(), Data: json.RawMessage(`{"statement":"portable"}`)}
	if err := provider.Emit(ctx, event); err != nil {
		t.Fatalf("emit: %v", err)
	}
	request := epistemic.DecisionRequest{SpecVersion: epistemic.Version, DecisionID: "provider-decision", Recommendation: "evaluate", Action: epistemic.Action{Type: "test", Subject: epistemic.Subject{Type: "fixture", ID: "provider"}}}
	result, err := provider.Evaluate(ctx, request)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.SpecVersion != epistemic.Version || result.DecisionID == "" {
		t.Fatalf("non-portable result: %+v", result)
	}
	if err = provider.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if err = provider.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
