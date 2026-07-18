package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Provider struct {
	BaseURL string
	Client  *http.Client
	Headers map[string]string
}

func New(baseURL string) *Provider {
	return &Provider{BaseURL: strings.TrimRight(baseURL, "/"), Client: &http.Client{Timeout: 30 * time.Second}, Headers: map[string]string{}}
}

func (p *Provider) Emit(ctx context.Context, event epistemic.Event) error {
	return p.post(ctx, "/v1/events", event, nil)
}

func (p *Provider) Evaluate(ctx context.Context, request epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	var result epistemic.DecisionResult
	err := p.post(ctx, "/v1/decisions:evaluate", request, &result)
	return result, err
}

func (p *Provider) Flush(context.Context) error    { return nil }
func (p *Provider) Shutdown(context.Context) error { return nil }

func (p *Provider) post(ctx context.Context, path string, value, target any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if protocolContext := contextHeader(value); protocolContext != "" {
		request.Header.Set("Epistemic-Context", protocolContext)
	}
	for key, header := range p.Headers {
		request.Header.Set(key, header)
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var protocolError epistemic.Error
		if json.Unmarshal(data, &protocolError) == nil && protocolError.Code != "" {
			return fmt.Errorf("%s: %s", protocolError.Code, protocolError.Message)
		}
		return fmt.Errorf("epistemic server returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	if target != nil {
		return json.Unmarshal(data, target)
	}
	return nil
}

func contextHeader(value any) string {
	var c epistemic.Context
	switch typed := value.(type) {
	case epistemic.Event:
		c = typed.Context
	case epistemic.DecisionRequest:
		c = typed.Context
	default:
		return ""
	}
	parts := make([]string, 0, 4)
	for _, pair := range [][2]string{{"decision", c.DecisionID}, {"run", c.RunID}, {"correlation", c.Correlation}, {"parent", c.ParentID}} {
		if pair[1] != "" {
			parts = append(parts, pair[0]+"="+pair[1])
		}
	}
	return strings.Join(parts, ";")
}

var _ epistemic.Provider = (*Provider)(nil)
