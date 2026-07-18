package providers_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	fileprovider "github.com/OlegGitH/epistemic-engine/sdk/go/providers/file"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/local"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/noop"
)

func TestP0ProviderSemantics(t *testing.T) {
	event := epistemic.Event{SpecVersion: epistemic.Version, ID: "evt-1", Type: "claim.contradicted", Source: epistemic.Source{Name: "test"}, Subject: epistemic.Subject{Type: "claim", ID: "claim-1"}, Time: time.Now().UTC(), Data: json.RawMessage(`{}`)}
	request := epistemic.DecisionRequest{SpecVersion: epistemic.Version, DecisionID: "decision-1", Recommendation: "deploy", Action: epistemic.Action{Type: "deployment", Subject: epistemic.Subject{Type: "repository", ID: "demo"}}, Events: []epistemic.Event{event}}
	localProvider := local.New()
	result, err := localProvider.Evaluate(context.Background(), request)
	if err != nil || result.Status != "block" {
		t.Fatalf("local result=%+v err=%v", result, err)
	}
	noopResult, err := noop.New().Evaluate(context.Background(), request)
	if err != nil || noopResult.Status != "indeterminate" {
		t.Fatalf("noop result=%+v err=%v", noopResult, err)
	}
	path := filepath.Join(t.TempDir(), "events.jsonl")
	fileProvider := fileprovider.New(path)
	if err = fileProvider.Emit(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err = fileProvider.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = fileProvider.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("file provider wrote no data")
	}
}
