package epistemic

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHumanReportExplainsAuthorizationAndProof(t *testing.T) {
	result := DecisionResult{SpecVersion: Version, DecisionID: "decision-report", Status: "allow", ActionAllowed: false, Conditions: []string{"Human approval is required."}, Reasons: []Reason{}, Context: Context{RunID: "run-report"}}
	resultJSON, _ := json.Marshal(result)
	certificate := Certificate{SpecVersion: Version, ID: "proof-report", DecisionID: result.DecisionID, Result: resultJSON, ArtifactHashes: []string{"artifact"}, IssuedAt: time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC), Proof: Proof{Algorithm: "SHA-256", Digest: strings.Repeat("a", 64)}}
	report, err := HumanReport(certificate)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"DO NOT PROCEED", "authorization is incomplete", "Human approval is required", certificate.Proof.Digest, "run-report"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("report does not contain %q:\n%s", expected, report)
		}
	}
}
