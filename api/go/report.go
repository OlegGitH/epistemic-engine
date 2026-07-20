package epistemic

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// HumanReport renders a portable certificate for reviewers. The returned
// Markdown is derived from the certificate and is intentionally outside the
// signed certificate payload, so presentation changes never alter its digest.
func HumanReport(certificate Certificate) (string, error) {
	var result DecisionResult
	if err := json.Unmarshal(certificate.Result, &result); err != nil {
		return "", fmt.Errorf("decode certificate result: %w", err)
	}
	decision, headline, summary := portableDecisionSummary(result)
	var output strings.Builder
	fmt.Fprintln(&output, "# Epistemic Decision Report")
	fmt.Fprintf(&output, "\n## %s — %s\n\n%s\n", decision, headline, summary)
	fmt.Fprintf(&output, "\n- **Status:** %s\n- **Action allowed:** %t\n", result.Status, result.ActionAllowed)
	fmt.Fprintf(&output, "- **Decision ID:** `%s`\n- **Run ID:** `%s`\n- **Issued:** %s\n", certificate.DecisionID, result.Context.RunID, certificate.IssuedAt.Format(time.RFC3339))
	if len(result.Reasons) > 0 {
		fmt.Fprintln(&output, "\n## Reasons")
		for _, reason := range result.Reasons {
			fmt.Fprintf(&output, "\n- **%s:** %s", reason.Code, reason.Message)
		}
		fmt.Fprintln(&output)
	}
	if len(result.Conditions) > 0 {
		fmt.Fprintln(&output, "\n## Conditions")
		for _, condition := range result.Conditions {
			fmt.Fprintf(&output, "\n- %s", condition)
		}
		fmt.Fprintln(&output)
	}
	fmt.Fprintf(&output, "\n## Integrity proof\n\n- Algorithm: %s\n- Certificate digest: `%s`\n- Content-addressed artifacts: %d\n", certificate.Proof.Algorithm, certificate.Proof.Digest, len(certificate.ArtifactHashes))
	fmt.Fprintln(&output, "\nThis report is a human-readable view of the immutable machine certificate. The certificate JSON and its digest remain the source of truth.")
	return output.String(), nil
}

func portableDecisionSummary(result DecisionResult) (string, string, string) {
	if result.Status == "allow" && result.ActionAllowed {
		return "PROCEED", "The proposed action is authorized.", "The portable evidence gate passed and the required authorization is present."
	}
	if result.Status == "allow" {
		return "DO NOT PROCEED", "Evidence passed, but authorization is incomplete.", "The evidence gate passed, but a required approval or condition still blocks consequential action."
	}
	if result.Status == "indeterminate" {
		return "DO NOT PROCEED", "The decision is indeterminate.", "The Engine could not establish enough reliable evidence to authorize the proposed action."
	}
	return "DO NOT PROCEED", "The proposed action is blocked.", "One or more evidence or policy gates failed. Review the reasons before attempting the action again."
}
