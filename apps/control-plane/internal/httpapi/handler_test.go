package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
)

func TestVerifiedFlowRequiresApproval(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	run := request[domain.Run](t, h, http.MethodPost, "/v1/runs", map[string]any{"recommendation": "Safe to deploy"}, http.StatusCreated)
	request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", map[string]any{"type": "build.completed", "payload": map[string]any{"status": "passed"}}, http.StatusAccepted)
	request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", map[string]any{"type": "test.completed", "payload": map[string]any{"status": "passed"}}, http.StatusAccepted)
	g := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	if len(g.Claims) < 4 {
		t.Fatalf("expected at least four claims, got %d", len(g.Claims))
	}
	plan := request[struct {
		Verifications []domain.Verification `json:"verifications"`
	}](t, h, http.MethodPost, "/v1/decisions/"+g.Decision.ID+"/plan", nil, http.StatusCreated)
	if len(plan.Verifications) != 2 {
		t.Fatalf("expected compatibility and privacy checks, got %d", len(plan.Verifications))
	}
	for _, verification := range plan.Verifications {
		request[domain.Verification](t, h, http.MethodPost, "/v1/verifications/"+verification.ID+"/execute", map[string]any{"environment": "sandbox", "outcome": "passed", "artifact": map[string]any{"test": verification.Kind, "exit_code": 0}, "approved": true, "approved_by": "test-reviewer"}, http.StatusOK)
	}
	cert := request[domain.Certificate](t, h, http.MethodPost, "/v1/decisions/"+g.Decision.ID+"/evaluate", map[string]any{"human_approved": true}, http.StatusOK)
	if cert.Verdict != domain.VerdictVerifiedWithConditions || !cert.ActionAllowed || cert.Proof.Digest == "" {
		t.Fatalf("unexpected certificate: %+v", cert)
	}
	digest, err := service.CertificateDigest(cert)
	if err != nil || digest != cert.Proof.Digest {
		t.Fatalf("certificate proof is not reproducible: digest=%s err=%v", digest, err)
	}
	report := request[domain.HumanCertificateReport](t, h, http.MethodGet, "/v1/decisions/"+g.Decision.ID+"/certificate/report", nil, http.StatusOK)
	if report.Decision != "PROCEED" || report.Counts.CriticalClaims != 4 || report.Counts.PassedVerifications != 2 || !strings.Contains(report.Markdown, cert.Proof.Digest) {
		t.Fatalf("unexpected human certificate report: %+v", report)
	}
	markdownRequest := httptest.NewRequest(http.MethodGet, "/v1/decisions/"+g.Decision.ID+"/certificate/report?format=markdown", nil)
	markdownResponse := httptest.NewRecorder()
	h.ServeHTTP(markdownResponse, markdownRequest)
	if markdownResponse.Code != http.StatusOK || !strings.HasPrefix(markdownResponse.Header().Get("Content-Type"), "text/markdown") || !strings.Contains(markdownResponse.Body.String(), "The proposed action is authorized") {
		t.Fatalf("unexpected markdown report: status=%d headers=%v body=%s", markdownResponse.Code, markdownResponse.Header(), markdownResponse.Body.String())
	}
}

func TestHealthReportsStorageDurability(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()), WithStorage("postgresql", true))
	for _, path := range []string{"/health", "/healthz"} {
		health := request[struct {
			Status  string `json:"status"`
			Storage string `json:"storage"`
			Durable bool   `json:"durable"`
		}](t, h, http.MethodGet, path, nil, http.StatusOK)
		if health.Status != "ok" || health.Storage != "postgresql" || !health.Durable {
			t.Fatalf("unexpected health payload for %s: %+v", path, health)
		}
	}
}

func TestProductionVerificationIsRejected(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	run := request[domain.Run](t, h, http.MethodPost, "/v1/runs", map[string]any{"recommendation": "Safe"}, http.StatusCreated)
	g := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	plan := request[struct {
		Verifications []domain.Verification `json:"verifications"`
	}](t, h, http.MethodPost, "/v1/decisions/"+g.Decision.ID+"/plan", nil, http.StatusCreated)
	request[map[string]any](t, h, http.MethodPost, "/v1/verifications/"+plan.Verifications[0].ID+"/execute", map[string]any{"environment": "production", "outcome": "passed", "artifact": map[string]any{"exit_code": 0}}, http.StatusBadRequest)
}

func TestEventAndAnalysisReprocessingAreIdempotent(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	run := request[domain.Run](t, h, http.MethodPost, "/v1/runs", map[string]any{"recommendation": "Safe"}, http.StatusCreated)
	payload := map[string]any{"external_id": "trace-event-1", "sequence": 1, "type": "build.completed", "payload": map[string]any{"status": "passed"}}
	first := request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", payload, http.StatusAccepted)
	second := request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", payload, http.StatusAccepted)
	if first.ID != second.ID {
		t.Fatalf("duplicate event produced a new identity: %s != %s", first.ID, second.ID)
	}
	firstGraph := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	secondGraph := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	if len(secondGraph.Run.Events) != 1 || len(firstGraph.Claims) != len(secondGraph.Claims) || firstGraph.Claims[0].ID != secondGraph.Claims[0].ID {
		t.Fatalf("reprocessing changed stored graph: first=%+v second=%+v", firstGraph, secondGraph)
	}
}

func TestVerificationRequiresApproval(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	run := request[domain.Run](t, h, http.MethodPost, "/v1/runs", map[string]any{"recommendation": "Safe"}, http.StatusCreated)
	graph := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	plan := request[struct {
		Verifications []domain.Verification `json:"verifications"`
	}](t, h, http.MethodPost, "/v1/decisions/"+graph.Decision.ID+"/verification-plan", nil, http.StatusCreated)
	request[map[string]any](t, h, http.MethodPost, "/v1/verifications/"+plan.Verifications[0].ID+"/execute", map[string]any{"environment": "sandbox", "outcome": "passed", "artifact": map[string]any{"exit_code": 0}}, http.StatusBadRequest)
}

func TestPipelineToolListsAndGeneratesGitHubWorkflow(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	catalog := request[struct {
		Tools []struct {
			ID string `json:"id"`
		} `json:"tools"`
	}](t, h, http.MethodGet, "/v1/tools", nil, http.StatusOK)
	if len(catalog.Tools) != 1 || catalog.Tools[0].ID != "github-actions-pipeline" {
		t.Fatalf("unexpected tool catalog: %+v", catalog.Tools)
	}
	generated := request[struct {
		ToolID string `json:"tool_id"`
		Files  []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}](t, h, http.MethodPost, "/v1/tools/github-actions/pipelines", map[string]any{
		"config_path": ".epistemic.yaml",
	}, http.StatusCreated)
	if generated.ToolID != "github-actions-pipeline" || len(generated.Files) != 1 || !strings.Contains(generated.Files[0].Content, "Epistemic quality gate") {
		t.Fatalf("unexpected generated pipeline: %+v", generated)
	}
}

func TestAccountDashboardTracksProjectsAIKnowledgeAndCertificates(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	account := request[domain.Account](t, h, http.MethodPost, "/v1/accounts", map[string]any{"name": "Acme AI"}, http.StatusCreated)
	project := request[domain.Project](t, h, http.MethodPost, "/v1/accounts/"+account.ID+"/projects", map[string]any{"name": "Food Lens", "repository": "acme/food-lens", "owner": "Trust team"}, http.StatusCreated)
	system := request[domain.AISystem](t, h, http.MethodPost, "/v1/projects/"+project.ID+"/ai-systems", map[string]any{
		"name": "Food image analyzer", "provider": "OpenAI", "model": "gpt-5.6", "purpose": "Assess visible food evidence", "data_classes": []string{"user_image"},
	}, http.StatusCreated)
	run := request[domain.Run](t, h, http.MethodPost, "/v1/runs", map[string]any{
		"account_id": account.ID, "project_id": project.ID, "ai_system_id": system.ID, "title": "Release candidate", "recommendation": "Safe to deploy",
	}, http.StatusCreated)
	request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", map[string]any{"type": "build.completed", "payload": map[string]any{"status": "passed"}}, http.StatusAccepted)
	request[domain.Event](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/events", map[string]any{"type": "test.completed", "payload": map[string]any{"status": "passed"}}, http.StatusAccepted)
	graph := request[domain.Graph](t, h, http.MethodPost, "/v1/runs/"+run.ID+"/analyze", nil, http.StatusOK)
	plan := request[struct {
		Verifications []domain.Verification `json:"verifications"`
	}](t, h, http.MethodPost, "/v1/decisions/"+graph.Decision.ID+"/verification-plan", nil, http.StatusCreated)
	for _, verification := range plan.Verifications {
		request[domain.Verification](t, h, http.MethodPost, "/v1/verifications/"+verification.ID+"/execute", map[string]any{
			"environment": "sandbox", "outcome": "passed", "artifact": map[string]any{"exit_code": 0}, "approved": true, "approved_by": "portfolio-test",
		}, http.StatusOK)
	}
	request[domain.Certificate](t, h, http.MethodPost, "/v1/decisions/"+graph.Decision.ID+"/evaluate", map[string]any{"human_approved": true}, http.StatusOK)

	dashboard := request[domain.AccountDashboard](t, h, http.MethodGet, "/v1/accounts/"+account.ID+"/dashboard", nil, http.StatusOK)
	if dashboard.Account.ID != account.ID || dashboard.Metrics.Projects != 1 || dashboard.Metrics.AISystems != 1 || dashboard.Metrics.ValidCertificates != 1 {
		t.Fatalf("unexpected dashboard metrics: %+v", dashboard)
	}
	if len(dashboard.Projects) != 1 || dashboard.Projects[0].CertificationStatus != "valid" || dashboard.Projects[0].Runs != 1 {
		t.Fatalf("unexpected project summary: %+v", dashboard.Projects)
	}
	if len(dashboard.AISystems) != 1 || dashboard.AISystems[0].CertificationStatus != "valid" || dashboard.AISystems[0].CertificateDigest == "" {
		t.Fatalf("unexpected AI system summary: %+v", dashboard.AISystems)
	}
	if dashboard.Knowledge.Claims == 0 || dashboard.Knowledge.EvidenceArtifacts == 0 || len(dashboard.Certificates) != 1 {
		t.Fatalf("dashboard did not aggregate knowledge and certificates: %+v", dashboard)
	}
}

func TestConnectedProjectPublishesVerifiedCertificateAndReportIdempotently(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	account := request[domain.Account](t, h, http.MethodPost, "/v1/accounts", map[string]any{"name": "Connected account"}, http.StatusCreated)
	project := request[domain.Project](t, h, http.MethodPost, "/v1/accounts/"+account.ID+"/projects", map[string]any{"name": "Connected project", "repository": "acme/connected"}, http.StatusCreated)
	system := request[domain.AISystem](t, h, http.MethodPost, "/v1/projects/"+project.ID+"/ai-systems", map[string]any{"name": "Release reviewer", "provider": "OpenAI", "model": "gpt-5.6", "purpose": "Review release evidence"}, http.StatusCreated)
	connection := request[service.CreateConnectionResult](t, h, http.MethodPost, "/v1/projects/"+project.ID+"/connections", map[string]any{"provider": "github-actions", "endpoint": "https://epistemic.example.com"}, http.StatusCreated)
	if connection.Token == "" || connection.Connection.TokenHash != "" || !strings.Contains(connection.Workflow, "EPISTEMIC_INGEST_TOKEN") {
		t.Fatalf("unexpected connection response: %+v", connection)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	decisionResult := epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: "decision-connected", Status: "allow", ActionAllowed: true, Reasons: []epistemic.Reason{}, Conditions: []string{}, EvaluatedAt: now, Context: epistemic.Context{RunID: "github-run-42"}}
	resultJSON, _ := json.Marshal(decisionResult)
	certificate := epistemic.Certificate{SpecVersion: epistemic.Version, ID: "proof-connected", DecisionID: decisionResult.DecisionID, Result: resultJSON, ArtifactHashes: []string{}, IssuedAt: now, Proof: epistemic.Proof{Algorithm: "SHA-256"}}
	digest, err := epistemic.Hash(certificate)
	if err != nil {
		t.Fatal(err)
	}
	certificate.Proof.Digest = digest
	payload := map[string]any{
		"external_id": "workflow-42-1", "ai_system_id": system.ID,
		"context":     map[string]any{"repository": "acme/connected", "commit_sha": "abc123", "branch": "main", "workflow": "Epistemic CI"},
		"report":      map[string]any{"tool": "project-quality", "status": "passed", "exit_code": 0, "summary": "All project checks passed"},
		"certificate": certificate,
	}
	first := requestWithToken[service.IngestResult](t, h, connection.Token, payload, http.StatusAccepted)
	retry := requestWithToken[service.IngestResult](t, h, connection.Token, payload, http.StatusAccepted)
	if first.ReportID == "" || first.CertificateID == "" || retry.ReportID != first.ReportID || retry.CertificateID != first.CertificateID {
		t.Fatalf("retry did not return stable stored identities: first=%+v retry=%+v", first, retry)
	}

	dashboard := request[domain.AccountDashboard](t, h, http.MethodGet, "/v1/accounts/"+account.ID+"/dashboard", nil, http.StatusOK)
	if dashboard.Metrics.ConnectedProjects != 1 || dashboard.Metrics.Reports != 1 || len(dashboard.Certificates) != 1 {
		t.Fatalf("ingest was not tracked idempotently: %+v", dashboard)
	}
	if dashboard.Projects[0].ConnectionStatus != "active" || dashboard.Projects[0].CertificationStatus != "valid" || dashboard.AISystems[0].CertificationStatus != "valid" {
		t.Fatalf("connected certification did not update portfolio status: projects=%+v systems=%+v", dashboard.Projects, dashboard.AISystems)
	}

	certificate.Proof.Digest = strings.Repeat("0", 64)
	requestWithToken[map[string]any](t, h, connection.Token, map[string]any{"external_id": "tampered", "certificate": certificate}, http.StatusBadRequest)
	requestWithToken[map[string]any](t, h, "invalid-token", payload, http.StatusUnauthorized)

	revoked := request[domain.ProjectConnection](t, h, http.MethodDelete, "/v1/connections/"+connection.Connection.ID, nil, http.StatusOK)
	if revoked.Status != "revoked" {
		t.Fatalf("connection was not revoked: %+v", revoked)
	}
	payload["external_id"] = "workflow-after-revoke"
	requestWithToken[map[string]any](t, h, connection.Token, payload, http.StatusUnauthorized)
	replacement := request[service.CreateConnectionResult](t, h, http.MethodPost, "/v1/projects/"+project.ID+"/connections", map[string]any{"provider": "github-actions", "endpoint": "https://epistemic.example.com"}, http.StatusCreated)
	reconnected := requestWithToken[service.IngestResult](t, h, replacement.Token, payload, http.StatusAccepted)
	if reconnected.CertificateID != first.CertificateID {
		t.Fatalf("reconnection duplicated an existing project certificate: first=%+v reconnected=%+v", first, reconnected)
	}
	dashboard = request[domain.AccountDashboard](t, h, http.MethodGet, "/v1/accounts/"+account.ID+"/dashboard", nil, http.StatusOK)
	if dashboard.Metrics.ConnectedProjects != 1 || dashboard.Metrics.Reports != 2 || len(dashboard.Certificates) != 1 || dashboard.Projects[0].ConnectionStatus != "active" {
		t.Fatalf("reconnected project dashboard is inconsistent: %+v", dashboard)
	}
}

func request[T any](t *testing.T, h http.Handler, method, path string, body any, want int) T {
	t.Helper()
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d body=%s", method, path, w.Code, w.Body.String())
	}
	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return out
}

func requestWithToken[T any](t *testing.T, h http.Handler, token string, body any, want int) T {
	t.Helper()
	data, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/v1/ingest", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("POST /v1/ingest: status=%d body=%s", w.Code, w.Body.String())
	}
	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return out
}
