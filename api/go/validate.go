package epistemic

import (
	"errors"
	"fmt"
	"strings"
)

var EventTypes = []string{
	"decision.started", "decision.requested", "decision.evaluated", "decision.blocked", "decision.approved", "decision.completed",
	"claim.declared", "claim.updated", "claim.supported", "claim.contradicted", "claim.superseded", "claim.rejected",
	"evidence.discovered", "evidence.attached", "evidence.expired", "evidence.invalidated",
	"assumption.declared", "assumption.resolved", "unknown.declared", "unknown.resolved",
	"contradiction.detected", "contradiction.resolved",
	"verification.requested", "verification.approved", "verification.started", "verification.completed", "verification.failed",
	"proof.issued", "proof.revoked", "proof.superseded",
}

var eventTypeSet = func() map[string]bool {
	values := map[string]bool{}
	for _, value := range EventTypes {
		values[value] = true
	}
	return values
}()

func ValidateEvent(event Event) error {
	if event.SpecVersion != Version {
		return fmt.Errorf("unsupported spec_version %q", event.SpecVersion)
	}
	if strings.TrimSpace(event.ID) == "" {
		return errors.New("event id is required")
	}
	if !eventTypeSet[event.Type] {
		return fmt.Errorf("unsupported event type %q", event.Type)
	}
	if strings.TrimSpace(event.Source.Name) == "" {
		return errors.New("event source.name is required")
	}
	if event.Subject.Type == "" || event.Subject.ID == "" {
		return errors.New("event subject type and id are required")
	}
	if event.Time.IsZero() {
		return errors.New("event time is required")
	}
	if len(event.Data) == 0 {
		return errors.New("event data is required")
	}
	if event.Ordering.Sequence < 0 {
		return errors.New("event ordering.sequence cannot be negative")
	}
	return nil
}

func ValidateDecisionRequest(request DecisionRequest) error {
	if request.SpecVersion != Version {
		return fmt.Errorf("unsupported spec_version %q", request.SpecVersion)
	}
	if strings.TrimSpace(request.Recommendation) == "" {
		return errors.New("recommendation is required")
	}
	if request.Action.Type == "" || request.Action.Subject.Type == "" || request.Action.Subject.ID == "" {
		return errors.New("action type and subject are required")
	}
	if request.Mode != "" && request.Mode != "observe" && request.Mode != "advise" && request.Mode != "enforce" {
		return fmt.Errorf("unsupported mode %q", request.Mode)
	}
	for index, event := range request.Events {
		if err := ValidateEvent(event); err != nil {
			return fmt.Errorf("events[%d]: %w", index, err)
		}
	}
	return nil
}
