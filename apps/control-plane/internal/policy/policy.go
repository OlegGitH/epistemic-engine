package policy

import "github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"

const Version = "deployment-readiness/v1"

type Result struct {
	Verdict       domain.Verdict
	Conditions    []string
	ActionAllowed bool
}

func Evaluate(graph domain.Graph, humanApproved bool) Result {
	if len(graph.Claims) == 0 {
		return Result{Verdict: domain.VerdictInsufficientEvidence}
	}
	hasCriticalClaim := false
	for _, claim := range graph.Claims {
		hasCriticalClaim = hasCriticalClaim || claim.Critical
		if claim.State == domain.ClaimRejected {
			return Result{Verdict: domain.VerdictRejected}
		}
	}
	if !hasCriticalClaim {
		return Result{Verdict: domain.VerdictInsufficientEvidence}
	}
	for _, claim := range graph.Claims {
		if claim.State == domain.ClaimContradicted {
			return Result{Verdict: domain.VerdictContradicted}
		}
	}

	conditions := make([]string, 0)
	hasExternalVerification := false
	for _, claim := range graph.Claims {
		if !claim.Critical {
			continue
		}
		switch claim.State {
		case domain.ClaimSupported:
			// Passing state.
		case domain.ClaimExternallyVerified:
			hasExternalVerification = true
		case domain.ClaimPartiallySupported:
			conditions = append(conditions, "Resolve partial support: "+claim.Statement)
		default:
			return Result{Verdict: domain.VerdictInsufficientEvidence}
		}
	}
	for _, unknown := range graph.Unknowns {
		if unknown.Critical && !unknown.Resolved {
			return Result{Verdict: domain.VerdictInsufficientEvidence}
		}
	}

	if hasExternalVerification {
		conditions = append(conditions, "Retain verification artifacts and monitor the first staged rollout.")
	}
	verdict := domain.VerdictVerified
	if len(conditions) > 0 {
		verdict = domain.VerdictVerifiedWithConditions
	}
	return Result{Verdict: verdict, Conditions: conditions, ActionAllowed: humanApproved}
}
