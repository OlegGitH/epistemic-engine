package composite

import (
	"context"
	"errors"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Provider struct{ Providers []epistemic.Provider }

func New(providers ...epistemic.Provider) *Provider { return &Provider{Providers: providers} }

func (p *Provider) Emit(ctx context.Context, event epistemic.Event) error {
	var failures []error
	for _, provider := range p.Providers {
		if err := provider.Emit(ctx, event); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (p *Provider) Evaluate(ctx context.Context, request epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	var selected epistemic.DecisionResult
	var failures []error
	for _, provider := range p.Providers {
		result, err := provider.Evaluate(ctx, request)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if selected.SpecVersion == "" || (selected.Status == "indeterminate" && result.Status != "indeterminate") {
			selected = result
		}
	}
	if selected.SpecVersion == "" {
		return selected, errors.Join(failures...)
	}
	return selected, errors.Join(failures...)
}

func (p *Provider) Flush(ctx context.Context) error {
	var values []error
	for _, provider := range p.Providers {
		values = append(values, provider.Flush(ctx))
	}
	return errors.Join(values...)
}
func (p *Provider) Shutdown(ctx context.Context) error {
	var values []error
	for _, provider := range p.Providers {
		values = append(values, provider.Shutdown(ctx))
	}
	return errors.Join(values...)
}

var _ epistemic.Provider = (*Provider)(nil)
