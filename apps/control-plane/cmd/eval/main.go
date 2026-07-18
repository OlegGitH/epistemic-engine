package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/policy"
)

type scenario struct {
	Name   string `json:"name"`
	Events []struct {
		Type    string          `json:"type"`
		Source  string          `json:"source"`
		Payload json.RawMessage `json:"payload"`
	} `json:"events"`
	ExpectedStates  map[string]domain.ClaimState `json:"expected_states"`
	ExpectedVerdict domain.Verdict               `json:"expected_verdict"`
}

func main() {
	path := flag.String("fixtures", "../../evals/fixtures/scenarios.json", "golden scenario file")
	flag.Parse()
	data, err := os.ReadFile(*path)
	if err != nil {
		fatal(err)
	}
	var scenarios []scenario
	if err = json.Unmarshal(data, &scenarios); err != nil {
		fatal(err)
	}
	claimTotal, claimCorrect, contradictions, contradictionsFound, decisionCorrect := 0, 0, 0, 0, 0
	for _, s := range scenarios {
		run := domain.Run{ID: "eval_" + s.Name}
		for i, e := range s.Events {
			run.Events = append(run.Events, domain.Event{ID: fmt.Sprintf("event_%d", i), Type: e.Type, Source: e.Source, Payload: e.Payload, OccurredAt: time.Now().UTC()})
		}
		result, err := analysis.NewRulesAnalyzer().Analyze(context.Background(), run, domain.Decision{ID: "decision_" + s.Name})
		if err != nil {
			fatal(err)
		}
		states := map[string]domain.ClaimState{}
		for _, c := range result.Claims {
			states[c.ID] = c.State
		}
		for id, want := range s.ExpectedStates {
			claimTotal++
			if states[id] == want {
				claimCorrect++
			}
			if want == domain.ClaimContradicted {
				contradictions++
				if states[id] == want {
					contradictionsFound++
				}
			}
		}
		if policy.Evaluate(domain.Graph{Claims: result.Claims, Unknowns: result.Unknowns}, false).Verdict == s.ExpectedVerdict {
			decisionCorrect++
		}
	}
	metrics := map[string]any{"scenario_count": len(scenarios), "critical_claim_recall": ratio(claimCorrect, claimTotal), "contradiction_recall": ratio(contradictionsFound, contradictions), "decision_accuracy": ratio(decisionCorrect, len(scenarios))}
	encoded, _ := json.MarshalIndent(metrics, "", "  ")
	fmt.Println(string(encoded))
}
func ratio(a, b int) float64 {
	if b == 0 {
		return 1
	}
	return float64(a) / float64(b)
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
