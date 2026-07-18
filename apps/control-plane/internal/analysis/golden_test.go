package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/policy"
)

type goldenScenario struct {
	Name   string `json:"name"`
	Events []struct {
		Type    string          `json:"type"`
		Source  string          `json:"source"`
		Payload json.RawMessage `json:"payload"`
	} `json:"events"`
	ExpectedStates  map[string]domain.ClaimState `json:"expected_states"`
	ExpectedVerdict domain.Verdict               `json:"expected_verdict"`
}

func TestGoldenScenarios(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "evals", "fixtures", "scenarios.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var scenarios []goldenScenario
	if err = json.Unmarshal(data, &scenarios); err != nil {
		t.Fatal(err)
	}
	if len(scenarios) < 6 {
		t.Fatalf("expected at least six scenarios, got %d", len(scenarios))
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			run := domain.Run{ID: "run_golden"}
			for index, event := range scenario.Events {
				run.Events = append(run.Events, domain.Event{ID: fmt.Sprintf("%s_%d", scenario.Name, index+1), Sequence: int64(index + 1), Type: event.Type, Source: event.Source, Payload: event.Payload, OccurredAt: time.Now().UTC()})
			}
			result, err := NewRulesAnalyzer().Analyze(context.Background(), run, domain.Decision{ID: "decision_golden"})
			if err != nil {
				t.Fatal(err)
			}
			states := map[string]domain.ClaimState{}
			for _, claim := range result.Claims {
				states[claim.ID] = claim.State
			}
			for id, want := range scenario.ExpectedStates {
				if got := states[id]; got != want {
					t.Errorf("%s state=%s want=%s", id, got, want)
				}
			}
			graph := domain.Graph{Claims: result.Claims, Unknowns: result.Unknowns}
			if got := policy.Evaluate(graph, false).Verdict; got != scenario.ExpectedVerdict {
				t.Errorf("verdict=%s want=%s", got, scenario.ExpectedVerdict)
			}
		})
	}
}
