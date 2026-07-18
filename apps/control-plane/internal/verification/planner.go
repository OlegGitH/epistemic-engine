package verification

import (
	"encoding/json"
	"strings"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func SpecificationForClaim(claim domain.Claim) (string, json.RawMessage) {
	lower := strings.ToLower(claim.Statement)
	file := ""
	switch {
	case strings.Contains(lower, "backward-compatible") || strings.Contains(lower, "compatibility"):
		file = "test_compatibility.py"
	case strings.Contains(lower, "pii") || strings.Contains(lower, "personal") || strings.Contains(lower, "email"):
		file = "test_pii_logging.py"
	}
	if file == "" {
		specification, _ := json.Marshal(map[string]any{
			"instructions":            "A human reviewer must supply evidence for this bounded claim.",
			"required_evidence_types": claim.RequiredEvidenceTypes,
		})
		return "human", specification
	}
	specification, _ := json.Marshal(Specification{
		Image: "python:3.12-alpine", Repository: "unsafe-orders-pr",
		Command:        []string{"python", "-m", "unittest", "discover", "-s", "tests", "-p", file, "-v"},
		TimeoutSeconds: 45,
	})
	return "test", specification
}
