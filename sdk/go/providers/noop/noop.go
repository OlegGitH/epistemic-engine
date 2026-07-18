package noop

import (
	"context"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Provider struct{}

func New() Provider                                          { return Provider{} }
func (Provider) Emit(context.Context, epistemic.Event) error { return nil }
func (Provider) Evaluate(_ context.Context, request epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	id := request.DecisionID
	if id == "" {
		id = "noop"
	}
	return epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: id, Status: "indeterminate", Reasons: []epistemic.Reason{{Code: "provider_disabled", Message: "Epistemic evaluation is disabled."}}, Conditions: []string{}, EvaluatedAt: time.Now().UTC(), Context: request.Context}, nil
}
func (Provider) Flush(context.Context) error    { return nil }
func (Provider) Shutdown(context.Context) error { return nil }

var _ epistemic.Provider = Provider{}
