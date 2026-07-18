package epistemic

import "context"

type Provider interface {
	Emit(context.Context, Event) error
	Evaluate(context.Context, DecisionRequest) (DecisionResult, error)
	Flush(context.Context) error
	Shutdown(context.Context) error
}

type Client struct{ provider Provider }

func NewClient(provider Provider) *Client { return &Client{provider: provider} }

func (c *Client) Emit(ctx context.Context, event Event) error {
	if err := ValidateEvent(event); err != nil {
		return err
	}
	return c.provider.Emit(ctx, event)
}

func (c *Client) Evaluate(ctx context.Context, request DecisionRequest) (DecisionResult, error) {
	if err := ValidateDecisionRequest(request); err != nil {
		return DecisionResult{}, err
	}
	return c.provider.Evaluate(ctx, request)
}

func (c *Client) Flush(ctx context.Context) error    { return c.provider.Flush(ctx) }
func (c *Client) Shutdown(ctx context.Context) error { return c.provider.Shutdown(ctx) }
