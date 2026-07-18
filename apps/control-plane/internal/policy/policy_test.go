package policy

import (
	"testing"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func TestCriticalContradictionBlocks(t *testing.T) {
	g := domain.Graph{Claims: []domain.Claim{{Critical: true, State: domain.ClaimContradicted}}}
	got := Evaluate(g, true)
	if got.Verdict != domain.VerdictContradicted || got.ActionAllowed {
		t.Fatalf("got %+v", got)
	}
}
func TestCriticalMissingEvidenceBlocks(t *testing.T) {
	g := domain.Graph{Claims: []domain.Claim{{Critical: true, State: domain.ClaimVerificationPending}}}
	got := Evaluate(g, true)
	if got.Verdict != domain.VerdictInsufficientEvidence {
		t.Fatalf("got %+v", got)
	}
}
func TestVerifiedStillNeedsHumanApproval(t *testing.T) {
	g := domain.Graph{Claims: []domain.Claim{{Critical: true, State: domain.ClaimExternallyVerified}}}
	without := Evaluate(g, false)
	with := Evaluate(g, true)
	if without.Verdict != domain.VerdictVerifiedWithConditions || without.ActionAllowed || !with.ActionAllowed {
		t.Fatalf("without=%+v with=%+v", without, with)
	}
}

func TestUnanalyzedOrNonCriticalDecisionNeverPasses(t *testing.T) {
	for _, graph := range []domain.Graph{
		{},
		{Claims: []domain.Claim{{Critical: false, State: domain.ClaimSupported}}},
	} {
		got := Evaluate(graph, true)
		if got.Verdict != domain.VerdictInsufficientEvidence || got.ActionAllowed {
			t.Fatalf("got %+v", got)
		}
	}
}
