package local

import (
	"context"
	"sync"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Provider struct {
	mu     sync.Mutex
	events map[string]epistemic.Event
}

func New() *Provider { return &Provider{events: map[string]epistemic.Event{}} }

func (p *Provider) Emit(_ context.Context, event epistemic.Event) error {
	if err := epistemic.ValidateEvent(event); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.events[event.ID]; !exists {
		p.events[event.ID] = event
	}
	return nil
}

func (p *Provider) Evaluate(_ context.Context, request epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	if err := epistemic.ValidateDecisionRequest(request); err != nil {
		return epistemic.DecisionResult{}, err
	}
	p.mu.Lock()
	events := make([]epistemic.Event, 0, len(request.Events)+len(p.events))
	seen := map[string]bool{}
	for _, event := range request.Events {
		if !seen[event.ID] {
			events = append(events, event)
			seen[event.ID] = true
		}
	}
	for _, event := range p.events {
		if !seen[event.ID] && (request.DecisionID == "" || event.Context.DecisionID == request.DecisionID) {
			events = append(events, event)
			seen[event.ID] = true
		}
	}
	p.mu.Unlock()
	id := request.DecisionID
	if id == "" {
		id = "local-decision"
	}
	result := epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: id, Status: "allow", ActionAllowed: request.Approval.Approved, Reasons: []epistemic.Reason{}, Conditions: []string{}, EvaluatedAt: time.Now().UTC(), Context: request.Context}
	pending := false
	for _, event := range events {
		switch event.Type {
		case "claim.contradicted", "claim.rejected", "contradiction.detected", "verification.failed":
			result.Status, result.ActionAllowed = "block", false
			result.Reasons = append(result.Reasons, epistemic.Reason{Code: "contradicted", Message: "A blocking contradiction or failed verification is unresolved.", SubjectID: event.Subject.ID})
		case "verification.requested", "claim.declared", "unknown.declared":
			pending = true
		case "verification.completed", "claim.supported", "unknown.resolved":
			// These portable signals may satisfy a corresponding requirement.
		}
	}
	if result.Status != "block" && pending {
		result.Status, result.ActionAllowed = "indeterminate", false
		result.Reasons = append(result.Reasons, epistemic.Reason{Code: "verification_required", Message: "A critical portable requirement remains unresolved."})
	}
	if result.Status == "allow" && !request.Approval.Approved {
		result.ActionAllowed = false
		result.Conditions = append(result.Conditions, "Human approval is required before consequential action.")
	}
	return result, nil
}

func (p *Provider) Flush(context.Context) error    { return nil }
func (p *Provider) Shutdown(context.Context) error { return nil }

var _ epistemic.Provider = (*Provider)(nil)
