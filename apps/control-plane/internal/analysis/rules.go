package analysis

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

type RulesAnalyzer struct{}

func NewRulesAnalyzer() *RulesAnalyzer { return &RulesAnalyzer{} }

var emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func (a *RulesAnalyzer) Analyze(_ context.Context, run domain.Run, decision domain.Decision) (domain.Analysis, error) {
	evidence := evidenceFromEvents(run)
	byKind := map[string][]domain.Evidence{}
	for _, item := range evidence {
		byKind[item.Kind] = append(byKind[item.Kind], item)
	}

	var buildPass, buildFail, unitPass, unitFail, compatibilityPass, compatibilityFail, piiPass, piiLeak, rollbackReady bool
	for _, event := range run.Events {
		value := strings.ToLower(event.Type + " " + string(event.Payload))
		passed := strings.Contains(value, "pass") || strings.Contains(value, `"status":"success`) || strings.Contains(value, `"status": "success`)
		failed := strings.Contains(value, "fail") || strings.Contains(value, "error")
		isCompatibility := strings.Contains(value, "compatib") || strings.Contains(value, "legacy") || strings.Contains(value, "migration")
		isPII := strings.Contains(value, "pii") || strings.Contains(value, "privacy") || strings.Contains(value, "email")
		isTest := strings.Contains(value, "test")
		if strings.Contains(value, "build") {
			buildPass = buildPass || passed
			buildFail = buildFail || failed
		}
		if isTest && !isCompatibility && !isPII {
			unitPass = unitPass || passed
			unitFail = unitFail || failed
		}
		if isCompatibility {
			compatibilityPass = compatibilityPass || passed
			compatibilityFail = compatibilityFail || failed
		}
		if isPII && isTest {
			piiPass = piiPass || passed
		}
		if (strings.Contains(value, "log") || strings.Contains(value, "diff") || strings.Contains(value, "patch")) && (emailPattern.MatchString(value) || strings.Contains(value, "customer.email") || strings.Contains(value, "user.email")) {
			piiLeak = true
		}
		if strings.Contains(value, "rollback") && (passed || strings.Contains(value, "ready")) {
			rollbackReady = true
		}
	}

	claims := []domain.Claim{
		claim("claim_build", "The proposed revision builds successfully.", "proposed revision", true, []string{"build_result"}, observedState(buildPass, buildFail), "Build observations must match the exact proposed revision."),
		claim("claim_tests", "The relevant automated test suite passes for the proposed revision.", "unit and integration tests", true, []string{"test_result"}, observedState(unitPass, unitFail), "Test results are scoped to the revision and suite reported by CI."),
		claim("claim_compatibility", "The change is backward-compatible with deployed interfaces and data.", "public interfaces, stored data, and legacy clients", true, []string{"migration", "test_result"}, observedState(compatibilityPass, compatibilityFail), "Compatibility requires a migration or legacy-client check, not only unit tests."),
		claim("claim_privacy", "The change does not expose email addresses or other PII in logs.", "application and audit logs", true, []string{"code_diff", "log_output", "test_result"}, privacyState(piiPass, piiLeak), "Logging personal data contradicts the deployment privacy requirement."),
		claim("claim_rollback", "Rollback and operational signals are sufficient to recover safely.", "staging and production operations", false, []string{"trace_output", "custom"}, observedState(rollbackReady, false), "Operational recovery requires an explicit rollback or staging observation."),
	}

	bindings := map[string][]domain.Evidence{
		"claim_build":         byKind["build_result"],
		"claim_tests":         byKind["test_result"],
		"claim_compatibility": append(append([]domain.Evidence{}, byKind["migration"]...), filterEvidence(evidence, "compatib", "legacy", "migration")...),
		"claim_privacy":       filterEvidence(evidence, "pii", "privacy", "email", "log", "diff"),
		"claim_rollback":      filterEvidence(evidence, "rollback"),
	}

	result := domain.Analysis{Claims: claims, Evidence: evidence}
	for claimIndex := range result.Claims {
		bound := dedupeEvidence(bindings[result.Claims[claimIndex].ID])
		result.Claims[claimIndex].Support = supportFor(bound, result.Claims[claimIndex].State)
		for _, item := range bound {
			result.Claims[claimIndex].EvidenceIDs = append(result.Claims[claimIndex].EvidenceIDs, item.ID)
			relation := domain.RelationSupports
			if result.Claims[claimIndex].State == domain.ClaimContradicted {
				relation = domain.RelationContradicts
			}
			result.Relations = append(result.Relations, domain.Relation{FromID: item.ID, ToID: result.Claims[claimIndex].ID, Type: relation, Rationale: result.Claims[claimIndex].Justification})
		}
		if result.Claims[claimIndex].Critical && (result.Claims[claimIndex].State == domain.ClaimUnsupported || result.Claims[claimIndex].State == domain.ClaimVerificationPending) {
			result.Assumptions = append(result.Assumptions, domain.Assumption{ClaimID: result.Claims[claimIndex].ID, Statement: "Required evidence is assumed but not present: " + strings.Join(result.Claims[claimIndex].RequiredEvidenceTypes, ", "), Critical: true, Status: "open", Impact: "blocks_deployment"})
			result.Unknowns = append(result.Unknowns, domain.Unknown{Question: "What direct evidence verifies: " + result.Claims[claimIndex].Statement, Critical: true})
		}
	}
	return result, nil
}

func claim(id, statement, scope string, critical bool, required []string, state domain.ClaimState, justification string) domain.Claim {
	importance := "supporting"
	if critical {
		importance = "critical"
	}
	return domain.Claim{ID: id, Statement: statement, Scope: scope, Critical: critical, Importance: importance, RequiredEvidenceTypes: required, State: state, Justification: justification}
}

func observedState(passed, failed bool) domain.ClaimState {
	if failed {
		return domain.ClaimContradicted
	}
	if passed {
		return domain.ClaimSupported
	}
	return domain.ClaimVerificationPending
}

func privacyState(passed, leaked bool) domain.ClaimState {
	if leaked {
		return domain.ClaimContradicted
	}
	if passed {
		return domain.ClaimSupported
	}
	return domain.ClaimVerificationPending
}

func filterEvidence(items []domain.Evidence, terms ...string) []domain.Evidence {
	var result []domain.Evidence
	for _, item := range items {
		value := strings.ToLower(item.Kind + " " + item.Summary + " " + string(item.Raw))
		for _, term := range terms {
			if strings.Contains(value, term) {
				result = append(result, item)
				break
			}
		}
	}
	return result
}

func dedupeEvidence(items []domain.Evidence) []domain.Evidence {
	seen := map[string]bool{}
	result := make([]domain.Evidence, 0, len(items))
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			result = append(result, item)
		}
	}
	return result
}

func supportFor(items []domain.Evidence, state domain.ClaimState) domain.SupportScore {
	coverage := 0.0
	if len(items) > 0 {
		coverage = 1
	}
	sources := map[string]bool{}
	quality, totalFresh := 0.0, 0.0
	for _, item := range items {
		sources[item.Source] = true
		switch item.Kind {
		case "test_result", "build_result", "migration":
			quality += 1
		case "code_diff", "log_output":
			quality += .8
		default:
			quality += .6
		}
		if time.Since(item.ObservedAt) < 30*24*time.Hour {
			totalFresh++
		}
	}
	if len(items) > 0 {
		quality /= float64(len(items))
		totalFresh /= float64(len(items))
	}
	independence := 0.0
	if len(items) > 0 {
		independence = float64(len(sources)) / float64(len(items))
		if independence > 1 {
			independence = 1
		}
	}
	direct := 0.0
	if state == domain.ClaimSupported {
		direct = .8
	}
	if state == domain.ClaimExternallyVerified {
		direct = 1
	}
	burden := 0.0
	if state == domain.ClaimContradicted {
		burden = 1
	}
	value := (.25*coverage + .15*quality + .10*independence + .10*totalFresh + .15*coverage + .25*direct) * (1 - burden)
	return domain.SupportScore{Value: value, EvidenceCoverage: coverage, SourceQuality: quality, SourceIndependence: independence, Freshness: totalFresh, ScopeMatch: coverage, DirectVerificationStrength: direct, ContradictionBurden: burden, Semantics: "explainable evidence support; not probability of truth"}
}
