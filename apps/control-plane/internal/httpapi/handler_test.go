package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
