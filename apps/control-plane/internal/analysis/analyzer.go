package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

// Analyzer proposes observable claims and explicit justifications. Implementations
// must never represent model output as hidden chain-of-thought or ground truth.
type Analyzer interface {
	Analyze(context.Context, domain.Run, domain.Decision) (domain.Analysis, error)
}

func evidenceFromEvents(run domain.Run) []domain.Evidence {
	evidence := make([]domain.Evidence, 0, len(run.Events))
	for _, event := range run.Events {
		sum := sha256.Sum256(event.Payload)
		identity := sha256.Sum256([]byte(event.ID + hex.EncodeToString(sum[:])))
		kind := evidenceKind(event.Type, event.Payload)
		evidence = append(evidence, domain.Evidence{
			ID: "evidence_" + hex.EncodeToString(identity[:8]), RunID: run.ID, Kind: kind, Source: event.Source,
			Summary:     fmt.Sprintf("Imported %s event from %s", event.Type, event.Source),
			ContentHash: hex.EncodeToString(sum[:]), ObservedAt: event.OccurredAt, Raw: event.Payload,
		})
	}
	return evidence
}

func evidenceKind(eventType string, payload []byte) string {
	value := strings.ToLower(eventType + " " + string(payload))
	switch {
	case strings.Contains(value, "migration") || strings.Contains(value, "schema"):
		return "migration"
	case strings.Contains(value, "diff") || strings.Contains(value, "patch"):
		return "code_diff"
	case strings.Contains(value, "test"):
		return "test_result"
	case strings.Contains(value, "log"):
		return "log_output"
	case strings.Contains(value, "build"):
		return "build_result"
	case strings.Contains(value, "model") || strings.Contains(value, "tool") || strings.Contains(value, "guardrail") || strings.Contains(value, "trace"):
		return "trace_output"
	default:
		return "custom"
	}
}
