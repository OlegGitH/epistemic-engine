package analysis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func TestCodeChangeReviewOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		assessments []map[string]any
		states      []domain.ClaimState
		unknowns    int
	}{
		{
			name: "fully covered",
			assessments: []map[string]any{
				{"requirement_id": "cursor", "status": "passed", "confidence": .97, "evidence_refs": []string{"src/api.ts", "test/api.test.ts"}, "reviewer": "recorded-pr-reviewer"},
				{"requirement_id": "docs", "status": "passed", "confidence": .93, "evidence_refs": []string{"docs/api.md"}, "reviewer": "recorded-pr-reviewer"},
			},
			states: []domain.ClaimState{domain.ClaimSupported, domain.ClaimSupported},
		},
		{
			name: "partial coverage",
			assessments: []map[string]any{
				{"requirement_id": "cursor", "status": "passed", "confidence": .97, "evidence_refs": []string{"src/api.ts", "test/api.test.ts"}},
				{"requirement_id": "docs", "status": "partial", "confidence": .74, "evidence_refs": []string{"README.md"}},
			},
			states: []domain.ClaimState{domain.ClaimSupported, domain.ClaimPartiallySupported},
		},
		{
			name: "missing evidence",
			assessments: []map[string]any{
				{"requirement_id": "cursor", "status": "passed", "confidence": .97, "evidence_refs": []string{"src/api.ts", "test/api.test.ts"}},
			},
			states:   []domain.ClaimState{domain.ClaimSupported, domain.ClaimVerificationPending},
			unknowns: 1,
		},
		{
			name: "contradiction",
			assessments: []map[string]any{
				{"requirement_id": "cursor", "status": "failed", "confidence": .99, "evidence_refs": []string{"test/api.test.ts"}, "rationale": "The invalid cursor test fails."},
				{"requirement_id": "docs", "status": "passed", "confidence": .93, "evidence_refs": []string{"docs/api.md"}},
			},
			states: []domain.ClaimState{domain.ClaimContradicted, domain.ClaimSupported},
		},
		{
			name: "confidence alone is insufficient",
			assessments: []map[string]any{
				{"requirement_id": "cursor", "status": "passed", "confidence": .99, "evidence_refs": []string{}},
				{"requirement_id": "docs", "status": "passed", "confidence": .79, "evidence_refs": []string{"docs/api.md"}},
			},
			states:   []domain.ClaimState{domain.ClaimVerificationPending, domain.ClaimVerificationPending},
			unknowns: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			run := codeChangeRun(t, test.assessments)
			result, err := NewRulesAnalyzer().Analyze(context.Background(), run, domain.Decision{ID: "decision_review", ActionType: "code_change_review"})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Claims) != len(test.states) {
				t.Fatalf("got %d claims, want %d", len(result.Claims), len(test.states))
			}
			for index, want := range test.states {
				if got := result.Claims[index].State; got != want {
					t.Errorf("claim %d state = %s, want %s", index, got, want)
				}
			}
			if len(result.Unknowns) != test.unknowns {
				t.Errorf("unknowns = %d, want %d", len(result.Unknowns), test.unknowns)
			}
		})
	}
}

func TestCodeChangeReviewRejectsMalformedRequirements(t *testing.T) {
	run := codeChangeRun(t, nil)
	run.Events[0].Payload = json.RawMessage(`{"requirements":[{"id":"","text":"missing id","critical":true}]}`)
	if _, err := NewRulesAnalyzer().Analyze(context.Background(), run, domain.Decision{ActionType: "code_change_review"}); err == nil {
		t.Fatal("expected malformed requirement error")
	}
}

func TestEvidenceKindHonorsAValidDeclaredArtifactKind(t *testing.T) {
	if got := evidenceKind("change.artifact.observed", json.RawMessage(`{"kind":"custom","summary":"migration documentation"}`)); got != "custom" {
		t.Fatalf("evidence kind = %s, want custom", got)
	}
	if got := evidenceKind("change.artifact.observed", json.RawMessage(`{"kind":"invented","summary":"passing test"}`)); got != "test_result" {
		t.Fatalf("fallback evidence kind = %s, want test_result", got)
	}
}

func codeChangeRun(t *testing.T, assessments []map[string]any) domain.Run {
	t.Helper()
	now := time.Now()
	events := []domain.Event{{ID: "evt_requirements", Type: "requirements.declared", Source: "request", OccurredAt: now, Payload: marshalReviewPayload(t, map[string]any{
		"request": "Add cursor pagination and document it.",
		"requirements": []map[string]any{
			{"id": "cursor", "text": "The API supports cursor pagination and rejects invalid cursors.", "critical": true, "required_evidence_types": []string{"code_diff", "test_result"}},
			{"id": "docs", "text": "The API documentation explains cursor pagination.", "critical": true, "required_evidence_types": []string{"custom"}},
		},
	})}}
	artifacts := []map[string]any{
		{"id": "src/api.ts", "kind": "code_diff", "path": "src/api.ts"},
		{"id": "test/api.test.ts", "kind": "test_result", "path": "test/api.test.ts"},
		{"id": "docs/api.md", "kind": "custom", "path": "docs/api.md"},
		{"id": "README.md", "kind": "custom", "path": "README.md"},
	}
	for index, artifact := range artifacts {
		events = append(events, domain.Event{ID: "evt_artifact_" + string(rune('a'+index)), Type: "change.artifact.observed", Source: "github", OccurredAt: now, Payload: marshalReviewPayload(t, artifact)})
	}
	for index, assessment := range assessments {
		events = append(events, domain.Event{ID: "evt_assessment_" + string(rune('a'+index)), Type: "requirement.assessed", Source: "pr-review-ai", OccurredAt: now, Payload: marshalReviewPayload(t, assessment)})
	}
	return domain.Run{ID: "run_review", Events: events}
}

func marshalReviewPayload(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
