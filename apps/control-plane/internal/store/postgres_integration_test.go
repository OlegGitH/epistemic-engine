package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func TestPostgresRepositoryRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("EPISTEMIC_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("EPISTEMIC_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	repository, err := NewPostgres(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	if _, err = repository.pool.Exec(ctx, "TRUNCATE proofs, verifications, unknowns, assumptions, claim_evidence, relations, evidence, claims, run_events, decisions, runs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	run := domain.Run{ID: "run_postgres_test", Title: "Round trip", Goal: "Verify persistence", Source: "test", Recommendation: "Deploy", Status: domain.RunIngesting, DecisionID: "decision_postgres_test", CreatedAt: now}
	decision := domain.Decision{ID: run.DecisionID, RunID: run.ID, Recommendation: run.Recommendation, ActionType: "software_deployment", Subject: "fixture", RiskLevel: "high", PolicyVersion: "deployment-readiness/v1", Conditions: []string{}, CreatedAt: now}
	if err = repository.CreateRun(ctx, run, decision); err != nil {
		t.Fatal(err)
	}
	event := domain.Event{ID: "event_postgres_test", ExternalID: "producer-event-1", Type: "test.completed", Source: "ci", OccurredAt: now, Payload: json.RawMessage(`{"status":"passed"}`)}
	stored, created, err := repository.AddEvent(ctx, run.ID, event)
	if err != nil || !created || stored.Sequence != 1 {
		t.Fatalf("first event: stored=%+v created=%v err=%v", stored, created, err)
	}
	duplicate, created, err := repository.AddEvent(ctx, run.ID, domain.Event{ID: "different", ExternalID: event.ExternalID, Type: event.Type, Source: event.Source, OccurredAt: now, Payload: event.Payload})
	if err != nil || created || duplicate.ID != event.ID {
		t.Fatalf("duplicate event: stored=%+v created=%v err=%v", duplicate, created, err)
	}
	claim := domain.Claim{ID: "claim_postgres_test", DecisionID: decision.ID, Statement: "Tests pass", Scope: "fixture", Critical: true, Importance: "critical", RequiredEvidenceTypes: []string{"test_result"}, State: domain.ClaimVerificationPending, Justification: "Needs controlled verification", Support: domain.SupportScore{Semantics: "explainable evidence support; not probability of truth"}, CreatedAt: now}
	evidence := domain.Evidence{ID: "evidence_postgres_test", RunID: run.ID, Kind: "test_result", Source: "ci", Summary: "Test passed", ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ObservedAt: now, Raw: event.Payload}
	relation := domain.Relation{ID: "relation_postgres_test", FromID: evidence.ID, ToID: claim.ID, Type: domain.RelationSupports, Rationale: "Direct test result"}
	if err = repository.SaveAnalysis(ctx, run.ID, domain.Analysis{Claims: []domain.Claim{claim}, Evidence: []domain.Evidence{evidence}, Relations: []domain.Relation{relation}}); err != nil {
		t.Fatal(err)
	}
	verification := domain.Verification{ID: "verification_postgres_test", DecisionID: decision.ID, ClaimID: claim.ID, Check: "Run targeted test", Kind: "test", Specification: json.RawMessage(`{"image":"python:3.12-alpine","repository":"unsafe-orders-pr","command":["true"]}`), Environment: "sandbox", Status: "planned", RequiresApproval: true, CreatedAt: now}
	if err = repository.SaveVerifications(ctx, []domain.Verification{verification}); err != nil {
		t.Fatal(err)
	}
	verification.Status, verification.Outcome, verification.Approved = "completed", "passed", true
	verification.ApprovedBy = "integration-test"
	verification.Artifact = json.RawMessage(`{"exit_code":0}`)
	verification.ArtifactHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	verification.ExecutedAt = &now
	if err = repository.UpdateVerification(ctx, verification); err != nil {
		t.Fatal(err)
	}
	graph, err := repository.GetGraphByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Run.Events) != 1 || len(graph.Claims) != 1 || graph.Claims[0].State != domain.ClaimExternallyVerified || len(graph.Verifications) != 1 || len(graph.Decision.ClaimIDs) != 1 {
		t.Fatalf("unexpected graph: %+v", graph)
	}
}
