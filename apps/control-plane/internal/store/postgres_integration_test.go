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
	if _, err = repository.pool.Exec(ctx, "TRUNCATE published_certificates, project_reports, project_connections, proofs, verifications, unknowns, assumptions, claim_evidence, relations, evidence, claims, run_events, decisions, runs, ai_systems, projects, accounts RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	account := domain.Account{ID: "account_postgres_test", Name: "Test account", Slug: "test-account", CreatedAt: now}
	if err = repository.CreateAccount(ctx, account); err != nil {
		t.Fatal(err)
	}
	project := domain.Project{ID: "project_postgres_test", AccountID: account.ID, Name: "Test project", Slug: "test-project", CreatedAt: now}
	if err = repository.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	system := domain.AISystem{ID: "ai_postgres_test", AccountID: account.ID, ProjectID: project.ID, Name: "Test analyzer", Provider: "OpenAI", Model: "gpt-5.6", Purpose: "Exercise persistence", DataClasses: []string{"test_data"}, Tools: []string{}, Status: "active", CreatedAt: now, UpdatedAt: now}
	if err = repository.CreateAISystem(ctx, system); err != nil {
		t.Fatal(err)
	}
	run := domain.Run{ID: "run_postgres_test", AccountID: account.ID, ProjectID: project.ID, AISystemID: system.ID, Title: "Round trip", Goal: "Verify persistence", Source: "test", Recommendation: "Deploy", Status: domain.RunIngesting, DecisionID: "decision_postgres_test", CreatedAt: now}
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
	if graph.Assumptions == nil || graph.Unknowns == nil {
		t.Fatalf("empty graph collections must serialize as arrays: %+v", graph)
	}
	emptyVerifications, err := repository.getVerifications(ctx, "decision-without-verifications")
	if err != nil || emptyVerifications == nil {
		t.Fatalf("empty verifications must be a non-nil array: values=%+v err=%v", emptyVerifications, err)
	}
	connection := domain.ProjectConnection{ID: "connection_postgres_test", AccountID: account.ID, ProjectID: project.ID, Provider: "github-actions", Repository: "example/project", Status: "active", TokenHash: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", TokenPrefix: "epk_example", CreatedAt: now}
	if err = repository.CreateProjectConnection(ctx, connection); err != nil {
		t.Fatal(err)
	}
	report := domain.ProjectReport{ID: "report_postgres_test", ExternalID: "workflow-1-1", AccountID: account.ID, ProjectID: project.ID, ConnectionID: connection.ID, AISystemID: system.ID, Tool: "project-quality", Status: "passed", Summary: "Checks passed", Details: json.RawMessage(`{"status":"passed"}`), ReceivedAt: now}
	publication := domain.PublishedCertificate{ID: "publication_postgres_test", ExternalID: report.ExternalID, AccountID: account.ID, ProjectID: project.ID, ConnectionID: connection.ID, AISystemID: system.ID, DecisionID: "portable-decision", RunID: "github-run-1", Status: "allow", ActionAllowed: true, PolicyVersion: "epistemic.dev/v0.1", Digest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", ArtifactHashes: []string{}, IssuedAt: now, ReceivedAt: now, Raw: json.RawMessage(`{"proof":{"algorithm":"SHA-256"}}`)}
	ingest := domain.ProjectIngest{Connection: connection, Report: &report, Certificate: &publication}
	firstIngest, err := repository.SaveProjectIngest(ctx, ingest)
	if err != nil {
		t.Fatal(err)
	}
	firstReportID, firstCertificateID := firstIngest.Report.ID, firstIngest.Certificate.ID
	report.ID = "duplicate-report"
	publication.ID = "duplicate-publication"
	retryIngest, err := repository.SaveProjectIngest(ctx, ingest)
	if err != nil {
		t.Fatal(err)
	}
	if firstReportID != retryIngest.Report.ID || firstCertificateID != retryIngest.Certificate.ID {
		t.Fatalf("postgres ingest identities were not idempotent: first=%+v retry=%+v", firstIngest, retryIngest)
	}
	dashboard, err := repository.GetAccountDashboard(ctx, account.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Metrics.Projects != 1 || dashboard.Metrics.ConnectedProjects != 1 || dashboard.Metrics.Reports != 1 || dashboard.Metrics.AISystems != 1 || dashboard.Metrics.ValidCertificates != 1 || dashboard.Knowledge.Claims != 1 || dashboard.Projects[0].KnowledgeCoveragePct != 100 {
		t.Fatalf("unexpected persisted dashboard: %+v", dashboard)
	}
}
