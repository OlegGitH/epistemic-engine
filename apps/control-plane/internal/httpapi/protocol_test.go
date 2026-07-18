package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

func TestProtocolDiscoveryIdempotencyAndParentValidation(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	capabilities := request[epistemic.Capabilities](t, h, http.MethodGet, "/.well-known/epistemic", nil, http.StatusOK)
	if len(capabilities.ProtocolVersions) != 1 || capabilities.ProtocolVersions[0] != epistemic.Version || capabilities.Limits.MaxBatchSize != 100 {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	event := protocolEvent("evt-parent", "claim.declared", "claim", "claim-1", map[string]any{"statement": "Build passes"})
	first := request[map[string]any](t, h, http.MethodPost, "/v1/events", event, http.StatusAccepted)
	second := request[map[string]any](t, h, http.MethodPost, "/v1/events", event, http.StatusAccepted)
	if first["accepted"] != true || second["duplicate"] != true {
		t.Fatalf("unexpected idempotency responses: first=%+v second=%+v", first, second)
	}
	child := protocolEvent("evt-child", "evidence.attached", "evidence", "evidence-1", map[string]any{"claim_id": "claim-1"})
	child.Context.ParentID = "missing-parent"
	response := request[epistemic.Error](t, h, http.MethodPost, "/v1/events", child, http.StatusBadRequest)
	if response.Code != "invalid_message" {
		t.Fatalf("unexpected error: %+v", response)
	}
	ordered := protocolEvent("evt-ordered", "claim.updated", "claim", "claim-1", map[string]any{"statement": "updated"})
	ordered.Context = epistemic.Context{DecisionID: "ordered-decision", RunID: "ordered-run"}
	ordered.Ordering = epistemic.Ordering{Sequence: 1, Partition: "ordered-run"}
	request[map[string]any](t, h, http.MethodPost, "/v1/events", ordered, http.StatusAccepted)
	collision := protocolEvent("evt-collision", "claim.updated", "claim", "claim-2", map[string]any{"statement": "collision"})
	collision.Context, collision.Ordering = ordered.Context, ordered.Ordering
	request[epistemic.Error](t, h, http.MethodPost, "/v1/events", collision, http.StatusBadRequest)
	unsupported := protocolEvent("evt-unsupported", "claim.updated", "claim", "claim-3", map[string]any{})
	unsupported.SpecVersion = "9.0"
	versionError := request[epistemic.Error](t, h, http.MethodPost, "/v1/events", unsupported, http.StatusBadRequest)
	if versionError.Code != "unsupported_version" {
		t.Fatalf("unexpected version error: %+v", versionError)
	}
}

func TestProtocolEvaluationReturnsPortableBlockAndProof(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	decisionID := "portable-decision-1"
	events := []epistemic.Event{
		protocolEvent("evt-build", "evidence.discovered", "evidence", "build", map[string]any{"kind": "build", "status": "passed"}),
		protocolEvent("evt-unit", "evidence.discovered", "evidence", "unit", map[string]any{"kind": "unit_test", "status": "passed"}),
		protocolEvent("evt-compat", "verification.failed", "verification", "compatibility", map[string]any{"kind": "migration compatibility test", "status": "failed"}),
		protocolEvent("evt-pii", "contradiction.detected", "contradiction", "privacy", map[string]any{"kind": "code diff", "line": "customer_email=alice@example.com"}),
	}
	for index := range events {
		events[index].Context = epistemic.Context{DecisionID: decisionID, RunID: "portable-run-1", Correlation: "correlation-1"}
		events[index].Ordering = epistemic.Ordering{Sequence: int64(index + 1), Partition: "portable-run-1"}
	}
	result := request[epistemic.DecisionResult](t, h, http.MethodPost, "/v1/decisions:evaluate", epistemic.DecisionRequest{SpecVersion: epistemic.Version, DecisionID: decisionID, Recommendation: "The change is safe to deploy.", Action: epistemic.Action{Type: "software_deployment", Subject: epistemic.Subject{Type: "repository", ID: "orders"}, RiskLevel: "high"}, Context: epistemic.Context{DecisionID: decisionID, RunID: "portable-run-1", Correlation: "correlation-1"}, Mode: "enforce", Events: events, Approval: epistemic.Approval{Approved: true, Actor: "test-reviewer"}}, http.StatusOK)
	if result.Status != "block" || result.ActionAllowed || result.Certificate == nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	certificate := request[epistemic.Certificate](t, h, http.MethodGet, "/v1/decisions/"+decisionID+"/certificate", nil, http.StatusOK)
	expected := certificate.Proof.Digest
	certificate.Proof.Digest = ""
	digest, err := epistemic.Hash(certificate)
	if err != nil || digest != expected {
		t.Fatalf("portable proof is not reproducible: got=%s want=%s err=%v", digest, expected, err)
	}
	history := request[struct {
		Events []epistemic.Event `json:"events"`
	}](t, h, http.MethodGet, "/v1/decisions/"+decisionID+"/events", nil, http.StatusOK)
	if len(history.Events) < len(events)+3 {
		t.Fatalf("expected imported and generated lifecycle events, got %d", len(history.Events))
	}
	stored := request[epistemic.DecisionResult](t, h, http.MethodGet, "/v1/decisions/"+decisionID, nil, http.StatusOK)
	if stored.Status != result.Status || stored.Context.Correlation != "correlation-1" {
		t.Fatalf("stored portable result changed: %+v", stored)
	}
}

func protocolEvent(id, eventType, subjectType, subjectID string, data any) epistemic.Event {
	payload, _ := json.Marshal(data)
	return epistemic.Event{SpecVersion: epistemic.Version, ID: id, Type: eventType, Source: epistemic.Source{Name: "protocol-test", Version: "0.1"}, Subject: epistemic.Subject{Type: subjectType, ID: subjectID}, Time: time.Now().UTC(), Data: payload}
}
